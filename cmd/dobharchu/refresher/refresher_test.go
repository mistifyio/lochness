package refresher_test

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"regexp"
	"strings"
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/dobharchu/refresher"
	"github.com/mistifyio/lochness/cmd/dobharchu/testhelper"
)

type (
	TestingData struct {
		hypervisors map[string]*TestingHypervisorData
		guests      map[string]*TestingGuestData
	}
	TestingHypervisorData struct {
		mac     string
		ip      string
		gateway string
		netmask string
	}
	TestingGuestData struct {
		mac     string
		ip      string
		gateway string
		cidr    string
	}
)

func doTestSetup(context *lochness.Context, etcdClient *etcd.Client) (*TestingData, error) {
	data := &TestingData{
		hypervisors: make(map[string]*TestingHypervisorData),
		guests:      make(map[string]*TestingGuestData),
	}

	// Clear out the lochness keys
	_, err := etcdClient.Delete("lochness/", true)
	if err != nil {
		return nil, errors.New("Could not clear data from etcd: " + err.Error())
	}

	// Add flavors, network, and firewall group
	f1, err := testhelper.NewTestFlavor(context, 4, 4096, 8192)
	if err != nil {
		return nil, err
	}
	f2, err := testhelper.NewTestFlavor(context, 6, 8192, 1024)
	if err != nil {
		return nil, err
	}
	n, err := testhelper.NewTestNetwork(context)
	if err != nil {
		return nil, err
	}
	fw, err := testhelper.NewTestFirewallGroup(context)
	if err != nil {
		return nil, err
	}

	// Add subnet
	s, err := testhelper.NewTestSubnet(context, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	if err != nil {
		return nil, err
	}

	// Add hypervisors
	h1, err := testhelper.NewTestHypervisor(context, "de:ad:be:ef:7f:21", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	if err != nil {
		return nil, err
	}
	data.hypervisors[h1.ID] = &TestingHypervisorData{"DE:AD:BE:EF:7F:21", "192.168.100.200", "192.168.100.1", "255.255.255.0"}
	h2, err := testhelper.NewTestHypervisor(context, "de:ad:be:ef:7f:23", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	if err != nil {
		return nil, err
	}
	data.hypervisors[h2.ID] = &TestingHypervisorData{"DE:AD:BE:EF:7F:23", "192.168.100.203", "192.168.100.1", "255.255.255.0"}

	// Add guests
	g1, err := testhelper.NewTestGuest(context, "01:23:45:67:89:ab", n, s, f1, fw, h1)
	if err != nil {
		return nil, err
	}
	data.guests[g1.ID] = &TestingGuestData{"01:23:45:67:89:AB", g1.IP.String(), "10.10.10.1", "10.10.10.0"}
	g2, err := testhelper.NewTestGuest(context, "23:45:67:89:ab:cd", n, s, f2, fw, h1)
	if err != nil {
		return nil, err
	}
	data.guests[g2.ID] = &TestingGuestData{"23:45:67:89:AB:CD", g2.IP.String(), "10.10.10.1", "10.10.10.0"}
	g3, err := testhelper.NewTestGuest(context, "45:67:89:ab:cd:ef", n, s, f1, fw, h2)
	if err != nil {
		return nil, err
	}
	data.guests[g3.ID] = &TestingGuestData{"45:67:89:AB:CD:EF", g3.IP.String(), "10.10.10.1", "10.10.10.0"}
	g4, err := testhelper.NewTestGuest(context, "67:89:ab:cd:ef:01", n, s, f2, fw, h2)
	if err != nil {
		return nil, err
	}
	data.guests[g4.ID] = &TestingGuestData{"67:89:AB:CD:EF:01", g4.IP.String(), "10.10.10.1", "10.10.10.0"}

	return data, nil
}

func getLines(t *testing.T, b *bytes.Buffer) []string {
	s := bufio.NewScanner(b)
	re := regexp.MustCompile("\\s+")
	var lines []string
	for s.Scan() {
		line := s.Text()
		if line != "" {
			cleaned := re.ReplaceAllString(strings.TrimSpace(line), " ")
			lines = append(lines, cleaned)
		}
	}
	h.Ok(t, s.Err())
	return lines
}

func TestHypervisorsConf(t *testing.T) {

	// Setup
	r := refresher.NewRefresher("example.com", "http://127.0.0.1:4001")
	h.Equals(t, r.Domain, "example.com")
	testData, err := doTestSetup(r.Context, r.EtcdClient)
	h.Ok(t, err)

	// Write to buffer
	b := new(bytes.Buffer)
	err = r.WriteHypervisorsConfigFile(b)
	h.Ok(t, err)

	// Define tests and regexes
	fkeys := []string{
		"group start",
		"domain",
		"if iPXE",
		"filename for iPXE",
		"next-server",
		"filename for non-iPXE",
		"first hypervisor start",
		"first hypervisor mac",
		"first hypervisor ip",
		"first hypervisor gateway",
		"first hypervisor netmask",
		"first hypervisor end",
		"second hypervisor start",
		"second hypervisor mac",
		"second hypervisor ip",
		"second hypervisor gateway",
		"second hypervisor netmask",
		"second hypervisor end",
		"group end",
	}
	found := make(map[string]bool)
	for _, k := range fkeys {
		found[k] = false
	}
	groupre := regexp.MustCompile("^group ([0-9a-zA-Z]+) {$")
	hostre := regexp.MustCompile("^host ([0-9a-f\\-]+) {$")
	ifre := regexp.MustCompile("^if .* {$")
	errors := make([]string, 0)

	// Check the returned file line by line
	s := bufio.NewScanner(b)
	spacere := regexp.MustCompile("\\s+")
	ingroup := false
	inhost := false
	inif := false
	hostprefix := ""
	var hostmatch *TestingHypervisorData
	var line string
	for s.Scan() {
		line = s.Text()
		if line == "" {
			continue
		}
		line = spacere.ReplaceAllString(strings.TrimSpace(line), " ")
		t.Log(line)

		// End blocks
		if line == "}" {
			if inif {
				inif = false
				t.Log(" -- CLOSE IF")
			} else if inhost {
				inhost = false
				found[hostprefix+" hypervisor end"] = true
				t.Log(" -- CLOSE " + strings.ToUpper(hostprefix) + " HYPERVISOR")
			} else if ingroup {
				ingroup = false
				found["group end"] = true
				t.Log(" -- CLOSE GROUP")
			}
			continue
		}

		// Start blocks
		if ifre.FindString(line) == line {
			inif = true
			t.Log(" -- OPEN IF")
		}
		m1 := hostre.FindStringSubmatch(line)
		if len(m1) == 2 {
			inhost = true
			if found["first hypervisor start"] {
				found["second hypervisor start"] = true
				hostprefix = "second"
				t.Log(" -- OPEN SECOND HYPERVISOR")
			} else {
				found["first hypervisor start"] = true
				hostprefix = "first"
				t.Log(" -- OPEN FIRST HYPERVISOR")
			}
			id := m1[1]
			if _, ok := testData.hypervisors[id]; !ok {
				errors = append(errors, "Host block not matching a hypervisor")
				hostmatch = nil
			} else {
				hostmatch = testData.hypervisors[id]
			}
			continue
		}
		m2 := groupre.FindStringSubmatch(line)
		if len(m2) == 2 {
			ingroup = true
			if m2[1] != "hypervisors" {
				errors = append(errors, "Group name is not hypervisors")
			} else {
				found["group start"] = true
			}
			t.Log(" -- OPEN GROUP")
			continue
		}

		// Host lines
		if inhost {
			if hostmatch == nil {
				continue
			}
			if line == "hardware ethernet \""+hostmatch.mac+"\";" {
				found[hostprefix+" hypervisor mac"] = true
			} else if line == "fixed-address \""+hostmatch.ip+"\";" {
				found[hostprefix+" hypervisor ip"] = true
			} else if line == "option routers \""+hostmatch.gateway+"\";" {
				found[hostprefix+" hypervisor gateway"] = true
			} else if line == "option subnet-mask \""+hostmatch.netmask+"\";" {
				found[hostprefix+" hypervisor netmask"] = true
			}
			continue
		}

		// Group lines
		if ingroup {
			if line == "option domain-name \"nodes.example.com\";" {
				found["domain"] = true
			}
			if line == "if exists user-class and option user-class = \"iPXE\" {" {
				found["if iPXE"] = true
			}
			if line == "filename \"http://ipxe.services.example.com:8888/ipxe/${net0/ip}\";" {
				found["filename for iPXE"] = true
			}
			if line == "next-server tftp.services.example.com;" {
				found["next-server"] = true
			}
			if line == "filename \"undionly.kpxe\";" {
				found["filename for non-iPXE"] = true
			}
		}
	}
	h.Ok(t, s.Err())

	// Report anything not found or wrong
	for _, key := range fkeys {
		if !found[key] {
			t.Error("Config file missing " + key)
		}
	}
	for _, err := range errors {
		t.Error(err)
	}

}

func TestGuestsConf(t *testing.T) {

	// Setup
	r := refresher.NewRefresher("example.com", "http://127.0.0.1:4001")
	h.Equals(t, r.Domain, "example.com")
	testData, err := doTestSetup(r.Context, r.EtcdClient)
	h.Ok(t, err)

	// Write to buffer
	b := new(bytes.Buffer)
	err = r.WriteGuestsConfigFile(b)
	h.Ok(t, err)

	// Define tests and regexes
	fkeys := []string{
		"group start",
		"domain",
		"first guest start",
		"first guest mac",
		"first guest ip",
		"first guest gateway",
		"first guest cidr",
		"first guest end",
		"second guest start",
		"second guest mac",
		"second guest ip",
		"second guest gateway",
		"second guest cidr",
		"second guest end",
		"third guest start",
		"third guest mac",
		"third guest ip",
		"third guest gateway",
		"third guest cidr",
		"third guest end",
		"fourth guest start",
		"fourth guest mac",
		"fourth guest ip",
		"fourth guest gateway",
		"fourth guest cidr",
		"fourth guest end",
		"group end",
	}
	found := make(map[string]bool)
	for _, k := range fkeys {
		found[k] = false
	}
	groupre := regexp.MustCompile("^group ([0-9a-zA-Z]+) {$")
	hostre := regexp.MustCompile("^host ([0-9a-f\\-]+) {$")
	ifre := regexp.MustCompile("^if .* {$")
	errors := make([]string, 0)

	// Check the returned file line by line
	s := bufio.NewScanner(b)
	spacere := regexp.MustCompile("\\s+")
	ingroup := false
	inhost := false
	inif := false
	hostprefix := ""
	var hostmatch *TestingGuestData
	var line string
	for s.Scan() {
		line = s.Text()
		if line == "" {
			continue
		}
		line = spacere.ReplaceAllString(strings.TrimSpace(line), " ")
		t.Log(line)

		// End blocks
		if line == "}" {
			if inif {
				inif = false
				t.Log(" -- CLOSE IF")
			} else if inhost {
				inhost = false
				found[hostprefix+" guest end"] = true
				t.Log(" -- CLOSE " + strings.ToUpper(hostprefix) + " GUEST")
			} else if ingroup {
				ingroup = false
				found["group end"] = true
				t.Log(" -- CLOSE GROUP")
			}
			continue
		}

		// Start blocks
		if ifre.FindString(line) == line {
			inif = true
			t.Log(" -- OPEN IF")
		}
		m1 := hostre.FindStringSubmatch(line)
		if len(m1) == 2 {
			inhost = true
			if found["third guest start"] {
				found["fourth guest start"] = true
				hostprefix = "fourth"
				t.Log(" -- OPEN FOURTH GUEST")
			} else if found["second guest start"] {
				found["third guest start"] = true
				hostprefix = "third"
				t.Log(" -- OPEN THIRD GUEST")
			} else if found["first guest start"] {
				found["second guest start"] = true
				hostprefix = "second"
				t.Log(" -- OPEN SECOND GUEST")
			} else if found["fourth guest start"] {
				hostprefix = "extra"
				t.Log(" -- OPEN EXTRA GUEST")
			} else {
				found["first guest start"] = true
				hostprefix = "first"
				t.Log(" -- OPEN FIRST GUEST")
			}
			id := m1[1]
			if _, ok := testData.guests[id]; !ok {
				errors = append(errors, "Host block not matching a guest")
				hostmatch = nil
			} else {
				hostmatch = testData.guests[id]
			}
			continue
		}
		m2 := groupre.FindStringSubmatch(line)
		if len(m2) == 2 {
			ingroup = true
			if m2[1] != "guests" {
				errors = append(errors, "Group name is not guests")
			} else {
				found["group start"] = true
			}
			t.Log(" -- OPEN GROUP")
			continue
		}

		// Host lines
		if inhost {
			if hostmatch == nil {
				continue
			}
			if line == "hardware ethernet \""+hostmatch.mac+"\";" {
				found[hostprefix+" guest mac"] = true
			} else if line == "fixed-address \""+hostmatch.ip+"\";" {
				found[hostprefix+" guest ip"] = true
			} else if line == "option routers \""+hostmatch.gateway+"\";" {
				found[hostprefix+" guest gateway"] = true
			} else if line == "option subnet-mask \""+hostmatch.cidr+"\";" {
				found[hostprefix+" guest cidr"] = true
			}
			continue
		}

		// Group lines
		if ingroup {
			if line == "option domain-name \"guests.example.com\";" {
				found["domain"] = true
			}
		}
	}
	h.Ok(t, s.Err())

	// Report anything not found or wrong
	for _, key := range fkeys {
		if !found[key] {
			t.Error("Config file missing " + key)
		}
	}
	for _, err := range errors {
		t.Error(err)
	}

}

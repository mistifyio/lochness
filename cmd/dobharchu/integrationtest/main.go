package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/testhelper"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
)

const etcdClientAddress = "http://localhost:44001"
const etcdPeerAddress = "http://localhost:29001"

var testDir, hconfPath, gconfPath string
var confLastMod = make(map[string]time.Time)
var confLastSize = make(map[string]int64)
var selfLog *os.File
var testOk bool

// testProcess describes a long-running process whose output we're logging
type testProcess struct {
	name   string
	cmd    *exec.Cmd
	writer *bufio.Writer
	file   *os.File
	path   string
}

func newTestProcess(name string, cmd *exec.Cmd) *testProcess {
	p := new(testProcess)
	p.name = name
	p.cmd = cmd
	p.path = testDir + "/" + name + ".log"
	return p
}

func (p *testProcess) captureOutput(stderr bool) error {
	var err error
	p.file, err = os.Create(p.path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
			"path":  p.path,
		}).Error("Could not open log path")
		return err
	}
	p.writer = bufio.NewWriter(p.file)
	if stderr {
		p.cmd.Stderr = p.writer
	} else {
		p.cmd.Stdout = p.writer
	}
	return nil
}

func (p *testProcess) start() error {
	if err := p.cmd.Start(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "exec.Cmd.Start",
			"name":  p.name,
		}).Fatal("Could not start process")
		return err
	}
	return nil
}

func (p *testProcess) finish() error {
	if p.writer != nil {
		if err := p.writer.Flush(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "bufio.Writer.Flush",
				"name":  p.name,
				"path":  p.path,
			}).Error("Could not flush buffer")
			return err
		}
		if err := p.file.Close(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "os.File.Close",
				"name":  p.name,
				"path":  p.path,
			}).Error("Could not close log file")
			return err
		}
	}
	if err := p.cmd.Process.Signal(os.Kill); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "exec.Cmd.Process.Signal",
			"name":  p.name,
		}).Error("Could not kill process")
		return err
	}
	_ = p.cmd.Wait()
	return nil
}

func reportConfStatus(r *bytes.Buffer, testdsc string, hexp string, gexp string) bool {
	var status, ok string
	ret := true
	status = fileStatus(hconfPath)
	if hexp == status {
		ok = "OK"
	} else {
		ok = "FAIL"
		ret = false
	}
	_, _ = r.WriteString("Hypervisors conf status " + testdsc + ": " + status + " [" + ok + "]\n")
	status = fileStatus(gconfPath)
	if gexp == status {
		ok = "OK"
	} else {
		ok = "FAIL"
		ret = false
	}
	_, _ = r.WriteString("Groups conf status " + testdsc + ": " + status + " [" + ok + "]\n")
	return ret
}

func fileStatus(path string) string {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "not present"
		}
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Stat",
			"path":  path,
		}).Error("Could not stat file")
		return "error"
	}

	modtime := fi.ModTime()
	size := fi.Size()
	if _, ok := confLastMod[path]; !ok {
		confLastMod[path] = modtime
		confLastSize[path] = size
		return "created"
	}

	var status string
	if confLastMod[path] == modtime {
		status = "not touched"
	} else if confLastSize[path] == size {
		status = "touched"
	} else {
		status = "changed"
	}
	confLastMod[path] = modtime
	confLastSize[path] = size
	return status
}

func reportHasHosts(r *bytes.Buffer, testdsc string, hs map[string]*lochness.Hypervisor, gs map[string]*lochness.Guest) bool {
	ret := true

	// Hypervisors
	var hostStatus, hostOk, macStatus, macOk string
	hlines, err := getLines(hconfPath)
	if err != nil {
		hostStatus = "error"
		macStatus = "error"
		hostOk = "FAIL"
		macOk = "FAIL"
		ret = false
	} else {
		var expCount = len(hs)
		hostExp := make(map[string]bool, expCount)
		macExp := make(map[string]bool, expCount)
		for id, h := range hs {
			hostExp["host "+id+" {"] = false
			macExp["hardware ethernet "+strings.ToUpper(h.MAC.String())+";"] = false
		}
		hostCount := countLines(hostExp, hlines)
		hostStatus = fmt.Sprintf("%d/%d", hostCount, expCount)
		if hostCount == expCount {
			hostOk = "OK"
		} else {
			hostOk = "FAIL"
			ret = false
		}
		macCount := countLines(macExp, hlines)
		macStatus = fmt.Sprintf("%d/%d", macCount, expCount)
		if macCount == expCount {
			macOk = "OK"
		} else {
			macOk = "FAIL"
			ret = false
		}
	}
	_, _ = r.WriteString("Hypervisor hosts found in conf " + testdsc + ": " + hostStatus + " [" + hostOk + "]\n")
	_, _ = r.WriteString("Hypervisor MACs found in conf " + testdsc + ": " + macStatus + " [" + macOk + "]\n")

	// Groups
	glines, err := getLines(gconfPath)
	if err != nil {
		hostStatus = "error"
		macStatus = "error"
		hostOk = "FAIL"
		macOk = "FAIL"
		ret = false
	} else {
		var expCount = len(gs)
		hostExp := make(map[string]bool, expCount)
		macExp := make(map[string]bool, expCount)
		for id, g := range gs {
			hostExp["host "+id+" {"] = false
			macExp["hardware ethernet "+strings.ToUpper(g.MAC.String())+";"] = false
		}
		hostCount := countLines(hostExp, glines)
		hostStatus = fmt.Sprintf("%d/%d", hostCount, expCount)
		if hostCount == expCount {
			hostOk = "OK"
		} else {
			hostOk = "FAIL"
			ret = false
		}
		macCount := countLines(macExp, glines)
		macStatus = fmt.Sprintf("%d/%d", macCount, expCount)
		if macCount == expCount {
			macOk = "OK"
		} else {
			macOk = "FAIL"
			ret = false
		}
	}
	_, _ = r.WriteString("Guest hosts found in conf " + testdsc + ": " + hostStatus + " [" + hostOk + "]\n")
	_, _ = r.WriteString("Guest MACs found in conf " + testdsc + ": " + macStatus + " [" + macOk + "]\n")
	return ret
}

func countLines(exp map[string]bool, lines []string) int {
	for _, line := range lines {
		for k := range exp {
			if line == k {
				exp[k] = true
			}
		}
	}
	count := 0
	for _, found := range exp {
		if found {
			count++
		}
	}
	return count
}

func getLines(path string) ([]string, error) {
	var lines []string
	f, err := os.Open(path)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Open",
		}).Error("Could not open conf file for reading")
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "os.File.Close",
			}).Error("Could not close conf file after reading")
		}
	}()
	s := bufio.NewScanner(f)
	re := regexp.MustCompile("\\s+")
	for s.Scan() {
		line := s.Text()
		if line != "" {
			cleaned := re.ReplaceAllString(strings.TrimSpace(line), " ")
			lines = append(lines, cleaned)
		}
	}
	if err := s.Err(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "bufio.Scan",
		}).Error("Could not get lines from conf file")
	}
	return lines, nil
}

func cleanup(r *bytes.Buffer, e *etcd.Client, ep *testProcess, dp *testProcess) {
	if r != nil {
		log.Debug("Writing report")
		rpath := testDir + "/report.txt"
		if err := ioutil.WriteFile(rpath, r.Bytes(), 0644); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "ioutil.WriteFile",
				"path":  rpath,
			}).Warning("Could not write report")
		}
	}
	if dp != nil {
		log.Debug("Exiting Dobharchu")
		_ = dp.finish()
		time.Sleep(time.Second)
	}
	if e != nil {
		log.Debug("Clearing test data")
		if _, err := e.Delete("/lochness", true); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "etcd.Delete",
			}).Warning("Could not clear test-created data from etcd")
		}
		time.Sleep(time.Second)
	}
	if ep != nil {
		log.Debug("Exiting etcd")
		_ = ep.finish()
	}
	log.Info("Done")
}

func cleanupAfterError(err error, errfunc string, r *bytes.Buffer, e *etcd.Client, ep *testProcess, dp *testProcess) {
	testOk = false
	if r != nil {
		_, _ = r.WriteString("Finished with error: " + err.Error() + "\n")
	}
	log.WithFields(log.Fields{
		"error": err,
		"func":  errfunc,
	}).Error(err)
	showTestStatus(false)
	cleanup(r, e, ep, dp)
	os.Exit(1)
}

func showTestStatus(completed bool) {
	status := "incomplete"
	if completed {
		status = "complete"
	}
	outcome := "PASS"
	if !testOk {
		outcome = "FAIL"
	}
	fmt.Println("Test " + status + " [" + outcome + "]")
	fmt.Println("Details may be found in test directory: " + testDir)
}

func main() {

	// Command line options
	var logLevel string
	flag.StringVarP(&logLevel, "log-level", "l", "info", "log level: debug/info/warning/error/critical/fatal")
	flag.Parse()

	// Logging
	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
		}).Fatal("Could not set up logrus")
	}

	// Write logs to this directory
	testDir = "dobharchu-integration-test-" + uuid.New()
	if err := os.Mkdir(testDir, 0755); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Mkdir",
			"path":  testDir,
		}).Fatal("Could not create directory for test logs")
	}
	hconfPath = testDir + "/hypervisors.conf"
	gconfPath = testDir + "/guests.conf"

	// From now on, write the logs from this script to a file in the test directory as well
	var err error
	selfLog, err = os.Create(testDir + "/integrationtest.log")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Open",
		}).Fatal("Could not open self-log file for writing")
	}
	defer func() {
		if err := selfLog.Sync(); err != nil {
			fmt.Println("Could not sync self-log file")
		}
		if err := selfLog.Close(); err != nil {
			fmt.Println("Could not close self-log file")
			os.Exit(1)
		}
	}()
	log.SetOutput(selfLog)

	// Set up report and global ok
	r := &bytes.Buffer{}
	_, _ = r.WriteString("Dobharchu Integration Test Results\n")
	_, _ = r.WriteString("==================================\n")
	testOk = true

	// Start up processes
	log.Info("Starting etcd")
	ep := newTestProcess("etcd", exec.Command("etcd", "--data-dir", testDir+"/data.etcd",
		"--listen-client-urls", etcdClientAddress,
		"--listen-peer-urls", etcdPeerAddress,
		"--initial-advertise-peer-urls", etcdPeerAddress,
		"--initial-cluster", "default="+etcdPeerAddress,
		"--advertise-client-urls", etcdClientAddress,
	))
	if err := ep.captureOutput(true); err != nil {
		cleanupAfterError(err, "testProcess.captureOutput", r, nil, ep, nil)
	}
	if err := ep.start(); err != nil {
		cleanupAfterError(err, "testProcess.start", r, nil, ep, nil)
	}
	log.Info("Starting dobharchu")
	dp := newTestProcess("dobharchu", exec.Command("dobharchu", "-e", etcdClientAddress,
		"-d", "example.com",
		"-l", logLevel,
		"--hypervisors-path="+hconfPath,
		"--guests-path="+gconfPath,
	))
	if err := dp.captureOutput(false); err != nil {
		if err := ep.finish(); err != nil {
			log.Error("Could not close out etcd")
		}
		cleanupAfterError(err, "testProcess.captureOutput", r, nil, ep, dp)
	}
	if err := dp.start(); err != nil {
		if err := ep.finish(); err != nil {
			log.Error("Could not close out etcd")
		}
		cleanupAfterError(err, "testProcess.start", r, nil, ep, dp)
	}

	// Begin test
	log.Info("Running test")
	time.Sleep(time.Second)
	if ok := reportConfStatus(r, "on start", "created", "created"); !ok {
		log.Warning("Failure testing conf status on start")
		testOk = false
	}

	// Set up context
	e := etcd.NewClient([]string{etcdClientAddress})
	c := lochness.NewContext(e)

	// Add flavors, network, and firewall group
	log.Debug("Creating two flavors, a network, and a firewall group for building the other objects")
	f1, err := testhelper.NewFlavor(c, 4, 4096, 8192)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewFlavor", r, e, ep, dp)
	}
	f2, err := testhelper.NewFlavor(c, 6, 8192, 1024)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewFlavor", r, e, ep, dp)
	}
	n, err := testhelper.NewNetwork(c)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewNetwork", r, e, ep, dp)
	}
	fw, err := testhelper.NewFirewallGroup(c)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewFirewallGroup", r, e, ep, dp)
	}
	time.Sleep(time.Second)
	if ok := reportConfStatus(r, "after setup", "not touched", "not touched"); !ok {
		log.Warning("Failure testing conf status after setup")
		testOk = false
	}

	// Add subnet
	log.Debug("Creating a new subnet")
	s, err := testhelper.NewSubnet(c, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewSubnet", r, e, ep, dp)
	}
	time.Sleep(time.Second)
	if ok := reportConfStatus(r, "after subnet creation", "touched", "touched"); !ok {
		log.Warning("Failure testing conf status after subnet creation")
		testOk = false
	}

	// Add hypervisors
	hs := make(map[string]*lochness.Hypervisor)
	gs := make(map[string]*lochness.Guest)
	log.Debug("Creating two new hypervisors")
	h1, err := testhelper.NewHypervisor(c, "fe:dc:ba:98:76:54", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewHypervisor", r, e, ep, dp)
	}
	hs[h1.ID] = h1
	h2, err := testhelper.NewHypervisor(c, "dc:ba:98:76:54:32", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewHypervisor", r, e, ep, dp)
	}
	hs[h2.ID] = h2
	time.Sleep(time.Second)
	if ok := reportConfStatus(r, "after hypervisor creation", "changed", "touched"); !ok {
		log.Warning("Failure testing conf status after hypervisor creation")
		testOk = false
	}
	if ok := reportHasHosts(r, "after hypervisor creation", hs, gs); !ok {
		log.Warning("Failure testing for hosts in confs after hypervisor creation")
		testOk = false
	}

	// Add guests
	log.Debug("Creating four new guests")
	g1, err := testhelper.NewGuest(c, "ba:98:76:54:32:10", n, s, f1, fw, h1)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewGuest", r, e, ep, dp)
	}
	gs[g1.ID] = g1
	g2, err := testhelper.NewGuest(c, "98:76:54:32:10:fe", n, s, f2, fw, h1)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewGuest", r, e, ep, dp)
	}
	gs[g2.ID] = g2
	g3, err := testhelper.NewGuest(c, "76:54:32:10:fe:dc", n, s, f1, fw, h2)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewGuest", r, e, ep, dp)
	}
	gs[g3.ID] = g3
	g4, err := testhelper.NewGuest(c, "54:32:10:fe:dc:ba", n, s, f2, fw, h2)
	if err != nil {
		cleanupAfterError(err, "testhelper.NewGuest", r, e, ep, dp)
	}
	gs[g4.ID] = g4
	time.Sleep(time.Second)
	if ok := reportConfStatus(r, "after group creation", "touched", "changed"); !ok {
		log.Warning("Failure testing conf status after group creation")
		testOk = false
	}
	if ok := reportHasHosts(r, "after group creation", hs, gs); !ok {
		log.Warning("Failure testing for hosts in confs after group creation")
		testOk = false
	}

	// Sleep for a few seconds to make sure everything finished, then clean up
	time.Sleep(2 * time.Second)
	log.WithField("path", testDir).Info("Creating test output directory")
	showTestStatus(true)
	cleanup(r, e, ep, dp)
}

// Package common contains common utilities and suites to be used in other tests
package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/kv"
	_ "github.com/mistifyio/lochness/pkg/kv/etcd"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

// Suite sets up a general test suite with setup/teardown.
type Suite struct {
	suite.Suite
	KVDir    string
	KVPrefix string
	KVURL    string
	KV       kv.KV
	KVCmd    *exec.Cmd
	Context  *lochness.Context
}

// SetupSuite runs a new kv instance.
func (s *Suite) SetupSuite() {
	// Start up a test kv
	s.KVDir, _ = ioutil.TempDir("", "lochnessTest-"+uuid.New())
	port := 54321
	clientURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", port+1)
	s.KVCmd = exec.Command("etcd",
		"-name", "lochnessTest",
		"-data-dir", s.KVDir,
		"-initial-cluster-state", "new",
		"-initial-cluster-token", "lochnessTest",
		"-initial-cluster", "lochnessTest="+peerURL,
		"-initial-advertise-peer-urls", peerURL,
		"-listen-peer-urls", peerURL,
		"-listen-client-urls", clientURL,
		"-advertise-client-urls", clientURL,
	)
	s.Require().NoError(s.KVCmd.Start())
	time.Sleep(100 * time.Millisecond) // Wait for test kv to be ready

	e, err := kv.New(clientURL)
	if err != nil {
		panic(err)
	}
	s.KVURL = clientURL

	s.KV = e
	s.Context = lochness.NewContext(s.KV)

	s.KVPrefix = "/lochness"
}

// SetupTest prepares anything needed per test.
func (s *Suite) SetupTest() {
}

// TearDownTest cleans the kv instance.
func (s *Suite) TearDownTest() {
	// Clean out kv
	_ = s.KV.Delete(s.KVPrefix, true)
}

// TearDownSuite stops the kv instance and removes all data.
func (s *Suite) TearDownSuite() {
	// Stop the test kv process
	_ = s.KVCmd.Process.Kill()
	_ = s.KVCmd.Wait()

	// Remove the test kv data directory
	s.Require().NoError(os.RemoveAll(s.KVDir))
}

// PrefixKey generates an kv key using the set prefix
func (s *Suite) PrefixKey(key string) string {
	return filepath.Join(s.KVPrefix, key)
}

// NewFlavor creates and saves a new Flavor.
func (s *Suite) NewFlavor() *lochness.Flavor {
	f := s.Context.NewFlavor()
	f.Image = uuid.New()
	f.Resources = lochness.Resources{
		Memory: 128,
		Disk:   1024,
		CPU:    1,
	}
	_ = f.Save()
	return f
}

// NewFWGroup creates and saves a new FWGroup.
func (s *Suite) NewFWGroup() *lochness.FWGroup {
	fw := s.Context.NewFWGroup()
	_ = fw.Save()
	return fw
}

// NewVLAN creates and saves a new VLAN.
func (s *Suite) NewVLAN() *lochness.VLAN {
	v := s.Context.NewVLAN()
	v.Tag = rand.Intn(4096)
	_ = v.Save()
	return v
}

// NewVLANGroup creates and saves a new VLANGroup.
func (s *Suite) NewVLANGroup() *lochness.VLANGroup {
	v := s.Context.NewVLANGroup()
	s.NoError(v.Save())
	return v
}

// NewNetwork creates and saves a new Netework.
func (s *Suite) NewNetwork() *lochness.Network {
	n := s.Context.NewNetwork()
	_ = n.Save()
	return n
}

// NewSubnet creates and saves a new Subnet.
func (s *Suite) NewSubnet() *lochness.Subnet {
	sub := s.Context.NewSubnet()
	_, sub.CIDR, _ = net.ParseCIDR("192.168.100.1/24")
	sub.StartRange = net.ParseIP("192.168.100.2")
	sub.EndRange = net.ParseIP("192.168.100.10")
	_ = sub.Save()
	return sub
}

// NewHypervisor creates and saves a new Hypervisor.
func (s *Suite) NewHypervisor() *lochness.Hypervisor {
	h := s.Context.NewHypervisor()
	h.IP = net.ParseIP("192.168.100.11")
	h.Netmask = net.ParseIP("225.225.225.225")
	h.Gateway = net.ParseIP("192.168.100.1")
	h.MAC, _ = net.ParseMAC("96:E0:51:F9:31:C1")
	h.TotalResources = lochness.Resources{
		Memory: 16 * 1024,
		Disk:   1024 * 1024,
		CPU:    32,
	}
	h.AvailableResources = h.TotalResources
	_ = h.Save()
	return h
}

// NewGuest creates and saves a new Guest. Creates any necessary resources.
func (s *Suite) NewGuest() *lochness.Guest {
	flavor := s.NewFlavor()
	network := s.NewNetwork()
	mac, _ := net.ParseMAC("4C:3F:B1:7E:54:64")

	guest := s.Context.NewGuest()
	guest.FlavorID = flavor.ID
	guest.NetworkID = network.ID
	guest.MAC = mac

	_ = guest.Save()
	return guest
}

// NewHypervisorWithGuest creates and saves a new Hypervisor and Guest, with
// the Guest added to the Hypervisor.
func (s *Suite) NewHypervisorWithGuest() (*lochness.Hypervisor, *lochness.Guest) {
	guest := s.NewGuest()
	hypervisor := s.NewHypervisor()

	subnet := s.NewSubnet()
	network, _ := s.Context.Network(guest.NetworkID)
	s.Require().NoError(network.AddSubnet(subnet))
	s.Require().NoError(hypervisor.AddSubnet(subnet, "mistify0"))

	s.Require().NoError(hypervisor.AddGuest(guest))

	return hypervisor, guest
}

// DoRequest is a convenience method for making an http request and doing basic handling of the response.
func (s *Suite) DoRequest(method, url string, expectedRespCode int, postBodyStruct interface{}, respBody interface{}) *http.Response {
	var postBody io.Reader
	if postBodyStruct != nil {
		bodyBytes, _ := json.Marshal(postBodyStruct)
		postBody = bytes.NewBuffer(bodyBytes)
	}

	req, err := http.NewRequest(method, url, postBody)
	if postBody != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	s.NoError(err)
	correctResponse := s.Equal(expectedRespCode, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()

	body, err := ioutil.ReadAll(resp.Body)
	s.NoError(err)

	if correctResponse {
		s.NoError(json.Unmarshal(body, respBody))
	} else {
		s.T().Log(string(body))
	}
	return resp
}

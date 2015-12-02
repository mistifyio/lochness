/*
ct contains common utilities and suites to be used in other tests
*/
package ct

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type CommonTestSuite struct {
	suite.Suite
	EtcdDir    string
	EtcdPrefix string
	EtcdClient *etcd.Client
	EtcdCmd    *exec.Cmd
	Context    *lochness.Context
}

func (s *CommonTestSuite) SetupSuite() {
	//	log.SetLevel(log.ErrorLevel)

	// Start up a test etcd
	s.EtcdDir, _ = ioutil.TempDir("", "lochnessTest-"+uuid.New())
	port := 54321
	clientURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", port+1)
	s.EtcdCmd = exec.Command("etcd",
		"-name", "lochnessTest",
		"-data-dir", s.EtcdDir,
		"-initial-cluster-state", "new",
		"-initial-cluster-token", "lochnessTest",
		"-initial-cluster", "lochnessTest="+peerURL,
		"-initial-advertise-peer-urls", peerURL,
		"-listen-peer-urls", peerURL,
		"-listen-client-urls", clientURL,
		"-advertise-client-urls", clientURL,
	)
	s.Require().NoError(s.EtcdCmd.Start())
	s.EtcdClient = etcd.NewClient([]string{clientURL})

	// Wait for test etcd to be ready
	for !s.EtcdClient.SyncCluster() {
		time.Sleep(10 * time.Millisecond)
	}

	// s.EtcdPrefix = uuid.New()
	s.EtcdPrefix = "/lochness"
}

func (s *CommonTestSuite) SetupTest() {
	s.Context = lochness.NewContext(s.EtcdClient)
}

func (s *CommonTestSuite) TearDownTest() {
	// Clean out etcd
	_, _ = s.EtcdClient.Delete(s.EtcdPrefix, true)
}

func (s *CommonTestSuite) TearDownSuite() {
	// Stop the test etcd process
	_ = s.EtcdCmd.Process.Kill()
	_ = s.EtcdCmd.Wait()

	// Remove the test etcd data directory
	s.Require().NoError(os.RemoveAll(s.EtcdDir))
}

func (s *CommonTestSuite) PrefixKey(key string) string {
	return filepath.Join(s.EtcdPrefix, key)
}

func (s *CommonTestSuite) NewFlavor() *lochness.Flavor {
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

func (s *CommonTestSuite) NewFWGroup() *lochness.FWGroup {
	fw := s.Context.NewFWGroup()
	_ = fw.Save()
	return fw
}

func (s *CommonTestSuite) NewVLAN() *lochness.VLAN {
	v := s.Context.NewVLAN()
	v.Tag = rand.Intn(4066)
	_ = v.Save()
	return v
}

func (s *CommonTestSuite) NewVLANGroup() *lochness.VLANGroup {
	v := s.Context.NewVLANGroup()
	s.NoError(v.Save())
	return v
}

func (s *CommonTestSuite) NewNetwork() *lochness.Network {
	n := s.Context.NewNetwork()
	_ = n.Save()
	return n
}

func (s *CommonTestSuite) NewSubnet() *lochness.Subnet {
	sub := s.Context.NewSubnet()
	_, sub.CIDR, _ = net.ParseCIDR("192.168.100.1/24")
	sub.StartRange = net.ParseIP("192.168.100.2")
	sub.EndRange = net.ParseIP("192.168.100.10")
	_ = sub.Save()
	return sub
}

func (s *CommonTestSuite) NewHypervisor() *lochness.Hypervisor {
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

func (s *CommonTestSuite) NewGuest() *lochness.Guest {
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

func (s *CommonTestSuite) NewHypervisorWithGuest() (*lochness.Hypervisor, *lochness.Guest) {
	guest := s.NewGuest()
	hypervisor := s.NewHypervisor()

	subnet := s.NewSubnet()
	network, _ := s.Context.Network(guest.NetworkID)
	_ = network.AddSubnet(subnet)
	_ = hypervisor.AddSubnet(subnet, "mistify0")

	_ = hypervisor.AddGuest(guest)

	return hypervisor, guest
}

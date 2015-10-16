package lochness_test

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
	s.EtcdCmd = exec.Command("etcd",
		"-name=lochnessTest",
		"-data-dir="+string(s.EtcdDir),
		fmt.Sprintf("-listen-client-urls=http://127.0.0.1:%d", port),
		fmt.Sprintf("-listen-peer-urls=http://127.0.0.1:%d", port+1),
	)
	s.Require().NoError(s.EtcdCmd.Start())
	s.EtcdClient = etcd.NewClient([]string{fmt.Sprintf("http://127.0.0.1:%d", port)})

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
	s.EtcdCmd.Process.Kill()
	s.EtcdCmd.Wait()

	// Remove the test etcd data directory
	s.Require().NoError(os.RemoveAll(s.EtcdDir))
}

func (s *CommonTestSuite) prefixKey(key string) string {
	return filepath.Join(s.EtcdPrefix, key)
}

func (s *CommonTestSuite) newFlavor() *lochness.Flavor {
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

func (s *CommonTestSuite) newFWGroup() *lochness.FWGroup {
	fw := s.Context.NewFWGroup()
	_ = fw.Save()
	return fw
}

func (s *CommonTestSuite) newVLAN() *lochness.VLAN {
	v := s.Context.NewVLAN()
	v.Tag = rand.Intn(4066)
	_ = v.Save()
	return v
}

func (s *CommonTestSuite) newVLANGroup() *lochness.VLANGroup {
	v := s.Context.NewVLANGroup()
	s.NoError(v.Save())
	return v
}

func (s *CommonTestSuite) newNetwork() *lochness.Network {
	n := s.Context.NewNetwork()
	_ = n.Save()
	return n
}

func (s *CommonTestSuite) newSubnet() *lochness.Subnet {
	sub := s.Context.NewSubnet()
	_, sub.CIDR, _ = net.ParseCIDR("192.168.100.1/24")
	sub.StartRange = net.ParseIP("192.168.100.2")
	sub.EndRange = net.ParseIP("192.168.100.10")
	_ = sub.Save()
	return sub
}

func (s *CommonTestSuite) newHypervisor() *lochness.Hypervisor {
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

func (s *CommonTestSuite) newGuest() *lochness.Guest {
	flavor := s.newFlavor()
	network := s.newNetwork()
	mac, _ := net.ParseMAC("4C:3F:B1:7E:54:64")

	guest := s.Context.NewGuest()
	guest.FlavorID = flavor.ID
	guest.NetworkID = network.ID
	guest.MAC = mac

	_ = guest.Save()
	return guest
}

func (s *CommonTestSuite) newHypervisorWithGuest() (*lochness.Hypervisor, *lochness.Guest) {
	guest := s.newGuest()
	hypervisor := s.newHypervisor()

	subnet := s.newSubnet()
	network, _ := s.Context.Network(guest.NetworkID)
	_ = network.AddSubnet(subnet)
	_ = hypervisor.AddSubnet(subnet, "mistify0")

	_ = hypervisor.AddGuest(guest)

	return hypervisor, guest
}

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

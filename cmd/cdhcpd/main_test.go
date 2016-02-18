package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/suite"
)

type MainTestSuite struct {
	ct.CommonTestSuite
	BinName          string
	ConfDir          string
	HypervisorConfig string
	GuestConfig      string
}

func (s *MainTestSuite) SetupSuite() {
	s.CommonTestSuite.SetupSuite()
	s.Require().NoError(ct.Build())
	s.BinName = "cdhcpd"
}

func (s *MainTestSuite) SetupTest() {
	s.CommonTestSuite.SetupTest()
	s.ConfDir, _ = ioutil.TempDir("", "cdhcpd-Test")
	s.HypervisorConfig = filepath.Join(s.ConfDir, "hypervisors.conf")
	s.GuestConfig = filepath.Join(s.ConfDir, "guests.conf")
}

func (s *MainTestSuite) TearDownTest() {
	s.CommonTestSuite.TearDownTest()
	_ = os.RemoveAll(s.ConfDir)
}

func TestMainTestSuite(t *testing.T) {
	suite.Run(t, new(MainTestSuite))
}

func (s *MainTestSuite) TestCmd() {
	hypervisor, guest := s.NewHypervisorWithGuest()

	args := []string{
		"-d", "cdhcpdTest",
		"-e", s.EtcdURL,
		"-c", s.ConfDir,
		"-l", "fatal",
	}
	cmd, err := ct.Exec("./"+s.BinName, args...)
	s.Require().NoError(err)
	time.Sleep(1 * time.Second)

	s.checkConfFiles(hypervisor, guest)

	hypervisor2, guest2 := s.NewHypervisorWithGuest()
	time.Sleep(1 * time.Second)

	s.checkConfFiles(hypervisor2, guest2)

	_, _ = s.EtcdClient.Delete(s.EtcdPrefix, true)
	time.Sleep(1 * time.Second)
	hData, _ := ioutil.ReadFile(s.HypervisorConfig)
	s.NotContains(string(hData), hypervisor.ID, "hypervisor not removed")

	gData, _ := ioutil.ReadFile(s.GuestConfig)
	s.NotContains(string(gData), guest.ID, "guest not removed")

	// Stop the daemon
	_ = cmd.Stop()
	status, err := cmd.ExitStatus()
	s.Equal(-1, status, "expected status code to be that of a killed process")
}

func (s *MainTestSuite) checkConfFiles(hypervisor *lochness.Hypervisor, guest *lochness.Guest) bool {
	passed := true
	hData, err := ioutil.ReadFile(s.HypervisorConfig)
	passed = s.NoError(err) && passed
	passed = s.Contains(string(hData), hypervisor.ID, "hypervisor not present") && passed

	gData, err := ioutil.ReadFile(s.GuestConfig)
	passed = s.NoError(err) && passed
	passed = s.Contains(string(gData), guest.ID, "guest not present") && passed

	return passed
}

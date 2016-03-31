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

func TestCDhcpd(t *testing.T) {
	suite.Run(t, new(CmdSuite))
}

type CmdSuite struct {
	common.Suite
	BinName          string
	ConfDir          string
	HypervisorConfig string
	GuestConfig      string
}

func (s *CmdSuite) SetupSuite() {
	s.Suite.SetupSuite()
	s.Require().NoError(common.Build())
	s.BinName = "cdhcpd"
}

func (s *CmdSuite) SetupTest() {
	s.Suite.SetupTest()
	s.ConfDir, _ = ioutil.TempDir("", "cdhcpd-test")
	s.HypervisorConfig = filepath.Join(s.ConfDir, "hypervisors.conf")
	s.GuestConfig = filepath.Join(s.ConfDir, "guests.conf")
}

func (s *CmdSuite) TearDownTest() {
	s.Suite.TearDownTest()
	_ = os.RemoveAll(s.ConfDir)
}

func (s *CmdSuite) TestCmd() {
	hypervisor, guest := s.NewHypervisorWithGuest()

	args := []string{
		"-d", "cdhcpdTest",
		"-k", s.KVURL,
		"-c", s.ConfDir,
		"-l", "fatal",
	}
	cmd, err := common.Start("./"+s.BinName, args...)
	s.Require().NoError(err)
	time.Sleep(1 * time.Second)

	s.checkConfFiles(hypervisor, guest)

	hypervisor2, guest2 := s.NewHypervisorWithGuest()
	time.Sleep(1 * time.Second)

	s.checkConfFiles(hypervisor2, guest2)

	s.Require().NoError(s.KV.Delete(s.KVPrefix, true))
	time.Sleep(1 * time.Second)
	hData, _ := ioutil.ReadFile(s.HypervisorConfig)
	s.NotContains(string(hData), hypervisor.ID, "hypervisor not removed")

	gData, _ := ioutil.ReadFile(s.GuestConfig)
	s.NotContains(string(gData), guest.ID, "guest not removed")

	// Stop the daemon
	_ = cmd.Stop()
	status, err := cmd.ExitStatus()
	s.Equal(-1, status, "expected status code to be that of a killed process")

	// so that common.Suite.TearDownTest does not fail
	s.KV.Set("lochness", "hi")
}

func (s *CmdSuite) checkConfFiles(hypervisor *lochness.Hypervisor, guest *lochness.Guest) bool {
	passed := true
	hData, err := ioutil.ReadFile(s.HypervisorConfig)
	passed = s.NoError(err) && passed
	passed = s.Contains(string(hData), hypervisor.ID, "hypervisor not present") && passed

	gData, err := ioutil.ReadFile(s.GuestConfig)
	passed = s.NoError(err) && passed
	passed = s.Contains(string(gData), guest.ID, "guest not present") && passed

	return passed
}

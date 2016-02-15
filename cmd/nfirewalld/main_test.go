package main_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/common_test"
	"github.com/stretchr/testify/suite"
)

type NFirewalldTestSuite struct {
	ct.CommonTestSuite
	ConfigPath string
	BinName    string
	Hypervisor *lochness.Hypervisor
	Guest      *lochness.Guest
	FWGroup    *lochness.FWGroup
}

func (s *NFirewalldTestSuite) SetupSuite() {
	s.CommonTestSuite.SetupSuite()

	s.Require().NoError(ct.Build(), "failed to build nfirewalld")
	s.BinName = "nfirewalld"
}

func (s *NFirewalldTestSuite) SetupTest() {
	s.CommonTestSuite.SetupTest()

	configFile, err := ioutil.TempFile("", "nfirewalldTest-")
	s.Require().NoError(err, "failed to create config file")
	s.ConfigPath = configFile.Name()

	s.Hypervisor, s.Guest = s.NewHypervisorWithGuest()
	s.FWGroup = s.NewFWGroup()
	s.Require().NoError(newFWRule(s.FWGroup, "deny", "192.168.1.100/16", 2000, 3000), "failed to add FWRule")
	s.Guest.FWGroupID = s.FWGroup.ID
	s.Require().NoError(s.Guest.Save(), "failed to save guest with FWGroup ID")
}

func (s *NFirewalldTestSuite) TearDownTest() {
	//	os.Remove(s.ConfigPath)

	s.CommonTestSuite.TearDownTest()
}

func TestNFirewalldTestSuite(t *testing.T) {
	suite.Run(t, new(NFirewalldTestSuite))
}

func (s *NFirewalldTestSuite) TestCmd() {
	args := []string{
		"-e", s.EtcdURL,
		"-f", s.ConfigPath,
		"-i", s.Hypervisor.ID,
	}
	cmd, err := ct.Exec("./"+s.BinName, args...)
	s.NoError(err)

	time.Sleep(1 * time.Second)
	s.NoError(cmd.Stop())
	status, _ := cmd.ExitStatus()
	s.Equal(-1, status, "expected status code to be that of a killed process")

	output := strings.TrimSpace(cmd.Out.String())
	fmt.Println("OUTPUT:\n", output)
}

func newFWRule(fwgroup *lochness.FWGroup, action, source string, start, end uint) error {
	_, n, _ := net.ParseCIDR("source")
	fwrule := &lochness.FWRule{
		Source:    n,
		PortStart: start,
		PortEnd:   end,
		Action:    action,
	}
	fwgroup.Rules = lochness.FWRules{fwrule}
	return fwgroup.Save()
}

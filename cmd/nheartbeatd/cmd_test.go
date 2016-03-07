package main_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestNHeartbeatd(t *testing.T) {
	suite.Run(t, new(CmdSuite))
}

type CmdSuite struct {
	common.Suite
	Hypervisor *lochness.Hypervisor
	BinName    string
}

func (s *CmdSuite) SetupSuite() {
	s.Suite.SetupSuite()
	s.Require().NoError(common.Build())
	s.BinName = "nheartbeatd"
}

func (s *CmdSuite) SetupTest() {
	s.Suite.SetupTest()
	s.Hypervisor = s.NewHypervisor()
	s.Require().NoError(s.Hypervisor.SetConfig("guestDiskDir", "/dev/null"))
}

func (s *CmdSuite) TestCmd() {
	tests := []struct {
		description string
		id          string
		ttl         int
		interval    int
		loglevel    string
		expectedErr bool
	}{
		{"invalid id", "asdf", 2, 1, "error", true},
		{"no hypervisor for id", uuid.New(), 2, 1, "error", true},
		{"ttl less than interval", s.Hypervisor.ID, 1, 2, "error", true},
		{"valid", s.Hypervisor.ID, 2, 1, "error", false},
	}

	for _, test := range tests {
		msg := common.TestMsgFunc(test.description)
		args := []string{
			"-k", s.KVURL,
			"-d", test.id,
			"-i", strconv.Itoa(test.interval),
			"-t", strconv.Itoa(test.ttl),
			"-l", test.loglevel,
		}
		cmd, err := common.Start("./"+s.BinName, args...)
		if !s.NoError(err, msg("command exec should not error")) {
			continue
		}
		start := time.Now()

		if test.expectedErr {
			s.Error(cmd.Wait(), msg("daemon should have exited with error"))
			continue
		}

		for i := 0; i < 2; i++ {
			time.Sleep(time.Duration(test.interval) * time.Second)
			if !s.True(cmd.Alive(), msg("daemon should still be running")) {
				break
			}

			resp, err := s.KVClient.Get(fmt.Sprintf("/lochness/hypervisors/%s/heartbeat", test.id), false, false)
			if !s.NoError(err, msg("heartbeat key should exist")) {
				continue
			}

			s.EqualValues(test.ttl, resp.Node.TTL)

			v, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", resp.Node.Value)
			if !s.NoError(err, msg("heartbeat value should be a go time string")) {
				continue
			}

			s.WithinDuration(start.Add(time.Duration(i*test.interval)*time.Second), v, 100*time.Millisecond, i,
				msg("heartbeat value should be time around when it was set"),
			)
		}

		// Stop the daemon
		_ = cmd.Stop()
		status, err := cmd.ExitStatus()
		s.Equal(-1, status, msg("expected status code to be that of a killed process"))

		// Check that the key expires
		time.Sleep(time.Duration(test.ttl) * time.Second)
		_, err = s.KVClient.Get(fmt.Sprintf("/lochness/hypervisors/%s/heartbeat", test.id), false, false)
		s.Error(err, msg("heartbeat should have expired"))
	}
}

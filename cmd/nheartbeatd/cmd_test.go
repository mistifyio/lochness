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
	ttl := 2 * time.Second
	interval := 1 * time.Second
	tests := []struct {
		description string
		id          string
		ttl         time.Duration
		interval    time.Duration
		loglevel    string
		expectedErr bool
	}{
		{"invalid id", "asdf", ttl, interval, "error", true},
		{"no hypervisor for id", uuid.New(), ttl, interval, "error", true},
		{"ttl less than interval", s.Hypervisor.ID, interval, ttl, "error", true},
		{"valid", s.Hypervisor.ID, ttl, interval, "error", false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		args := []string{
			"-k", s.KVURL,
			"-d", test.id,
			"-i", strconv.Itoa(test.interval.Seconds()),
			"-t", strconv.Itoa(test.ttl.Seconds()),
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
			time.Sleep(test.interval)
			if !s.True(cmd.Alive(), msg("daemon should still be running")) {
				break
			}

			resp, err := s.KV.Get(fmt.Sprintf("/lochness/hypervisors/%s/heartbeat", test.id))
			if !s.NoError(err, msg("heartbeat key should exist")) {
				continue
			}

			// this is unfortunate but necessary if we actually want to check contents
			v, err := time.Parse("locked=true:2006-01-02 15:04:05.999999999 -0700 MST", string(resp.Data))
			if !s.NoError(err, msg("heartbeat lock value should be a go time string")) {
				continue
			}

			start := start.Add(time.Duration(i) * test.interval)
			s.Require().WithinDuration(start, v, 100*time.Millisecond, msg("heartbeat value should be time around when it was set"))
		}

		// Stop the daemon
		_ = cmd.Stop()
		status, err := cmd.ExitStatus()
		s.Equal(-1, status, msg("expected status code to be that of a killed process"))

		// Check that the key expires
		time.Sleep(2 * test.ttl)
		_, err = s.KV.Get(fmt.Sprintf("/lochness/hypervisors/%s/heartbeat", test.id))
		s.Error(err, msg("heartbeat should have expired"))
	}
}

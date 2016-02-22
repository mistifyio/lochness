package sd_test

import (
	"os"
	"strconv"
	"testing"

	"github.com/mistifyio/lochness/pkg/sd"
	"github.com/stretchr/testify/suite"
)

func TestSDWatchdogEnabled(t *testing.T) {
	suite.Run(t, new(SDWatchdogEnabledSuite))
}

type SDWatchdogEnabledSuite struct {
	suite.Suite
}

func (s *SDWatchdogEnabledSuite) TestWatchdogEnabled() {
	spid := strconv.Itoa(os.Getpid())
	tests := []struct {
		description     string
		pid             string
		usec            string
		expectedEnabled bool
		expectedErr     bool
	}{
		{"no env set", "", "", false, true},
		{"missing pid", "", "10", true, false},
		{"invalid pid", "asdf", "10", false, true},
		{"missing usec", spid, "", false, true},
		{"invalid usec", spid, "asdf", false, true},
		{"self pid and valid usec", spid, "10", true, false},
		{"non-self pid and usec", spid + "1", "10", false, false},
	}

	for _, test := range tests {
		_ = os.Setenv("WATCHDOG_PID", test.pid)
		_ = os.Setenv("WATCHDOG_USEC", test.usec)
		msg := testMsgFunc(test.description)

		time, err := sd.WatchdogEnabled()
		if test.expectedErr {
			s.Error(err, msg("should have failed"))
		} else {
			s.NoError(err, msg("should not have failed"))
		}

		if test.expectedEnabled {
			s.NotZero(time, msg("should be enabled"))
		} else {
			s.Zero(time, msg("should not be enabled"))
		}
	}
}

package main_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/mistifyio/lochness/cmd/common_test"
	"github.com/mistifyio/lochness/cmd/locker"
	"github.com/mistifyio/lochness/pkg/lock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type LockerTestSuite struct {
	ct.CommonTestSuite
	BinName string
}

func (s *LockerTestSuite) SetupSuite() {
	s.CommonTestSuite.SetupSuite()
	s.Require().NoError(ct.Build())
	s.BinName = "locker"
}

func TestLockerTestSuite(t *testing.T) {
	suite.Run(t, new(LockerTestSuite))
}

func (s *LockerTestSuite) TestCmd() {
	shCmd := `set -e; sleep 1; echo -n "%s" > "%s"`

	tests := []struct {
		description  string
		interval     int
		ttl          int
		watchdog     uint64
		expectedOut  string
		fileOut      string
		expectedCode int
	}{
		{"valid", 1, 2, 2, "", "valid", 0},
		{"mismatched watchdog ttl", 1, 2, 0, "params and systemd ttls do not match", "", 1},
	}

	for id, test := range tests {
		file, _ := ioutil.TempFile("", "lockerTest-")
		defer func() { _ = os.Remove(file.Name()) }()

		msg := ct.TestMsgFunc(test.description)
		params := &main.Params{
			Interval: 1,
			TTL:      2,
			Key:      uuid.New(),
			Addr:     s.EtcdURL,
			Blocking: false,
			ID:       id,
		}
		params.Args = []string{"/bin/sh", "-c", fmt.Sprintf(shCmd, test.fileOut, file.Name())}
		params.Lock, _ = lock.Acquire(s.EtcdClient, params.Key, uuid.New(), params.TTL, params.Blocking)
		defer func() { _ = params.Lock.Release() }()

		// Time in microseconds
		_ = os.Setenv("WATCHDOG_USEC", strconv.FormatUint(test.watchdog*uint64(time.Second/time.Microsecond), 10))

		args, _ := json.Marshal(&params)
		arg := base64.StdEncoding.EncodeToString(args)
		cmd, err := ct.Exec("./"+s.BinName, arg)
		if !s.NoError(err, msg("should not have errored execing command")) {
			continue
		}

		_ = cmd.Wait()
		status, err := cmd.ExitStatus()
		out, err := ioutil.ReadAll(file)

		s.Equal(test.expectedCode, status, msg("unexpected exit status code"))
		fmt.Println("command output:", cmd.Out.String())
		s.Contains(cmd.Out.String(), test.expectedOut, msg("unexpected output message"))
		s.Equal(test.fileOut, string(out), msg("unexpected output"))
	}
}

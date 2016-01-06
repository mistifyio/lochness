package sd_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/mistifyio/lochness/pkg/sd"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type SDNotifyTestSuite struct {
	suite.Suite
	SocketDir string
}

func (s *SDNotifyTestSuite) SetupTest() {
	s.SocketDir, _ = ioutil.TempDir("", "sdTest-")
}

func (s *SDNotifyTestSuite) TearDownTest() {
	_ = os.RemoveAll(s.SocketDir)
}

func TestSDNotifyTestSuite(t *testing.T) {
	suite.Run(t, new(SDNotifyTestSuite))
}

func (s *SDNotifyTestSuite) socketPath(name string) string {
	return filepath.Join(s.SocketDir, name)
}
func (s *SDNotifyTestSuite) TestNotify() {
	tests := []struct {
		description string
		socket      string
		state       string
		expectedErr bool
	}{
		{"missing socket", "", "asdf", true},
		{"missing state", uuid.New(), "", false},
		{"good socket", uuid.New(), "asdf", false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		socket := s.socketPath(test.socket)
		_ = os.Setenv("NOTIFY_SOCKET", socket)

		unixAddr, _ := net.ResolveUnixAddr("unixgram", socket)
		lConn, lErr := net.ListenUnixgram("unixgram", unixAddr)
		if lErr == nil {
			defer func() { _ = lConn.Close() }()
		}

		err := sd.Notify(test.state)
		data := make([]byte, 1024)
		if lErr == nil {
			_, _ = lConn.Read(data)
		}
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			if test.state != "" {
				s.NotContains(string(data), test.state, msg("state should not be written to socket"))
			}
		} else {
			s.NoError(err, msg("should succeed"))
			s.Contains(string(data), test.state, msg("state should be written to socket"))
		}
	}
}

func testMsgFunc(prefix string) func(...interface{}) string {
	return func(val ...interface{}) string {
		if len(val) == 0 {
			return prefix
		}
		msgPrefix := prefix + " : "
		if len(val) == 1 {
			return msgPrefix + val[0].(string)
		} else {
			return msgPrefix + fmt.Sprintf(val[0].(string), val[1:]...)
		}
	}
}

package sd_test

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/mistifyio/lochness/pkg/sd"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestSDNotify(t *testing.T) {
	suite.Run(t, new(SDNotifySuite))
}

type SDNotifySuite struct {
	common.Suite
	SocketDir string
}

func (s *SDNotifySuite) SetupSuite() {
}

func (s *SDNotifySuite) TearDownSuite() {
}

func (s *SDNotifySuite) SetupTest() {
	s.SocketDir, _ = ioutil.TempDir("", "sdTest-")
}

func (s *SDNotifySuite) TearDownTest() {
	_ = os.RemoveAll(s.SocketDir)
}

func (s *SDNotifySuite) socketPath(name string) string {
	return filepath.Join(s.SocketDir, name)
}

func (s *SDNotifySuite) TestNotify() {
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
		msg := s.Messager(test.description)
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

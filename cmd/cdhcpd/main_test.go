package main_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/common_test"
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

	s.EtcdClient.Delete(s.EtcdPrefix, true)
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

/*

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	log "github.com/Sirupsen/logrus"
)

// func writeConfig(confType, path string, checksum []byte, generator func(io.Writer) error) ([]byte, error) {
func TestWriteConfig(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal("could not allocate temporary file", err)
	}
	if err := f.Close(); err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"filename": f.Name(),
		}).Error("failed to close temp file")
	}

	wrapper := func(contents []byte) func(io.Writer) error {
		return func(w io.Writer) error {
			_, err := w.Write(contents)
			return err
		}
	}

	tests := []struct {
		contents []byte
		sum      string
		err      error
		fn       func(io.Writer) error
	}{
		{
			err: fmt.Errorf("test failure"),
			fn: func(w io.Writer) error {
				return errors.New("test failure")
			},
		},
		{
			contents: []byte("testing\n"),
			sum:      "eb1a3227cdc3fedbaec2fe38bf6c044a",
			err:      nil,
		},
		{
			contents: []byte("testing123\n"),
			sum:      "bad9425ff652b1bd52b49720abecf0ba",
			err:      nil,
		},
	}

	for _, test := range tests {
		fn := test.fn
		if fn == nil {
			fn = wrapper(test.contents)
		}

		hash, err := writeConfig("test", f.Name(), test.contents, fn)
		if !reflect.DeepEqual(err, test.err) {
			t.Fatalf("unexpected error, want:|%s|, got:|%s|\n", test.err, err)
		}
		if test.err != nil {
			continue
		}

		sum := fmt.Sprintf("%x", hash)
		if !reflect.DeepEqual(sum, test.sum) {
			t.Fatal("checksum mismatch, want:", test.sum, "got:", sum)
		}
		b, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Fatal("failed to read file for verification", err)
		}
		if !reflect.DeepEqual(b, test.contents) {
			t.Fatalf("incorrect contents, want: %s, got: %s", test.contents, b)
		}

		hash, err = writeConfig("test", f.Name(), hash, fn)
		if err != nil {
			t.Fatal("got an unexpected error", err)
		}

		if hash != nil {
			t.Fatal("wanted a nil checksum to signify no changes, got:", string(hash))
		}

		if err := os.Remove(f.Name()); err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"filename": f.Name(),
			}).Error("failed to remove temp file")
		}
	}
}
*/

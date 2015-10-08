package lochness_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	suite.Suite
	EtcdDir    string
	EtcdPrefix string
	EtcdClient *etcd.Client
	EtcdCmd    *exec.Cmd
	Context    *lochness.Context
}

func (s *ContextTestSuite) SetupSuite() {
	// Start up a test etcd
	s.EtcdDir, _ = ioutil.TempDir("", "lochnessTest-"+uuid.New())
	port := 54321
	s.EtcdCmd = exec.Command("etcd",
		"-name=lochnessTest",
		"-data-dir="+string(s.EtcdDir),
		fmt.Sprintf("-listen-client-urls=http://127.0.0.1:%d", port),
		fmt.Sprintf("-listen-peer-urls=http://127.0.0.1:%d", port+1),
	)
	s.Require().NoError(s.EtcdCmd.Start())
	s.EtcdClient = etcd.NewClient([]string{fmt.Sprintf("http://127.0.0.1:%d", port)})

	// Wait for test etcd to be ready
	for !s.EtcdClient.SyncCluster() {
		time.Sleep(10 * time.Millisecond)
	}

	// s.EtcdPrefix = uuid.New()
	s.EtcdPrefix = "/lochness"
}

func (s *ContextTestSuite) SetupTest() {
	s.Context = lochness.NewContext(s.EtcdClient)
}

func (s *ContextTestSuite) TearDownTest() {
	// Clean out etcd
	_, _ = s.EtcdClient.Delete(s.EtcdPrefix, true)
}

func (s *ContextTestSuite) TearDownSuite() {
	// Stop the test etcd process
	s.EtcdCmd.Process.Kill()
	s.EtcdCmd.Wait()

	// Remove the test etcd data directory
	s.Require().NoError(os.RemoveAll(s.EtcdDir))
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}

func (s *ContextTestSuite) prefixKey(key string) string {
	return filepath.Join(s.EtcdPrefix, key)
}

func (s *ContextTestSuite) newFlavor() *lochness.Flavor {
	f := s.Context.NewFlavor()
	f.Image = uuid.New()
	_ = f.Save()
	return f
}

func (s *ContextTestSuite) newFWGroup() *lochness.FWGroup {
	fw := s.Context.NewFWGroup()
	_ = fw.Save()
	return fw
}

func (s *ContextTestSuite) TestNewContext() {
	s.NotNil(s.Context)
}

func (s *ContextTestSuite) TestIsKeyNotFound() {
	_, err := s.EtcdClient.Get(s.prefixKey("some-randon-non-existent-key"), false, false)

	s.Error(err)
	s.True(lochness.IsKeyNotFound(err))

	err = errors.New("some-random-non-key-not-found-error")
	s.False(lochness.IsKeyNotFound(err))
}

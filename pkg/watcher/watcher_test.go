package watcher_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/watcher"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type WatcherTestSuite struct {
	suite.Suite
	EtcdDir    string
	EtcdPrefix string
	EtcdClient *etcd.Client
	EtcdCmd    *exec.Cmd
	Watcher    *watcher.Watcher
}

func (s *WatcherTestSuite) SetupSuite() {
	// Start up a test etcd
	s.EtcdDir, _ = ioutil.TempDir("", "watcherTestEtcd-"+uuid.New())
	port := 54444
	clientURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", port+1)
	s.EtcdCmd = exec.Command("etcd",
		"-name", "watcherTest",
		"-data-dir", s.EtcdDir,
		"-initial-cluster-state", "new",
		"-initial-cluster-token", "watcherTest",
		"-initial-cluster", "watcherTest="+peerURL,
		"-initial-advertise-peer-urls", peerURL,
		"-listen-peer-urls", peerURL,
		"-listen-client-urls", clientURL,
		"-advertise-client-urls", clientURL,
	)
	s.Require().NoError(s.EtcdCmd.Start())
	s.EtcdClient = etcd.NewClient([]string{clientURL})

	// Wait for test etcd to be ready
	for !s.EtcdClient.SyncCluster() {
		time.Sleep(10 * time.Millisecond)
	}

	// s.EtcdPrefix = uuid.New()
	s.EtcdPrefix = "/lochness"
}

func (s *WatcherTestSuite) SetupTest() {
	s.Watcher, _ = watcher.New(s.EtcdClient)
}

func (s *WatcherTestSuite) TearDownTest() {
	s.NoError(s.Watcher.Close())
	_, _ = s.EtcdClient.Delete(s.EtcdPrefix, true)
}

func (s *WatcherTestSuite) TearDownSuite() {
	_ = s.EtcdCmd.Process.Kill()
	_ = s.EtcdCmd.Wait()
	_ = os.RemoveAll(s.EtcdDir)
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}

func (s *WatcherTestSuite) prefixKey(key string) string {
	return filepath.Join(s.EtcdPrefix, key)
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

func (s *WatcherTestSuite) TestNew() {
	s.NotNil(s.Watcher)
	watcher, err := watcher.New(nil)
	s.Error(err)
	s.Nil(watcher)
}

func (s *WatcherTestSuite) TestAdd() {
	tests := []struct {
		description string
		prefix      string
	}{
		{"empty", ""},
		{"no leading slash", uuid.New()},
		{"leading slash", "/addTest"},
		{"duplicate", "/addTest"},
		{"nested", "/nested/" + uuid.New()},
	}
	for _, test := range tests {
		s.NoError(s.Watcher.Add(test.prefix), test.description)
	}

	s.NoError(s.Watcher.Close())
	s.Error(s.Watcher.Add(uuid.New()), "after close should fail")
}

func (s *WatcherTestSuite) TestNextResponse() {

	prefixes := make([]string, 5)
	for i := 0; i < 5; i++ {
		prefixes[i] = uuid.New()
		// Using existing prefixes for more consistent test results. See comment
		// in Watcher.Add() internals for more details.
		_, _ = s.EtcdClient.SetDir(prefixes[i], 0)
		_ = s.Watcher.Add(prefixes[i])
	}

	go func() {
		for i := 0; i < len(prefixes); i++ {
			for j := 0; j < len(prefixes); j++ {
				_, _ = s.EtcdClient.Set(prefixes[j]+"/subkey", fmt.Sprintf("%d", i+j), 0)
			}
		}
	}()

	lastModifiedIndex := uint64(0)
	for i := len(prefixes) * len(prefixes); i > 0 && s.Watcher.Next(); i-- {
		s.NoError(s.Watcher.Err())
		resp := s.Watcher.Response()
		s.NotNil(resp, "should return a response")
		s.Equal(resp, s.Watcher.Response(), "response should only change after Next()")
		s.NotEqual(lastModifiedIndex, resp.Node.ModifiedIndex, "response should change after Next()")
		lastModifiedIndex = resp.Node.ModifiedIndex
	}

}

func (s *WatcherTestSuite) TestRemove() {
	prefix := uuid.New()
	s.Error(s.Watcher.Remove(prefix), "not watched prefix should fail")
	_ = s.Watcher.Add(prefix)
	s.NoError(s.Watcher.Remove(prefix), "watched prefix should succeed")
}

func (s *WatcherTestSuite) TestClose() {
	s.Watcher.Add(uuid.New())
	s.NoError(s.Watcher.Close())
	s.NoError(s.Watcher.Close())
}

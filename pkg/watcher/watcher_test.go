package watcher_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mistifyio/lochness/pkg/kv"
	_ "github.com/mistifyio/lochness/pkg/kv/etcd"
	"github.com/mistifyio/lochness/pkg/watcher"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestWatcherCmd(t *testing.T) {
	suite.Run(t, new(WatcherSuite))
}

type WatcherSuite struct {
	suite.Suite
	KVDir    string
	KVPrefix string
	KV       kv.KV
	KVCmd    *exec.Cmd
	Watcher  *watcher.Watcher
}

func (s *WatcherSuite) SetupSuite() {
	// Start up a test etcd
	s.KVDir, _ = ioutil.TempDir("", "watcherTest-"+uuid.New())
	port := 54444
	clientURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", port+1)
	s.KVCmd = exec.Command("etcd",
		"-name", "watcherTest",
		"-data-dir", s.KVDir,
		"-initial-cluster-state", "new",
		"-initial-cluster-token", "watcherTest",
		"-initial-cluster", "watcherTest="+peerURL,
		"-initial-advertise-peer-urls", peerURL,
		"-listen-peer-urls", peerURL,
		"-listen-client-urls", clientURL,
		"-advertise-client-urls", clientURL,
	)
	s.Require().NoError(s.KVCmd.Start())
	time.Sleep(10 * time.Millisecond) // Wait for test kv to be ready

	KV, err := kv.New(clientURL)
	if err != nil {
		panic(err)
	}

	s.KV = KV
	s.KVPrefix = "/lochness"
}

func (s *WatcherSuite) SetupTest() {
	s.Watcher, _ = watcher.New(s.KV)
}

func (s *WatcherSuite) TearDownTest() {
	s.NoError(s.Watcher.Close())
	_ = s.KV.Delete(s.KVPrefix, true)
}

func (s *WatcherSuite) TearDownSuite() {
	_ = s.KVCmd.Process.Kill()
	_ = s.KVCmd.Wait()
	_ = os.RemoveAll(s.KVDir)
}

func (s *WatcherSuite) prefixKey(key string) string {
	return filepath.Join(s.KVPrefix, key)
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

func (s *WatcherSuite) TestNew() {
	s.NotNil(s.Watcher)
	watcher, err := watcher.New(nil)
	s.Error(err)
	s.Nil(watcher)
}

func (s *WatcherSuite) TestAdd() {
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

func (s *WatcherSuite) TestNextResponse() {
	prefixes := make([]string, 5)
	for i := 0; i < 5; i++ {
		// Using existing prefixes for more consistent test results.
		// See comment in Watcher.Add() internals for more details.
		prefixes[i] = uuid.New()

		// ensure prefix is a "directory" not a key
		s.Require().NoError(s.KV.Set(prefixes[i]+"/foo", "foo"))
		s.Require().NoError(s.KV.Delete(prefixes[i]+"/foo", false))
		s.Require().NoError(s.Watcher.Add(prefixes[i]))
	}

	go func() {
		for i := 0; i < len(prefixes); i++ {
			for j := 0; j < len(prefixes); j++ {
				_ = s.KV.Set(prefixes[j]+"/subkey", fmt.Sprint(i+j))
			}
		}
	}()

	lastModifiedIndex := uint64(0)
	for i := len(prefixes) * len(prefixes); i > 0 && s.Watcher.Next(); i-- {
		s.NoError(s.Watcher.Err())
		event := s.Watcher.Event()
		s.NotNil(event, "should return an event")
		s.Equal(event, s.Watcher.Event(), "event should only change after Next()")
		s.NotEqual(lastModifiedIndex, event.Index, "response should change after Next()")
		lastModifiedIndex = event.Index
	}

}

func (s *WatcherSuite) TestRemove() {
	prefix := uuid.New()
	s.Error(s.Watcher.Remove(prefix), "not watched prefix should fail")
	_ = s.Watcher.Add(prefix)
	s.NoError(s.Watcher.Remove(prefix), "watched prefix should succeed")
}

func (s *WatcherSuite) TestClose() {
	_ = s.Watcher.Add(uuid.New())
	s.NoError(s.Watcher.Close())
	s.NoError(s.Watcher.Close())
}

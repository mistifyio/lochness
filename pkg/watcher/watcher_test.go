package watcher_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/mistifyio/lochness/internal/tests/common"
	_ "github.com/mistifyio/lochness/pkg/kv/consul"
	"github.com/mistifyio/lochness/pkg/watcher"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestWatcherCmd(t *testing.T) {
	suite.Run(t, new(WatcherSuite))
}

type WatcherSuite struct {
	common.Suite
	Watcher *watcher.Watcher
}

func (s *WatcherSuite) SetupSuite() {
	s.KVPort = 54444
	s.TestPrefix = "watcher-test"
	s.Suite.SetupSuite()
}

func (s *WatcherSuite) SetupTest() {
	s.Suite.SetupTest()
	s.Watcher, _ = watcher.New(s.KV)
}

func (s *WatcherSuite) TearDownTest() {
	s.NoError(s.Watcher.Close())
	s.Suite.TearDownTest()
}

func (s *WatcherSuite) TearDownSuite() {
	s.Suite.TearDownSuite()
}

func (s *WatcherSuite) prefixKey(key string) string {
	return filepath.Join(s.KVPrefix, key)
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

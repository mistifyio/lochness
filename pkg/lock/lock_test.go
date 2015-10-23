package lock_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/lock"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type LockTestSuite struct {
	suite.Suite
	EtcdDir    string
	EtcdPrefix string
	EtcdClient *etcd.Client
	EtcdCmd    *exec.Cmd
}

func (s *LockTestSuite) SetupSuite() {
	// Start up a test etcd
	s.EtcdDir, _ = ioutil.TempDir("", "lockTestEtcd-"+uuid.New())
	port := 54444
	clientURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", port+1)
	s.EtcdCmd = exec.Command("etcd",
		"-name", "lockTest",
		"-data-dir", s.EtcdDir,
		"-initial-cluster-state", "new",
		"-initial-cluster-token", "lockTest",
		"-initial-cluster", "lockTest="+peerURL,
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

func (s *LockTestSuite) TearDownTest() {
	_, _ = s.EtcdClient.Delete(s.EtcdPrefix, true)
}

func (s *LockTestSuite) TearDownSuite() {
	_ = s.EtcdCmd.Process.Kill()
	_ = s.EtcdCmd.Wait()
	_ = os.RemoveAll(s.EtcdDir)
}

func TestLockTestSuite(t *testing.T) {
	suite.Run(t, new(LockTestSuite))
}

func (s *LockTestSuite) prefixKey(key string) string {
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

func (s *LockTestSuite) TestAcquire() {
	// Key/Value to test conflicts
	lockKey := uuid.New()
	lockValue := uuid.New()

	tests := []struct {
		description string
		key         string
		value       string
		ttl         uint64
		blocking    bool
		expectedErr bool
	}{
		{"key missing", "", uuid.New(), 5, false, true},
		{"value missing", uuid.New(), "", 5, false, false},
		{"0 ttl", uuid.New(), uuid.New(), 0, false, false},
		{"all present", lockKey, lockValue, 5, false, false},
		{"repeated request", lockKey, lockValue, 5, false, true},
		{"lock already held, blocking", lockKey, uuid.New(), 5, true, false},
		{"lock already held, nonblocking", lockKey, uuid.New(), 5, false, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		l, err := lock.Acquire(s.EtcdClient, test.key, test.value, test.ttl, test.blocking)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(l, msg("should not return lock"))
		} else {
			s.NoError(err, msg("should acquire lock"))
			s.NotNil(l, msg("should return lock"))
		}
	}
}

func (s *LockTestSuite) TestRefresh() {
	l, _ := lock.Acquire(s.EtcdClient, uuid.New(), uuid.New(), 1, false)

	// Refresh
	s.NoError(l.Refresh(), "before expiration should succeed")

	// Expire
	time.Sleep(2 * time.Second)
	s.Error(l.Refresh(), "after expiration should fail")

	// Not held
	s.Error(l.Refresh(), "lock not held should fail")
}

func (s *LockTestSuite) TestRelease() {
	l, _ := lock.Acquire(s.EtcdClient, uuid.New(), uuid.New(), 1, false)

	// Release held lock
	s.NoError(l.Release(), "held lock should succeed")

	// Release not held lock
	s.Error(l.Release(), "not held lock should fail")

	// Release expired lock
	l, _ = lock.Acquire(s.EtcdClient, uuid.New(), uuid.New(), 1, false)
	time.Sleep(2 * time.Second)
	s.Error(l.Release(), "expired lock should fail")
}

func (s *LockTestSuite) TestJSON() {
	l, _ := lock.Acquire(s.EtcdClient, uuid.New(), uuid.New(), 5, false)
	lockBytes, err := json.Marshal(l)
	s.NoError(err)

	lockFromJSON := &lock.Lock{}
	s.NoError(json.Unmarshal(lockBytes, lockFromJSON))
	// Since all fields are unexported, convert back to JSON and compare that
	lockBytes2, err := json.Marshal(lockFromJSON)
	s.NoError(err)
	s.Equal(lockBytes, lockBytes2)
	s.NoError(lockFromJSON.Refresh())
}

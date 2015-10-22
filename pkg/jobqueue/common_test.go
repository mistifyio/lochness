package jobqueue_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type CommonTestSuite struct {
	suite.Suite
	EtcdDir    string
	EtcdPrefix string
	EtcdClient *etcd.Client
	EtcdCmd    *exec.Cmd
	BStalkAddr string
	BStalkCmd  *exec.Cmd
	Client     *jobqueue.Client
}

func (s *CommonTestSuite) SetupSuite() {
	// Start up a test etcd
	s.EtcdDir, _ = ioutil.TempDir("", "jobqueueTestEtcd-"+uuid.New())
	port := 54333
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

func (s *CommonTestSuite) SetupTest() {
	// Start up a test beanstalk
	bPort := "4321"
	s.BStalkCmd = exec.Command("beanstalkd", "-p", bPort)
	s.Require().NoError(s.BStalkCmd.Start())
	s.BStalkAddr = fmt.Sprintf("127.0.0.1:%s", bPort)

	time.Sleep(100 * time.Millisecond)
	client, err := jobqueue.NewClient(s.BStalkAddr, s.EtcdClient)
	s.Require().NoError(err)
	s.Client = client
}

func (s *CommonTestSuite) TearDownTest() {
	_, _ = s.EtcdClient.Delete(s.EtcdPrefix, true)

	_ = s.BStalkCmd.Process.Kill()
	_ = s.BStalkCmd.Wait()
}

func (s *CommonTestSuite) TearDownSuite() {
	_ = s.EtcdCmd.Process.Kill()
	_ = s.EtcdCmd.Wait()
	_ = os.RemoveAll(s.EtcdDir)
}

func (s *CommonTestSuite) prefixKey(key string) string {
	return filepath.Join(s.EtcdPrefix, key)
}

func (s *CommonTestSuite) newJob(action string) *jobqueue.Job {
	if action == "" {
		action = "restart"
	}

	context := lochness.NewContext(s.EtcdClient)
	guest := context.NewGuest()
	guest.FlavorID = uuid.New()
	guest.NetworkID = uuid.New()
	_ = guest.Save()

	j := s.Client.NewJob()
	j.Guest = guest.ID
	j.Action = action
	_ = j.Save(60 * time.Second)
	return j
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

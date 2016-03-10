package jobqueue_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/mistifyio/lochness/pkg/kv"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type JobQCommonSuite struct {
	suite.Suite
	KVDir      string
	KVPrefix   string
	KV         kv.KV
	KVCmd      *exec.Cmd
	BStalkAddr string
	BStalkCmd  *exec.Cmd
	Client     *jobqueue.Client
}

func (s *JobQCommonSuite) SetupSuite() {
	// Start up a test etcd
	s.KVDir, _ = ioutil.TempDir("", "jobqueueTest-"+uuid.New())
	port := 54333
	clientURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	peerURL := fmt.Sprintf("http://127.0.0.1:%d", port+1)
	s.KVCmd = exec.Command("etcd",
		"-name", "jobqueueTest",
		"-data-dir", s.KVDir,
		"-initial-cluster-state", "new",
		"-initial-cluster-token", "jobqueueTest",
		"-initial-cluster", "jobqueueTest="+peerURL,
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

func (s *JobQCommonSuite) SetupTest() {
	// Start up a test beanstalk
	bPort := "4321"
	s.BStalkCmd = exec.Command("beanstalkd", "-p", bPort)
	s.Require().NoError(s.BStalkCmd.Start())
	s.BStalkAddr = fmt.Sprintf("127.0.0.1:%s", bPort)

	time.Sleep(100 * time.Millisecond)
	client, err := jobqueue.NewClient(s.BStalkAddr, s.KV)
	s.Require().NoError(err)
	s.Client = client

	s.Require().NoError(s.KV.Set(s.KVPrefix+"/foo.test", "testing"))
	s.Require().NoError(s.KV.Delete(s.KVPrefix+"/foo.test", false))
}

func (s *JobQCommonSuite) TearDownTest() {
	s.Require().NoError(s.KV.Delete(s.KVPrefix, true))

	s.Require().NoError(s.BStalkCmd.Process.Kill())
	s.Require().Error(s.BStalkCmd.Wait())
}

func (s *JobQCommonSuite) TearDownSuite() {
	s.Require().NoError(s.KVCmd.Process.Kill())
	s.Require().Error(s.KVCmd.Wait())
	s.Require().NoError(os.RemoveAll(s.KVDir))
}

func (s *JobQCommonSuite) prefixKey(key string) string {
	return filepath.Join(s.KVPrefix, key)
}

func (s *JobQCommonSuite) newJob(action string) *jobqueue.Job {
	if action == "" {
		action = "restart"
	}

	context := lochness.NewContext(s.KV)
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

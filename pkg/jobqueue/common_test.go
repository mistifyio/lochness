package jobqueue_test

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/pborman/uuid"
)

type JobQCommonSuite struct {
	common.Suite
	BStalkAddr string
	BStalkCmd  *exec.Cmd
	Client     *jobqueue.Client
}

func (s *JobQCommonSuite) SetupSuite() {
	s.KVPort = 54333
	s.TestPrefix = "jobqueue-test"
	s.Suite.SetupSuite()
}

func (s *JobQCommonSuite) SetupTest() {
	s.Suite.SetupTest()

	// Start up a test beanstalk
	bPort := "4321"
	s.BStalkCmd = exec.Command("beanstalkd", "-p", bPort)
	s.Require().NoError(s.BStalkCmd.Start())
	s.BStalkAddr = fmt.Sprintf("127.0.0.1:%s", bPort)

	time.Sleep(500 * time.Millisecond)
	client, err := jobqueue.NewClient(s.BStalkAddr, s.KV)
	s.Require().NoError(err)
	s.Client = client

	s.Require().NoError(s.KV.Set(s.KVPrefix+"/foo.test", "testing"))
	s.Require().NoError(s.KV.Delete(s.KVPrefix+"/foo.test", false))
}

func (s *JobQCommonSuite) TearDownTest() {
	s.Require().NoError(s.BStalkCmd.Process.Kill())
	s.Require().Error(s.BStalkCmd.Wait())

	s.Suite.TearDownTest()
}

func (s *JobQCommonSuite) TearDownSuite() {
	s.Suite.TearDownSuite()
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
	s.Require().NoError(guest.Save())

	j := s.Client.NewJob()
	j.Guest = guest.ID
	j.Action = action
	s.Require().NoError(j.Save(60 * time.Second))
	s.Require().NoError(j.Release())
	return j
}

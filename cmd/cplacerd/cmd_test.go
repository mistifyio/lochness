package main_test

import (
	"fmt"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/stretchr/testify/suite"
)

func TestCPlacerd(t *testing.T) {
	suite.Run(t, new(CmdSuite))
}

type CmdSuite struct {
	common.Suite
	BinName        string
	BeanstalkdCmd  *exec.Cmd
	BeanstalkdPath string
	JobQueue       *jobqueue.Client
	Port           string
}

func (s *CmdSuite) SetupSuite() {
	s.Suite.SetupSuite()
	s.Require().NoError(common.Build())
	s.BinName = "cplacerd"
	s.Port = "45362"

	bPort := "59872"
	s.BeanstalkdPath = fmt.Sprintf("127.0.0.1:%s", bPort)
	s.BeanstalkdCmd = exec.Command("beanstalkd", "-p", bPort)
	s.Require().NoError(s.BeanstalkdCmd.Start())
	beanstalkdReady := false
	for i := 0; i < 10; i++ {
		if _, err := beanstalk.Dial("tcp", s.BeanstalkdPath); err == nil {
			beanstalkdReady = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().True(beanstalkdReady)

	jobQueue, err := jobqueue.NewClient(s.BeanstalkdPath, s.KV)
	s.Require().NoError(err)
	s.JobQueue = jobQueue
}

func (s *CmdSuite) TearDownTest() {
	_ = s.BeanstalkdCmd.Process.Kill()
	_ = s.BeanstalkdCmd.Wait()
	s.Suite.TearDownTest()
}

func (s *CmdSuite) TestCmd() {
	hypervisor := s.NewHypervisor()
	_, _ = lochness.SetHypervisorID(hypervisor.ID)
	subnet := s.NewSubnet()
	network := s.NewNetwork()
	_ = network.AddSubnet(subnet)
	_ = hypervisor.AddSubnet(subnet, "mistify0")

	tests := []struct {
		description  string
		jobStatus    string
		jobAction    string
		hypervisorID string
		expectedErr  bool
	}{
		{"wrong job status",
			"foobar", "select-hypervisor", "", true},
		{"wrong job action",
			jobqueue.JobStatusNew, "foobar", "", true},
		{"guest has hypervisor id",
			jobqueue.JobStatusNew, "select-hypervisor", hypervisor.ID, true},
		{"no live hypervisors",
			jobqueue.JobStatusNew, "select-hypervisor", "", true},
		{"valid",
			jobqueue.JobStatusNew, "select-hypervisor", "", false},
	}

	for _, test := range tests {
		msg := common.TestMsgFunc(test.description)
		if test.description == "valid" {
			s.NoError(hypervisor.Heartbeat(1 * time.Hour))
		}

		guest := s.NewGuest()
		guest.NetworkID = network.ID
		if test.hypervisorID != "" {
			guest.HypervisorID = test.hypervisorID
		}
		_ = guest.Save()

		job, _ := s.JobQueue.AddJob(guest.ID, "select-hypervisor")
		job.Refresh() // lock the job
		if test.jobAction != job.Action {
			job.Action = test.jobAction
		}
		if test.jobStatus != jobqueue.JobStatusNew {
			job.Status = test.jobStatus
		}
		s.Require().NoError(job.Save(1 * time.Hour))
		job.Release()

		// Start the daemon
		args := []string{
			"-p", s.Port,
			"-k", s.KVURL,
			"-b", s.BeanstalkdPath,
			"-l", "fatal",
		}
		cmd, err := common.Start("./"+s.BinName, args...)
		s.Require().NoError(err)

		// Wait for processing
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			if err := job.Refresh(); err != nil {
				continue
			}
			if job.Status == jobqueue.JobStatusError || job.Action == "fetch" {
				break
			}
			// job.Refresh acquires job lock, we need to release it
			// so binary can continue processing
			s.Require().NoError(job.Release())
		}

		_ = guest.Refresh()
		workStats, _ := s.JobQueue.StatsWork()
		totalWorkJobs, _ := strconv.Atoi(workStats["current-jobs-total"])
		if test.expectedErr {
			s.Equal(jobqueue.JobStatusError, job.Status, msg("should have errored"))
			s.NotEmpty(job.Error, msg("should have error msg"))
			s.Equal(test.hypervisorID, guest.HypervisorID, msg("should not have changed hypervisor ID"))
			s.Equal(test.jobAction, job.Action, msg("should not have changed actions"))
			s.Equal(0, totalWorkJobs, msg("should not have created new work task"))
		} else {
			s.Equal(jobqueue.JobStatusNew, job.Status, msg("should not have errored"))
			s.Empty(job.Error, msg("should not have error msg"))
			s.Equal(hypervisor.ID, guest.HypervisorID, msg("should have been assigned to hypervisor"))
			s.Equal("fetch", job.Action, msg("should have changed actions"))
			s.Equal(1, totalWorkJobs, msg("should have created new work task"))
		}

		createStats, _ := s.JobQueue.StatsCreate()
		totalCreateJobs, _ := strconv.Atoi(createStats["current-jobs-total"])
		s.Equal(0, totalCreateJobs, msg("should not have task left in create queue"))

		// Stop the daemon
		_ = cmd.Stop()
		status, err := cmd.ExitStatus()
		s.Equal(-1, status, msg("expected status code to be that of a killed process"))
	}

}

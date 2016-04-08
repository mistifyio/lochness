package jobqueue_test

import (
	"strconv"
	"testing"

	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/mistifyio/lochness/pkg/kv"
	_ "github.com/mistifyio/lochness/pkg/kv/consul"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestJobQClient(t *testing.T) {
	suite.Run(t, new(ClientSuite))
}

type ClientSuite struct {
	JobQCommonSuite
}

func (s *ClientSuite) TestNewClient() {
	tests := []struct {
		description string
		bstalkAddr  string
		kv          kv.KV
		expectedErr bool
	}{
		{"missing both", "", nil, true},
		{"missing kv", s.BStalkAddr, nil, true},
		{"missing bstalk", "", s.KV, true},
		{"invalid bstalk", "asdf", s.KV, true},
		{"not running bstalk", "127.0.0.1:12345", s.KV, true},
		{"bstalk and kv", s.BStalkAddr, s.KV, false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		c, err := jobqueue.NewClient(test.bstalkAddr, test.kv)
		if test.expectedErr {
			s.Error(err, msg("should error"))
			s.Nil(c, msg("fail should not return client"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(c, msg("success should return client"))
		}
	}
}

func (s *ClientSuite) TestAddTask() {
	j := s.newJob("select-hypervisor")
	j2 := s.newJob("")

	tests := []struct {
		description string
		job         *jobqueue.Job
		expectedErr bool
	}{
		{"no job", nil, true},
		{"select job", j, false},
		{"restart job", j2, false},
	}
	for _, test := range tests {
		msg := s.Messager(test.description)
		id, err := s.Client.AddTask(test.job)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Equal(uint64(0), id, msg("should not return an id"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotEqual(uint64(0), id, msg("should return an id"))
		}
	}
}

func (s *ClientSuite) TestDeleteTask() {
	job := s.newJob("restart")
	taskID, _ := s.Client.AddTask(job)
	s.NoError(s.Client.DeleteTask(taskID), "existing should succeed")
	s.Error(s.Client.DeleteTask(taskID), "missing should fail")
}

func (s *ClientSuite) TestNextWorkTask() {
	job := s.newJob("restart")
	taskID, _ := s.Client.AddTask(job)
	task, err := s.Client.NextWorkTask()
	s.NoError(err)
	s.Equal(taskID, task.ID)
	s.Equal(job.ID, task.JobID)
}

func (s *ClientSuite) TestNextCreateTask() {
	job := s.newJob("select-hypervisor")
	taskID, _ := s.Client.AddTask(job)
	task, err := s.Client.NextCreateTask()
	s.NoError(err)
	s.Equal(taskID, task.ID)
	s.Equal(job.ID, task.JobID)
}

func (s *ClientSuite) TestAddJob() {
	tests := []struct {
		description string
		guest       string
		action      string
		expectedErr bool
	}{
		{"missing both", "", "", true},
		{"missing guest id", "", "restart", true},
		{"missing action", uuid.New(), "", true},
		{"all present", uuid.New(), "restart", false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		job, err := s.Client.AddJob(test.guest, test.action)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(job, msg("should not return job"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(job, msg("should return job"))
		}
	}
}

func (s *ClientSuite) TestStats() {
	stats, err := s.Client.StatsCreate()
	if connErr, ok := err.(beanstalk.ConnError); ok {
		err = connErr.Err
	}
	s.Equal(beanstalk.ErrNotFound, err, "fresh tube should not be found")

	_, _ = s.Client.AddJob(uuid.New(), "select-hypervisor")

	stats, err = s.Client.StatsCreate()
	s.NoError(err, "should not error")
	totalCreateJobs, _ := strconv.Atoi(stats["current-jobs-total"])
	s.Equal(1, totalCreateJobs, "should equal current total of all job types")

	_, _ = s.Client.AddJob(uuid.New(), "foo")
	_, _ = s.Client.AddJob(uuid.New(), "bar")

	stats, err = s.Client.StatsWork()
	s.NoError(err, "should not error")
	totalWorkJobs, _ := strconv.Atoi(stats["current-jobs-total"])
	s.Equal(2, totalWorkJobs, "should equal current total of all job types")
}

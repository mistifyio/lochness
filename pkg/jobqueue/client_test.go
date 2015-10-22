package jobqueue_test

import (
	"testing"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	CommonTestSuite
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TestNewClient() {
	tests := []struct {
		description string
		bstalkAddr  string
		etcdClient  *etcd.Client
		expectedErr bool
	}{
		{"missing both", "", nil, true},
		{"missing etcd", s.BStalkAddr, nil, true},
		{"missing bstalk", "", s.EtcdClient, true},
		{"invalid bstalk", "asdf", s.EtcdClient, true},
		{"not running bstalk", "127.0.0.1:12345", s.EtcdClient, true},
		{"bstalk and etcd", s.BStalkAddr, s.EtcdClient, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		c, err := jobqueue.NewClient(test.bstalkAddr, test.etcdClient)
		if test.expectedErr {
			s.Error(err, msg("should error"))
			s.Nil(c, msg("fail should not return client"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(c, msg("success should return client"))
		}
	}
}

func (s *ClientTestSuite) TestAddTask() {
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
		msg := testMsgFunc(test.description)
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

func (s *ClientTestSuite) TestDeleteTask() {
	job := s.newJob("restart")
	taskID, _ := s.Client.AddTask(job)
	s.NoError(s.Client.DeleteTask(taskID), "existing should succeed")
	s.Error(s.Client.DeleteTask(taskID), "missing should fail")
}

func (s *ClientTestSuite) TestNextWorkTask() {
	job := s.newJob("restart")
	taskID, _ := s.Client.AddTask(job)
	task, err := s.Client.NextWorkTask()
	s.NoError(err)
	s.Equal(taskID, task.ID)
	s.Equal(job.ID, task.JobID)
}

func (s *ClientTestSuite) TestNextCreateTask() {
	job := s.newJob("select-hypervisor")
	taskID, _ := s.Client.AddTask(job)
	task, err := s.Client.NextCreateTask()
	s.NoError(err)
	s.Equal(taskID, task.ID)
	s.Equal(job.ID, task.JobID)
}

func (s *ClientTestSuite) TestAddJob() {
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
		msg := testMsgFunc(test.description)
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

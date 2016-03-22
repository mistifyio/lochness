package jobqueue_test

import (
	"testing"
	"time"

	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestJobSuite(t *testing.T) {
	suite.Run(t, new(JobSuite))
}

type JobSuite struct {
	JobQCommonSuite
}

func (s *JobSuite) TestNewJob() {
	j := s.Client.NewJob()
	s.NotNil(uuid.Parse(j.ID))
	s.Equal(jobqueue.JobStatusNew, j.Status)
}

func (s *JobSuite) TestValidate() {
	tests := []struct {
		description string
		id          string
		action      string
		guest       string
		status      string
		expectedErr bool
	}{
		{"missing id", "", "restart", uuid.New(), "new", true},
		{"missing action", uuid.New(), "", uuid.New(), "new", true},
		{"missing guest", uuid.New(), "restart", "", "new", true},
		{"missing status", uuid.New(), "restart", uuid.New(), "", true},
		{"nothing missing", uuid.New(), "restart", uuid.New(), "new", false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		j := &jobqueue.Job{
			ID:     test.id,
			Action: test.action,
			Guest:  test.guest,
			Status: test.status,
		}
		err := j.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *JobSuite) TestSave() {
	goodJob := s.Client.NewJob()
	goodJob.Action = "restart"
	goodJob.Guest = uuid.New()

	clobberJob := &jobqueue.Job{}
	*clobberJob = *goodJob
	clobberJob.Guest = uuid.New()

	tests := []struct {
		description string
		job         *jobqueue.Job
		expectedErr bool
	}{
		{"invalid job", s.Client.NewJob(), true},
		{"valid job", goodJob, false},
		{"existing job", goodJob, false},
		{"existing job clobber changes", clobberJob, true},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		err := test.job.Save(60 * time.Second)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

func (s *JobSuite) TestRefresh() {
	job := s.Client.NewJob()
	jobCopy := &jobqueue.Job{}
	*jobCopy = *job
	job.Action = "restart"
	job.Guest = uuid.New()

	s.Require().NoError(job.Save(60 * time.Second))
	s.Require().NoError(job.Release())
	s.NoError(jobCopy.Refresh(), "refresh existing should succeed")
	// For some reason, assert.ObjectsAreEqualValues doesn't work here
	s.Equal(job.Action, jobCopy.Action, "refresh should pull new data")
	s.Equal(job.Guest, jobCopy.Guest, "refresh should pull new data")

	newJob := s.Client.NewJob()
	s.Error(newJob.Refresh(), "unsaved job refresh should fail")
}

func (s *JobSuite) TestJob() {
	job := s.newJob("")

	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid id", "asdf", true},
		{"nonexistant id", uuid.New(), true},
		{"real id", job.ID, false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		j, err := s.Client.Job(test.id)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(j, msg("failure shouldn't return a job"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			// For some reason, assert.ObjectsAreEqualValues doesn't work here
			s.Equal(job.Action, j.Action, "should pull correct data")
			s.Equal(job.Guest, j.Guest, "should pull correctdata")
		}
	}
}

package jobqueue_test

import (
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestTask(t *testing.T) {
	suite.Run(t, new(TaskSuite))
}

type TaskSuite struct {
	JobQCommonSuite
}

func (s *TaskSuite) TestDelete() {
	job := s.newJob("")
	_, _ = s.Client.AddTask(job)
	task, _ := s.Client.NextWorkTask()
	_ = task.Release()

	s.NoError(task.Delete())
}

func (s *TaskSuite) TestRelease() {
	job := s.newJob("")
	_, _ = s.Client.AddTask(job)
	task1, _ := s.Client.NextWorkTask()
	s.NoError(task1.Release())
	task2, _ := s.Client.NextWorkTask()
	s.Equal(task1.ID, task2.ID)
}

func (s *TaskSuite) TestRefreshJob() {
	job := s.newJob("")
	_, _ = s.Client.AddTask(job)
	task, _ := s.Client.NextWorkTask()
	job.Action = "stop"
	_ = job.Save(60 * time.Second)

	s.NoError(task.RefreshJob())
	s.Equal(job.Action, task.Job.Action)

}

func (s *TaskSuite) TestRefreshGuest() {
	job := s.newJob("")
	_, _ = s.Client.AddTask(job)
	task, _ := s.Client.NextWorkTask()
	s.NoError(task.RefreshGuest())

	s.Equal(job.Guest, task.Guest.ID)
	s.NotNil(uuid.Parse(task.Guest.FlavorID))

	task.Job.Guest = ""
	s.Error(task.RefreshGuest())
	task.Job = nil
	s.Error(task.RefreshGuest())
}

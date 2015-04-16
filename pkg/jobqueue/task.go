package jobqueue

import (
	"errors"

	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
)

// Task is a "helper" struct to pull together information from beanstalk and
// etcd
type Task struct {
	ID    uint64 // id from beanstalkd
	JobID string // body from beanstalkd
	Job   *lochness.Job
	Guest *lochness.Guest
	conn  *beanstalk.Conn
	ctx   *lochness.Context
}

// Delete removes a task from beanstalk
func (t *Task) Delete() error {
	return t.conn.Delete(t.ID)
}

// Release releases a task back to beanstalk
func (t *Task) Release() error {
	return t.conn.Release(t.ID, priority, delay)
}

// RefreshJob reloads a task's job information
func (t *Task) RefreshJob() error {
	job, err := t.ctx.Job(t.JobID)
	if err != nil {
		return err
	}
	t.Job = job
	return nil
}

// RefreshGuest reloads a task's guest information
func (t *Task) RefreshGuest() error {
	if t.Job == nil {
		return errors.New("trying to load guest from nil job")
	}
	if t.Job.Guest == "" {
		return errors.New("job missing guest id")
	}
	guest, err := t.ctx.Guest(t.Job.Guest)
	if err != nil {
		return err
	}
	t.Guest = guest
	return nil
}

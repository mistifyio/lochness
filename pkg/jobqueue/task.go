package jobqueue

import (
	"errors"

	"github.com/mistifyio/lochness"
)

// Task is a "helper" struct to pull together information from beanstalk and the kv
type Task struct {
	ID     uint64 // id from beanstalkd
	JobID  string // body from beanstalkd
	Job    *Job
	Guest  *lochness.Guest
	client *Client
}

// Delete removes a task from beanstalk
func (t *Task) Delete() error {
	if t.Job != nil {
		if err := t.Job.Release(); err != nil {
			return err
		}
	}
	return t.client.beanConn.Delete(t.ID)
}

// Release releases a task back to beanstalk
func (t *Task) Release() error {
	if t.Job != nil {
		if err := t.Job.Release(); err != nil {
			return err
		}
	}
	return t.client.beanConn.Release(t.ID, priority, delay)
}

// RefreshJob reloads a task's job information
func (t *Task) RefreshJob() error {
	job, err := t.client.Job(t.JobID)
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
	ctx := lochness.NewContext(t.client.kv)
	guest, err := ctx.Guest(t.Job.Guest)
	if err != nil {
		return err
	}
	t.Guest = guest
	return nil
}

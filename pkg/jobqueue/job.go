package jobqueue

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"time"

	kv "github.com/coreos/go-etcd/etcd"
	"github.com/pborman/uuid"
)

var (
	// JobPath is the path in the config store
	JobPath = "lochness/jobs/"
)

// Job Status
const (
	JobStatusNew     = "new"
	JobStatusWorking = "working"
	JobStatusDone    = "done"
	JobStatusError   = "error"
)

type (
	// Job is a single job for a guest such as create, delete, etc.
	Job struct {
		ID            string    `json:"id"`
		RemoteID      string    `json:"remote"` // ID of remote hypervisor/guest job
		Action        string    `json:"action"`
		Guest         string    `json:"guest"`
		Error         string    `json:"error,omitempty"`
		Status        string    `json:"status,omitempty"`
		StartedAt     time.Time `json:"started_at,omitempty"`
		FinishedAt    time.Time `json:"finished_at,omitempty"`
		modifiedIndex uint64
		client        *Client
	}
)

// NewJob creates a new job.
func (c *Client) NewJob() *Job {
	return &Job{
		ID:     uuid.New(),
		client: c,
		Status: JobStatusNew,
	}
}

// Validate ensures required fields are populated.
func (j *Job) Validate() error {

	//XXX: use global error definitions for these?

	if j.ID == "" {
		return errors.New("ID is required")
	}

	if j.Action == "" {
		return errors.New("Action is required")
	}

	if j.Guest == "" {
		return errors.New("Guest is required")
	}

	if j.Status == "" {
		return errors.New("Status is required")
	}

	return nil
}

// key is a helper to generate the config store key.
func (j *Job) key() string {
	return filepath.Join(JobPath, j.ID)
}

// Save persists a job.
func (j *Job) Save(ttl time.Duration) error {

	if err := j.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(j)

	if err != nil {
		return err
	}

	// if we changed something, don't clobber
	var resp *kv.Response
	if j.modifiedIndex != 0 {
		resp, err = j.client.kv.CompareAndSwap(j.key(), string(v), uint64(ttl.Seconds()), "", j.modifiedIndex)
	} else {
		resp, err = j.client.kv.Create(j.key(), string(v), uint64(ttl.Seconds()))
	}
	if err != nil {
		return err
	}

	j.modifiedIndex = resp.EtcdIndex

	return nil
}

// Refresh reloads a Job from the data store.
func (j *Job) Refresh() error {
	resp, err := j.client.kv.Get(j.key(), false, false)

	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(resp.Node.Value), &j); err != nil {
		return err
	}
	j.modifiedIndex = resp.Node.ModifiedIndex

	return nil
}

// Job retrieves a single job from the data store.
func (c *Client) Job(id string) (*Job, error) {
	j := &Job{
		ID:     id,
		client: c,
	}

	if err := j.Refresh(); err != nil {
		return nil, err
	}

	return j, nil
}

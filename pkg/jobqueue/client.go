package jobqueue

import (
	"errors"
	"strconv"
	"time"

	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness/pkg/kv"
)

// Default parameters
const (
	priority     = uint32(0)
	delay        = 5 * time.Second
	ttr          = 5 * time.Second
	timeout      = 10 * time.Hour
	reserveDelay = 5 * time.Second
	jobTTL       = 24 * time.Hour
)

// Client is for interacting with the job queue
type Client struct {
	beanConn *beanstalk.Conn
	kv       kv.KV
	tubes    *tubes
}

// NewClient creates a new Client and initializes the beanstalk connection + tubes
func NewClient(bstalk string, kv kv.KV) (*Client, error) {
	if kv == nil {
		return nil, errors.New("kv must not be nil")
	}

	conn, err := beanstalk.Dial("tcp", bstalk)
	if err != nil {
		return nil, err
	}

	client := &Client{
		beanConn: conn,
		kv:       kv,
		tubes:    newTubes(conn),
	}
	return client, nil
}

// AddTask creates a new task in the appropriate beanstalk queue
func (c *Client) AddTask(j *Job) (uint64, error) {
	if j == nil {
		return 0, errors.New("missing job")
	}

	ts := c.tubes.work
	if j.Action == "select-hypervisor" {
		ts = c.tubes.create
	}
	id, err := ts.Put(j.ID)
	return id, err
}

// DeleteTask removes a task from beanstalk by id
func (c *Client) DeleteTask(id uint64) error {
	return c.beanConn.Delete(id)
}

// NextCreateTask returns the next task from the create tube
func (c *Client) NextCreateTask() (*Task, error) {
	task, err := c.nextTask(c.tubes.create)
	return task, err
}

// NextWorkTask returns the next task from the work tube
func (c *Client) NextWorkTask() (*Task, error) {
	task, err := c.nextTask(c.tubes.work)
	return task, err
}

// nextTask returns the next task from a tubeSet and loads the Job and Guest
func (c *Client) nextTask(ts *tubeSet) (*Task, error) {
	id, body, err := ts.Reserve()
	if err != nil {
		return nil, err
	}

	// Build the Task object
	task := &Task{
		ID:     id,
		JobID:  body,
		client: c,
	}

	// Load the Job and Guest
	if err := task.RefreshJob(); err != nil {
		return task, err
	}
	if err := task.RefreshGuest(); err != nil {
		return task, err
	}

	return task, err
}

// AddJob creates a new job for a guest and adds a task for it
func (c *Client) AddJob(guestID, action string) (*Job, error) {
	job := c.NewJob()
	job.Guest = guestID
	job.Action = action
	if err := job.Save(jobTTL); err != nil {
		return nil, err
	}
	_, err := c.AddTask(job)
	return job, err
}

func tubeStats(tube *tubeSet) (map[string]string, error) {
	stats, err := tube.publish.Stats()
	if err != nil {
		return nil, err
	}
	ready, _ := strconv.Atoi(stats["current-jobs-ready"])
	reserved, _ := strconv.Atoi(stats["current-jobs-reserved"])
	buried, _ := strconv.Atoi(stats["current-jobs-buried"])
	delayed, _ := strconv.Atoi(stats["current-jobs-delayed"])
	stats["current-jobs-total"] = strconv.Itoa(ready + reserved + buried + delayed)
	return stats, err
}

// StatsCreate returns the stats for the create queue
func (c *Client) StatsCreate() (map[string]string, error) {
	return tubeStats(c.tubes.create)
}

// StatsWork returns the stats for the work queue
func (c *Client) StatsWork() (map[string]string, error) {
	return tubeStats(c.tubes.work)
}

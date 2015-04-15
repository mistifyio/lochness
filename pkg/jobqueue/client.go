package jobqueue

import (
	"time"

	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
)

// Beanstalk parameters
const (
	priority     = uint32(0)
	delay        = 5 * time.Second
	ttr          = 5 * time.Second
	timeout      = 10 * time.Hour
	reserveDelay = 5 * time.Second
)

// Client is for interacting with the job queue
type Client struct {
	conn  *beanstalk.Conn
	ctx   *lochness.Context
	tubes *tubes
}

// NewClient creates a new Client and initializes the beanstalk connection + tubes
func NewClient(bstalk string, ctx *lochness.Context) (*Client, error) {
	conn, err := beanstalk.Dial("tcp", bstalk)
	if err != nil {
		return nil, err
	}

	client := &Client{
		conn:  conn,
		ctx:   ctx,
		tubes: newTubes(conn),
	}
	return client, nil
}

// AddTask creates a new task in the appropriate beanstalk queue
func (c *Client) AddTask(j *lochness.Job) (uint64, error) {
	ts := c.tubes.work
	if j.Action != "select-hypervisor" {
		ts = c.tubes.create
	}
	id, err := ts.Put(j.ID)
	return id, err
}

// DeleteTask removes a task from beanstalk by id
func (c *Client) DeleteTask(id uint64) error {
	return c.conn.Delete(id)
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
		ID:    id,
		JobID: body,
		conn:  c.conn,
		ctx:   c.ctx,
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

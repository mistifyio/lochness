# jobqueue

[![jobqueue](https://godoc.org/github.com/mistifyio/lochness/pkg/jobqueue?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/jobqueue)

Package jobqueue manages the lochness guest job queue. Jobs are stored in a kv,
with references placed in beanstalk tubes for processing.

## Usage

```go
const (
	JobStatusNew     = "new"
	JobStatusWorking = "working"
	JobStatusDone    = "done"
	JobStatusError   = "error"
)
```
Job Status

```go
var (
	// JobPath is the path in the config store
	JobPath = "/lochness/jobs/"
)
```

#### type Client

```go
type Client struct {
}
```

Client is for interacting with the job queue

#### func  NewClient

```go
func NewClient(bstalk string, kv kv.KV) (*Client, error)
```
NewClient creates a new Client and initializes the beanstalk connection + tubes

#### func (*Client) AddJob

```go
func (c *Client) AddJob(guestID, action string) (*Job, error)
```
AddJob creates a new job for a guest and adds a task for it

#### func (*Client) AddTask

```go
func (c *Client) AddTask(j *Job) (uint64, error)
```
AddTask creates a new task in the appropriate beanstalk queue

#### func (*Client) DeleteTask

```go
func (c *Client) DeleteTask(id uint64) error
```
DeleteTask removes a task from beanstalk by id

#### func (*Client) Job

```go
func (c *Client) Job(id string) (*Job, error)
```
Job retrieves a single job from the data store.

#### func (*Client) NewJob

```go
func (c *Client) NewJob() *Job
```
NewJob creates a new job.

#### func (*Client) NextCreateTask

```go
func (c *Client) NextCreateTask() (*Task, error)
```
NextCreateTask returns the next task from the create tube

#### func (*Client) NextWorkTask

```go
func (c *Client) NextWorkTask() (*Task, error)
```
NextWorkTask returns the next task from the work tube

#### func (*Client) StatsCreate

```go
func (c *Client) StatsCreate() (map[string]string, error)
```
StatsCreate returns the stats for the create queue

#### func (*Client) StatsWork

```go
func (c *Client) StatsWork() (map[string]string, error)
```
StatsWork returns the stats for the work queue

#### type Job

```go
type Job struct {
	ID         string    `json:"id"`
	RemoteID   string    `json:"remote"` // ID of remote hypervisor/guest job
	Action     string    `json:"action"`
	Guest      string    `json:"guest"`
	Error      string    `json:"error,omitempty"`
	Status     string    `json:"status,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}
```

Job is a single job for a guest such as create, delete, etc.

#### func (*Job) Refresh

```go
func (j *Job) Refresh() error
```
Refresh reloads a Job from the data store.

#### func (*Job) Release

```go
func (j *Job) Release() error
```
Release releases control of the Job so that another component may acquire it.

#### func (*Job) Save

```go
func (j *Job) Save(ttl time.Duration) error
```
Save persists a job.

#### func (*Job) Validate

```go
func (j *Job) Validate() error
```
Validate ensures required fields are populated.

#### type Task

```go
type Task struct {
	ID    uint64 // id from beanstalkd
	JobID string // body from beanstalkd
	Job   *Job
	Guest *lochness.Guest
}
```

Task is a "helper" struct to pull together information from beanstalk and the kv

#### func (*Task) Delete

```go
func (t *Task) Delete() error
```
Delete removes a task from beanstalk

#### func (*Task) RefreshGuest

```go
func (t *Task) RefreshGuest() error
```
RefreshGuest reloads a task's guest information

#### func (*Task) RefreshJob

```go
func (t *Task) RefreshJob() error
```
RefreshJob reloads a task's job information

#### func (*Task) Release

```go
func (t *Task) Release() error
```
Release releases a task back to beanstalk

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

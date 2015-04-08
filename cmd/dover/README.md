# dover

[![dover](https://godoc.org/github.com/mistifyio/lochness/cmd/dover?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/dover)

Dover is the worker daemon for guest actions. It takes tasks off of a beanstalk
queue, communicates with agents to perform the work, and updates guest metadata.

### Command Usage

    $ dover -h
    Usage of dover:
    -b, --beanstalk="127.0.0.1:11300": address of beanstalkd server
    -e, --etcd="http://127.0.0.1:4001": address of etcd server
    -p, --http=7544: http port to publish metrics. set to 0 to disable
    -l, --log-level="warn": log level

Multiple instances may be run at the same time.

### Guest Action Workflow
https://github.com/mistifyio/lochness/wiki/Guest-Action-%22Workflows%22
## Usage

```go
const (
	// WorkTube is the name of the beanstalk tube for work tasks
	WorkTube = "work"
)
```

#### type Task

```go
type Task struct {
	ID    uint64 //id from beanstalkd
	Body  []byte // body from beanstalkd
	Job   *lochness.Job
	Guest *lochness.Guest
}
```

Task is a "helper" struct to pull together information from beanstalk and etcd

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

# loveland

[![loveland](https://godoc.org/github.com/mistifyio/lochness/cmd/loveland?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/loveland)

loveland is the guest placement daemon. It monitors a beanstalk queue for
requests to create new guests. It then decides which hypervisor a new guest
should be created under based on a variety of criteria. It does not actually
communicate with the hypervisor, but creates the job for `dover` to process.

### Command Usage

    $ loveland -h
    Usage of loveland:
    -b, --beanstalk="127.0.0.1:11300": address of beanstalkd server
    -e, --etcd="http://127.0.0.1:4001": address of etcd server
    -p, --http=7543: address for http interface. set to 0 to disable
    -l, --log-level="warn": log level

Only one instance should be run per cluster, typically ensured by running it via
`lock`.

### Guest Action Workflow
https://github.com/mistifyio/lochness/wiki/Guest-Action-%22Workflows%22
## Usage

```go
const (
	CreateTube = "create"
	WorkTube   = "work"
)
```
XXX: allow different tube names?

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

#### type TaskFunc

```go
type TaskFunc struct {
}
```

TaskFunc is a convenience wrapper for function calls on tasks

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

# cworkerd

[![cworkerd](https://godoc.org/github.com/mistifyio/lochness/cmd/cworkerd?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/cworkerd)

cworkerd is the worker daemon for guest actions. It takes tasks out of a
beanstalk queue, communicates with agents to perform the work, and updates guest
metadata.


### Command Usage

The following arguments are understood:

    $ cworkerd -h
    Usage of cworkerd:
    -a, --agent-port=8080: port on which agents listen
    -b, --beanstalk="127.0.0.1:11300": address of beanstalkd server
    -k, --kv="http://127.0.0.1:4001": address of kv server
    -p, --http=7544: http port to publish metrics. set to 0 to disable
    -l, --log-level="warn": log level

Multiple instances may be run at the same time.

### Guest Action Workflow
https://github.com/mistifyio/lochness/wiki/Guest-Action-%22Workflows%22


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

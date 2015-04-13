# lock

[![lock](https://godoc.org/github.com/mistifyio/lochness/cmd/lock?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/lock)

lock gurantees cluster wide singleton services for non-cluster aware programs.
The service is run by systemd, but does not need to have any integration with
it. We use systemd in order to make use of its one-cgroup-per-service
functionality and the automatic killing of everything in said cgroup when the
service is done.

The `lock` command is not the program that actually starts the service; it takes
care of parsing the command line, finding the program to run, and acquiring the
lock in `etcd`. Once the setup has been handled, it will generate and start a
systemd service (locker service) which is charged with starting the user
supplied program, as a systemd service. Such nested, so systemd. I know.

The systemd love fest is actually for good reason. The locker service is
configured with the `WatchdogSec=ttl` property so that if it hangs while trying
to do work systemd will kill it. Meanwhile, the user service is configured with
`BindsTo=locker.service` so that if the locker service dies (via watchdog or
other) the user service is killed. And since all the services are in their own
cgroup, when the service dies all child processes will be killed, *hooray*!


### Usage

The following arguments are understood:

    $ lock -h
    Usage of lock: [options] -- command args
    -b, --block=false: Block if we failed to acquire the lock
    -e, --etcd="http://localhost:4001": address of etcd machine
    -i, --interval=30: Interval in seconds to refresh lock
    -k, --key="/lock": Key to use as lock
    -t, --ttl=0: TTL for key in seconds, leave 0 for (2 * interval)

    command will be run with args via fork/exec not a shell


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

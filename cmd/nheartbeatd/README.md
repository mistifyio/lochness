# nheartbeatd

[![nheartbeatd](https://godoc.org/github.com/mistifyio/lochness/cmd/nheartbeatd?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/nheartbeatd)

nheartbeatd periodically confirms that the hypervisor node is alive and updates
the resource usage in etcd.


### Usage

The following arguments are understood:

    $ nheartbeatd -h
    Usage of nheartbeatd:
    -e, --etcd="http://localhost:4001": address of etcd machine
    -d, --id="": hypervisor id
    -i, --interval=60: update interval in seconds
    -t, --ttl=0: heartbeat ttl in seconds


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

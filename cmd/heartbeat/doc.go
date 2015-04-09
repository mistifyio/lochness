/*
heartbeat periodically confirms that the hypervisor node is alive.

Usage

The following arguments are understood:

	$ heartbeat -h
	Usage of heartbeat:
	-e, --etcd="http://localhost:4001": address of etcd machine
	-d, --id="": hypervisor id
	-i, --interval=60: update interval in seconds
	-t, --ttl=0: heartbeat ttl in seconds
*/
package main

/*
nheartbeatd periodically confirms that the hypervisor node is alive and updates the resource usage in a kv.

Usage

The following arguments are understood:

	$ nheartbeatd -h
	Usage of nheartbeatd:
	-k, --kv="http://localhost:4001": address of kv machine
	-d, --id="": hypervisor id
	-i, --interval=60: update interval in seconds
	-t, --ttl=0: heartbeat ttl in seconds
*/
package main

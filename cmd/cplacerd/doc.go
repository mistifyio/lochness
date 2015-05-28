/*
cplacerd is the guest placement daemon. It monitors a beanstalk queue for
requests to create new guests. It then decides which hypervisor a new guest
should be created under based on a variety of criteria. It does not actually
communicate with the hypervisor, but creates the job for `cworkerd` to process.

Usage

The following arguments are understood:

	$ cplacerd -h
	Usage of cplacerd:
	-b, --beanstalk="127.0.0.1:11300": address of beanstalkd server
	-e, --etcd="http://127.0.0.1:4001": address of etcd server
	-p, --http=7543: address for http interface. set to 0 to disable
	-l, --log-level="warn": log level

Only one instance should be run per cluster, typically ensured by running it via `lock`.

Guest Action Workflow
https://github.com/mistifyio/lochness/wiki/Guest-Action-%22Workflows%22
*/
package main

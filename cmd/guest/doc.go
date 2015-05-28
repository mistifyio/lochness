/*
guest is the command line interface to cguestd, the guest management service.
guest can list/create/modify/delete guests.

Usage

The following arguments are understood:

	Usage:
	guest [flags]
	guest [command]

	Available Commands:
	list        List the guests
	create      Create guests asynchronously
	modify      Modify guests
	delete      Delete guests asynchronously
	shutdown    Shutdown guests asynchronously
	reboot      Reboot guests asynchronously
	restart     Restart guests asynchronously
	poweroff    Poweroff guests asynchronously
	start       Start guests asynchronously
	suspend     Suspend guests asynchronously
	job         Check status of guest jobs
	help        Help about any command

	Flags:
	-h, --help=false: help for guest
	-j, --json=false: output in json
	-s, --server="http://localhost:18000/": server address to connect to


	Use "guest help [command]" for more information about a command.

Input is supported via command line or stdin. Async actions queue a job, which
can be checked on with the job command.

Output

All commands except job support two output formats, a list of ids or a list of JSON
objects, line separated. IDs are guest ids for synchronsous actions, job ids for async actions. The JSON is
a lochness.Guest for synchronous actions or the following for async actions:

	{
		"id": "1234abcd-1234-abcd-1234-abcd1234abcd", // Job ID
		"guest": {...}
	}

The job command either returns the job id or a JSON lochness.Job.

Examples

List guests

	$ guest list
	1d1af312-1100-49e2-b3ad-09532ffc4e77
	e41a5a67-b37b-4591-8f74-c1bd997ade84

	$ guest list -j
	{"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"1d1af312-1100-49e2-b3ad-09532ffc4e77","ip":"10.100.101.34","mac":"e3:80:38:b2:28:a1","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}
	{"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e41a5a67-b37b-4591-8f74-c1bd997ade84","ip":"10.100.101.55","mac":"7f:e3:d6:59:22:bd","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}

	$ guest list -j 1d1af312-1100-49e2-b3ad-09532ffc4e77
	{"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"1d1af312-1100-49e2-b3ad-09532ffc4e77","ip":"10.100.101.34","mac":"e3:80:38:b2:28:a1","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}

Create guests

	$ guest create '{"bridge":"br0", "flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234", "ip":"10.100.101.66", "mac":"A4-75-C1-6B-E3-49", "network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}' '{"bridge":"br0", "flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234", "ip":"10.100.101.66", "mac":"A4-75-C1-6B-E3-49", "network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}'
	fbd0c7c2-5532-4abc-b6d8-c0cef0e8c1eb
	52a27964-aeb8-49b5-9267-b3e98571e32d

	$ guest create -j '{"bridge":"br0", "flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234", "ip":"10.100.101.66", "mac":"A4-75-C1-6B-E3-49", "network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}'
	{"id":"fbd0c7c2-5532-4abc-b6d8-c0cef0e8c1eb","guest":{"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e217e622-b30b-41c1-87ac-a249152b3f32","ip":"10.100.101.66","mac":"a4:75:c1:6b:e3:49","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}}

Modify guests

	$ guest modify e2aae131-eff7-41ae-8541-73a48eb5295d '{"type":"qwerty"}' 41a7d3ca-685e-4a57-bc61-dce3e33b6b09 '{"type":"zxcv"}'
	e2aae131-eff7-41ae-8541-73a48eb5295d
	41a7d3ca-685e-4a57-bc61-dce3e33b6b09

	$ guest modify -j e2aae131-eff7-41ae-8541-73a48eb5295d '{"type":"qwerty"}'
	{"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e2aae131-eff7-41ae-8541-73a48eb5295d","ip":"10.100.101.66","mac":"a4:75:c1:6b:e3:49","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"qwerty"}

Delete guests (also applies to shutdown, reboot, restart, poweroff, start,
suspend)

	$ guest delete 41a7d3ca-685e-4a57-bc61-dce3e33b6b09 41a7d3ca-685e-4a57-bc61-dce3e33b6b09
	14e13848-e449-405a-ae04-b4bbc9016ac5

	$ guest delete -j e2aae131-eff7-41ae-8541-73a48eb5295d
	{"id":"14e13848-e449-405a-ae04-b4bbc9016ac5","guest":{"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e2aae131-eff7-41ae-8541-73a48eb5295d","ip":"10.100.101.66","mac":"a4:75:c1:6b:e3:49","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"qwerty"}}

Job status

	$ guest job a18d2ad3-64ed-47cd-9b3b-733542b9b51c
	a18d2ad3-64ed-47cd-9b3b-733542b9b51c

	$ guest job -j a18d2ad3-64ed-47cd-9b3b-733542b9b51c
	{"action":"select-hypervisor","finished_at":"0001-01-01T00:00:00Z","guest":"2bc2e856-8e79-4b83-9681-2eae31718275","id":"a18d2ad3-64ed-47cd-9b3b-733542b9b51c","remote":"","started_at":"0001-01-01T00:00:00Z","status":"new"}
*/
package main

/*
waheela is the guest management service. It exposes functionality over an HTTP
API with JSON formatting.

Command Usage

	$ waheela -h
	Usage of waheela:
	-e, --etcd="http://localhost:4001": address of etcd machine
	-l, --log-level="warn": log level
	-p, --port=18000: listen port
	-s, --statsd="": statsd address

HTTP API Endpoints

	/guests
		* GET - Retrieve a list of guests
		* POST - Create a new guest
	/guests/{guestID}
		* GET    - Retrieve information about a guest
		* PATCH  - Update information for a guest
		* DELETE - Delete a guest

Example Structs

Guest - lochness.Guest

	{
		"id": "f2011319-ad59-42fb-9bad-92e261f0651c",
		"metadata": {},
		"type": "",
		"flavor": "fe6de923-7230-416e-89d7-374b4b7b9362",
		"hypervisor": "e88a75a6-7ae6-487c-9634-6553d3793437",
		"network": "ac258bc2-4fc4-4713-a6fd-fc1afb65cd32",
		"subnet": "c6430cba-648a-41aa-aee4-b59dacfc790d",
		"fwgroup": "ecf5f19a-83e3-4dff-8f03-871d0d13ae65",
		"mac": "01:23:45:67:89:ac",
		"ip": "10.10.10.28",
		"bridge": "br0"
	}

Example Requests

GET /guests

	$ curl http://localhost:18000/guests

	[{"id":"f2011319-ad59-42fb-9bad-92e261f0651c","metadata":{},"type":"","flavor":"fe6de923-7230-416e-89d7-374b4b7b9362","hypervisor":"e88a75a6-7ae6-487c-9634-6553d3793437","network":"ac258bc2-4fc4-4713-a6fd-fc1afb65cd32","subnet":"c6430cba-648a-41aa-aee4-b59dacfc790d","fwgroup":"ecf5f19a-83e3-4dff-8f03-871d0d13ae65","mac":"01:23:45:67:89:ac","ip":"10.10.10.28","bridge":"br0"},{"id":"ad762efc-3c23-402b-8e1f-a248a005efb9","metadata":{},"type":"","flavor":"1f5acce3-96b4-4ccb-865f-e6c44f68900d","hypervisor":"e88a75a6-7ae6-487c-9634-6553d3793437","network":"ac258bc2-4fc4-4713-a6fd-fc1afb65cd32","subnet":"c6430cba-648a-41aa-aee4-b59dacfc790d","fwgroup":"9b2342a9-c1c1-4410-9b25-5984485cd247","mac":"01:23:45:67:89:ab","ip":"10.10.10.231","bridge":"br0"}]

POST /guests

	$ curl -XPOST http://localhost:18000/guests --data-binary '{"bridge":"br0", "flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234", "ip":"10.100.101.66", "mac":"A4-75-C1-6B-E3-49", "network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}'

	{"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}

GET /guests/{guestID}

	$ curl http://localhost:18000/guests/94ea0ba1-5ec2-460e-9c2e-8269593cdad3

	{"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}

PATCH /guests/{guestID}

	$ curl -X PATCH http://localhost:18000/guests/94ea0ba1-5ec2-460e-9c2e-8269593cdad3 --data-binary '{"metadata":{"foo":"bar"}}'

	{"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{"foo":"bar"},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}

DELETE /guests/{guestID}

	$ curl -X DELETE http://localhost:18000/guests/94ea0ba1-5ec2-460e-9c2e-8269593cdad3

	{"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{"foo":"bar"},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}
*/
package main

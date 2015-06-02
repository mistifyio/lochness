# chypervisord

[![chypervisord](https://godoc.org/github.com/mistifyio/lochness/cmd/chypervisord?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/chypervisord)

chypervisord is the hypervisor management service. It exposes functionality over
an HTTP API with JSON formatting.


### Usage

The following arguments are understood:

    $ chypervisord -h
    Usage of chypervisord:
    -e, --etcd="http://localhost:4001": address of etcd machine
    -l, --log-level="warn": log level
    -p, --port=17000: listen port

### HTTP API Endpoints

    /hypervisors
    	* GET  - Retrieve a list of hypervisors
    	* POST - Add a new hypervisor

    /hypervisors/{hypervisorID}
    	* GET 	 - Retrieve information about a hypervisor
    	* PATCH	 - Update a hypervisor's information
    	* DELETE - Remove a hypervisor

    /hypervisors/{hypervisorID}/config
    	* GET   - Retrieve a hypervisor's configuration
    	* PATCH - Update a hypervisor's configuration

    /hypervisors/{hypervisorID}/subnets
    	* GET   - Retrieve a list of subnets associated with the hypervisor
    	* PATCH - Update the list of subnets associated with the hypervisor

    /hypervisors/{hypervisorID}/subnets/{subnetID}
    	* DELETE - Remove a single subnet from a hypervisor

    /hypervisors/{hypervisorID}/guests
    	* GET - Retrieve a list of guests running under the hypervisor


### Example Structs

Hypervisor - lochness.Hypervisor

    {
    	"id": "abcd1234-abcd-1234-abcd-1234abcd1234",
    	"metadata": {},
    	"ip": "10.100.101.35",
    	"netmask": "255.255.255.255",
    	"gateway": "10.100.101.35",
    	"mac": "01:23:45:67:89:ac",
    	"total_resources": {
    		"memory": 1024,
    		"disk": 1024,
    		"cpu": 1
    	},
    	"available_resources": {
    		"memory": 1024,
    		"disk": 1024,
    		"cpu": 1
    	}
    }

Config - map of string keys and string values

    {
    	"foo": "bar"
    }

Subnet - map of string subnet ids to string interfaces

    	{
    		"c6430cba-648a-41aa-aee4-b59dacfc790d": "br0"
        }


### Example Requests

GET /hypervisors

    $ curl http://localhost:17000/hypervisors

    [{"id":"e88a75a6-7ae6-487c-9634-6553d3793437","metadata":{},"ip":"10.100.101.34","netmask":"","gateway":"","mac":"01:23:45:67:89:ab","total_resources":{"memory":0,"disk":0,"cpu":0},"available_resources":{"memory":0,"disk":0,"cpu":0}}]

POST /hypervisors

    $ curl -XPOST http://localhost:17000/hypervisors --data-binary '{"id":"abcd1234-abcd-1234-abcd-1234abcd1234","metadata":{},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":1024,"disk":1024,"cpu":1},"available_resources":{"memory":1024,"disk":1024,"cpu":1}}'

    {"id":"abcd1234-abcd-1234-abcd-1234abcd1234","metadata":{},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":1024,"disk":1024,"cpu":1},"available_resources":{"memory":1024,"disk":1024,"cpu":1}}

GET /hypervisors/{hypervisorID}

    $ curl http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234

    {"id":"abcd1234-abcd-1234-abcd-1234abcd1234","metadata":{},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":1024,"disk":1024,"cpu":1},"available_resources":{"memory":1024,"disk":1024,"cpu":1}}

PATCH /hypervisors/{hypervisorID}

    $ curl -XPATCH http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234 --data-binary '{"metadata":{"foo":"bar"}}'

    {"id":"abcd1234-abcd-1234-abcd-1234abcd1234","metadata":{"foo":"bar"},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":0,"disk":0,"cpu":0},"available_resources":{"memory":0,"disk":0,"cpu":0}}

DELETE /hypervisors/{hypervisorID}

    $ curl -XDELETE http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234

    {"id":"abcd1234-abcd-1234-abcd-1234abcd1234","metadata":{"foo":"bar"},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":0,"disk":0,"cpu":0},"available_resources":{"memory":0,"disk":0,"cpu":0}}

GET /hypervisors/{hypervisorID}/config

    $ curl http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config

    {"bar":"baz"}

PATCH /hypervisors/{hypervisorID}/config

    $ curl -XPATCH http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config --data-binary '{"bar":"","foobar":"asdf"}'

    {"foobar":"asdf"}

GET /hypervisors/{hypervisorID}/subnets

    $ curl http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/subnets

    {"c6430cba-648a-41aa-aee4-b59dacfc790d":"br0"}

PATCH /hypervisors/{hypervisorID}/subnets

    $ curl -XPATCH http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/subnets --data-binary '{"c6430cba-648a-41aa-aee4-b59dacfc790d":"br0"}'

    {"c6430cba-648a-41aa-aee4-b59dacfc790d":"br0"}

DELETE /hypervisors/{hypervisorID}/subnets/{subnetID}

    $ curl -XDELETE http://localhost:17000/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/subnets/c6430cba-648a-41aa-aee4-b59dacfc790d

    {}

GET /hypervisors/{hypervisorID}/guests

    $ curl http://localhost:17000/hypervisors/e88a75a6-7ae6-487c-9634-6553d3793437/guests

    ["ad762efc-3c23-402b-8e1f-a248a005efb9","f2011319-ad59-42fb-9bad-92e261f0651c"]


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

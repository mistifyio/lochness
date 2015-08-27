# cnetworkd

[![cnetworkd](https://godoc.org/github.com/mistifyio/lochness/cmd/cnetworkd?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/cnetworkd)

cnetworkd is the network configuration management service. It exposes
functionality over an HTTP API with JSON formatting.


### Usage

The following arguments are understood:

    ./cnetworkd -h
    Usage of ./cnetworkd:
    -e, --etcd="http://localhost:4001": address of etcd machine
    -l, --log-level="warn": log level
    -p, --port=19000: listen port

HTTP API endpoints

    /vlans/tags
    	* GET - Retrieve a list of VLAN tags
    	* POST - Add a new VLAN tag

    /vlans/tags/{vlanTag}
    	* GET - Retrieve information about a VLAN tag
    	* PATCH - Update a VLAN tag's information
    	* DELETE - Remove a VLAN tag

    /vlans/tags/{vlanTag}/groups
    	* GET - Retrieve a list of VLAN groups the VLAN tag belongs to
    	* POST - Set the list of VLAN groups the VLAN tag belongs to

    /vlans/groups
    	* GET - Retrieve a list of VLAN groups
    	* POST - Add a new VLAN tag

    /vlans/groups/{vlanGroupID}
    	* GET - Retrieve information about a VLAN group
    	* PATCH - Update a VLAN group's information
    	* DELETE - Remove a VLAN group

    /vlans/tags/{vlanGroupID}/tags
    	* GET - Retrieve a list of VLAN tags the VLAN group contains
    	* POST - Set the list of VLAN tags the VLAN group contains


### Example Structs

VLAN tag - lochness.VLAN

    {
    	"tag": 219,
    	"description": "foobar"
    }

VLAN Group - lochness.VLANGroup

    {
    	"id": "122be0b1-d621-4bf5-8b6b-6d0ce41d7c11",
    	"description": "foobar",
    	"metadata": {}
    }


### Example Requests

GET /vlans/tags

    $ curl http://localhost:19000/vlans/tags
    [{"tag":219,"description":"baz"}]

POST /vlans/tags

    $ curl -X POST http://localhost:19000/vlans/tags --data-binary '{"tag":123,"description":"another tag"}'
    {"tag":123,"description":"another tag"}

GET /vlans/tags/{vlanTag}

    $ curl http://localhost:19000/vlans/tags/219
    {"tag":219,"description":"baz"}

PATCH /vlans/tags/{vlanTag}

    $ curl -X PATCH http://localhost:19000/vlans/tags/123 --data-binary '{"description":"updated description"}'
    {"tag":123,"description":"updated description"}

DELETE /vlans/tags/{vlanTag}

    $ curl -X DELETE http://localhost:19000/vlans/tags/123
    {"tag":123,"description":"updated description"}

GET /vlans/tags/{vlanTag}/groups

    $ curl -X GET http://localhost:19000/vlans/tags/219/groups
    ["122be0b1-d621-4bf5-8b6b-6d0ce41d7c11"]

POST /vlans/tags/{vlanTag}/groups

    $ curl -X POST http://localhost:19000/vlans/tags/219/groups --data-binary '["122be0b1-d621-4bf5-8b6b-6d0ce41d7c11"]'
    ["122be0b1-d621-4bf5-8b6b-6d0ce41d7c11"]

GET /vlans/groups

    $ curl http://localhost:19000/vlans/groups
    [{"id":"122be0b1-d621-4bf5-8b6b-6d0ce41d7c11","description":"a group!","metadata":{}}]

POST /vlans/groups

    $ curl -X POST http://localhost:19000/vlans/groups --data-binary '{"description":"another group"}'
    {"id":"89be5c67-3ddc-4ca3-acbb-f78d96181aa9","description":"another group","metadata":{}}

GET /vlans/groups/{vlanGroupID}

    $ curl http://localhost:19000/vlans/groups/122be0b1-d621-4bf5-8b6b-6d0ce41d7c11
    {"id":"122be0b1-d621-4bf5-8b6b-6d0ce41d7c11","description":"a group!","metadata":{}}

PATCH /vlans/groups/{vlanGroupID}

    $ curl -X PATCH http://localhost:19000/vlans/groups/89be5c67-3ddc-4ca3-acbb-f78d96181aa9 --data-binary '{"description":"updated description"}'
    {"id":"89be5c67-3ddc-4ca3-acbb-f78d96181aa9","description":"updated description","metadata":{}}

DELETE /vlans/groups/{vlanGroupID}

    $ curl -X DELETE http://localhost:19000/vlans/groups/89be5c67-3ddc-4ca3-acbb-f78d96181aa9
    {"id":"89be5c67-3ddc-4ca3-acbb-f78d96181aa9","description":"updated description","metadata":{}}

GET /vlans/tags/{vlanGroupID}/tags

    $ curl -X GET http://localhost:19000/vlans/groups/122be0b1-d621-4bf5-8b6b-6d0ce41d7c11/tags
    [219]

POST /vlans/tags/{vlanGroupID}/tags

    $ curl -X POST http://localhost:19000/vlans/groups/122be0b1-d621-4bf5-8b6b-6d0ce41d7c11/tags --data-binary '[219]'
    [219]


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

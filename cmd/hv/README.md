# hv

[![hv](https://godoc.org/github.com/mistifyio/lochness/cmd/hv?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/hv)

hv is the command line interface to chypervisord, the hypervisor management
service. hv can list/modify/delete hypervisors, hypervisor guests, hypervisors
subnets, and hypervisor configs.

All commands support dual output formats, a tree like output for humans
(default) or a json output for further processing.

Most commands accept 0 or many arguments, a couple require at least 1 argument.


### Usage

The following arguments are understood:

    $ hv -h
    hv is the cli interface to chypervisord. All commands support arguments via command line or stdin

    Usage:
    hv [flags]
    hv [command]

    Available Commands:
    list        List the hypervisors
    create      Create new hypervisors
    delete      Delete hypervisors
    modify      Modify hypervisors
    guests      Operate on hypervisor guests
    config      Operate on hypervisor config
    subnets     Operate on hypervisor subnets
    help        Help about any command

    Flags:
    -h, --help=false: help for hv
    -j, --json=false: output in json
    -s, --server="http://localhost:17000": server address to connect to

    Use "hv help [command]" for more information about a command.


### Examples

List hypervisors

    $ hv list
    aa44c6e8-3ee3-4671-86da-31b6b060795c
    f403a417-f973-48f1-bea4-0283da8645a2
    f718449c-ed60-4e70-ac70-9b7710d2d68d

    $ hv list -j
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f403a417-f973-48f1-bea4-0283da8645a2","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f718449c-ed60-4e70-ac70-9b7710d2d68d","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

    $ hv list f718449c-ed60-4e70-ac70-9b7710d2d68d aa44c6e8-3ee3-4671-86da-31b6b060795c f403a417-f973-48f1-bea4-0283da8645a2
    f718449c-ed60-4e70-ac70-9b7710d2d68d
    aa44c6e8-3ee3-4671-86da-31b6b060795c
    f403a417-f973-48f1-bea4-0283da8645a2

    $ hv list -j f718449c-ed60-4e70-ac70-9b7710d2d68d aa44c6e8-3ee3-4671-86da-31b6b060795c f403a417-f973-48f1-bea4-0283da8645a2
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f718449c-ed60-4e70-ac70-9b7710d2d68d","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f403a417-f973-48f1-bea4-0283da8645a2","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

List guests on hypervisors

    $ hv guests list
    aa44c6e8-3ee3-4671-86da-31b6b060795c
    ├── 333434fe-2743-4b35-87cc-13fd62ba13fc
    └── 9c931fd1-9851-4658-83c3-0cb994266264
    f403a417-f973-48f1-bea4-0283da8645a2
    ├── 12d9f8af-dfa0-4dba-805e-2c528c3f05f9
    └── 5e32dada-fc99-4ad9-88ec-563e3639a751
    f718449c-ed60-4e70-ac70-9b7710d2d68d
    ├── 6d01bcdc-7985-4c0e-9436-2e726932ee13
    └── ede86f63-8008-4a09-a432-71e4c21742e8

    $ hv guests list -j
    {"guests":["333434fe-2743-4b35-87cc-13fd62ba13fc","9c931fd1-9851-4658-83c3-0cb994266264"],"id":"aa44c6e8-3ee3-4671-86da-31b6b060795c"}
    {"guests":["5e32dada-fc99-4ad9-88ec-563e3639a751","12d9f8af-dfa0-4dba-805e-2c528c3f05f9"],"id":"f403a417-f973-48f1-bea4-0283da8645a2"}
    {"guests":["6d01bcdc-7985-4c0e-9436-2e726932ee13","ede86f63-8008-4a09-a432-71e4c21742e8"],"id":"f718449c-ed60-4e70-ac70-9b7710d2d68d"}

Create hypervisors

    $ hv create '{"id":"bbcd1234-abcd-1234-abcd-1234abcd1234","metadata":{},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":1024,"disk":1024,"cpu":1}, "available_resources": {"memory":1024,"disk":1024,"cpu":1}}'
    bbcd1234-abcd-1234-abcd-1234abcd1234

    $ hv create -j '{"id":"cbcd1234-abcd-1234-abcd-1234abcd1234","metadata":{},"ip":"10.100.101.35","netmask":"255.255.255.255","gateway":"10.100.101.35","mac":"01:23:45:67:89:ac","total_resources":{"memory":1024,"disk":1024,"cpu":1}, "available_resources": {"memory":1024,"disk":1024,"cpu":1}}'
    {"available_resources":{"cpu":1,"disk":1024,"memory":1024},"gateway":"10.100.101.35","id":"cbcd1234-abcd-1234-abcd-1234abcd1234","ip":"10.100.101.35","mac":"01:23:45:67:89:ac","metadata":{},"netmask":"255.255.255.255","total_resources":{"cpu":1,"disk":1024,"memory":1024}}

Modify hypervisors

    $ hv modify aa44c6e8-3ee3-4671-86da-31b6b060795c '{"gateway":"10.0.0.254"}'
    aa44c6e8-3ee3-4671-86da-31b6b060795c

    $ hv list -j aa44c6e8-3ee3-4671-86da-31b6b060795c
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"10.0.0.254","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

    $ hv modify -j aa44c6e8-3ee3-4671-86da-31b6b060795c '{"gateway":"10.0.0.253"}'
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"10.0.0.253","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

    $ hv list -j aa44c6e8-3ee3-4671-86da-31b6b060795c
    {"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"10.0.0.253","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

Delete hypervisors

    $ hv list
    aa44c6e8-3ee3-4671-86da-31b6b060795c
    └── dae637b7-8abd-41d9-b7a2-9d7c2bdd3ef9:br0
    f403a417-f973-48f1-bea4-0283da8645a2
    └── 349c3aa5-3e95-4571-b0f2-1727b8806eba:br0
    f718449c-ed60-4e70-ac70-9b7710d2d68d
    └── f6ac4816-b9c8-4b77-b976-d4be70507754:br0

    # deleting a hypervisor requires deleting it's subnets and guests first, but we
    # need to delete guests using another tool, using etcdctl for now
    $ etcdctl rm /lochness/hypervisors/f403a417-f973-48f1-bea4-0283da8645a2/guests/5e32dada-fc99-4ad9-88ec-563e3639a751
    $ etcdctl rm /lochness/hypervisors/f403a417-f973-48f1-bea4-0283da8645a2/guests/12d9f8af-dfa0-4dba-805e-2c528c3f05f9

    $ hv delete f403a417-f973-48f1-bea4-0283da8645a2
    f403a417-f973-48f1-bea4-0283da8645a2

List subnets for hypervisors

    $ hv subnets list
    aa44c6e8-3ee3-4671-86da-31b6b060795c
    └── dae637b7-8abd-41d9-b7a2-9d7c2bdd3ef9:br0
    f718449c-ed60-4e70-ac70-9b7710d2d68d
    └── f6ac4816-b9c8-4b77-b976-d4be70507754:br0

Delete subnet from hypervisor

    # deleting a subnet requires the hypervisor id also
    $ hv subnets delete -j aa44c6e8-3ee3-4671-86da-31b6b060795c dae637b7-8abd-41d9-b7a2-9d7c2bdd3ef9 f718449c-ed60-4e70-ac70-9b7710d2d68d f6ac4816-b9c8-4b77-b976-d4be70507754
    {}
    {}


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

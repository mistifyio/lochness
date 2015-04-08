# guest

[![guest](https://godoc.org/github.com/mistifyio/lochness/cmd/guest?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/guest)

guest is the command line interface to waheela, the guest management service.
guest can list/create/modify/delete guests. Input is supported via command line
or stdin.

All commands support two output formats, a list of guest ids or a list of guest
json objects, line separated.

### Command Usage

    Usage:
    guest [flags]
    guest [command]

    Available Commands:
    list [<id>...]            List the guests
    create <spec>...          Create guests
    modify (<id> <spec>)...   Modify guests
    delete <id>...            Delete guests
    help [command]            Help about any command

    Available Flags:
    -h, --help=false: help for guest
    -j, --jsonout=false: output in json
    -s, --server="http://localhost:18000/": server address to connect to

    Use "guest help [command]" for more information about that command.


### Examples

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
    e2aae131-eff7-41ae-8541-73a48eb5295d
    41a7d3ca-685e-4a57-bc61-dce3e33b6b09

    $ guest create -j '{"bridge":"br0", "flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234", "ip":"10.100.101.66", "mac":"A4-75-C1-6B-E3-49", "network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}'
    {"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e217e622-b30b-41c1-87ac-a249152b3f32","ip":"10.100.101.66","mac":"a4:75:c1:6b:e3:49","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}

Modify guests

    $ guest modify e2aae131-eff7-41ae-8541-73a48eb5295d '{"type":"qwerty"}' 41a7d3ca-685e-4a57-bc61-dce3e33b6b09 '{"type":"zxcv"}'
    e2aae131-eff7-41ae-8541-73a48eb5295d
    41a7d3ca-685e-4a57-bc61-dce3e33b6b09

    $ guest modify -j e2aae131-eff7-41ae-8541-73a48eb5295d '{"type":"qwerty"}'
    {"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e2aae131-eff7-41ae-8541-73a48eb5295d","ip":"10.100.101.66","mac":"a4:75:c1:6b:e3:49","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"qwerty"}

Delete guests

    $ guest delete 41a7d3ca-685e-4a57-bc61-dce3e33b6b09 41a7d3ca-685e-4a57-bc61-dce3e33b6b09
    41a7d3ca-685e-4a57-bc61-dce3e33b6b09

    $ guest delete -j e2aae131-eff7-41ae-8541-73a48eb5295d
    {"bridge":"br0","flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","hypervisor":"","id":"e2aae131-eff7-41ae-8541-73a48eb5295d","ip":"10.100.101.66","mac":"a4:75:c1:6b:e3:49","metadata":{},"network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"qwerty"}
## Usage

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

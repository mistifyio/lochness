# dobharchu

[![dobharchu](https://godoc.org/github.com/mistifyio/lochness/cmd/dobharchu?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/dobharchu)

dobharchu is a service to monitor etcd for changes to hyperviors and guests and
rebuild the DHCP config files as needed.


### Usage

The following arguments are accepted:

    $ dobharchu -h
    Usage of dobharchu:
    -d, --domain="": domain for lochness; required
    -e, --etcd="http://127.0.0.1:4001": address of etcd server
    	--guests-path="/etc/dhcpd/guests.conf": alternative path to guests.conf
    	--hypervisors-path="/etc/dhcpd/hypervisors.conf": alternative path to hypervisors.conf
    -l, --log-level="warning": log level: debug/info/warning/error/critical/fatal


### Watched

The following etcd prefixes are watched for changes:

    /lochness/hypervisors
    /lochness/guests
    /lochness/subnets


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

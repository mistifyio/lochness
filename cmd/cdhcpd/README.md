# cdhcpd

[![cdhcpd](https://godoc.org/github.com/mistifyio/lochness/cmd/cdhcpd?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/cdhcpd)

cdhcpd is a service to monitor a kv for changes to hyperviors and guests and
rebuild the DHCP config files as needed.


### Usage

The following arguments are accepted:

    $ cdhcpd -h
    Usage of cdhcpd:
      -c, --conf-dir="/etc/dhcp/": dhcpd configuration directory
      -d, --domain="": domain for lochness; required
      -k, --kv="http://127.0.0.1:4001": address of kv server
      -l, --log-level="warning": log level: debug/info/warning/error/critical/fatal


### Watched

The following prefixes are watched for changes:

    /lochness/hypervisors
    /lochness/guests
    /lochness/subnets


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

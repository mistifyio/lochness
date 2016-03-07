# nfirewalld

[![nfirewalld](https://godoc.org/github.com/mistifyio/lochness/cmd/nfirewalld?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/nfirewalld)

nfirewalld is a simple firewall daemon that monitors a kv for firewall
configuration. The firewall is implemented using nftables. When guests or
firewall groups are added, modified, or removed, a new firewall configuration is
generated and nftables is reloaded.


### Usage

The following arguments are understood:

    $ nfirewalld -h
    Usage of nfirewalld:
    -k, --kv="http://localhost:4001": kv cluster address
    -f, --file="/etc/nftables.conf": nft configuration file
    -i, --id="": hypervisor id


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

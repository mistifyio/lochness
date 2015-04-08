# cherufe

[![cherufe](https://godoc.org/github.com/mistifyio/lochness/cmd/cherufe?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/cherufe)

cherufe is a simple firewall daemon that monitors etcd for firewall
configuration. The firewall is implemented using nftables. When guests or
firewall groups are added, modified, or removed, a new firewall configuration is
generated and nftables is reloaded.

### Command Usage

    $ cherufe -h
    Usage of cherufe:
    -e, --etcd="http://localhost:4001": etcd cluster address
    -f, --file="/etc/nftables.conf": nft configuration file
    -i, --id="": hypervisor id
## Usage

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

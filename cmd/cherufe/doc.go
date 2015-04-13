/*
cherufe is a simple firewall daemon that monitors etcd for firewall
configuration. The firewall is implemented using nftables. When guests or
firewall groups are added, modified, or removed, a new firewall configuration
is generated and nftables is reloaded.

Usage

The following arguments are understood:

	$ cherufe -h
	Usage of cherufe:
	-e, --etcd="http://localhost:4001": etcd cluster address
	-f, --file="/etc/nftables.conf": nft configuration file
	-i, --id="": hypervisor id
*/
package main

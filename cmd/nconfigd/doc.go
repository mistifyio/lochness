/*
nconfigd is a service to monitor a kv and run ansible on change.
The prefixes to watch and which ansible role(s) to run for each are specified in a config file.

Usage

The following arguments are understood:

	$ nconfigd -h
	Usage of nconfigd:
	-a, --ansible="/root/lochness-ansible": directory containing the ansible run command
	-c, --config="": path to config file with prefixs
	-k, --kv="http://127.0.0.1:4001": address of kv server
	-l, --log-level="warn": log level

Config

Config consists of a map of watched kv prefixes to an array of ansible role names

Example config

	{
		"/lochness/config": [],
		"/lochness/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config/cbootstrapd": ["cbootstrapd"],
		"/lochness/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config/dhcpd": ["dhcpd","dhcrelay"],
		"/lochness/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config/dhcrelay": ["dhcrelay"],
		"/lochness/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config/dns": ["dns","dhcpd"],
		"/lochness/hypervisors/abcd1234-abcd-1234-abcd-1234abcd1234/config/tftpd": ["tftpd"]
	}
*/
package main

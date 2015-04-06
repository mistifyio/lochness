# dobharchu

[dobharchu](http://en.wikipedia.org/wiki/Dobhar-ch%C3%BA) is a service to monitor etcd for changes to hyperviors and guests and rebuild the DHCP config files as needed

## Usage

```
$ dobharchu -h
Usage of dobharchu:
  -d, --domain="": domain for lochness; required
  -e, --etcd="http://127.0.0.1:4001": address of etcd server
      --guests-path="/etc/dhcpd/guests.conf": alternative path to guests.conf
      --hypervisors-path="/etc/dhcpd/hypervisors.conf": alternative path to hypervisors.conf
  -l, --log-level="warning": log level: debug/info/warning/error/critical/fatal
```

### Watching

Dobharchu watches for changes to the following prefixes in etcd:

* `/lochness/hypervisors`
* `/lochness/guests`
* `/lochness/subnets`

### Example


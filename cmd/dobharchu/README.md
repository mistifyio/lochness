# Dobharchu

Dobharchu is a script that watches for changes to hypervisors and groups and
rebuilds the DHCP config files when needed.

#### Watching

Dobharchu watches for changes to the following prefixes in etcd:

* `/lochness/hypervisors`
* `/lochness/guests`
* `/lochness/subnets`

#### Config files built by Dobharchu

* `/etc/dhcpd/hypervisors.conf`
* `/etc/dhcpd/guests.conf`

## Flags

| Flag               | Shorthand | Type   | Default                       | Description                                               |
| ------------------ | --------- | ------ | ----------------------------- | --------------------------------------------------------- |
| `domain`           | `d`       | string | *none*                        | domain for lochness; required                             |
| `etcd`             | `e`       | string | `http://127.0.0.1:4001`       | address of etcd server                                    |
| `hypervisors-path` | *none*    | string | `/etc/dhcpd/hypervisors.conf` | alternative path to hypervisors.conf                      |
| `guests-path`      | *none*    | string | `/etc/dhcpd/guests.conf`      | alternative path to guests.conf                           |
| `test`             | `t`       | bool   | false                         | run in test mode; do not require etcd.SyncCluster to work |
| `log-level`        | `l`       | string | `warning`                     | log level: debug/info/warning/error/critical/fatal        |

## Testing

Dobharchu has unit tests for its primary work package `refresher` and a manual
integration test for ensuring that the watch mechanism works properly.

To run the integration test:

```sh
$ go run cmd/dobharchu/integrationtest/main.go
```

It will create a new set of hypervisors and guests and give you the IDs so that
you can check that they appear correctly in the conf files.

When you're testing, we recommend that you give Dobharchu alternate config
files to use:

```sh
$ ./dobharchu -d example.com --hypervisors-path=htest.conf --guests-path=gtest.conf
```

If you're running etcd locally while you test rather than using a cluster, you
can bypass Dobharchu's initial sync test by sending it the test-mode flag:

```sh
$ ./dobharchu -d example.com --hypervisors-path=htest.conf --guests-path=gtest.conf -t
```


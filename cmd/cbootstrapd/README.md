# cbootstrapd

[![cbootstrapd](https://godoc.org/github.com/mistifyio/lochness/cmd/cbootstrapd?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/cbootstrapd)

cbootstrapd is a simple web service to enable boot and pre-init configuration of
LochNess nodes.


### Usage

The following arguments are understood:

    $ cbootstrapd -h
    Usage of cbootstrapd:
    -b, --base="http://ipxe.mistify.local:8888": base address of bits request
    -e, --etcd="http://127.0.0.1:4001": address of etcd machine
    -i, --images="/var/lib/images": directory containing the images
    -o, --options="": additional options to add to boot kernel
    -p, --port=8888: address to listen
    -s, --statsd="": statsd address
    -v, --version="0.1.0": If all else fails, what version to serve

### HTTP API Endpoints

    /ipxe/{ip}
    	iPXE config
    	* GET - Get an ipxe script that corresponds to this hv

    /images/{version}/{file}
    	boot images (kernel, rootfs)
    	* GET - Get kernel/rootfs

    /configs/{ip}
    	pre-init node configuration (zfs, etcd, ...)
    	* GET - Get a shell style file of K=V pairs for pre-init configuration


### Example Requests

GET /ipxe/{ip}

    $ curl http://192.168.100.100:8888/ipxe/192.168.100.200

    #!ipxe
    kernel http://192.168.100.100:8888/images/0.1.0/vmlinuz uuid=ed5df266-1416-497b-ac96-da42a77c5410
    initrd http://192.168.100.100:8888/images/0.1.0/initrd
    boot

GET /images/{version}/{file}

    $ wget http://192.168.100.100:8888/images/0.1.0/vmlinuz

    wget ipxe.services.lochness.local:8888/images/0.1.0/vmlinuz
    --2015-03-12 19:24:49--  http://ipxe.services.lochness.local:8888/images/0.1.0/vmlinuz
    Resolving ipxe.services.lochness.local (ipxe.services.lochness.local)... 192.168.100.100
    Connecting to ipxe.services.lochness.local (ipxe.services.lochness.local)|192.168.100.100|:8888... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 5765360 (5.5M) [application/octet-stream]
    Saving to: 'vmlinuz'

    vmlinuz	100%[=========================================================>]   5.50M  --.-KB/s   in 0.01s

    2015-03-12 19:24:49 (426 MB/s) - 'vmlinuz' saved [5765360/5765360]

GET /configs/{ip}

    $ curl http://192.168.100.100:8888/configs/192.168.100.200

    ETCD_DISCOVERY=https://discovery.etcd.io/3e86b59982e49066c5d813af1c2e2579cbf573de
    ZFS_POOL=raid0


--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

---
title: Enfield
---

# Enfield

Enfield is a three legged, short bodied, short clawed armed grayish cryptid with
big reddish eyes ([Wikipedia](http://en.wikipedia.org/wiki/Enfield_Monster)).

Also a simple web service to enable boot and pre-init configuration of LochNess
nodes.

Current service endpoints served by enfield are:

 - `/ipxe` - iPXE config
 - `/images` - boot images (_kernel_, _rootfs_)

# ipxe

## GET /ipxe/:ip

Get an ipxe script that corresponds to this hv.

#### request

    curl http://192.168.100.100:8888/ipxe/192.168.100.200

#### response

    #!ipxe
    kernel http://192.168.100.100:8888/images/0.1.0/vmlinuz uuid=ed5df266-1416-497b-ac96-da42a77c5410
    initrd http://192.168.100.100:8888/images/0.1.0/initrd
    boot

# images

## GET /images/:version/:file

Get kernel/rootfs

#### request

    wget http://192.168.100.100:8888/images/0.1.0/vmlinuz

#### response

    wget ipxe.services.lochness.local:8888/images/0.1.0/vmlinuz
    --2015-03-12 19:24:49--  http://ipxe.services.lochness.local:8888/images/0.1.0/vmlinuz
    Resolving ipxe.services.lochness.local (ipxe.services.lochness.local)... 192.168.100.100
    Connecting to ipxe.services.lochness.local (ipxe.services.lochness.local)|192.168.100.100|:8888... connected.
    HTTP request sent, awaiting response... 200 OK
    Length: 5765360 (5.5M) [application/octet-stream]
    Saving to: 'vmlinuz'

    vmlinuz                        100%[=========================================================>]   5.50M  --.-KB/s   in 0.01s

    2015-03-12 19:24:49 (426 MB/s) - 'vmlinuz' saved [5765360/5765360]

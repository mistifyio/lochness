#!/bin/bash
set -e

etcdctl rm --recursive /lochness || true

FID=`uuidgen`
etcdctl set /lochness/flavors/$FID/metadata "$(printf '{ "id": "%s", "memory": 512, "disk": 4096, "cpu": 2 }' $FID)"

HID=`uuidgen`
etcdctl set /lochness/hypervisors/$HID/metadata "$(printf '{ "id": "%s", "ip": "10.99.100.45", "netmask": "255.255.255.0", "gateway": "10.99.100.1", "mac": "3e:15:c2:ed:31:00", "available_resources": { "memory": 262144, "cpu": 8, "disk": 268435456}}' $HID)"

etcdctl set /lochness/hypervisors/$HID/heartbeat ""

SID=`uuidgen`
etcdctl set /lochness/subnets/$SID/metadata "$(printf '{"id": "%s", "cidr": "192.168.100.0/24", "start": "192.168.100.10", "end": "192.168.100.200", "gateway": "192.168.100.1" }' $SID)"

etcdctl set /lochness/hypervisors/$HID/subnets/$SID br0

NID=`uuidgen`
etcdctl set /lochness/networks/$NID/metadata "$(printf '{"id": "%s" }' $NID)"

etcdctl set /lochness/networks/$NID/subnets/$SID $SID

FWID=`uuidgen`
etcdctl set /lochness/fwgroups/$FWID/metadata "$(printf '{"id": "%s", "rules": [ {"source": "192.168.1.0/24", "portStart": 80, "portEnd": 80, "protocol": "tcp", "action": "allow"}, {"group": "%s", "portStart": 22, "portEnd": 22, "protocol": "tcp", "action": "allow"} ] }' $FWID $FWID)"

etcdctl get /lochness/fwgroups/$FWID/metadata

etcdctl get /lochness/fwgroups/$FWID/metadata | jq .

for i in {100..110}; do
    GID=`uuidgen`
    etcdctl set /lochness/guests/$GID/metadata "$(printf '{ "id": "%s", "network": "%s", "flavor": "%s", "fwgroup": "%s", "ip": "192.168.100.%s", "hypervisor": "%s" }' $GID $NID $FID $FWID $i $HID)"

    etcdctl set /lochness/hypervisors/$HID/guests/$GID ""
done
export HYPERVISOR_ID=$HID
go run *.go


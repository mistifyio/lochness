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

GID=`uuidgen`
etcdctl set /lochness/guests/$GID/metadata "$(printf '{ "id": "%s", "network": "%s", "flavor": "%s" }' $GID $NID $FID)"

JID=`uuidgen`
etcdctl set /lochness/jobs/$JID "$(printf '{ "id": "%s", "guest": "%s", "action": "select-hypervisor", "status": "new" }' $JID $GID)"

printf "use create\r\nput 0 0 300 %d\r\n%s\r\n" $(echo -n $JID | wc -c) $JID| nc 127.0.0.1 11300

sleep 5

etcdctl get /lochness/jobs/$JID | jq .
etcdctl get /lochness/guests/$GID/metadata | jq .

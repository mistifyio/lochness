/*
Package lochness provides primitives for orchestrating Mistify Agents.

Loch Ness is a simple virtual machine manager ("cloud controller").  It is a proof-of-concept, straight forward implementation for testing various ideas.  It is targeted at clusters of up to around 100 physical machines.

Data Model

A Hypervisor is a physical machine

A subnet is an actual  IP subnet with a range of usable IP addresses. A
hypervisor can have one of more subnets, while a subnet can span multiple
hypervisors.  This assumes a rather simple network layout.

A Network is a logical collection of subnets. It is generally used by guest
creation to decide what logical segment to place a guest on.

A flavor is a virtual resource "Template" for guest creation. A guest has a
single flavor.

A FW Group is a collection of firewall rules for incoming IP traffic.  A Guest
has a single fwgroup.

A guest is a virtual machine.  At creation time, a network, fwgroup, and network
is required.
*/
package lochness

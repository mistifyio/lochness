# Intro
hv is the command line interface to `grootslang`, the hypervisor management
service. hv can list/modify/delete hypervisors, hypervisor guests, hypervisors
subnets, and hypervisor configs.

All commands support dual output formats, a `tree` like output for humans
(default) or a json output for further processing.

Most commands accept 0 or many arguments, a couple require at least 1 argument.
Run `hv help` for more information.

## Examples

### List

```
$ hv list
aa44c6e8-3ee3-4671-86da-31b6b060795c
f403a417-f973-48f1-bea4-0283da8645a2
f718449c-ed60-4e70-ac70-9b7710d2d68d

$ hv list -j
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f403a417-f973-48f1-bea4-0283da8645a2","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f718449c-ed60-4e70-ac70-9b7710d2d68d","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

$ hv list f718449c-ed60-4e70-ac70-9b7710d2d68d aa44c6e8-3ee3-4671-86da-31b6b060795c f403a417-f973-48f1-bea4-0283da8645a2
f718449c-ed60-4e70-ac70-9b7710d2d68d
aa44c6e8-3ee3-4671-86da-31b6b060795c
f403a417-f973-48f1-bea4-0283da8645a2

$ hv list -j f718449c-ed60-4e70-ac70-9b7710d2d68d aa44c6e8-3ee3-4671-86da-31b6b060795c f403a417-f973-48f1-bea4-0283da8645a2
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f718449c-ed60-4e70-ac70-9b7710d2d68d","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"","id":"f403a417-f973-48f1-bea4-0283da8645a2","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

$ hv guests list
aa44c6e8-3ee3-4671-86da-31b6b060795c
├── 333434fe-2743-4b35-87cc-13fd62ba13fc
└── 9c931fd1-9851-4658-83c3-0cb994266264
f403a417-f973-48f1-bea4-0283da8645a2
├── 12d9f8af-dfa0-4dba-805e-2c528c3f05f9
└── 5e32dada-fc99-4ad9-88ec-563e3639a751
f718449c-ed60-4e70-ac70-9b7710d2d68d
├── 6d01bcdc-7985-4c0e-9436-2e726932ee13
└── ede86f63-8008-4a09-a432-71e4c21742e8

$ hv guests list -j
{"guests":["333434fe-2743-4b35-87cc-13fd62ba13fc","9c931fd1-9851-4658-83c3-0cb994266264"],"id":"aa44c6e8-3ee3-4671-86da-31b6b060795c"}
{"guests":["5e32dada-fc99-4ad9-88ec-563e3639a751","12d9f8af-dfa0-4dba-805e-2c528c3f05f9"],"id":"f403a417-f973-48f1-bea4-0283da8645a2"}
{"guests":["6d01bcdc-7985-4c0e-9436-2e726932ee13","ede86f63-8008-4a09-a432-71e4c21742e8"],"id":"f718449c-ed60-4e70-ac70-9b7710d2d68d"}
```

### Modify

```
$ hv modify aa44c6e8-3ee3-4671-86da-31b6b060795c '{"gateway":"10.0.0.254"}'
aa44c6e8-3ee3-4671-86da-31b6b060795c

$ hv list -j aa44c6e8-3ee3-4671-86da-31b6b060795c
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"10.0.0.254","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

$ hv modify -j aa44c6e8-3ee3-4671-86da-31b6b060795c '{"gateway":"10.0.0.253"}'
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"10.0.0.253","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}

$ hv list -j aa44c6e8-3ee3-4671-86da-31b6b060795c
{"available_resources":{"cpu":0,"disk":0,"memory":0},"gateway":"10.0.0.253","id":"aa44c6e8-3ee3-4671-86da-31b6b060795c","ip":"10.100.101.34","mac":"01:23:45:67:89:ab","metadata":{},"netmask":"","total_resources":{"cpu":0,"disk":0,"memory":0}}
```


### Delete

```
$ hv list
aa44c6e8-3ee3-4671-86da-31b6b060795c
└── dae637b7-8abd-41d9-b7a2-9d7c2bdd3ef9:br0
f403a417-f973-48f1-bea4-0283da8645a2
└── 349c3aa5-3e95-4571-b0f2-1727b8806eba:br0
f718449c-ed60-4e70-ac70-9b7710d2d68d
└── f6ac4816-b9c8-4b77-b976-d4be70507754:br0

# deleting a hypervisor requires deleting it's subnets and guests first, but we
# need to delete guests using another tool, using etcdctl for now
$ etcdctl rm /lochness/hypervisors/f403a417-f973-48f1-bea4-0283da8645a2/guests/5e32dada-fc99-4ad9-88ec-563e3639a751
$ etcdctl rm /lochness/hypervisors/f403a417-f973-48f1-bea4-0283da8645a2/guests/12d9f8af-dfa0-4dba-805e-2c528c3f05f9

$ hv delete f403a417-f973-48f1-bea4-0283da8645a2
f403a417-f973-48f1-bea4-0283da8645a2

$ hv subnets list
aa44c6e8-3ee3-4671-86da-31b6b060795c
└── dae637b7-8abd-41d9-b7a2-9d7c2bdd3ef9:br0
f718449c-ed60-4e70-ac70-9b7710d2d68d
└── f6ac4816-b9c8-4b77-b976-d4be70507754:br0

# deleting a subnet requires the hypervisor id also
$ hv subnets delete -j aa44c6e8-3ee3-4671-86da-31b6b060795c dae637b7-8abd-41d9-b7a2-9d7c2bdd3ef9 f718449c-ed60-4e70-ac70-9b7710d2d68d f6ac4816-b9c8-4b77-b976-d4be70507754
{}
{}
```

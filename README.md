# lochness

[![lochness](https://godoc.org/github.com/mistifyio/lochness?status.png)](https://godoc.org/github.com/mistifyio/lochness)

Package lochness provides primitives for orchestrating Mistify Agents.

Loch Ness is a simple virtual machine manager ("cloud controller"). It is a
proof-of-concept, straight forward implementation for testing various ideas. It
is targeted at clusters of up to around 100 physical machines.


### Data Model

A Hypervisor is a physical machine

A subnet is an actual IP subnet with a range of usable IP addresses. A
hypervisor can have one of more subnets, while a subnet can span multiple
hypervisors. This assumes a rather simple network layout.

A Network is a logical collection of subnets. It is generally used by guest
creation to decide what logical segment to place a guest on.

A flavor is a virtual resource "Template" for guest creation. A guest has a
single flavor.

A FW Group is a collection of firewall rules for incoming IP traffic. A Guest
has a single fwgroup.

A guest is a virtual machine. At creation time, a network, fwgroup, and network
is required.

## Usage

```go
const (
	JobStatusNew     = "new"
	JobStatusWorking = "working"
	JobStatusDone    = "done"
	JobStatusError   = "error"
)
```
Job Status

```go
var (
	// ConfigPath is the path in the config store.
	ConfigPath = "lochness/config/"
)
```

```go
var DefaultCandidateFunctions = []CandidateFunction{
	CandidateIsAlive,
	CandidateHasSubnet,
	CandidateHasResources,
	CandidateRandomize,
}
```
DefaultCandidateFunctions is a default list of CandidateFunctions for general
use

```go
var (
	// FWGroupPath is the path in the config store
	FWGroupPath = "lochness/fwgroups/"
)
```

```go
var (
	// FlavorPath is the path in the config store
	FlavorPath = "lochness/flavors/"
)
```

```go
var (
	// GuestPath is the path in the config store
	GuestPath = "lochness/guests/"
)
```

```go
var (
	// HypervisorPath is the path in the config store
	HypervisorPath = "lochness/hypervisors/"
)
```

```go
var (
	// JobPath is the path in the config store
	JobPath = "lochness/jobs/"
)
```

```go
var (
	// NetworkPath is the path in the config store.
	NetworkPath = "lochness/networks/"
)
```

```go
var (
	// SubnetPath is the key prefix for subnets
	SubnetPath = "lochness/subnets/"
)
```

#### func  GetHypervisorID

```go
func GetHypervisorID() string
```
GetHypervisorID gets the hypervisor id as set with SetHypervisorID. It does not
make an attempt to discover the id if not set.

#### func  IsKeyNotFound

```go
func IsKeyNotFound(err error) bool
```
IsKeyNotFound is a helper to determine if the error is a key not found error

#### func  SetHypervisorID

```go
func SetHypervisorID(id string) (string, error)
```
SetHypervisorID sets the id of the current hypervisor. It should be used by all
daemons that are ran on a hypervisor and are expected to interact with the data
stores directly. Passing in a blank string will fall back to first checking the
environment variable "HYPERVISOR_ID" and then using the hostname. ID must be a
valid UUID. ID will be lowercased.

#### func  ToBool

```go
func ToBool(val string) bool
```
ToBool is a wrapper around strconv.ParseBool for easy boolean values

#### type Agent

```go
type Agent interface {
	GetGuest(string) (*client.Guest, error)
	CreateGuest(string) (string, error)
	DeleteGuest(string) (string, error)
	GuestAction(string, string) (string, error)
	CheckJobStatus(string, string, string) (bool, error)
}
```

Agent is an interface that allows for communication with a hypervisor agent

#### type CandidateFunction

```go
type CandidateFunction func(*Guest, Hypervisors) (Hypervisors, error)
```

CandidateFunction is used to select hypervisors that can run the given guest.

#### type Context

```go
type Context struct {
}
```

Context carries around data/structs needed for operations

#### func  NewContext

```go
func NewContext(e *etcd.Client) *Context
```
NewContext creates a new context

#### func (*Context) FWGroup

```go
func (c *Context) FWGroup(id string) (*FWGroup, error)
```
FWGroup fetches a FWGroup from the config store

#### func (*Context) FirstGuest

```go
func (c *Context) FirstGuest(f func(*Guest) bool) (*Guest, error)
```
FirstGuest will return the first guest for which the function returns true.

#### func (*Context) FirstHypervisor

```go
func (c *Context) FirstHypervisor(f func(*Hypervisor) bool) (*Hypervisor, error)
```
FirstHypervisor will return the first hypervisor for which the function returns
true.

#### func (*Context) Flavor

```go
func (c *Context) Flavor(id string) (*Flavor, error)
```
Flavor fetches a single Flavor from the config store

#### func (*Context) ForEachConfig

```go
func (c *Context) ForEachConfig(f func(key, val string) error) error
```
ForEachConfig will run f on each config. It will stop iteration if f returns an
error.

#### func (*Context) ForEachGuest

```go
func (c *Context) ForEachGuest(f func(*Guest) error) error
```
ForEachGuest will run f on each Guest. It will stop iteration if f returns an
error.

#### func (*Context) ForEachHypervisor

```go
func (c *Context) ForEachHypervisor(f func(*Hypervisor) error) error
```
ForEachHypervisor will run f on each Hypervisor. It will stop iteration if f
returns an error.

#### func (*Context) ForEachSubnet

```go
func (c *Context) ForEachSubnet(f func(*Subnet) error) error
```
ForEachSubnet will run f on each Subnet. It will stop iteration if f returns an
error.

#### func (*Context) GetConfig

```go
func (c *Context) GetConfig(key string) (string, error)
```
GetConfig gets a single value from the config store. The key can contain slashes
("/")

#### func (*Context) Guest

```go
func (c *Context) Guest(id string) (*Guest, error)
```
Guest fetches a Guest from the config store

#### func (*Context) Hypervisor

```go
func (c *Context) Hypervisor(id string) (*Hypervisor, error)
```
Hypervisor fetches a Hypervisor from the config store.

#### func (*Context) Job

```go
func (c *Context) Job(id string) (*Job, error)
```
Job retrieves a single job from the data store.

#### func (*Context) Network

```go
func (c *Context) Network(id string) (*Network, error)
```
Network fetches a Network from the data store.

#### func (*Context) NewFWGroup

```go
func (c *Context) NewFWGroup() *FWGroup
```
NewFWGroup creates a new, blank FWGroup

#### func (*Context) NewFlavor

```go
func (c *Context) NewFlavor() *Flavor
```
NewFlavor creates a blank Flavor

#### func (*Context) NewGuest

```go
func (c *Context) NewGuest() *Guest
```
NewGuest create a new blank Guest

#### func (*Context) NewHypervisor

```go
func (c *Context) NewHypervisor() *Hypervisor
```
NewHypervisor create a new blank Hypervisor.

#### func (*Context) NewJob

```go
func (c *Context) NewJob() *Job
```
NewJob creates a new job.

#### func (*Context) NewMistifyAgent

```go
func (context *Context) NewMistifyAgent() *MistifyAgent
```
NewMistifyAgent creates a new MistifyAgent instance within the context

#### func (*Context) NewNetwork

```go
func (c *Context) NewNetwork() *Network
```
NewNetwork creates a new, blank Network.

#### func (*Context) NewStubAgent

```go
func (context *Context) NewStubAgent(failPercent int) *StubAgent
```
NewStubAgent creates a new StubAgent instance within the context and initialies
the random number generator for failures

#### func (*Context) NewSubnet

```go
func (c *Context) NewSubnet() *Subnet
```
NewSubnet creates a new "blank" subnet. Fill in the needed values and then call
Save

#### func (*Context) SetConfig

```go
func (c *Context) SetConfig(key, val string) error
```
SetConfig sets a single value from the config store. The key can contain slashes
("/")

#### func (*Context) Subnet

```go
func (c *Context) Subnet(id string) (*Subnet, error)
```
Subnet fetches a single subnet by ID

#### type ErrorHTTPCode

```go
type ErrorHTTPCode struct {
	Expected int
	Code     int
}
```

ErrorHTTPCode should be used for errors resulting from an http response code not
matching the expected code

#### func (ErrorHTTPCode) Error

```go
func (e ErrorHTTPCode) Error() string
```
Error returns a string error message

#### type FWGroup

```go
type FWGroup struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
	Rules    FWRules           `json:"rules"`
}
```

FWGroup represents a group of firewall rules

#### func (FWGroup) MarshalJSON

```go
func (f FWGroup) MarshalJSON() ([]byte, error)
```
MarshalJSON is a helper for marshalling a FWGroup

#### func (*FWGroup) Refresh

```go
func (f *FWGroup) Refresh() error
```
Refresh reloads from the data store

#### func (*FWGroup) Save

```go
func (f *FWGroup) Save() error
```
Save persists a FWGroup. It will call Validate.

#### func (*FWGroup) UnmarshalJSON

```go
func (f *FWGroup) UnmarshalJSON(input []byte) error
```
UnmarshalJSON is a helper for unmarshalling a FWGroup

#### func (*FWGroup) Validate

```go
func (f *FWGroup) Validate() error
```
Validate ensures a FWGroup has reasonable data. It currently does nothing.

#### type FWGroups

```go
type FWGroups []*FWGroup
```

FWGroups is an alias to FWGroup slices

#### type FWRule

```go
type FWRule struct {
	Source    *net.IPNet `json:"source,omitempty"`
	Group     string     `json:"group"`
	PortStart uint       `json:"portStart"`
	PortEnd   uint       `json:"portEnd"`
	Protocol  string     `json:"protocol"`
	Action    string     `json:"action"`
}
```

FWRule represents a single firewall rule

#### type FWRules

```go
type FWRules []*FWRule
```

FWRules is an alias to a slice of *FWRule

#### type Flavor

```go
type Flavor struct {
	ID       string            `json:"id"`
	Image    string            `json:"image"`
	Metadata map[string]string `json:"metadata"`
	Resources
}
```

Flavor defines the virtual resources for a guest

#### func (*Flavor) Refresh

```go
func (f *Flavor) Refresh() error
```
Refresh reloads from the data store

#### func (*Flavor) Save

```go
func (f *Flavor) Save() error
```
Save persists a Flavor. It will call Validate.

#### func (*Flavor) Validate

```go
func (f *Flavor) Validate() error
```
Validate ensures a Flavor has reasonable data. It currently does nothing.

#### type Flavors

```go
type Flavors []*Flavor
```

Flavors is an alias to a slice of *Flavor

#### type Guest

```go
type Guest struct {
	ID           string            `json:"id"`
	Metadata     map[string]string `json:"metadata"`
	Type         string            `json:"type"`       // type of guest. currently just kvm
	FlavorID     string            `json:"flavor"`     // resource flavor
	HypervisorID string            `json:"hypervisor"` // hypervisor. may be blank if not assigned yet
	NetworkID    string            `json:"network"`
	SubnetID     string            `json:"subnet"`
	FWGroupID    string            `json:"fwgroup"`
	MAC          net.HardwareAddr  `json:"mac"`
	IP           net.IP            `json:"ip"`
	Bridge       string            `json:"bridge"`
}
```

Guest is a virtual machine

#### func (*Guest) Candidates

```go
func (g *Guest) Candidates(f ...CandidateFunction) (Hypervisors, error)
```
Candidates returns a list of Hypervisors that may run this Guest.

#### func (*Guest) Destroy

```go
func (g *Guest) Destroy() error
```
Destroy removes a guest

#### func (*Guest) MarshalJSON

```go
func (g *Guest) MarshalJSON() ([]byte, error)
```
MarshalJSON is a helper for marshalling a Guest

#### func (*Guest) Refresh

```go
func (g *Guest) Refresh() error
```
Refresh reloads from the data store

#### func (*Guest) Save

```go
func (g *Guest) Save() error
```
Save persists the Guest to the data store.

#### func (*Guest) UnmarshalJSON

```go
func (g *Guest) UnmarshalJSON(input []byte) error
```
UnmarshalJSON is a helper for unmarshalling a Guest

#### func (*Guest) Validate

```go
func (g *Guest) Validate() error
```
Validate ensures a Guest has reasonable data. It currently does nothing. TODO: a
guest needs a valid flavor, firewall group, and network

#### type Guests

```go
type Guests []*Guest
```

Guests is an alias to a slice of *Guest

#### type Hypervisor

```go
type Hypervisor struct {
	ID                 string            `json:"id"`
	Metadata           map[string]string `json:"metadata"`
	IP                 net.IP            `json:"ip"`
	Netmask            net.IP            `json:"netmask"`
	Gateway            net.IP            `json:"gateway"`
	MAC                net.HardwareAddr  `json:"mac"`
	TotalResources     Resources         `json:"total_resources"`
	AvailableResources Resources         `json:"available_resources"`

	// Config is a set of key/values for driving various config options. writes should
	// only be done using SetConfig
	Config map[string]string
}
```

Hypervisor is a physical box on which guests run

#### func (*Hypervisor) AddGuest

```go
func (h *Hypervisor) AddGuest(g *Guest) error
```
AddGuest adds a Guest to the Hypervisor. It reserves an IPaddress for the Guest.
/ It also updates the Guest.

#### func (*Hypervisor) AddSubnet

```go
func (h *Hypervisor) AddSubnet(s *Subnet, bridge string) error
```
AddSubnet adds a subnet to a Hypervisor.

#### func (*Hypervisor) Destroy

```go
func (h *Hypervisor) Destroy() error
```
Destroy removes a hypervisor. The Hypervisor must not have any guests.

#### func (*Hypervisor) ForEachGuest

```go
func (h *Hypervisor) ForEachGuest(f func(*Guest) error) error
```
ForEachGuest will run f on each Guest. It will stop iteration if f returns an
error.

#### func (*Hypervisor) Guests

```go
func (h *Hypervisor) Guests() []string
```
Guests returns a slice of GuestIDs assigned to the Hypervisor.

#### func (*Hypervisor) Heartbeat

```go
func (h *Hypervisor) Heartbeat(ttl time.Duration) error
```
Heartbeat announces the avilibility of a hypervisor. In general, this is useful
for service announcement/discovery. Should be ran from the hypervisor, or
something monitoring it.

#### func (*Hypervisor) IsAlive

```go
func (h *Hypervisor) IsAlive() bool
```
IsAlive returns true if the heartbeat is present.

#### func (*Hypervisor) MarshalJSON

```go
func (h *Hypervisor) MarshalJSON() ([]byte, error)
```
MarshalJSON is a helper for marshalling a Hypervisor

#### func (*Hypervisor) Refresh

```go
func (h *Hypervisor) Refresh() error
```
Refresh reloads a Hypervisor from the data store.

#### func (*Hypervisor) RemoveGuest

```go
func (h *Hypervisor) RemoveGuest(g *Guest) error
```
RemoveGuest removes a guest from the hypervisor. Also releases the IP

#### func (*Hypervisor) RemoveSubnet

```go
func (h *Hypervisor) RemoveSubnet(s *Subnet) error
```
RemoveSubnet removes a subnet from a Hypervisor.

#### func (*Hypervisor) Save

```go
func (h *Hypervisor) Save() error
```
Save persists a FWGroup. It will call Validate.

#### func (*Hypervisor) SetConfig

```go
func (h *Hypervisor) SetConfig(key, value string) error
```
SetConfig sets a single Hypervisor Config value. Set value to "" to unset.

#### func (*Hypervisor) Subnets

```go
func (h *Hypervisor) Subnets() map[string]string
```
Subnets returns the subnet/bridge mappings for a Hypervisor.

#### func (*Hypervisor) UnmarshalJSON

```go
func (h *Hypervisor) UnmarshalJSON(input []byte) error
```
UnmarshalJSON is a helper for unmarshalling a Hypervisor

#### func (*Hypervisor) UpdateResources

```go
func (h *Hypervisor) UpdateResources() error
```
UpdateResources syncs Hypervisor resource usage to the data store. It should
only be ran on the actual hypervisor.

#### func (*Hypervisor) Validate

```go
func (h *Hypervisor) Validate() error
```
Validate ensures a Hypervisor has reasonable data. It currently does nothing.

#### func (*Hypervisor) VerifyOnHV

```go
func (h *Hypervisor) VerifyOnHV() error
```
VerifyOnHV verifies that it is being ran on hypervisor with same hostname as id.

#### type Hypervisors

```go
type Hypervisors []*Hypervisor
```

Hypervisors is an alias to a slice of *Hypervisor

#### func  CandidateHasResources

```go
func CandidateHasResources(g *Guest, hs Hypervisors) (Hypervisors, error)
```
CandidateHasResources returns Hypervisors that have availible resources based on
the request Flavor of the Guest.

#### func  CandidateHasSubnet

```go
func CandidateHasSubnet(g *Guest, hs Hypervisors) (Hypervisors, error)
```
CandidateHasSubnet returns Hypervisors that have subnets with availible
addresses in the request Network of the Guest.

#### func  CandidateIsAlive

```go
func CandidateIsAlive(g *Guest, hs Hypervisors) (Hypervisors, error)
```
CandidateIsAlive returns Hypervisors that are "alive" based on heartbeat

#### func  CandidateRandomize

```go
func CandidateRandomize(g *Guest, hs Hypervisors) (Hypervisors, error)
```
CandidateRandomize shuffles the list of Hypervisors.

#### type Job

```go
type Job struct {
	ID         string    `json:"id"`
	RemoteID   string    `json:"remote"` // ID of remote hypervisor/guest job
	Action     string    `json:"action"`
	Guest      string    `json:"guest"`
	Error      string    `json:"error,omitempty"`
	Status     string    `json:"status,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
}
```

Job is a single job for a guest such as create, delete, etc.

#### func (*Job) Refresh

```go
func (j *Job) Refresh() error
```
Refresh reloads a Job from the data store.

#### func (*Job) Save

```go
func (j *Job) Save(ttl time.Duration) error
```
Save persists a job.

#### func (*Job) Validate

```go
func (j *Job) Validate() error
```
Validate ensures required fields are populated.

#### type MistifyAgent

```go
type MistifyAgent struct {
}
```

MistifyAgent is an Agent that communicates with a hypervisor agent to perform
actions relating to guests

#### func (*MistifyAgent) CheckJobStatus

```go
func (agent *MistifyAgent) CheckJobStatus(action, guestID, jobID string) (bool, error)
```
CheckJobStatus looks up whether a guest job has been completed or not.

#### func (*MistifyAgent) CreateGuest

```go
func (agent *MistifyAgent) CreateGuest(guestID string) (string, error)
```
CreateGuest tries to create a new guest on a hypervisor selected from a list of
viable candidates

#### func (*MistifyAgent) DeleteGuest

```go
func (agent *MistifyAgent) DeleteGuest(guestID string) (string, error)
```
DeleteGuest deletes a guest from a hypervisor

#### func (*MistifyAgent) FetchImage

```go
func (agent *MistifyAgent) FetchImage(guestID string) (string, error)
```
FetchImage fetches a disk image that can be used for guest creation

#### func (*MistifyAgent) GetGuest

```go
func (agent *MistifyAgent) GetGuest(guestID string) (*client.Guest, error)
```
GetGuest retrieves information on a guest from an agent

#### func (*MistifyAgent) GuestAction

```go
func (agent *MistifyAgent) GuestAction(guestID, actionName string) (string, error)
```
GuestAction is used to run various actions on a guest under a hypervisor
Actions: "shutdown", "reboot", "restart", "poweroff", "start", "suspend"

#### type Network

```go
type Network struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata"`
}
```

Network is a logical collection of subnets.

#### func (*Network) AddSubnet

```go
func (n *Network) AddSubnet(s *Subnet) error
```
AddSubnet adds a Subnet to the Network.

#### func (*Network) Refresh

```go
func (n *Network) Refresh() error
```
Refresh reloads the Network from the data store.

#### func (*Network) Save

```go
func (n *Network) Save() error
```
Save persists a Network. It will call Validate.

#### func (*Network) Subnets

```go
func (n *Network) Subnets() []string
```
Subnets returns the IDs of the Subnets associated with the network.

#### func (*Network) Validate

```go
func (n *Network) Validate() error
```
Validate ensures a Network has reasonable data. It currently does nothing.

#### type Networks

```go
type Networks []*Network
```

Networks is an alias to a slice of *Network

#### type Resources

```go
type Resources struct {
	Memory uint64 `json:"memory"` // memory in MB
	Disk   uint64 `json:"disk"`   // disk in MB
	CPU    uint32 `json:"cpu"`    // virtual cpus
}
```

Resources represents compute resources

#### type StubAgent

```go
type StubAgent struct {
}
```

StubAgent is an Agenter with stubbed methods for testing

#### func (*StubAgent) CheckJobStatus

```go
func (agent *StubAgent) CheckJobStatus(action, guestID, jobID string) (bool, error)
```
CheckJobStatus looks up whether a guest job has been completed or not.

#### func (*StubAgent) CreateGuest

```go
func (agent *StubAgent) CreateGuest(guestID string) (string, error)
```
CreateGuest is a stub for creating a guest via request to the agent.

#### func (*StubAgent) DeleteGuest

```go
func (agent *StubAgent) DeleteGuest(guestID string) (string, error)
```
DeleteGuest is a stub for deleting a guest via request to the agent.

#### func (*StubAgent) GetGuest

```go
func (agent *StubAgent) GetGuest(guestID string) (*client.Guest, error)
```
GetGuest is a stub for retrieving a guest via request to the agent.

#### func (*StubAgent) GuestAction

```go
func (agent *StubAgent) GuestAction(guestID, actionName string) (string, error)
```
GuestAction is a stub for issuing other basic guest actions via request to the
agent

#### type Subnet

```go
type Subnet struct {
	ID         string            `json:"id"`
	Metadata   map[string]string `json:"metadata"`
	NetworkID  string            `json:"network"`
	Gateway    net.IP            `json:"gateway"`
	CIDR       *net.IPNet        `json:"cidr"`
	StartRange net.IP            `json:"start"` // first usable IP in range
	EndRange   net.IP            `json:"end"`   // last usable IP in range
}
```

Subnet is an actual ip subnet for assigning addresses

#### func (*Subnet) Addresses

```go
func (s *Subnet) Addresses() map[string]string
```
Addresses returns used IP addresses.

#### func (*Subnet) AvailibleAddresses

```go
func (s *Subnet) AvailibleAddresses() []net.IP
```
AvailibleAddresses returns the availible ip addresses. this is probably a
horrible idea for ipv6.

#### func (*Subnet) Delete

```go
func (s *Subnet) Delete() error
```
Delete removes a subnet. It does not ensure it is unused, so use with extreme
caution.

#### func (*Subnet) MarshalJSON

```go
func (s *Subnet) MarshalJSON() ([]byte, error)
```
MarshalJSON is used by the json package

#### func (*Subnet) Refresh

```go
func (s *Subnet) Refresh() error
```
Refresh reloads the Subnet from the data store.

#### func (*Subnet) ReleaseAddress

```go
func (s *Subnet) ReleaseAddress(ip net.IP) error
```
ReleaseAddress releases an address. This does not change any thing that may also
be referring to this address.

#### func (*Subnet) ReserveAddress

```go
func (s *Subnet) ReserveAddress(id string) (net.IP, error)
```
ReserveAddress reserves an ip address. The id is a guest id.

#### func (*Subnet) Save

```go
func (s *Subnet) Save() error
```
Save persists the subnet to the datastore.

#### func (*Subnet) UnmarshalJSON

```go
func (s *Subnet) UnmarshalJSON(input []byte) error
```
UnmarshalJSON is used by the json package

#### func (*Subnet) Validate

```go
func (s *Subnet) Validate() error
```
Validate ensures the values are reasonable.

#### type Subnets

```go
type Subnets []*Subnet
```

Subnets is an alias to a slice of *Subnet

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

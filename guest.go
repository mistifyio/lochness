package lochness

import (
	"encoding/json"
	"math/rand"
	"net"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	// GuestPath is the path in the config store
	GuestPath = "lochness/guests/"
)

type (
	// Guest is a virtual machine
	Guest struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		Type          string            `json:"type"`       // type of guest. currently just kvm
		FlavorID      string            `json:"flavor"`     // resource flavor
		HypervisorID  string            `json:"hypervisor"` // hypervisor. may be blank if not assigned yet
		NetworkID     string            `json:"network"`
		SubnetID      string            `json:"subnet"`
		FWGroupID     string            `json:"fwgroup"`
		MAC           net.HardwareAddr  `json:"mac"`
		IP            net.IP            `json:"ip"`
		Bridge        string            `json:"bridge"`
	}

	// Guests is a convenience type for Guest slices
	Guests []*Guests

	// guestJSON is used to ease json marshal/unmarshal
	guestJSON struct {
		ID           string            `json:"id"`
		Metadata     map[string]string `json:"metadata"`
		Type         string            `json:"type"`       // type of guest. currently just kvm
		FlavorID     string            `json:"flavor"`     // resource flavor
		HypervisorID string            `json:"hypervisor"` // hypervisor. may be blank if not assigned yet
		NetworkID    string            `json:"network"`
		SubnetID     string            `json:"subnet"`
		FWGroupID    string            `json:"fwgroup"`
		MAC          string            `json:"mac"`
		IP           net.IP            `json:"ip"`
		Bridge       string            `json:"bridge"`
	}
)

// MarshalJSON is a helper for marshalling a Guest
func (g *Guest) MarshalJSON() ([]byte, error) {
	data := guestJSON{
		ID:           g.ID,
		Metadata:     g.Metadata,
		Type:         g.Type,
		FlavorID:     g.FlavorID,
		NetworkID:    g.NetworkID,
		SubnetID:     g.SubnetID,
		FWGroupID:    g.FWGroupID,
		HypervisorID: g.HypervisorID,
		IP:           g.IP,
		MAC:          g.MAC.String(),
		Bridge:       g.Bridge,
	}

	return json.Marshal(data)
}

// UnmarshalJSON is a helper for unmarshalling a Guest
func (g *Guest) UnmarshalJSON(input []byte) error {
	data := guestJSON{}

	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	g.ID = data.ID
	g.Metadata = data.Metadata
	g.Type = data.Type
	g.FlavorID = data.FlavorID
	g.NetworkID = data.NetworkID
	g.SubnetID = data.SubnetID
	g.FWGroupID = data.FWGroupID
	g.HypervisorID = data.HypervisorID
	g.IP = data.IP
	g.Bridge = data.Bridge

	a, err := net.ParseMAC(data.MAC)
	if err != nil {
		return err
	}

	g.MAC = a
	return nil

}

// NewGuest create a new blank Guest
func (c *Context) NewGuest() *Guest {
	g := &Guest{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return g
}

// Guest fetches a Guest from the config store
func (c *Context) Guest(id string) (*Guest, error) {
	g := &Guest{
		context: c,
		ID:      id,
	}

	err := g.Refresh()
	if err != nil {
		return nil, err
	}
	return g, nil
}

// key is a helper to generate the config store key
func (g *Guest) key() string {
	return filepath.Join(GuestPath, g.ID, "metadata")
}

// fromResponse is a helper to unmarshal a Guest
func (g *Guest) fromResponse(resp *etcd.Response) error {
	g.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &g)
}

// Refresh reloads from the data store
func (g *Guest) Refresh() error {
	resp, err := g.context.etcd.Get(g.key(), false, false)

	if err != nil {
		return err
	}

	if resp == nil || resp.Node == nil {
		// should this be an error??
		return nil
	}

	return g.fromResponse(resp)
}

// Validate ensures a Guest has reasonable data. It currently does nothing.
// TODO: a guest needs a valid flavor, firewall group, and network
func (g *Guest) Validate() error {
	// do validation stuff...
	return nil
}

// Save persists the Guest to the data store.
func (g *Guest) Save() error {

	if err := g.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(g)

	if err != nil {
		return err
	}

	// if we changed something, don't clobber
	var resp *etcd.Response
	if g.modifiedIndex != 0 {
		resp, err = g.context.etcd.CompareAndSwap(g.key(), string(v), 0, "", g.modifiedIndex)
	} else {
		resp, err = g.context.etcd.Create(g.key(), string(v), 0)
	}
	if err != nil {
		return err
	}

	g.modifiedIndex = resp.EtcdIndex
	return nil
}

// Candidates returns a list of Hypervisors that may run this Guest.
func (g *Guest) Candidates() (Hypervisors, error) {

	// should probably be holding a lock when calling this whole function
	// this whole thing is brute force and nasty...

	f, err := g.context.Flavor(g.FlavorID)
	if err != nil {
		return nil, err
	}

	n, err := g.context.Network(g.NetworkID)
	if err != nil {
		return nil, err
	}
	s := n.Subnets()
	subnets := make(map[string]bool, len(s))
	for _, k := range s {
		subnet, err := g.context.Subnet(k)
		if err != nil {
			return nil, err
		}
		// only include subnets that have availible addresses
		avail := subnet.AvailibleAddresses()
		if len(avail) > 0 {
			subnets[k] = true
		}
	}

	var hypervisors Hypervisors
	err = g.context.ForEachHypervisor(func(h *Hypervisor) error {
		if !h.IsAlive() {
			return nil
		}
		hasSubnet := false
		for k := range h.Subnets() {
			if _, ok := subnets[k]; ok {
				// we want to see if we have any availible ip's?
				hasSubnet = true
				break
			}
		}

		if !hasSubnet {
			return nil
		}

		avail := h.AvailableResources
		if avail.Disk <= f.Disk || avail.Memory <= f.Memory {
			return nil
		}

		hypervisors = append(hypervisors, h)
		return nil
	})

	if err != nil && len(hypervisors) == 0 {
		return nil, err
	}
	return randomizeHypervisors(hypervisors), nil
}

// based on code found on stackoverflow(?)
func randomizeHypervisors(s Hypervisors) Hypervisors {
	for i := range s {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}

	return s
}

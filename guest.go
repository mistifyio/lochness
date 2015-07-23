package lochness

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
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

	// Guests is an alias to a slice of *Guest
	Guests []*Guest

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

	// CandidateFunction is used to select hypervisors that can run the given guest.
	CandidateFunction func(*Guest, Hypervisors) (Hypervisors, error)
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

	if data.ID != "" {
		g.ID = data.ID
	}
	if data.Metadata != nil {
		g.Metadata = data.Metadata
	}
	if data.Type != "" {
		g.Type = data.Type
	}
	if data.FlavorID != "" {
		g.FlavorID = data.FlavorID
	}
	if data.NetworkID != "" {
		g.NetworkID = data.NetworkID
	}
	if data.SubnetID != "" {
		g.SubnetID = data.SubnetID
	}
	if data.FWGroupID != "" {
		g.FWGroupID = data.FWGroupID
	}
	if data.HypervisorID != "" {
		g.HypervisorID = data.HypervisorID
	}
	if data.IP != nil {
		g.IP = data.IP
	}
	if data.Bridge != "" {
		g.Bridge = data.Bridge
	}

	if data.MAC != "" {
		a, err := net.ParseMAC(data.MAC)
		if err != nil {
			return err
		}

		g.MAC = a
	}
	return nil

}

// NewGuest create a new blank Guest
func (c *Context) NewGuest() *Guest {
	g := &Guest{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	// Generate a MAC based on the ID. May be overwritten later.
	md5ID := md5.Sum([]byte(g.ID))
	mac := fmt.Sprintf("02:%x:%x:%x:%x:%x",
		md5ID[0:1],
		md5ID[1:2],
		md5ID[2:3],
		md5ID[3:4],
		md5ID[4:5],
	)
	g.MAC, _ = net.ParseMAC(mac)

	return g
}

// Guest fetches a Guest from the config store
func (c *Context) Guest(id string) (*Guest, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	g := &Guest{
		context: c,
		ID:      id,
	}

	err = g.Refresh()
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
	if g.FlavorID == "" {
		return errors.New("missing flavor id")
	}

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

// Destroy removes a guest
func (g *Guest) Destroy() error {
	if g.modifiedIndex == 0 {
		// it has not been saved?
		return errors.New("not persisted")
	}

	if g.HypervisorID != "" {
		hypervisor, err := g.context.Hypervisor(g.HypervisorID)
		if err != nil {
			return err
		}
		if err := hypervisor.RemoveGuest(g); err != nil {
			return err
		}
	}

	// XXX: another instance where transactions would be helpful
	if _, err := g.context.etcd.CompareAndDelete(g.key(), "", g.modifiedIndex); err != nil {
		return err
	}

	if _, err := g.context.etcd.Delete(filepath.Join(GuestPath, g.ID), true); err != nil {
		return err
	}
	return nil
}

// Candidates returns a list of Hypervisors that may run this Guest.
func (g *Guest) Candidates(f ...CandidateFunction) (Hypervisors, error) {
	// this is not terribly effecient, but is fairly easy to understand

	var hypervisors Hypervisors
	_ = g.context.ForEachHypervisor(func(h *Hypervisor) error {
		hypervisors = append(hypervisors, h)
		return nil
	})

	if len(hypervisors) == 0 {
		return nil, errors.New("no hypervisors")
	}

	for _, fn := range f {
		hs, err := fn(g, hypervisors)
		if err != nil {
			return nil, err
		}
		hypervisors = hs
		if len(hypervisors) == 0 {
			return nil, errors.New("no hypervisors")
		}
	}

	return hypervisors, nil
}

// CandidateIsAlive returns Hypervisors that are "alive" based on heartbeat
func CandidateIsAlive(g *Guest, hs Hypervisors) (Hypervisors, error) {

	var hypervisors Hypervisors
	for _, h := range hs {
		if h.IsAlive() {
			hypervisors = append(hypervisors, h)
		}
	}
	return hypervisors, nil
}

// CandidateHasResources returns Hypervisors that have availible resources
// based on the request Flavor of the Guest.
func CandidateHasResources(g *Guest, hs Hypervisors) (Hypervisors, error) {
	f, err := g.context.Flavor(g.FlavorID)
	if err != nil {
		return nil, err
	}

	var hypervisors Hypervisors
	for _, h := range hs {
		avail := h.AvailableResources
		if avail.Disk >= f.Disk && avail.Memory >= f.Memory && avail.CPU >= f.CPU {
			hypervisors = append(hypervisors, h)
		}
	}
	return hypervisors, nil
}

// CandidateHasSubnet returns Hypervisors that have subnets with availible addresses
// in the request Network of the Guest.
func CandidateHasSubnet(g *Guest, hs Hypervisors) (Hypervisors, error) {
	n, err := g.context.Network(g.NetworkID)
	if err != nil {
		return nil, err
	}
	s := n.Subnets()
	subnets := make(map[string]bool, len(s))

	var hypervisors Hypervisors

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

	for _, h := range hs {
		hasSubnet := false
		for k := range h.Subnets() {
			if _, ok := subnets[k]; ok {
				hasSubnet = true
				break
			}
		}
		if hasSubnet {
			hypervisors = append(hypervisors, h)
		}
	}

	return hypervisors, nil
}

// CandidateRandomize shuffles the list of Hypervisors.
func CandidateRandomize(g *Guest, hs Hypervisors) (Hypervisors, error) {
	return randomizeHypervisors(hs), nil
}

// based on code found on stackoverflow(?)
func randomizeHypervisors(s Hypervisors) Hypervisors {
	for i := range s {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}

	return s
}

// DefaultCandidateFunctions is a default list of CandidateFunctions for general use
var DefaultCandidateFunctions = []CandidateFunction{
	CandidateIsAlive,
	CandidateHasSubnet,
	CandidateHasResources,
	CandidateRandomize,
}

// FirstGuest will return the first guest for which the function returns true.
func (c *Context) FirstGuest(f func(*Guest) bool) (*Guest, error) {
	// we could get this recursively, but we may want to change how refresh works just a bit
	resp, err := c.etcd.Get(GuestPath, false, false)
	if err != nil {
		return nil, err
	}
	for _, n := range resp.Node.Nodes {
		h, err := c.Guest(filepath.Base(n.Key))
		if err != nil {
			return nil, err
		}

		if f(h) {
			return h, nil
		}
	}
	return nil, nil
}

// ForEachGuest will run f on each Guest. It will stop iteration if f returns an error.
func (c *Context) ForEachGuest(f func(*Guest) error) error {
	resp, err := c.etcd.Get(GuestPath, false, false)
	if err != nil {
		return err
	}
	for _, n := range resp.Node.Nodes {
		g, err := c.Guest(filepath.Base(n.Key))
		if err != nil {
			return err
		}

		if err := f(g); err != nil {
			return err
		}
	}
	return nil
}

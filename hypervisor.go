package lochness

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mistifyio/lochness/pkg/kv"
	"github.com/pborman/uuid"
)

var (
	// HypervisorPath is the path in the config store
	HypervisorPath = "/lochness/hypervisors/"
	// id of currently running hypervisor
	hypervisorID = ""
)

type (
	// Hypervisor is a physical box on which guests run
	Hypervisor struct {
		context            *Context
		modifiedIndex      uint64
		ID                 string            `json:"id"`
		Metadata           map[string]string `json:"metadata"`
		IP                 net.IP            `json:"ip"`
		Netmask            net.IP            `json:"netmask"`
		Gateway            net.IP            `json:"gateway"`
		MAC                net.HardwareAddr  `json:"mac"`
		TotalResources     Resources         `json:"total_resources"`
		AvailableResources Resources         `json:"available_resources"`
		subnets            map[string]string
		guests             []string
		alive              bool
		lock               kv.Lock
		// Config is a set of key/values for driving various config options. writes should
		// only be done using SetConfig
		Config map[string]string
	}

	// Hypervisors is an alias to a slice of *Hypervisor
	Hypervisors []*Hypervisor

	// hypervisorJSON is used to ease json marshal/unmarshal
	hypervisorJSON struct {
		ID                 string            `json:"id"`
		Metadata           map[string]string `json:"metadata"`
		IP                 net.IP            `json:"ip"`
		Netmask            net.IP            `json:"netmask"`
		Gateway            net.IP            `json:"gateway"`
		MAC                string            `json:"mac"`
		TotalResources     Resources         `json:"total_resources"`
		AvailableResources Resources         `json:"available_resources"`
	}
)

// MarshalJSON is a helper for marshalling a Hypervisor
func (h *Hypervisor) MarshalJSON() ([]byte, error) {
	data := hypervisorJSON{
		ID:                 h.ID,
		Metadata:           h.Metadata,
		IP:                 h.IP,
		Netmask:            h.Netmask,
		Gateway:            h.Gateway,
		MAC:                h.MAC.String(),
		TotalResources:     h.TotalResources,
		AvailableResources: h.AvailableResources,
	}

	return json.Marshal(data)
}

// UnmarshalJSON is a helper for unmarshalling a Hypervisor
func (h *Hypervisor) UnmarshalJSON(input []byte) error {
	data := hypervisorJSON{}

	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	if data.ID != "" {
		h.ID = data.ID
	}

	if data.Metadata != nil {
		h.Metadata = data.Metadata
	}

	if data.IP != nil {
		h.IP = data.IP
	}
	if data.Netmask != nil {
		h.Netmask = data.Netmask
	}
	if data.Gateway != nil {
		h.Gateway = data.Gateway
	}
	if &data.TotalResources != nil {
		h.TotalResources = data.TotalResources
	}
	if &data.AvailableResources != nil {
		h.AvailableResources = data.AvailableResources
	}

	if data.MAC != "" {
		a, err := net.ParseMAC(data.MAC)
		if err != nil {
			return err
		}

		h.MAC = a
	}

	if h.Config == nil {
		h.Config = make(map[string]string)
	}

	return nil

}

// blankHypervisor is a helper for creating a blank Hypervisor.
func (c *Context) blankHypervisor(id string) *Hypervisor {
	h := &Hypervisor{
		context:  c,
		ID:       id,
		Metadata: make(map[string]string),
		subnets:  make(map[string]string),
		Config:   make(map[string]string),
		guests:   make([]string, 0, 0),
	}

	if id == "" {
		h.ID = uuid.New()
	}

	return h
}

// NewHypervisor create a new blank Hypervisor.
func (c *Context) NewHypervisor() *Hypervisor {
	return c.blankHypervisor("")
}

// Hypervisor fetches a Hypervisor from the config store.
func (c *Context) Hypervisor(id string) (*Hypervisor, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	h := c.blankHypervisor(id)

	err = h.Refresh()
	if err != nil {
		return nil, err
	}

	return h, nil
}

// key is a helper to generate the config store key.
func (h *Hypervisor) key() string {
	return filepath.Join(HypervisorPath, h.ID, "metadata")
}

// Refresh reloads a Hypervisor from the data store.
func (h *Hypervisor) Refresh() error {
	prefix := filepath.Join(HypervisorPath, h.ID)

	nodes, err := h.context.kv.GetAll(prefix)
	if err != nil {
		return err
	}

	// handle metadata
	key := filepath.Join(prefix, "metadata")
	value, ok := nodes[key]
	if !ok {
		return errors.New("metadata key is missing")
	}

	if err := json.Unmarshal(value.Data, &h); err != nil {
		return err
	}
	h.modifiedIndex = value.Index
	delete(nodes, key)

	// handle heartbeat
	key = filepath.Join(prefix, "heartbeat")
	_, ok = nodes[key]
	if ok {
		//if exists, then it's alive
		h.alive = true
		delete(nodes, key)
	}

	config := map[string]string{}
	guests := []string{}
	subnets := map[string]string{}

	// TODO(needs tests)
	for k, v := range nodes {
		elements := strings.Split(k, "/")
		base := elements[len(elements)-1]
		dir := elements[len(elements)-2]

		switch dir {
		case "subnets":
			subnets[base] = string(v.Data)
		case "guests":
			guests = append(guests, base)
		case "config":
			config[base] = string(v.Data)
		}
	}

	h.Config = config
	h.guests = guests
	h.subnets = subnets

	return nil
}

// TODO: figure out safe amount of memory to report and how to limit it
func memory() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(f)
	mem := 0
	for scanner.Scan() {
		if !strings.HasPrefix(scanner.Text(), "MemTotal:") {
			continue
		}
		vals := strings.Split(scanner.Text(), " ")
		mem, err = strconv.Atoi(vals[len(vals)-2])
		if err != nil {
			return 0, err
		}
	}
	return uint64(mem) * 80 / 100 / 1024, scanner.Err()
}

// TODO: parameterize this
func disk(path string) (uint64, error) {
	stat := &syscall.Statfs_t{}
	err := syscall.Statfs(path, stat)
	return uint64(stat.Bsize) * stat.Bavail * 80 / 100 / 1024 / 1024, err
}

// cpu gets number of CPU's.
func cpu() (uint32, error) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "processor") {
			count++
		}
	}
	return uint32(count - 1), scanner.Err()
}

// canonicalizeUUID is a helper to ensure UUID's are in a single form and case
func canonicalizeUUID(id string) (string, error) {
	i := uuid.Parse(id)
	if i == nil {
		return "", fmt.Errorf("invalid UUID: %s", id)
	}

	return strings.ToLower(i.String()), nil
}

// SetHypervisorID sets the id of the current hypervisor.
// It should be used by all daemons that are ran on a hypervisor and are expected to interact with the data stores directly.
// Passing in a blank string will fall back to first checking the environment variable "HYPERVISOR_ID" and then using the hostname.
// ID must be a valid UUID.
// ID will be lowercased.
func SetHypervisorID(id string) (string, error) {
	// the if statement approach is clunky and probably needs refining

	var err error

	if id == "" {
		id = os.Getenv("HYPERVISOR_ID")
	}

	if id == "" {
		id, err = os.Hostname()
		if err != nil {
			return "", err
		}
	}

	if id == "" {
		return "", errors.New("unable to discover an id to set")
	}

	id, err = canonicalizeUUID(id)
	if err != nil {
		return "", err
	}

	// purposefully set here rather than above, in case, for some reason,
	// caller knows there is a previous, usable value
	hypervisorID = id
	return hypervisorID, nil
}

// GetHypervisorID gets the hypervisor id as set with SetHypervisorID.
// It does not make an attempt to discover the id if not set.
func GetHypervisorID() string {
	return hypervisorID
}

// VerifyOnHV verifies that it is being ran on hypervisor with same hostname as id.
func (h *Hypervisor) VerifyOnHV() error {
	if GetHypervisorID() != h.ID {
		return errors.New("Hypervisor ID does not match hostname/environment")
	}
	return nil
}

// calcGuestsUsage calculates total resource usage of managed guests.
// Note that CPU "usage" is intentionally ignored as cores are not directly allocated to guests.
func (h *Hypervisor) calcGuestsUsage() (Resources, error) {
	usage := Resources{}
	err := h.ForEachGuest(func(guest *Guest) error {
		// cache?
		flavor, err := h.context.Flavor(guest.FlavorID)
		if err != nil {
			return err
		}
		usage.Memory += flavor.Memory
		usage.Disk += flavor.Disk
		return nil
	})
	if err != nil {
		return Resources{}, err
	}
	return usage, nil
}

// UpdateResources syncs Hypervisor resource usage to the data store.
// It should only be ran on the actual hypervisor.
func (h *Hypervisor) UpdateResources() error {
	if err := h.VerifyOnHV(); err != nil {
		return err
	}

	m, err := memory()
	if err != nil {
		return err
	}
	guestDiskDir, ok := h.Config["guestDiskDir"]
	if !ok {
		guestDiskDir = "/mistify/guests"
	}
	d, err := disk(guestDiskDir)
	if err != nil {
		return err
	}
	c, err := cpu()
	if err != nil {
		return err
	}

	h.TotalResources = Resources{Memory: m, Disk: d, CPU: c}

	usage, err := h.calcGuestsUsage()
	if err != nil {
		return err
	}

	h.AvailableResources = Resources{
		Memory: h.TotalResources.Memory - usage.Memory,
		Disk:   h.TotalResources.Disk - usage.Disk,
		CPU:    h.TotalResources.CPU - usage.CPU,
	}

	return h.Save()
}

// Validate ensures a Hypervisor has reasonable data.
// It currently does nothing.
func (h *Hypervisor) Validate() error {
	// TODO: do validation stuff...
	if h.ID == "" {
		return errors.New("no id")
	}
	if uuid.Parse(h.ID) == nil {
		return errors.New("invalid id")
	}
	return nil
}

// Save persists a FWGroup.
// It will call Validate.
func (h *Hypervisor) Save() error {
	if err := h.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(h)

	if err != nil {
		return err
	}

	index, err := h.context.kv.Update(h.key(), kv.Value{Data: v, Index: h.modifiedIndex})
	if err != nil {
		return err
	}
	h.modifiedIndex = index
	return nil
}

// the many side of many:one relationships is done with nested keys
func (h *Hypervisor) subnetKey(s *Subnet) string {
	var key string
	if s != nil {
		key = s.ID
	}
	return filepath.Join(HypervisorPath, h.ID, "subnets", key)
}

// AddSubnet adds a subnet to a Hypervisor.
func (h *Hypervisor) AddSubnet(s *Subnet, bridge string) error {
	// Make sure the hypervisor exists
	if h.modifiedIndex == 0 {
		if err := h.Refresh(); err != nil {
			return err
		}
	}

	// Make sure the subnet exists
	if s.modifiedIndex == 0 {
		if err := s.Refresh(); err != nil {
			return err
		}
	}

	err := h.context.kv.Set(filepath.Join(h.subnetKey(s)), bridge)
	if err == nil {
		h.subnets[s.ID] = bridge
	}
	return err
}

// RemoveSubnet removes a subnet from a Hypervisor.
func (h *Hypervisor) RemoveSubnet(s *Subnet) error {
	if err := h.context.kv.Delete(filepath.Join(h.subnetKey(s)), false); err != nil {
		return err
	}
	delete(h.subnets, s.ID)
	return nil
}

// Subnets returns the subnet/bridge mappings for a Hypervisor.
func (h *Hypervisor) Subnets() map[string]string {
	return h.subnets
}

// heartbeatKey is a helper for generating a key for config store.
func (h *Hypervisor) heartbeatKey() string {
	return filepath.Join(HypervisorPath, h.ID, "heartbeat")
}

// Heartbeat announces the availability of a hypervisor.
// In general, this is useful for service announcement/discovery.
// Should be ran from the hypervisor, or something monitoring it.
func (h *Hypervisor) Heartbeat(ttl time.Duration) error {
	if err := h.VerifyOnHV(); err != nil {
		return err
	}

	if h.lock == nil {
		lock, err := h.context.kv.Lock(h.heartbeatKey(), ttl)
		if err != nil {
			return err
		}
		h.lock = lock
	}

	if err := h.lock.Set([]byte(time.Now().String())); err != nil {
		return err
	}

	h.alive = true
	return nil
}

// IsAlive returns true if the heartbeat is present.
func (h *Hypervisor) IsAlive() bool {
	return h.alive
}

// guestKey for generating a key for config store.
func (h *Hypervisor) guestKey(g *Guest) string {
	var key string
	if g != nil {
		key = g.ID
	}
	return filepath.Join(HypervisorPath, h.ID, "guests", key)
}

// AddGuest adds a Guest to the Hypervisor.
// It reserves an IPaddress for the Guest.
// It also updates the Guest.
func (h *Hypervisor) AddGuest(g *Guest) error {

	// make sure we have subnet guest wants.  we should have this figured out
	// when we selected this hypervisor, so this is sort of silly to do again
	// we need to rethink how we do this

	n, err := h.context.Network(g.NetworkID)
	if err != nil {
		return err
	}
	var s *Subnet
	var bridge string
LOOP:
	for _, k := range n.Subnets() {

		for id, br := range h.subnets {
			if id == k {
				subnet, err := h.context.Subnet(id)
				if err != nil {
					return err
				}
				avail := subnet.AvailableAddresses()
				if len(avail) > 0 {
					s = subnet
					bridge = br
					break LOOP
				}
			}
		}
	}

	if s == nil {
		return errors.New("no suitable subnet found")
	}

	ip, err := s.ReserveAddress(g.ID)

	if err != nil {
		return err
	}

	// an instance where transactions would be cool...
	g.HypervisorID = h.ID
	g.IP = ip
	g.SubnetID = s.ID
	g.Bridge = bridge

	err = h.context.kv.Set(filepath.Join(h.guestKey(g)), g.ID)

	if err != nil {
		return err
	}

	err = g.Save()

	if err != nil {
		return err
	}

	h.guests = append(h.guests, g.ID)

	return nil
}

// RemoveGuest removes a guest from the hypervisor.
// Also releases the IP.
func (h *Hypervisor) RemoveGuest(g *Guest) error {
	if g.HypervisorID != h.ID {
		return errors.New("guest does not belong to hypervisor")
	}

	subnet, err := h.context.Subnet(g.SubnetID)
	if err != nil {
		return err
	}
	if err := subnet.ReleaseAddress(g.IP); err != nil {
		return err
	}

	if err := h.context.kv.Delete(filepath.Join(h.guestKey(g)), false); err != nil {
		return err
	}

	g.HypervisorID = ""
	g.IP = nil
	g.SubnetID = ""
	g.Bridge = ""

	if err := g.Save(); err != nil {
		return err
	}

	newGuests := make([]string, 0, len(h.guests)-1)
	for i := 0; i < len(h.guests); i++ {
		if h.guests[i] != g.ID {
			newGuests = append(newGuests, h.guests[i])
		}
	}
	h.guests = newGuests

	return nil
}

// Guests returns a slice of GuestIDs assigned to the Hypervisor.
func (h *Hypervisor) Guests() []string {
	return h.guests
}

// ForEachGuest will run f on each Guest.
// It will stop iteration if f returns an error.
func (h *Hypervisor) ForEachGuest(f func(*Guest) error) error {
	for _, id := range h.guests {
		guest, err := h.context.Guest(id)
		if err != nil {
			return err
		}

		if err := f(guest); err != nil {
			return err
		}
	}
	return nil
}

// FirstHypervisor will return the first hypervisor for which the function returns true.
func (c *Context) FirstHypervisor(f func(*Hypervisor) bool) (*Hypervisor, error) {
	keys, err := c.kv.Keys(HypervisorPath)
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		h, err := c.Hypervisor(filepath.Base(k))
		if err != nil {
			return nil, err
		}

		if f(h) {
			return h, nil
		}
	}
	return nil, nil
}

// ForEachHypervisor will run f on each Hypervisor.
// It will stop iteration if f returns an error.
func (c *Context) ForEachHypervisor(f func(*Hypervisor) error) error {
	// should we condense this to a single kv call?
	// We would need to rework how we "load" hypervisor a bit

	keys, err := c.kv.Keys(HypervisorPath)
	if err != nil {
		return err
	}
	for _, k := range keys {
		h, err := c.Hypervisor(filepath.Base(k))
		if err != nil {
			return err
		}

		if err := f(h); err != nil {
			return err
		}
	}
	return nil
}

// SetConfig sets a single Hypervisor Config value.
// Set value to "" to unset.
func (h *Hypervisor) SetConfig(key, value string) error {
	if key == "" {
		return errors.New("empty config key")
	}

	if value != "" {
		if err := h.context.kv.Set(filepath.Join(HypervisorPath, h.ID, "config", key), value); err != nil {
			return err
		}

		h.Config[key] = value
	} else {
		err := h.context.kv.Delete(filepath.Join(HypervisorPath, h.ID, "config", key), false)
		if err != nil && !h.context.kv.IsKeyNotFound(err) {
			return err
		}
		delete(h.Config, key)
	}
	return nil
}

// Destroy removes a hypervisor.
// The Hypervisor must not have any guests.
func (h *Hypervisor) Destroy() error {
	if len(h.guests) != 0 {
		// XXX: should use an error var?
		return errors.New("not empty")
	}

	if h.modifiedIndex == 0 {
		// it has not been saved?
		return errors.New("not persisted")
	}

	if err := h.context.kv.Remove(h.key(), h.modifiedIndex); err != nil {
		return err
	}

	return h.context.kv.Delete(filepath.Join(HypervisorPath, h.ID), true)
}

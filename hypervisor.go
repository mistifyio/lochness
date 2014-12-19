package lochness

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	HypervisorPath = "lochness/hypervisors/"
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
	}

	Hypervisors []*Hypervisor

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

func (t *Hypervisor) MarshalJSON() ([]byte, error) {
	data := hypervisorJSON{
		ID:                 t.ID,
		Metadata:           t.Metadata,
		IP:                 t.IP,
		Netmask:            t.Netmask,
		Gateway:            t.Gateway,
		MAC:                t.MAC.String(),
		TotalResources:     t.TotalResources,
		AvailableResources: t.AvailableResources,
	}

	return json.Marshal(data)
}

func (t *Hypervisor) UnmarshalJSON(input []byte) error {
	data := hypervisorJSON{}

	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	t.ID = data.ID
	t.Metadata = data.Metadata
	t.IP = data.IP
	t.Netmask = data.Netmask
	t.Gateway = data.Gateway
	t.TotalResources = data.TotalResources
	t.AvailableResources = data.AvailableResources

	if data.MAC != "" {
		a, err := net.ParseMAC(data.MAC)
		if err != nil {
			return err
		}

		t.MAC = a
	}

	return nil

}

func (c *Context) blankHypervisor(id string) *Hypervisor {
	h := &Hypervisor{
		context: c,
		ID:      id,
		subnets: make(map[string]string),
		guests:  make([]string, 0, 0),
	}

	if id == "" {
		h.ID = uuid.New()
	}

	return h
}

func (c *Context) NewHypervisor() *Hypervisor {
	return c.blankHypervisor("")
}

func (c *Context) Hypervisor(id string) (*Hypervisor, error) {
	t := c.blankHypervisor(id)

	err := t.Refresh()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Hypervisor) key() string {
	return filepath.Join(HypervisorPath, t.ID, "metadata")
}

// Refresh reloads from the data store
func (t *Hypervisor) Refresh() error {
	resp, err := t.context.etcd.Get(filepath.Join(HypervisorPath, t.ID), false, true)

	if err != nil {
		return err
	}

	for _, n := range resp.Node.Nodes {
		key := filepath.Base(n.Key)
		switch key {

		case "metadata":
			if err := json.Unmarshal([]byte(n.Value), &t); err != nil {
				return err
			}
			t.modifiedIndex = n.ModifiedIndex
		case "heartbeat":
			//if exists, then its alive
			t.alive = true

		case "subnets":
			for _, n := range n.Nodes {
				t.subnets[filepath.Base(n.Key)] = n.Value
			}
		case "guests":
			for _, n := range n.Nodes {
				t.guests = append(t.guests, filepath.Base(n.Key))
			}
		}
	}

	return nil
}

// TODO: figure out safe amount of memory to report and how to limit it (etcd?)
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

// TODO: figure out real zfs disk size
func disk() (uint64, error) {
	stat := &syscall.Statfs_t{}
	err := syscall.Statfs("/", stat)
	return uint64(stat.Bsize) * stat.Bavail * 80 / 100 / 1024 / 1024, err
}

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

func (t *Hypervisor) verifyOnHV() error {
	hostname := os.Getenv("TEST_HOSTNAME")
	if hostname == "" {
		var err error
		hostname, err = os.Hostname()
		if err != err {
			return err
		}
	}
	if hostname != t.ID {
		return errors.New("Hypervisor ID does not match hostname")
	}
	return nil
}

func (t *Hypervisor) calcGuestsUsage() (Resources, error) {
	usage := Resources{}
	for _, guestID := range t.Guests() {
		guest, err := t.context.Guest(guestID)
		if err != nil {
			return Resources{}, err
		}

		// cache?
		flavor, err := t.context.Flavor(guest.FlavorID)
		if err != nil {
			return Resources{}, err
		}
		usage.Memory += flavor.Memory
		usage.Disk += flavor.Disk
	}
	return usage, nil
}

// UpdateResources syncs resource usage to the data store
func (t *Hypervisor) UpdateResources() error {
	if err := t.verifyOnHV(); err != nil {
		return err
	}

	m, err := memory()
	if err != nil {
		return err
	}
	d, err := disk()
	if err != nil {
		return err
	}
	c, err := cpu()
	if err != nil {
		return err
	}

	t.TotalResources = Resources{Memory: m, Disk: d, CPU: c}

	usage, err := t.calcGuestsUsage()
	if err != nil {
		return err
	}

	t.AvailableResources = Resources{
		Memory: t.TotalResources.Memory - usage.Memory,
		Disk:   t.TotalResources.Disk - usage.Disk,
		CPU:    t.TotalResources.CPU - usage.CPU,
	}

	return t.Save()
}

func (t *Hypervisor) Validate() error {
	// do validation stuff...
	return nil
}

func (t *Hypervisor) Save() error {

	if err := t.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(t)

	if err != nil {
		return err
	}

	// if we changed something, don't clobber
	var resp *etcd.Response
	if t.modifiedIndex != 0 {
		resp, err = t.context.etcd.CompareAndSwap(t.key(), string(v), 0, "", t.modifiedIndex)
	} else {
		resp, err = t.context.etcd.Create(t.key(), string(v), 0)
	}
	if err != nil {
		return err
	}

	t.modifiedIndex = resp.EtcdIndex

	return nil
}

// the many side of many:one relationships is done with nested keys

func (t *Hypervisor) subnetKey(s *Subnet) string {
	var key string
	if s != nil {
		key = s.ID
	}
	return filepath.Join(HypervisorPath, t.ID, "subnets", key)
}

func (t *Hypervisor) AddSubnet(s *Subnet, bridge string) error {
	_, err := t.context.etcd.Set(filepath.Join(t.subnetKey(s)), bridge, 0)
	return err
}

func (t *Hypervisor) Subnets() map[string]string {
	return t.subnets
}

func (t *Hypervisor) heartbeatKey() string {
	return filepath.Join(HypervisorPath, t.ID, "heartbeat")
}

// Heartbeat announces the avilibility of a hypervisor.  In general, this is useful for
// service announcement/discovery. Should be ran from the hypervisor, or something monitoring it.
func (t *Hypervisor) Heartbeat(ttl time.Duration) error {
	if err := t.verifyOnHV(); err != nil {
		return err
	}

	v := time.Now().String()
	_, err := t.context.etcd.Set(t.heartbeatKey(), v, uint64(ttl.Seconds()))
	return err
}

// IsAlive checks if the Heartbeat is availible
func (t *Hypervisor) IsAlive() bool {
	return t.alive
}

func (t *Hypervisor) guestKey(g *Guest) string {
	var key string
	if g != nil {
		key = g.ID
	}
	return filepath.Join(HypervisorPath, t.ID, "guests", key)
}

func (t *Hypervisor) AddGuest(g *Guest) error {

	// make sure we have subnet guest wants.  we should have this figured out
	// when we selected this hypervisor, so this is sort of silly to do again
	// we need to rethink how we do this

	n, err := t.context.Network(g.NetworkID)
	if err != nil {
		return err
	}
	subnets, err := n.Subnets()
	if err != nil {
		return err
	}

	var s *Subnet
	var bridge string
LOOP:
	for _, k := range subnets {

		for id, br := range t.subnets {
			if id == k {
				subnet, err := t.context.Subnet(id)
				if err != nil {
					return err
				}
				avail := subnet.AvailibleAddresses()
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
	g.HypervisorID = t.ID
	g.IP = ip
	g.SubnetID = s.ID
	g.Bridge = bridge

	_, err = t.context.etcd.Set(filepath.Join(t.guestKey(g)), g.ID, 0)

	if err != nil {
		return err
	}

	return g.Save()
}

func (t *Hypervisor) Guests() []string {
	return t.guests
}

func (c *Context) ForEachHypervisor(f func(*Hypervisor) error) error {
	// should we condense this to a single etcd call?
	// We would need to rework how we "load" hypervisor a bit
	resp, err := c.etcd.Get(HypervisorPath, false, false)
	if err != nil {
		return err
	}
	for _, n := range resp.Node.Nodes {
		h, err := c.Hypervisor(filepath.Base(n.Key))
		if err != nil {
			return err
		}

		if err := f(h); err != nil {
			return err
		}
	}
	return nil
}

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

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	HypervisorPath = "lochness/hypervisors/"
)

type (
	// Hypervisor is a physical box on which guests run
	Hypervisor struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		IP            net.IP            `json:"ip"`
		Netmask       net.IP            `json:"netmask"`
		Gateway       net.IP            `json:"gateway"`
		MAC           net.HardwareAddr  `json:"mac"`
		Resources
	}

	// helper struct for bridge-to-subnet mapping
	subnetInfo struct {
		Bridge string `json:"bridge"`
	}

	Hypervisors []*Hypervisor

	hypervisorJSON struct {
		ID       string            `json:"id"`
		Metadata map[string]string `json:"metadata"`
		IP       net.IP            `json:"ip"`
		Netmask  net.IP            `json:"netmask"`
		Gateway  net.IP            `json:"gateway"`
		MAC      string            `json:"mac"`
		Resources
	}
)

func (t *Hypervisor) MarshalJSON() ([]byte, error) {
	data := hypervisorJSON{
		ID:       t.ID,
		Metadata: t.Metadata,
		IP:       t.IP,
		Netmask:  t.Netmask,
		Gateway:  t.Gateway,
		MAC:      t.MAC.String(),
		Resources: Resources{
			Memory: t.Memory,
			Disk:   t.Disk,
			CPU:    t.CPU,
		},
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
	t.Memory = data.Memory
	t.Disk = data.Disk
	t.CPU = data.CPU

	a, err := net.ParseMAC(data.MAC)
	if err != nil {
		return err
	}

	t.MAC = a
	return nil

}

func (c *Context) NewHypervisor() *Hypervisor {
	t := &Hypervisor{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return t
}

func (c *Context) Hypervisor(id string) (*Hypervisor, error) {
	t := &Hypervisor{
		context: c,
		ID:      id,
	}

	err := t.Refresh()
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Hypervisor) key() string {
	return filepath.Join(HypervisorPath, t.ID, "metadata")
}

func (t *Hypervisor) fromResponse(resp *etcd.Response) error {
	t.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &t)
}

// Refresh reloads from the data store
func (t *Hypervisor) Refresh() error {
	resp, err := t.context.etcd.Get(t.key(), false, false)

	if err != nil {
		return err
	}

	if resp == nil || resp.Node == nil {
		// should this be an error??
		return nil
	}

	return t.fromResponse(resp)
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

	t.Memory = m
	t.Disk = d
	t.CPU = c

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
	i := subnetInfo{
		Bridge: bridge,
	}

	v, err := json.Marshal(&i)
	if err != nil {
		return err
	}

	_, err = t.context.etcd.Set(filepath.Join(t.subnetKey(s)), string(v), 0)
	return err
}

func (t *Hypervisor) Subnets() (map[string]string, error) {
	resp, err := t.context.etcd.Get(t.subnetKey(nil), true, true)
	if err != nil {
		return nil, err
	}

	subnets := make(map[string]string, resp.Node.Nodes.Len())
	for _, n := range resp.Node.Nodes {
		var i subnetInfo
		if err := json.Unmarshal([]byte(n.Value), &i); err != nil {
			return nil, err
		}
		subnets[filepath.Base(n.Key)] = i.Bridge
	}

	return subnets, nil
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
func (t *Hypervisor) IsAlive() (bool, error) {
	resp, err := t.context.etcd.Get(t.heartbeatKey(), false, false)

	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			return false, nil
		}
		return false, err
	}

	if resp == nil || resp.Node == nil {
		return false, nil
	}

	return true, nil
}

func (t *Hypervisor) guestKey(g *Guest) string {
	var key string
	if g != nil {
		key = g.ID
	}
	return filepath.Join(HypervisorPath, t.ID, "guests", key)
}

func (t *Hypervisor) AddGuest(g *Guest) error {
	_, err := t.context.etcd.Set(filepath.Join(t.guestKey(g)), "", 0)

	if err != nil {
		return err
	}

	// an instance where transactions would be cool...
	g.HypervisorID = t.ID

	fmt.Println(t.ID)
	return g.Save()
}

func (t *Hypervisor) Guests() ([]string, error) {
	resp, err := t.context.etcd.Get(t.guestKey(nil), true, true)
	if err != nil {
		return nil, err
	}

	var guests []string

	for _, n := range resp.Node.Nodes {
		guests = append(guests, filepath.Base(n.Key))
	}

	return guests, nil
}

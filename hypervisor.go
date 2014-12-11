package lochness

import (
	"encoding/json"
	"net"
	"path/filepath"

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
		Memory        uint64            `json:"memory"` // memory in MB that we can use for guests
		Disk          uint64            `json:"disk"`   // disk in MB that we can use for guests
		CPU           uint32            `json:"cpu"`    // maximum number of virtual cpu's
	}

	// helper struct for bridge-to-subnet mapping
	subnetInfo struct {
		Bridge string `json:"bridge"`
	}

	Hypervisors []*Hypervisor
)

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

// the many side of many:one relations shiops is done with nested keys

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

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
		Memory        uint64            `json:"memory"`  // memory in MB that we can use for guests
		Disk          uint64            `json:"disk"`    // disk in MB that we can use for guests
		CPU           uint32            `json:"cpu"`     // maximum number of virtual cpu's
		Subnets       map[string]string `json:"subnets"` // a map of subnet id's to bridges
	}

	Hypervisors []*Hypervisor
)

func (c *Context) NewHypervisor() *Hypervisor {
	t := &Hypervisor{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
		Subnets:  make(map[string]string),
	}

	return t
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

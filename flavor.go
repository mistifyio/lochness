package lochness

import (
	"encoding/json"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	FlavorPath = "lochness/flavors/"
)

type (
	// Flavor defines the virtual resources for a guest
	Flavor struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		Memory        uint64            `json:"memory"` // memory in MB
		Disk          uint64            `json:"disk"`   // disk in MB
		CPU           uint32            `json:"cpu"`    // Virtual CPUs
	}

	Flavors []*Flavor
)

func (c *Context) NewFlavor() *Flavor {
	t := &Flavor{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return t
}

func (t *Flavor) key() string {
	return filepath.Join(FlavorPath, t.ID, "metadata")
}

func (t *Flavor) fromResponse(resp *etcd.Response) error {
	t.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &t)
}

// Refresh reloads from the data store
func (t *Flavor) Refresh() error {
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

func (t *Flavor) Validate() error {
	// do validation stuff...
	return nil
}

func (t *Flavor) Save() error {

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

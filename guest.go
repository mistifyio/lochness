package lochness

import (
	"encoding/json"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
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
	}

	Guests []*Guest
)

func (c *Context) NewGuest() *Guest {
	t := &Guest{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return t
}

func (t *Guest) key() string {
	return filepath.Join(GuestPath, t.ID, "metadata")
}

func (t *Guest) fromResponse(resp *etcd.Response) error {
	t.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &t)
}

// Refresh reloads from the data store
func (t *Guest) Refresh() error {
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

func (t *Guest) Validate() error {
	// do validation stuff...
	return nil
}

func (t *Guest) Save() error {

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

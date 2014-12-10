package lochness

import (
	"encoding/json"
	"net"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	FWGroupPath = "lochness/fwgroups/"
)

type (
	FWRule struct {
		Source    *net.IPNet `json:"source,omitempty"`
		Group     string     `json:"group"`
		PortStart uint       `json:"portStart"`
		PortEnd   uint       `json:"portEnd"`
		Protocol  string     `json:"protocol"`
		Action    string     `json:"action"`
	}

	FWRules []*FWRule

	FWGroup struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		Rules         FWRules           `json:"rules"`
	}

	FWGroups []*FWGroup
)

func (c *Context) NewFWGroup() *FWGroup {
	t := &FWGroup{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return t
}

func (c *Context) FWGroup(id string) (*FWGroup, error) {
	t := &FWGroup{
		context: c,
		ID:      id,
	}

	err := t.Refresh()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *FWGroup) key() string {
	return filepath.Join(FWGroupPath, t.ID, "metadata")
}

func (t *FWGroup) fromResponse(resp *etcd.Response) error {
	t.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &t)
}

// Refresh reloads from the data store
func (t *FWGroup) Refresh() error {
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

func (t *FWGroup) Validate() error {
	// do validation stuff...
	return nil
}

func (t *FWGroup) Save() error {

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

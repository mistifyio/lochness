package lochness

import (
	"encoding/json"
	"net"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	// FWGroupPath is the path in the config store
	FWGroupPath = "lochness/fwgroups/"
)

// XXX: should individual rules be their own keys??

type (

	// FWRule represents a single firewall rule
	FWRule struct {
		Source    *net.IPNet `json:"source,omitempty"`
		Group     string     `json:"group"`
		PortStart uint       `json:"portStart"`
		PortEnd   uint       `json:"portEnd"`
		Protocol  string     `json:"protocol"`
		Action    string     `json:"action"`
	}

	// FWRules is an alias to a slice of *FWRule
	FWRules []*FWRule

	// FWGroup represents a group of firewall rules
	FWGroup struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		Rules         FWRules           `json:"rules"`
	}

	// FWGroups is an alias to FWGroup slices
	FWGroups []*FWGroup

	fwRuleJSON struct {
		Source    string `json:"source,omitempty"`
		Group     string `json:"group,omitempty"`
		PortStart uint   `json:"portStart"`
		PortEnd   uint   `json:"portEnd"`
		Protocol  string `json:"protocol"`
		Action    string `json:"action"`
	}

	fwGroupJSON struct {
		ID       string            `json:"id"`
		Metadata map[string]string `json:"metadata"`
		Rules    []*fwRuleJSON     `json:"rules"`
	}
)

// MarshalJSON is a helper for marshalling a FWGroup
func (f FWGroup) MarshalJSON() ([]byte, error) {
	data := fwGroupJSON{
		ID:       f.ID,
		Metadata: f.Metadata,
		Rules:    make([]*fwRuleJSON, 0, len(f.Rules)),
	}

	for _, r := range f.Rules {
		rule := fwRuleJSON{
			Group:     r.Group,
			PortStart: r.PortStart,
			PortEnd:   r.PortEnd,
			Protocol:  r.Protocol,
			Action:    r.Action,
		}

		if r.Source != nil {
			rule.Source = r.Source.String()
		}

		data.Rules = append(data.Rules, &rule)
	}
	return json.Marshal(data)
}

// UnmarshalJSON is a helper for unmarshalling a FWGroup
func (f *FWGroup) UnmarshalJSON(input []byte) error {
	data := fwGroupJSON{}

	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	f.ID = data.ID
	f.Metadata = data.Metadata
	f.Rules = make(FWRules, 0, len(data.Rules))

	for _, r := range data.Rules {
		rule := &FWRule{
			Group:     r.Group,
			PortStart: r.PortStart,
			PortEnd:   r.PortEnd,
			Protocol:  r.Protocol,
			Action:    r.Action,
		}

		if r.Source != "" {
			_, n, err := net.ParseCIDR(r.Source)
			if err != nil {
				return err
			}
			rule.Source = n
		}
		f.Rules = append(f.Rules, rule)
	}
	return nil

}

// NewFWGroup creates a new, blank FWGroup
func (c *Context) NewFWGroup() *FWGroup {
	f := &FWGroup{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return f
}

// FWGroup fetches a FWGroup from the config store
func (c *Context) FWGroup(id string) (*FWGroup, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	f := &FWGroup{
		context: c,
		ID:      id,
	}

	err = f.Refresh()
	if err != nil {
		return nil, err
	}
	return f, nil
}

// key is a helper to generate the config store key
func (f *FWGroup) key() string {
	return filepath.Join(FWGroupPath, f.ID, "metadata")
}

// fromResponse is a helper to unmarshal a FWGroup
func (f *FWGroup) fromResponse(resp *etcd.Response) error {
	f.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &f)
}

// Refresh reloads from the data store
func (f *FWGroup) Refresh() error {
	resp, err := f.context.etcd.Get(f.key(), false, false)

	if err != nil {
		return err
	}

	if resp == nil || resp.Node == nil {
		// should this be an error??
		return nil
	}

	return f.fromResponse(resp)
}

// Validate ensures a FWGroup has reasonable data. It currently does nothing.
func (f *FWGroup) Validate() error {
	// do validation stuff...
	return nil
}

// Save persists a FWGroup.  It will call Validate.
func (f *FWGroup) Save() error {

	if err := f.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(f)

	if err != nil {
		return err
	}

	// if we changed something, don't clobber
	var resp *etcd.Response
	if f.modifiedIndex != 0 {
		resp, err = f.context.etcd.CompareAndSwap(f.key(), string(v), 0, "", f.modifiedIndex)
	} else {
		resp, err = f.context.etcd.Create(f.key(), string(v), 0)
	}
	if err != nil {
		return err
	}

	f.modifiedIndex = resp.EtcdIndex
	return nil
}

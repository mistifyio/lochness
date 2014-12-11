package lochness

import (
	"encoding/json"
	"net"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
)

var (
	SubnetPath = "lochness/subnets/"
)

type (
	// Subnet is an actual ip subnet for assigning addresses
	Subnet struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		NetworkID     string            `json:"network"`
		Gateway       net.IP            `json:"gateway"`
		CIDR          *net.IPNet        `json:"cidr"`
		StartRange    net.IP            `json:"start"` // first usable IP in range
		EndRange      net.IP            `json:"end"`   // last usable IP in range
	}

	Subnets []*Subnet

	//helper struct for json
	subnetJSON struct {
		ID         string            `json:"id"`
		Metadata   map[string]string `json:"metadata"`
		NetworkID  string            `json:"network"`
		Gateway    net.IP            `json:"gateway"`
		CIDR       string            `json:"cidr"`
		StartRange net.IP            `json:"start"`
		EndRange   net.IP            `json:"end"`
	}
)

// issues with (un)marshal of net.IPnet

func (t *Subnet) MarshalJSON() ([]byte, error) {
	data := subnetJSON{
		ID:         t.ID,
		Metadata:   t.Metadata,
		NetworkID:  t.NetworkID,
		Gateway:    t.Gateway,
		CIDR:       t.CIDR.String(),
		StartRange: t.StartRange,
		EndRange:   t.EndRange,
	}

	return json.Marshal(data)
}

func (t *Subnet) UnmarshalJSON(input []byte) error {
	data := subnetJSON{}

	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	t.ID = data.ID
	t.Metadata = data.Metadata
	t.NetworkID = data.NetworkID
	t.Gateway = data.Gateway
	t.StartRange = data.StartRange
	t.EndRange = data.EndRange

	_, n, err := net.ParseCIDR(data.CIDR)
	if err != nil {
		return err
	}

	t.CIDR = n
	return nil

}

func (c *Context) NewSubnet() *Subnet {
	t := &Subnet{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return t
}

func (c *Context) Subnet(id string) (*Subnet, error) {
	t := &Subnet{
		context: c,
		ID:      id,
	}

	err := t.Refresh()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Subnet) key() string {
	return filepath.Join(SubnetPath, t.ID, "metadata")
}

func (t *Subnet) fromResponse(resp *etcd.Response) error {
	t.modifiedIndex = resp.Node.ModifiedIndex
	return json.Unmarshal([]byte(resp.Node.Value), &t)
}

// Refresh reloads from the data store
func (t *Subnet) Refresh() error {
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

func (t *Subnet) Validate() error {
	// do validation stuff...
	return nil
}

func (t *Subnet) Save() error {

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

package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"

	"github.com/coreos/go-etcd/etcd"
)

var (
	// VLANPath is the path in the config store for VLANs
	VLANPath = "/lochness/vlans/"
)

type (
	// VLAN devines the virtual lan for a guest interface
	VLAN struct {
		context       *Context
		modifiedIndex uint64
		Tag           int    `json:"tag"`
		Description   string `json:"description"`
		vlanGroups    []string
	}

	// VLANs is an alias to a slice of *VLAN
	VLANs []*VLAN
)

func (c *Context) blankVLAN(tag int) *VLAN {
	if tag == 0 {
		tag = 1
	}

	v := &VLAN{
		context:    c,
		Tag:        tag,
		vlanGroups: make([]string, 0, 0),
	}

	return v
}

// NewVLAN creates a new blank VLAN.
func (c *Context) NewVLAN() *VLAN {
	return c.blankVLAN(0)
}

// key is a helper to generate the config store key.
func (v *VLAN) key() string {
	return filepath.Join(VLANPath, strconv.Itoa(v.Tag), "metadata")
}

func (v *VLAN) vlanGroupKey(vlanGroup *VLANGroup) string {
	var key string
	if vlanGroup != nil {
		key = vlanGroup.ID
	}
	return filepath.Join(VLANPath, strconv.Itoa(v.Tag), "vlangroups", key)
}

// VLAN fetches a VLAN from the data store.
func (c *Context) VLAN(tag int) (*VLAN, error) {
	v := c.blankVLAN(tag)
	if err := v.Refresh(); err != nil {
		return nil, err
	}
	return v, nil
}

// Refresh reloads the VLAN from the data store.
func (v *VLAN) Refresh() error {
	resp, err := v.context.etcd.Get(filepath.Dir(v.key()), false, true)
	if err != nil {
		return err
	}

	for _, node := range resp.Node.Nodes {
		key := filepath.Base(node.Key)
		switch key {
		case "metadata":
			if err := json.Unmarshal([]byte(node.Value), &v); err != nil {
				return err
			}
			v.modifiedIndex = node.ModifiedIndex
		case "vlangroups":
			for _, x := range node.Nodes {
				v.vlanGroups = append(v.vlanGroups, filepath.Base(x.Key))
			}
		}
	}

	return nil
}

// Validate ensures a VLAN has resonable data.
func (v *VLAN) Validate() error {
	// Tag must be positive and fit in 12bit
	if v.Tag <= 0 || v.Tag > 4095 {
		return errors.New("invalid tag")
	}
	return nil
}

// Save persists a VLAN. It will call Validate.
func (v *VLAN) Save() error {
	if err := v.Validate(); err != nil {
		return err
	}

	value, err := json.Marshal(v)
	if err != nil {
		return err
	}

	// if something changed, don't clobber
	var resp *etcd.Response
	if v.modifiedIndex != 0 {
		resp, err = v.context.etcd.CompareAndSwap(v.key(), string(value), 0, "", v.modifiedIndex)
	} else {
		resp, err = v.context.etcd.Create(v.key(), string(value), 0)
	}
	if err != nil {
		return err
	}

	v.modifiedIndex = resp.EtcdIndex

	return nil
}

// Destroy removes the VLAN
func (v *VLAN) Destroy() error {
	// Unlink VLANGroups
	for _, vlanGroupID := range v.vlanGroups {
		vlanGroup, err := v.context.VLANGroup(vlanGroupID)
		if err != nil {
			return err
		}
		if err := vlanGroup.RemoveVLAN(v); err != nil {
			return err
		}
	}

	// Delete the VLAN
	if _, err := v.context.etcd.Delete(filepath.Dir(v.key()), true); err != nil {
		return err
	}
	return nil
}

// ForEachVLAN will run f on each VLAN. It will stop iteration if f returns an error.
func (c *Context) ForEachVLAN(f func(*VLAN) error) error {
	resp, err := c.etcd.Get(VLANPath, false, false)
	if err != nil {
		return err
	}
	for _, n := range resp.Node.Nodes {
		vlanTag, _ := strconv.Atoi(filepath.Base(n.Key))
		vlan, err := c.VLAN(vlanTag)
		if err != nil {
			return err
		}

		if err := f(vlan); err != nil {
			return err
		}
	}
	return nil
}

// VLANGroups returns the IDs of the VLANGroups associated with the VLAN
func (v *VLAN) VLANGroups() []string {
	return v.vlanGroups
}

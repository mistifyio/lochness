package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mistifyio/lochness/pkg/kv"
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
	prefix := filepath.Join(VLANPath, strconv.Itoa(v.Tag))

	nodes, err := v.context.kv.GetAll(prefix)
	if err != nil {
		return err
	}

	// handle metadata
	key := filepath.Join(prefix, "metadata")
	value, ok := nodes[key]
	if !ok {
		return errors.New("metadata key is missing")
	}

	if err := json.Unmarshal(value.Data, &v); err != nil {
		return err
	}
	v.modifiedIndex = value.Index
	delete(nodes, key)

	groups := []string{}

	// TODO(needs tests)
	for k := range nodes {
		elements := strings.Split(k, "/")
		base := elements[len(elements)-1]
		dir := elements[len(elements)-2]

		if dir != "vlangroups" {
			continue
		}
		groups = append(groups, base)
	}

	v.vlanGroups = groups
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

	index, err := v.context.kv.Update(v.key(), kv.Value{Data: value, Index: v.modifiedIndex})
	if err != nil {
		return err
	}
	v.modifiedIndex = index
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
	return v.context.kv.Delete(filepath.Dir(v.key()), true)
}

// ForEachVLAN will run f on each VLAN. It will stop iteration if f returns an error.
func (c *Context) ForEachVLAN(f func(*VLAN) error) error {
	keys, err := c.kv.Keys(VLANPath)
	if err != nil {
		return err
	}

	for _, k := range keys {
		vlanTag, _ := strconv.Atoi(filepath.Base(k))
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

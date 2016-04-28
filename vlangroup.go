package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mistifyio/lochness/pkg/kv"
	"github.com/pborman/uuid"
)

var (
	// VLANGroupPath is the path in the config store for VLAN groups
	VLANGroupPath = "lochness/vlangroups/"
)

type (
	// VLANGroup defines a set of VLANs for a guest interface
	VLANGroup struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Description   string            `json:"description"`
		Metadata      map[string]string `json:"metadata"`
		vlans         []int
	}

	// VLANGroups is an alias to a slice of *VLANGroup
	VLANGroups []*VLANGroup
)

func (c *Context) blankVLANGroup(id string) *VLANGroup {
	vg := &VLANGroup{
		context:  c,
		ID:       id,
		Metadata: make(map[string]string),
		vlans:    []int{},
	}

	if id == "" {
		vg.ID = uuid.New()
	}

	return vg
}

// key is a helper to generate the config store key.
func (vg *VLANGroup) key() string {
	return filepath.Join(VLANGroupPath, vg.ID, "metadata")
}

func (vg *VLANGroup) vlanKey(vlan *VLAN) string {
	var key int
	if vlan != nil {
		key = vlan.Tag
	}
	return filepath.Join(VLANGroupPath, vg.ID, "vlans", strconv.Itoa(key))
}

// NewVLANGroup creates a new blank VLANGroup.
func (c *Context) NewVLANGroup() *VLANGroup {
	return c.blankVLANGroup("")
}

// VLANGroup fetches a VLAN from the data store.
func (c *Context) VLANGroup(id string) (*VLANGroup, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	vg := c.blankVLANGroup(id)
	if err = vg.Refresh(); err != nil {
		return nil, err
	}
	return vg, nil
}

// Refresh reloads the VLAN from the data store.
func (vg *VLANGroup) Refresh() error {
	prefix := filepath.Dir(vg.key())

	nodes, err := vg.context.kv.GetAll(prefix)
	if err != nil {
		return err
	}

	// handle metadata
	key := filepath.Join(prefix, "metadata")
	value, ok := nodes[key]
	if !ok {
		return errors.New("metadata key is missing")
	}

	if err := json.Unmarshal(value.Data, &vg); err != nil {
		return err
	}
	vg.modifiedIndex = value.Index
	delete(nodes, key)

	vlans := []int{}

	// TODO(needs tests)
	for k := range nodes {
		elements := strings.Split(k, "/")
		base := elements[len(elements)-1]
		dir := elements[len(elements)-2]

		if dir != "vlans" {
			continue
		}
		tag, _ := strconv.Atoi(base)
		vlans = append(vlans, tag)
	}

	vg.vlans = vlans
	return nil
}

// Validate ensures a VLANGroup has resonable data.
func (vg *VLANGroup) Validate() error {
	if _, err := canonicalizeUUID(vg.ID); err != nil {
		return errors.New("invalid ID")
	}
	return nil
}

// Save persists a VLANgroup. It will call Validate.
func (vg *VLANGroup) Save() error {
	if err := vg.Validate(); err != nil {
		return err
	}

	value, err := json.Marshal(vg)
	if err != nil {
		return err
	}

	index, err := vg.context.kv.Update(vg.key(), kv.Value{Data: value, Index: vg.modifiedIndex})
	if err != nil {
		return err
	}
	vg.modifiedIndex = index
	return nil
}

// Destroy removes a VLANGroup
func (vg *VLANGroup) Destroy() error {
	if vg.ID == "" {
		return errors.New("missing id")
	}

	// Unlink VLANs
	for _, vlanTag := range vg.vlans {
		vlan, err := vg.context.VLAN(vlanTag)
		if err != nil {
			return err
		}

		if err := vg.RemoveVLAN(vlan); err != nil {
			return err
		}
	}

	// Delete the VLANGroup
	return vg.context.kv.Delete(filepath.Dir(vg.key()), true)
}

// AddVLAN adds a VLAN to the VLANGroup
func (vg *VLANGroup) AddVLAN(vlan *VLAN) error {
	// Make sure the VLANGroup exists
	if vg.modifiedIndex == 0 {
		if err := vg.Refresh(); err != nil {
			return err
		}
	}

	// Make sure vlan exists
	if vlan.modifiedIndex == 0 {
		if err := vlan.Refresh(); err != nil {
			return err
		}
	}

	// VLANGroup side
	if err := vg.context.kv.Set(vg.vlanKey(vlan), ""); err != nil {
		return err
	}
	vg.vlans = append(vg.vlans, vlan.Tag)

	// VLAN side
	if err := vlan.context.kv.Set(vlan.vlanGroupKey(vg), ""); err != nil {
		return err
	}
	vlan.vlanGroups = append(vlan.vlanGroups, vg.ID)

	return nil
}

// RemoveVLAN removes a VLAN from the VLANGroup
func (vg *VLANGroup) RemoveVLAN(vlan *VLAN) error {
	// VLANGroup side
	if err := vg.context.kv.Delete(vg.vlanKey(vlan), false); err != nil {
		return err
	}

	if len(vg.vlans) == 0 {
		return nil
	}

	newVLANs := make([]int, 0, len(vg.vlans)-1)
	for _, vlanTag := range vg.vlans {
		if vlanTag != vlan.Tag {
			newVLANs = append(newVLANs, vlanTag)
		}
	}
	vg.vlans = newVLANs

	// VLAN side
	if err := vlan.context.kv.Delete(vlan.vlanGroupKey(vg), false); err != nil {
		return err
	}

	if len(vlan.vlanGroups) == 0 {
		return nil
	}
	newVLANGroups := make([]string, 0, len(vlan.vlanGroups)-1)
	for _, vlanGroup := range vlan.vlanGroups {
		if vlanGroup != vg.ID {
			newVLANGroups = append(newVLANGroups, vlanGroup)
		}
	}
	vlan.vlanGroups = newVLANGroups

	return nil
}

// ForEachVLANGroup will run f on each VLAN. It will stop iteration if f returns an error.
func (c *Context) ForEachVLANGroup(f func(*VLANGroup) error) error {
	keys, err := c.kv.Keys(VLANGroupPath)
	if err != nil {
		return err
	}

	for _, k := range keys {
		groupID := filepath.Base(k)
		vlanGroup, err := c.VLANGroup(groupID)
		if err != nil {
			return err
		}

		if err := f(vlanGroup); err != nil {
			return err
		}
	}
	return nil
}

// VLANs returns the Tags of the VLANs associated with the VLANGroup
func (vg *VLANGroup) VLANs() []int {
	return vg.vlans
}

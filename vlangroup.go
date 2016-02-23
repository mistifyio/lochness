package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"

	"github.com/pborman/uuid"

	"github.com/coreos/go-etcd/etcd"
)

var (
	// VLANGroupPath is the path in the config store for VLAN groups
	VLANGroupPath = "/lochness/vlangroups/"
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
		vlans:    make([]int, 0, 0),
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
	resp, err := vg.context.etcd.Get(filepath.Dir(vg.key()), false, true)
	if err != nil {
		return err
	}

	for _, node := range resp.Node.Nodes {
		key := filepath.Base(node.Key)
		switch key {
		case "metadata":
			if err := json.Unmarshal([]byte(node.Value), &vg); err != nil {
				return err
			}
			vg.modifiedIndex = node.ModifiedIndex
		case "vlans":
			vg.vlans = make([]int, len(node.Nodes))
			for i, x := range node.Nodes {
				vlanTag, _ := strconv.Atoi(filepath.Base(x.Key))
				vg.vlans[i] = vlanTag
			}
		}
	}

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

	// if something changed, don't clobber
	var resp *etcd.Response
	if vg.modifiedIndex != 0 {
		resp, err = vg.context.etcd.CompareAndSwap(vg.key(), string(value), 0, "", vg.modifiedIndex)
	} else {
		resp, err = vg.context.etcd.Create(vg.key(), string(value), 0)
	}
	if err != nil {
		return err
	}

	vg.modifiedIndex = resp.EtcdIndex
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
	if _, err := vg.context.etcd.Delete(filepath.Dir(vg.key()), true); err != nil {
		return err
	}
	return nil
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
	if _, err := vg.context.etcd.Set(vg.vlanKey(vlan), "", 0); err != nil {
		return err
	}
	vg.vlans = append(vg.vlans, vlan.Tag)

	// VLAN side
	if _, err := vlan.context.etcd.Set(vlan.vlanGroupKey(vg), "", 0); err != nil {
		return err
	}
	vlan.vlanGroups = append(vlan.vlanGroups, vg.ID)

	return nil
}

// RemoveVLAN removes a VLAN from the VLANGroup
func (vg *VLANGroup) RemoveVLAN(vlan *VLAN) error {
	// VLANGroup side
	if _, err := vg.context.etcd.Delete(vg.vlanKey(vlan), false); err != nil {
		return err
	}

	newVLANs := make([]int, 0, len(vg.vlans)-1)
	for _, vlanTag := range vg.vlans {
		if vlanTag != vlan.Tag {
			newVLANs = append(newVLANs, vlanTag)
		}
	}
	vg.vlans = newVLANs

	// VLAN side
	if _, err := vlan.context.etcd.Delete(vlan.vlanGroupKey(vg), false); err != nil {
		return err
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
	resp, err := c.etcd.Get(VLANGroupPath, false, false)
	if err != nil {
		return err
	}
	for _, n := range resp.Node.Nodes {
		groupID := filepath.Base(n.Key)
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

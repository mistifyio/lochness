package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"

	"github.com/mistifyio/lochness/pkg/kv"
	"github.com/pborman/uuid"
)

var (
	// FlavorPath is the path in the config store
	FlavorPath = "/lochness/flavors/"
)

type (
	// Flavor defines the virtual resources for a guest
	Flavor struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Image         string            `json:"image"`
		Metadata      map[string]string `json:"metadata"`
		Resources
	}

	// Flavors is an alias to a slice of *Flavor
	Flavors []*Flavor

	// Resources represents compute resources
	Resources struct {
		Memory uint64 `json:"memory"` // memory in MB
		Disk   uint64 `json:"disk"`   // disk in MB
		CPU    uint32 `json:"cpu"`    // virtual cpus
	}
)

// NewFlavor creates a blank Flavor
func (c *Context) NewFlavor() *Flavor {
	f := &Flavor{
		context:  c,
		ID:       uuid.New(),
		Metadata: make(map[string]string),
	}

	return f
}

// Flavor fetches a single Flavor from the config store
func (c *Context) Flavor(id string) (*Flavor, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	f := &Flavor{
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
func (f *Flavor) key() string {
	return filepath.Join(FlavorPath, f.ID, "metadata")
}

// fromResponse is a helper to unmarshal a Flavor
func (f *Flavor) fromResponse(value kv.Value) error {
	f.modifiedIndex = value.Index
	return json.Unmarshal(value.Data, &f)
}

// Refresh reloads from the data store
func (f *Flavor) Refresh() error {
	resp, err := f.context.kv.Get(f.key())

	if err != nil {
		return err
	}

	return f.fromResponse(resp)
}

// Validate ensures a Flavor has reasonable data. It currently does nothing.
func (f *Flavor) Validate() error {
	if f.ID == "" {
		return errors.New("flavor ID required")
	}
	if uuid.Parse(f.ID) == nil {
		return errors.New("flavor ID must be uuid")
	}

	if f.Image == "" {
		return errors.New("flavor image required")
	}
	if uuid.Parse(f.Image) == nil {
		return errors.New("flavor image must be uuid")
	}
	return nil
}

// Save persists a Flavor.
// It will call Validate.
func (f *Flavor) Save() error {
	if err := f.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(f)

	if err != nil {
		return err
	}

	index, err := f.context.kv.Update(f.key(), kv.Value{Data: v, Index: f.modifiedIndex})
	if err == nil {
		f.modifiedIndex = index
	}
	return err
}

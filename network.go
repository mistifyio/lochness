package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/mistifyio/lochness/pkg/kv"
	"github.com/pborman/uuid"
)

var (
	// NetworkPath is the path in the config store.
	NetworkPath = "/lochness/networks/"
)

type (
	// Network is a logical collection of subnets.
	Network struct {
		context       *Context
		modifiedIndex uint64
		ID            string            `json:"id"`
		Metadata      map[string]string `json:"metadata"`
		subnets       []string
	}

	// Networks is an alias to a slice of *Network
	Networks []*Network
)

// blankHypervisor is a helper for creating a blank Network.
func (c *Context) blankNetwork(id string) *Network {
	n := &Network{
		context:  c,
		ID:       id,
		Metadata: make(map[string]string),
		subnets:  make([]string, 0, 0),
	}

	if id == "" {
		n.ID = uuid.New()
	}

	return n
}

// NewNetwork creates a new, blank Network.
func (c *Context) NewNetwork() *Network {
	return c.blankNetwork("")
}

// Network fetches a Network from the data store.
func (c *Context) Network(id string) (*Network, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	n := c.blankNetwork(id)
	err = n.Refresh()
	if err != nil {
		return nil, err
	}
	return n, nil
}

// key is a helper to generate the config store key.
func (n *Network) key() string {
	return filepath.Join(NetworkPath, n.ID, "metadata")
}

// Refresh reloads the Network from the data store.
func (n *Network) Refresh() error {
	prefix := filepath.Join(NetworkPath, n.ID)

	nodes, err := n.context.kv.GetAll(prefix)
	if err != nil {
		return err
	}

	// handle metadata
	key := filepath.Join(prefix, "metadata")
	value, ok := nodes[key]
	if !ok {
		return errors.New("metadata key is missing")
	}

	if err := json.Unmarshal(value.Data, &n); err != nil {
		return err
	}
	n.modifiedIndex = value.Index
	delete(nodes, key)

	subnets := []string{}
	for k := range nodes {
		elements := strings.Split(k, "/")
		base := elements[len(elements)-1]
		dir := elements[len(elements)-2]
		if dir != "subnets" {
			continue
		}

		subnets = append(subnets, base)
	}

	n.subnets = subnets

	return nil

}

// Validate ensures a Network has reasonable data. It currently does nothing.
func (n *Network) Validate() error {
	if _, err := canonicalizeUUID(n.ID); err != nil {
		return errors.New("invalid ID")
	}
	return nil
}

// Save persists a Network.
// It will call Validate.
func (n *Network) Save() error {

	if err := n.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(n)

	if err != nil {
		return err
	}

	index, err := n.context.kv.Update(n.key(), kv.Value{Data: v, Index: n.modifiedIndex})
	if err != nil {
		return err
	}
	n.modifiedIndex = index
	return nil
}

func (n *Network) subnetKey(s *Subnet) string {
	var key string
	if s != nil {
		key = s.ID
	}
	return filepath.Join(NetworkPath, n.ID, "subnets", key)
}

// when we load one, should we make sure the networkid actually matches us?

// AddSubnet adds a Subnet to the Network.
func (n *Network) AddSubnet(s *Subnet) error {
	// Make sure the Network exists
	if n.modifiedIndex == 0 {
		if err := n.Refresh(); err != nil {
			return err
		}
	}

	// Make sure the subnet exists
	if s.modifiedIndex == 0 {
		if err := s.Refresh(); err != nil {
			return err
		}
	}

	if err := n.context.kv.Set(n.subnetKey(s), ""); err != nil {
		return err
	}
	n.subnets = append(n.subnets, s.ID)

	// an instance where transactions would be cool...
	s.NetworkID = n.ID
	if err := s.Save(); err != nil {
		return err
	}

	return nil
}

// RemoveSubnet removes a subnet from the network
func (n *Network) RemoveSubnet(s *Subnet) error {
	if err := n.context.kv.Delete(n.subnetKey(s), false); err != nil {
		return err
	}

	newSubnets := make([]string, 0, len(n.subnets)-1)
	for _, subnetID := range n.subnets {
		if subnetID != s.ID {
			newSubnets = append(newSubnets, subnetID)
		}
	}
	n.subnets = newSubnets

	s.NetworkID = ""
	if err := s.Save(); err != nil {
		return err
	}

	return nil
}

// Subnets returns the IDs of the Subnets associated with the network.
func (n *Network) Subnets() []string {
	return n.subnets
}

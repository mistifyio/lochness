package lochness

import (
	"encoding/json"
	"errors"
	"path/filepath"

	kv "github.com/coreos/go-etcd/etcd"
	"github.com/pborman/uuid"
)

var (
	// NetworkPath is the path in the config store.
	NetworkPath = "lochness/networks/"
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

	resp, err := n.context.kv.Get(filepath.Join(NetworkPath, n.ID), false, true)

	if err != nil {
		return err
	}

	for _, node := range resp.Node.Nodes {
		key := filepath.Base(node.Key)
		switch key {

		case "metadata":
			if err := json.Unmarshal([]byte(node.Value), &n); err != nil {
				return err
			}
			n.modifiedIndex = node.ModifiedIndex

		case "subnets":
			n.subnets = make([]string, len(node.Nodes))
			for i, x := range node.Nodes {
				n.subnets[i] = filepath.Base(x.Key)
			}
		}
	}

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

	// if we changed something, don't clobber
	var resp *kv.Response
	if n.modifiedIndex != 0 {
		resp, err = n.context.kv.CompareAndSwap(n.key(), string(v), 0, "", n.modifiedIndex)
	} else {
		resp, err = n.context.kv.Create(n.key(), string(v), 0)
	}
	if err != nil {
		return err
	}

	n.modifiedIndex = resp.EtcdIndex
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

	if _, err := n.context.kv.Set(n.subnetKey(s), "", 0); err != nil {
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
	if _, err := n.context.kv.Delete(n.subnetKey(s), false); err != nil {
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

package lochness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"path/filepath"

	kv "github.com/coreos/go-etcd/etcd"
	"github.com/pborman/uuid"
)

var (
	// SubnetPath is the key prefix for subnets
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
		addresses     map[uint32]string //all allocated addresses. use int as its quickest to go back and forth
	}

	// Subnets is an alias to a slice of *Subnet
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

// MarshalJSON is used by the json package
func (s *Subnet) MarshalJSON() ([]byte, error) {
	data := subnetJSON{
		ID:         s.ID,
		Metadata:   s.Metadata,
		NetworkID:  s.NetworkID,
		Gateway:    s.Gateway,
		CIDR:       s.CIDR.String(),
		StartRange: s.StartRange,
		EndRange:   s.EndRange,
	}

	return json.Marshal(data)
}

// UnmarshalJSON is used by the json package
func (s *Subnet) UnmarshalJSON(input []byte) error {
	data := subnetJSON{}

	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}

	s.ID = data.ID
	s.Metadata = data.Metadata
	s.NetworkID = data.NetworkID
	s.Gateway = data.Gateway
	s.StartRange = data.StartRange
	s.EndRange = data.EndRange

	_, n, err := net.ParseCIDR(data.CIDR)
	if err != nil {
		return err
	}

	s.CIDR = n
	return nil

}

func (c *Context) blankSubnet(id string) *Subnet {
	s := &Subnet{
		context:   c,
		ID:        id,
		Metadata:  make(map[string]string),
		addresses: make(map[uint32]string),
	}

	if id == "" {
		s.ID = uuid.New()
	}

	return s
}

// NewSubnet creates a new "blank" subnet.
// Fill in the needed values and then call Save.
func (c *Context) NewSubnet() *Subnet {
	return c.blankSubnet("")
}

// Subnet fetches a single subnet by ID
func (c *Context) Subnet(id string) (*Subnet, error) {
	var err error
	id, err = canonicalizeUUID(id)
	if err != nil {
		return nil, err
	}
	s := c.blankSubnet(id)
	err = s.Refresh()
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Subnet) key() string {
	return filepath.Join(SubnetPath, s.ID, "metadata")
}

// Refresh reloads the Subnet from the data store.
func (s *Subnet) Refresh() error {
	resp, err := s.context.kv.Get(filepath.Join(SubnetPath, s.ID), false, true)

	if err != nil {
		return err
	}

	for _, n := range resp.Node.Nodes {
		key := filepath.Base(n.Key)
		switch key {

		case "metadata":
			if err := json.Unmarshal([]byte(n.Value), &s); err != nil {
				return err
			}
			s.modifiedIndex = n.ModifiedIndex

		case "addresses":
			for _, n := range n.Nodes {
				if ip := net.ParseIP(filepath.Base(n.Key)); ip != nil {
					// just skip on error
					s.addresses[ipToI32(ip)] = n.Value
				}
			}
		}
	}

	return nil
}

// Delete removes a subnet. It does not ensure it is unused, so use with extreme caution.
func (s *Subnet) Delete() error {
	// Unlink network
	if s.NetworkID != "" {
		network, err := s.context.Network(s.NetworkID)
		if err != nil {
			return err
		}
		if err := network.RemoveSubnet(s); err != nil {
			return err
		}
	}

	// Delete the subnet
	_, err := s.context.kv.Delete(filepath.Join(SubnetPath, s.ID), true)
	return err
}

// Validate ensures the values are reasonable.
func (s *Subnet) Validate() error {
	if _, err := canonicalizeUUID(s.ID); err != nil {
		return errors.New("invalid ID")
	}

	if s.CIDR == nil {
		return errors.New("CIDR cannot be nil")
	}

	if s.StartRange == nil {
		return errors.New("StartRange cannot be nil")
	}
	if !s.CIDR.Contains(s.StartRange) {
		return fmt.Errorf("%s does not contain %s", s.CIDR, s.StartRange)
	}

	if s.EndRange == nil {
		return errors.New("EndRange cannot be nil")
	}
	if !s.CIDR.Contains(s.EndRange) {
		return fmt.Errorf("%s does not contain %s", s.CIDR, s.EndRange)
	}

	if bytes.Compare(s.StartRange, s.EndRange) > 0 {
		return errors.New("EndRange cannot be less than StartRange")
	}
	return nil
}

// Save persists the subnet to the datastore.
func (s *Subnet) Save() error {

	if err := s.Validate(); err != nil {
		return err
	}

	v, err := json.Marshal(s)

	if err != nil {
		return err
	}

	// if we changed something, don't clobber
	var resp *kv.Response
	if s.modifiedIndex != 0 {
		resp, err = s.context.kv.CompareAndSwap(s.key(), string(v), 0, "", s.modifiedIndex)
	} else {
		resp, err = s.context.kv.Create(s.key(), string(v), 0)
	}
	if err != nil {
		return err
	}

	s.modifiedIndex = resp.EtcdIndex
	return nil
}

func (s *Subnet) addressKey(address string) string {
	return filepath.Join(SubnetPath, s.ID, "addresses", address)
}

// Addresses returns used IP addresses.
func (s *Subnet) Addresses() map[string]string {

	addresses := make(map[string]string)

	start := ipToI32(s.StartRange)
	end := ipToI32(s.EndRange)

	// this is a horrible way to do this. should this be simple set math?
	for i := start; i <= end; i++ {
		if id, ok := s.addresses[i]; ok {
			ip := i32ToIP(i)
			addresses[ip.String()] = id
		}
	}

	return addresses
}

// based on https://github.com/ziutek/utils/
func ipToI32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func i32ToIP(a uint32) net.IP {
	return net.IPv4(byte(a>>24), byte(a>>16), byte(a>>8), byte(a))
}

// AvailableAddresses returns the available ip addresses.
// this is probably a horrible idea for ipv6.
func (s *Subnet) AvailableAddresses() []net.IP {
	addresses := make([]net.IP, 0, 0)
	start := ipToI32(s.StartRange)
	end := ipToI32(s.EndRange)

	// this is a horrible way to do this. should this be simple set math?
	for i := start; i <= end; i++ {
		if _, ok := s.addresses[i]; !ok {
			addresses = append(addresses, i32ToIP(i))
		}
	}

	return addresses
}

func randomizeAddresses(a []net.IP) []net.IP {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}

	return a
}

// ReserveAddress reserves an ip address. The id is a guest id.
func (s *Subnet) ReserveAddress(id string) (net.IP, error) {

	// hacky...
	//should this lock?? or do we assume lock is held?

	avail := s.AvailableAddresses()

	if len(avail) == 0 {
		return nil, errors.New("no available addresses")
	}

	avail = randomizeAddresses(avail)

	var chosen net.IP
	for _, ip := range avail {
		v := ip.String()
		_, err := s.context.kv.Create(s.addressKey(v), id, 0)
		if err == nil {
			chosen = ip
			s.addresses[ipToI32(ip)] = id
			break
		}
	}

	// what is nothing was chosen?
	return chosen, nil
}

// ReleaseAddress releases an address. This does not change any thing that may also be referring to this address.
func (s *Subnet) ReleaseAddress(ip net.IP) error {

	_, err := s.context.kv.Delete(s.addressKey(ip.String()), false)
	if err == nil {
		delete(s.addresses, ipToI32(ip))
	}
	return err
}

// ForEachSubnet will run f on each Subnet. It will stop iteration if f returns an error.
func (c *Context) ForEachSubnet(f func(*Subnet) error) error {
	resp, err := c.kv.Get(SubnetPath, false, false)
	if err != nil {
		return err
	}
	for _, n := range resp.Node.Nodes {
		s, err := c.Subnet(filepath.Base(n.Key))
		if err != nil {
			return err
		}

		if err := f(s); err != nil {
			return err
		}
	}
	return nil
}

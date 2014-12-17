package lochness

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"path/filepath"

	"code.google.com/p/go-uuid/uuid"
	"github.com/coreos/go-etcd/etcd"
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

	// Subnets is a helper for slices of subnets
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

// UnmarshalJSON is used by the json package
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

func (c *Context) blankSubnet(id string) *Subnet {
	t := &Subnet{
		context:   c,
		ID:        id,
		Metadata:  make(map[string]string),
		addresses: make(map[uint32]string),
	}

	if id == "" {
		t.ID = uuid.New()
	}

	return t
}

// NewSubnet creates a new "blank" subnet.  Fill in the needed values and then call Save
func (c *Context) NewSubnet() *Subnet {
	return c.blankSubnet("")
}

// Subnet fetches a single subnet by ID
func (c *Context) Subnet(id string) (*Subnet, error) {
	t := c.blankSubnet(id)
	err := t.Refresh()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Subnet) key() string {
	return filepath.Join(SubnetPath, t.ID, "metadata")
}

// Refresh reloads from the data store
func (t *Subnet) Refresh() error {
	resp, err := t.context.etcd.Get(filepath.Join(SubnetPath, t.ID), false, true)

	if err != nil {
		return err
	}

	for _, n := range resp.Node.Nodes {
		key := filepath.Base(n.Key)
		switch key {

		case "metadata":
			if err := json.Unmarshal([]byte(n.Value), &t); err != nil {
				return err
			}
			t.modifiedIndex = n.ModifiedIndex

		case "addresses":
			for _, n := range n.Nodes {
				if ip := net.ParseIP(filepath.Base(n.Key)); ip != nil {
					// just skip on error
					t.addresses[ipToI32(ip)] = n.Value
				}
			}
		}
	}

	return nil
}

// Validate ensures the values are reasonable. It currently does nothing
func (t *Subnet) Validate() error {
	// do validation stuff...
	return nil
}

// Save persists the subnet to the datastore
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

func (t *Subnet) addressKey(address string) string {
	return filepath.Join(SubnetPath, t.ID, "addresses", address)
}

// Addresses returns used IP addresses
func (t *Subnet) Addresses() (map[string]string, error) {

	addresses := make(map[string]string)

	start := ipToI32(t.StartRange)
	end := ipToI32(t.EndRange)

	// this is a horrible way to do this. should this be simple set math?
	for i := start; i <= end; i++ {
		if id, ok := t.addresses[i]; ok {
			ip := i32ToIP(i)
			addresses[ip.String()] = id
		}
	}

	return addresses, nil
}

// based on https://github.com/ziutek/utils/
func ipToI32(ip net.IP) uint32 {
	ip = ip.To4()
	return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func i32ToIP(a uint32) net.IP {
	return net.IPv4(byte(a>>24), byte(a>>16), byte(a>>8), byte(a))
}

// AvailibleAddresses returns the availible ip addresses.
// this is probably a horrible idea for ipv6
func (t *Subnet) AvailibleAddresses() []net.IP {
	addresses := make([]net.IP, 0, 0)
	start := ipToI32(t.StartRange)
	end := ipToI32(t.EndRange)

	// this is a horrible way to do this. should this be simple set math?
	for i := start; i <= end; i++ {
		if _, ok := t.addresses[i]; !ok {
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

// ReserveAddress reserves an ip address. The id is guest id
func (t *Subnet) ReserveAddress(id string) (net.IP, error) {

	// hacky...
	//should this lock?? or do we assume lock is held?

	avail := t.AvailibleAddresses()

	if len(avail) == 0 {
		return nil, errors.New("no availible addresses")
	}

	avail = randomizeAddresses(avail)

	var chosen net.IP
	for _, ip := range avail {
		v := ip.String()
		_, err := t.context.etcd.Create(t.addressKey(v), id, 0)
		if err == nil {
			chosen = ip
			t.addresses[ipToI32(ip)] = id
			break
		}
	}

	// what is nothing was chosen?
	return chosen, nil
}

// ReleaseAddress releases an address
func (t *Subnet) ReleaseAddress(ip net.IP) error {

	_, err := t.context.etcd.Delete(t.addressKey(ip.String()), false)
	if err != nil {
		delete(t.addresses, ipToI32(ip))
	}
	return err
}

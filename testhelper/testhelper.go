package testhelper

import (
	"errors"
	"net"

	"github.com/mistifyio/lochness"
)

// NewFlavor uses the lochness context and the given values to build a new flavor
func NewFlavor(context *lochness.Context, cpu uint32, memory, disk uint64) (*lochness.Flavor, error) {
	f := context.NewFlavor()
	f.CPU = cpu
	f.Memory = memory
	f.Disk = disk
	if err := f.Save(); err != nil {
		return nil, errors.New("Could not save flavor: " + err.Error())
	}
	return f, nil
}

// NewNetwork uses the lochness context and the given values to build a new network
func NewNetwork(context *lochness.Context) (*lochness.Network, error) {
	n := context.NewNetwork()
	if err := n.Save(); err != nil {
		return nil, errors.New("Could not save network: " + err.Error())
	}
	return n, nil
}

// NewFirewallGroup uses the lochness context and the given values to build a new firewall group
func NewFirewallGroup(context *lochness.Context) (*lochness.FWGroup, error) {
	fw := context.NewFWGroup()
	fw.Rules = append(fw.Rules, &lochness.FWRule{})
	if err := fw.Save(); err != nil {
		return nil, errors.New("Could not save firewall group: " + err.Error())
	}
	return fw, nil
}

// NewSubnet uses the lochness context and the given values to build a new subnet
func NewSubnet(context *lochness.Context, cidr string, gateway, start, end net.IP, n *lochness.Network) (*lochness.Subnet, error) {
	var err error
	s := context.NewSubnet()
	_, s.CIDR, err = net.ParseCIDR(cidr)
	if err != nil {
		return nil, errors.New("Could not parse subnet CIDR '" + cidr + "': " + err.Error())
	}
	s.Gateway = gateway
	s.StartRange = start
	s.EndRange = end
	if err := s.Save(); err != nil {
		return nil, errors.New("Could not save subnet: " + err.Error())
	}
	if err := n.AddSubnet(s); err != nil {
		return nil, errors.New("Could not add subnet to network: " + err.Error())
	}
	return s, nil
}

// NewHypervisor uses the lochness context and the given values to build a new hypervisor
func NewHypervisor(context *lochness.Context, mac string, ip, gateway, netmask net.IP, ifname string, s *lochness.Subnet) (*lochness.Hypervisor, error) {
	var err error
	h := context.NewHypervisor()
	h.IP = ip
	h.MAC, err = net.ParseMAC(mac)
	if err != nil {
		return nil, errors.New("Could not parse hypervisor MAC '" + mac + "': " + err.Error())
	}
	h.Gateway = gateway
	h.Netmask = netmask
	if err = h.Save(); err != nil {
		return nil, errors.New("Could not save hypervisor: " + err.Error())
	}
	if err := h.AddSubnet(s, ifname); err != nil {
		return nil, errors.New("Could not add subnet to hypervisor: " + err.Error())
	}
	return h, nil
}

// NewGuest uses the lochness context and the given values to build a new guest
func NewGuest(context *lochness.Context, mac string, n *lochness.Network, s *lochness.Subnet, f *lochness.Flavor, fw *lochness.FWGroup, h *lochness.Hypervisor) (*lochness.Guest, error) {
	var err error
	g := context.NewGuest()
	g.MAC, err = net.ParseMAC(mac)
	if err != nil {
		return nil, errors.New("Could not parse guest MAC '" + mac + "': " + err.Error())
	}
	g.NetworkID = n.ID
	g.SubnetID = s.ID
	g.FlavorID = f.ID
	g.FWGroupID = fw.ID
	if err := g.Save(); err != nil {
		return nil, errors.New("Could not save guest: " + err.Error())
	}

	if err := h.AddGuest(g); err != nil {
		return nil, errors.New("Could not add guest to hypervisor: " + err.Error())
	}

	g.IP, err = s.ReserveAddress(g.ID)
	if err != nil {
		return nil, errors.New("Could not reserve guest address in subnet: " + err.Error())
	}

	if err := g.Save(); err != nil {
		return nil, errors.New("Could not resave guest: " + err.Error())
	}

	return g, nil
}

package testhelper

import (
	"errors"
	"net"
	"path/filepath"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

// NewTestFlavor uses the lochness context and the given values to build a new flavor
func NewTestFlavor(context *lochness.Context, cpu uint32, memory, disk uint64) (*lochness.Flavor, error) {
	f := context.NewFlavor()
	f.CPU = cpu
	f.Memory = memory
	f.Disk = disk
	err := f.Save()
	if err != nil {
		return nil, errors.New("Could not save flavor: " + err.Error())
	}
	return f, nil
}

// NewTestNetwork uses the lochness context and the given values to build a new network
func NewTestNetwork(context *lochness.Context) (*lochness.Network, error) {
	n := context.NewNetwork()
	if err := n.Save(); err != nil {
		return nil, errors.New("Could not save network: " + err.Error())
	}
	return n, nil
}

// NewTestFirewallGroup uses the lochness context and the given values to build a new firewall group
func NewTestFirewallGroup(context *lochness.Context) (*lochness.FWGroup, error) {
	fw := context.NewFWGroup()
	fw.Rules = append(fw.Rules, &lochness.FWRule{})
	if err := fw.Save(); err != nil {
		return nil, errors.New("Could not save firewall group: " + err.Error())
	}
	return fw, nil
}

// NewTestSubnet uses the lochness context and the given values to build a new subnet
func NewTestSubnet(context *lochness.Context, cidr string, gateway, start, end net.IP, n *lochness.Network) (*lochness.Subnet, error) {
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

// NewTestHypervisor uses the lochness context and the given values to build a new hypervisor
func NewTestHypervisor(context *lochness.Context, mac string, ip, gateway, netmask net.IP, ifname string, s *lochness.Subnet) (*lochness.Hypervisor, error) {
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

// NewTestGuest uses the lochness context and the given values to build a new guest
func NewTestGuest(context *lochness.Context, mac string, n *lochness.Network, s *lochness.Subnet, f *lochness.Flavor, fw *lochness.FWGroup, h *lochness.Hypervisor) (*lochness.Guest, error) {
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

// Cleanup removes a list of objects that have been created during the test
func Cleanup(e *etcd.Client, created map[string]string) []error {
	var path string
	var errs []error
	for id, t := range created {
		switch {
		case t == "flavor":
			path = filepath.Join(lochness.FlavorPath, id)
		case t == "network":
			path = filepath.Join(lochness.NetworkPath, id)
		case t == "fwgroup":
			path = filepath.Join(lochness.FWGroupPath, id)
		case t == "subnet":
			path = filepath.Join(lochness.SubnetPath, id)
		case t == "hypervisor":
			path = filepath.Join(lochness.HypervisorPath, id)
		case t == "guest":
			path = filepath.Join(lochness.GuestPath, id)
		}
		_, err := e.Delete(path, true)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

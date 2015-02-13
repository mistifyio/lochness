package main

import (
	"encoding/json"
	"math/rand"
	"net"
	"os"
	"reflect"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

func print(i interface{}) {
	log.Printf("%s: %#v\n", reflect.TypeOf(i).String(), i)
	if data, err := json.Marshal(i); err == nil {
		log.Printf("%s: %s\n", reflect.TypeOf(i).String(), data)
	}

}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	c := lochness.NewContext(e)

	f1 := c.NewFlavor()
	f1.CPU = 4
	f1.Memory = 4096
	f1.Disk = 8192
	if err := f1.Save(); err != nil {
		log.Fatal(err)
	}
	print(f1)

	f2 := c.NewFlavor()
	f2.CPU = 6
	f2.Memory = 8192
	f2.Disk = 1024
	if err := f2.Save(); err != nil {
		log.Fatal(err)
	}
	print(f2)

	n := c.NewNetwork()
	if err := n.Save(); err != nil {
		log.Fatal(err)
	}
	print(n)

	fw := c.NewFWGroup()
	fw.Rules = append(fw.Rules, &lochness.FWRule{})

	if err := fw.Save(); err != nil {
		log.Fatal(err)
	}
	print(fw)

	s := c.NewSubnet()
	var err error
	_, s.CIDR, err = net.ParseCIDR("10.10.10.0/24")
	if err != nil {
		log.Fatal(err)
	}
	s.Gateway = net.IPv4(10, 10, 10, 1)
	s.StartRange = net.IPv4(10, 10, 10, 10)
	s.EndRange = net.IPv4(10, 10, 10, 250)
	if err := s.Save(); err != nil {
		log.Fatal(err)
	}
	print(s)

	addresses := s.Addresses()
	print(addresses)

	print(s.AvailibleAddresses())

	if err := n.AddSubnet(s); err != nil {
		log.Fatal(err)
	}

	network_subnets := n.Subnets()
	if len(network_subnets) == 0 {
		log.Fatal("no subnets available on network")
	}
	for _, k := range network_subnets {
		s, err := c.Subnet(k)
		if err != nil {
			log.Fatal(err)
		}
		print(s)
	}

	print(n)

	var h *lochness.Hypervisor
	if hv := os.Getenv("TEST_HV"); hv != "" {
		var err error
		h, err = c.Hypervisor(hv)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		h = c.NewHypervisor()
		h.IP = net.IPv4(10, 100, 101, 34)
		h.MAC, err = net.ParseMAC("01:23:45:67:89:ab")
		if err != nil {
			log.Fatal(err)
		}
		if err := h.Save(); err != nil {
			log.Fatal(err)
		}
	}
	h.AddSubnet(s, "br0")

	print(h)

	subnets := h.Subnets()
	if len(subnets) == 0 {
		log.Fatal("no subnets available on hypervisor")
	}

	for k := range subnets {
		s, err := c.Subnet(k)
		if err != nil {
			log.Fatal(err)
		}
		print(s)
	}

	fw1 := c.NewFWGroup()
	fw2 := c.NewFWGroup()

	fw1.Rules = lochness.FWRules{&lochness.FWRule{
		Group:     fw2.ID,
		PortStart: 80,
		PortEnd:   82,
		Protocol:  "tcp",
	}}
	fw2.Rules = lochness.FWRules{&lochness.FWRule{
		Group:     fw1.ID,
		PortStart: 80,
		PortEnd:   82,
		Protocol:  "tcp",
	}}

	fw1.Save()
	fw2.Save()

	g1 := c.NewGuest()
	g1.SubnetID = s.ID
	g1.NetworkID = n.ID
	g1.MAC, err = net.ParseMAC("01:23:45:67:89:ab")
	g1.FlavorID = f1.ID
	g1.FWGroupID = fw1.ID
	if err := g1.Save(); err != nil {
		log.Fatal(err)
	}
	if err := h.AddGuest(g1); err != nil {
		log.Fatal(err)
	}
	g1.IP, err = s.ReserveAddress(g1.ID)
	if err != nil {
		log.Fatal(err)
	}
	if err := g1.Save(); err != nil {
		log.Fatal(err)
	}
	print(g1)

	g2 := c.NewGuest()
	g2.SubnetID = s.ID
	g2.NetworkID = n.ID
	g2.MAC, err = net.ParseMAC("01:23:45:67:89:ac")
	g2.FlavorID = f2.ID
	g2.FWGroupID = fw2.ID
	if err := g2.Save(); err != nil {
		log.Fatal(err)
	}
	if err := h.AddGuest(g2); err != nil {
		log.Fatal(err)
	}
	g2.IP, err = s.ReserveAddress(g2.ID)
	if err != nil {
		log.Fatal(err)
	}
	if err := g2.Save(); err != nil {
		log.Fatal(err)
	}
	print(g2)
}

package main

import (
	"encoding/json"
	"fmt"
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
	log.WithField(reflect.TypeOf(i).String(), fmt.Sprintf("%#v", i)).Print("")
	if data, err := json.Marshal(i); err == nil {
		log.WithField(reflect.TypeOf(i).String(), string(data)).Print("")
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
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    f1.ID,
			"item":  "flavor",
		}).Fatal("failed to save flavor")
	}
	print(f1)

	f2 := c.NewFlavor()
	f2.CPU = 6
	f2.Memory = 8192
	f2.Disk = 1024
	if err := f2.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    f2.ID,
			"item":  "flavor",
		}).Fatal("failed to save flavor")
	}
	print(f2)

	n := c.NewNetwork()
	if err := n.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    n.ID,
			"item":  "network",
		}).Fatal("failed to save network")
	}
	print(n)

	fw := c.NewFWGroup()
	fw.Rules = append(fw.Rules, &lochness.FWRule{})

	if err := fw.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    fw.ID,
			"item":  "fwgroup",
		}).Fatal("failed to save fwgroup")
	}
	print(fw)

	s := c.NewSubnet()
	var err error
	cidr := "10.10.10.0/24"
	_, s.CIDR, err = net.ParseCIDR(cidr)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "net.ParseCIDR",
			"error": err,
			"id":    s.ID,
			"cidr":  cidr,
		}).Fatal("failed to parse subnet CIDR")
	}
	s.Gateway = net.IPv4(10, 10, 10, 1)
	s.StartRange = net.IPv4(10, 10, 10, 10)
	s.EndRange = net.IPv4(10, 10, 10, 250)
	if err := s.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    s.ID,
			"item":  "subnet",
		}).Fatal("failed to save subnet")
	}
	print(s)

	addresses := s.Addresses()
	print(addresses)

	print(s.AvailableAddresses())

	if err := n.AddSubnet(s); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddSubnet",
			"error": err,
			"id":    n.ID,
			"item":  "network",
		}).Fatal("failed to add subnet to network")
	}

	networkSubnets := n.Subnets()
	if len(networkSubnets) == 0 {
		log.Fatal("no subnets available on network")
	}
	for _, k := range networkSubnets {
		_, err := c.Subnet(k)
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
			log.WithFields(log.Fields{
				"error": err,
				"id":    hv,
				"func":  "context.Hypervsior",
			}).Fatal("failted to instantiate hypervisor")
		}
	} else {
		h = c.NewHypervisor()
		h.IP = net.IPv4(10, 100, 101, 34)
		mac := "01:23:45:67:89:ab"
		h.MAC, err = net.ParseMAC(mac)
		if err != nil {
			log.WithFields(log.Fields{
				"func":  "net.ParseMAC",
				"error": err,
				"id":    h.ID,
				"mac":   mac,
				"item":  "hypervisor",
			}).Fatal("failed to parse hypervisor mac")
		}
		if err := h.Save(); err != nil {
			log.WithFields(log.Fields{
				"func":  "Save",
				"error": err,
				"id":    h.ID,
				"item":  "hypervisor",
			}).Fatal("failed to save hypervisor")
		}
	}
	if err := h.AddSubnet(s, "br0"); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddSubnet",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
		}).Fatal("failed to add subnet")
	}

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

	if err := fw1.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    fw1.ID,
			"item":  "fwgroup",
		}).Fatal("failed to save fwgroup")
	}
	if err := fw2.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    fw2.ID,
			"item":  "fwgroup",
		}).Fatal("failed to save fwgroup")
	}

	g1 := c.NewGuest()
	g1.SubnetID = s.ID
	g1.NetworkID = n.ID
	g1.MAC, err = net.ParseMAC("01:23:45:67:89:ab")
	g1.FlavorID = f1.ID
	g1.FWGroupID = fw1.ID
	if err := g1.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    g1.ID,
			"item":  "guest",
		}).Fatal("failed to save guest")
	}
	if err := h.AddGuest(g1); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddGuest",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
			"guest": g1.ID,
		}).Fatal("failed to add guest to hypervisor")
	}
	g1.IP, err = s.ReserveAddress(g1.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "ReserverAddress",
			"error": err,
			"id":    s.ID,
			"item":  "subnet",
		}).Fatal("failed to reserver address")
	}
	if err := g1.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    g1.ID,
			"item":  "guest",
		}).Fatal("failed to save guest")
	}
	print(g1)

	g2 := c.NewGuest()
	g2.SubnetID = s.ID
	g2.NetworkID = n.ID
	g2.MAC, err = net.ParseMAC("01:23:45:67:89:ac")
	g2.FlavorID = f2.ID
	g2.FWGroupID = fw2.ID
	if err := g2.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    g2.ID,
			"item":  "guest",
		}).Fatal("failed to save guest")
	}
	if err := h.AddGuest(g2); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddGuest",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
			"guest": g2.ID,
		}).Fatal("failed to add guest to hypervisor")
	}
	g2.IP, err = s.ReserveAddress(g2.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "ReserverAddress",
			"error": err,
			"id":    s.ID,
			"item":  "subnet",
		}).Fatal("failed to reserver address")
	}
	if err := g2.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    g2.ID,
			"item":  "guest",
		}).Fatal("failed to save guest")
	}
	print(g2)
}

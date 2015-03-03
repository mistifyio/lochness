package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
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

	f := c.NewFlavor()
	f.CPU = 1
	f.Memory = 512
	f.Disk = 1024
	if err := f.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    f.ID,
			"item":  "flavor",
		}).Fatal("failed to save flavor")
	}

	n := c.NewNetwork()
	if err := n.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    n.ID,
			"item":  "network",
		}).Fatal("failed to save network")
	}

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

	s := c.NewSubnet()
	var err error
	cidr := "127.0.0.0/24"
	_, s.CIDR, err = net.ParseCIDR(cidr)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "net.ParseCIDR",
			"error": err,
			"id":    s.ID,
			"cidr":  cidr,
		}).Fatal("failed to parse subnet CIDR")
	}
	s.Gateway = net.IPv4(127, 0, 0, 1)
	s.StartRange = net.IPv4(127, 0, 0, 10)
	s.EndRange = net.IPv4(127, 0, 0, 250)
	if err := s.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    s.ID,
			"item":  "subnet",
		}).Fatal("failed to save subnet")
	}

	if err := n.AddSubnet(s); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddSubnet",
			"error": err,
			"id":    n.ID,
			"item":  "network",
		}).Fatal("failed to add subnet to network")
	}

	h := c.NewHypervisor()
	h.IP = net.IPv4(127, 0, 0, 1)
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
	h.TotalResources = f.Resources
	h.AvailableResources = f.Resources
	if err := h.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
		}).Fatal("failed to save hypervisor")
	}
	if err := h.AddSubnet(s, "br0"); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddSubnet",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
		}).Fatal("failed to add subnet")
	}

	g := c.NewGuest()
	g.SubnetID = s.ID
	g.NetworkID = n.ID
	g.FlavorID = f.ID
	mac = "01:23:45:67:89:ac"
	g.MAC, err = net.ParseMAC(mac)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "net.ParseMAC",
			"error": err,
			"id":    g.ID,
			"mac":   mac,
			"item":  "guest",
		}).Fatal("failed to parse guest mac")
	}
	if err := g.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    g.ID,
			"item":  "guest",
		}).Fatal("failed to save guest")
	}
	if err := h.AddGuest(g); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddGuest",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
			"guest": g.ID,
		}).Fatal("failed to add guest to hypervisor")
	}

	g.IP, err = s.ReserveAddress(g.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "ReserverAddress",
			"error": err,
			"id":    s.ID,
			"item":  "subnet",
		}).Fatal("failed to reserver address")
	}

	agent := c.NewMistifyAgent()
	clientGuest, err := agent.CreateGuest(g.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "agent.CreateGuest",
			"error": err,
			"id":    g.ID,
			"item":  "agent",
		}).Fatal("failed to create guest")
	}
	print(clientGuest)
}

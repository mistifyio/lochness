package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net"
	"reflect"
	"time"

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
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	c := lochness.NewContext(e)

	f := c.NewFlavor()
	f.CPU = 1
	f.Memory = 512
	f.Disk = 1024
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}

	n := c.NewNetwork()
	if err := n.Save(); err != nil {
		log.Fatal(err)
	}

	fw := c.NewFWGroup()
	fw.Rules = append(fw.Rules, &lochness.FWRule{})

	if err := fw.Save(); err != nil {
		log.Fatal(err)
	}

	s := c.NewSubnet()
	var err error
	_, s.CIDR, err = net.ParseCIDR("127.0.0.0/24")
	if err != nil {
		log.Fatal(err)
	}
	s.Gateway = net.IPv4(127, 0, 0, 1)
	s.StartRange = net.IPv4(127, 0, 0, 10)
	s.EndRange = net.IPv4(127, 0, 0, 250)
	if err := s.Save(); err != nil {
		log.Fatal(err)
	}

	if err := n.AddSubnet(s); err != nil {
		log.Fatal(err)
	}

	h := c.NewHypervisor()
	h.IP = net.IPv4(127, 0, 0, 1)
	h.MAC, err = net.ParseMAC("01:23:45:67:89:ab")
	if err != nil {
		log.Fatal(err)
	}
	h.TotalResources = f.Resources
	h.AvailableResources = f.Resources
	if err := h.Save(); err != nil {
		log.Fatal(err)
	}
	h.AddSubnet(s, "br0")

	g := c.NewGuest()
	g.SubnetID = s.ID
	g.NetworkID = n.ID
	g.MAC, err = net.ParseMAC("01:23:45:67:89:ac")
	g.FlavorID = f.ID
	if err := g.Save(); err != nil {
		log.Fatal(err)
	}
	if err := h.AddGuest(g); err != nil {
		log.Fatal(err)
	}

	g.IP, err = s.ReserveAddress(g.ID)
	if err != nil {
		log.Fatal(err)
	}

	agent := c.NewMistifyAgent()
	clientGuest, err := agent.CreateGuest(g.ID)
	if err != nil {
		log.Fatal(err)
	}
	print(clientGuest)
}

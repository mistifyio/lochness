package main

import (
	"log"
	"net"
	"reflect"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

func print(i interface{}) {
	log.Printf("%s: %+v\n", reflect.TypeOf(i).String(), i)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	c := lochness.NewContext(e)

	f := c.NewFlavor()
	f.CPU = 4
	f.Memory = 4096
	f.Disk = 8192
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}
	print(f)

	n := c.NewNetwork()
	if err := n.Save(); err != nil {
		log.Fatal(err)
	}
	print(n)

	s := c.NewSubnet()
	_, s.CIDR, _ = net.ParseCIDR("10.10.10.0/24")
	s.Gateway = net.IPv4(10, 10, 10, 1)
	s.StartRange = net.IPv4(10, 10, 10, 10)
	s.EndRange = net.IPv4(10, 10, 10, 250)
	if err := s.Save(); err != nil {
		log.Fatal(err)
	}
	print(s)

	if err := n.AddSubnet(s); err != nil {
		log.Fatal(err)
	}

	network_subnets, err := n.Subnets()
	if err != nil {
		log.Fatal(err)
	}
	for _, k := range network_subnets {
		s, err := c.Subnet(k)
		if err != nil {
			log.Fatal(err)
		}
		print(s)
	}

	print(n)

	h := c.NewHypervisor()
	if err := h.Save(); err != nil {
		log.Fatal(err)
	}
	h.AddSubnet(s, "br0")

	print(h)

	subnets, err := h.Subnets()
	if err != nil {
		log.Fatal(err)
	}

	for k, _ := range subnets {
		s, err := c.Subnet(k)
		if err != nil {
			log.Fatal(err)
		}
		print(s)
	}

}

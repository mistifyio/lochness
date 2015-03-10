package main

import (
	"fmt"
	"net"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	flag "github.com/spf13/pflag"
)

func doTestSetup(context *lochness.Context, etcdClient *etcd.Client) {

	// Clear out the lochness keys
	_, err := etcdClient.Delete("lochness/", true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Fatal("Could not clear data from etcd")
	}

	// Add flavors, network, and subnet
	f1 := newTestFlavor(context, 4, 4096, 8192)
	f2 := newTestFlavor(context, 6, 8192, 1024)
	n := newTestNetwork(context)
	fw := newTestFirewallGroup(context)
	s := newTestSubnet(context, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)

	// Add hypervisors
	h1 := newTestHypervisor(context, "de:ad:be:ef:7f:21", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	h2 := newTestHypervisor(context, "de:ad:be:ef:7f:23", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)

	// Add guests
	_ = newTestGuest(context, "01:23:45:67:89:ab", n, s, f1, fw, h1)
	_ = newTestGuest(context, "23:45:67:89:ab:cd", n, s, f2, fw, h1)
	_ = newTestGuest(context, "45:67:89:ab:cd:ef", n, s, f1, fw, h2)
	_ = newTestGuest(context, "67:89:ab:cd:ef:01", n, s, f2, fw, h2)
}

func newTestFlavor(context *lochness.Context, cpu uint32, memory, disk uint64) *lochness.Flavor {
	f := context.NewFlavor()
	f.CPU = cpu
	f.Memory = memory
	f.Disk = disk
	if err := f.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    f.ID,
			"item":  "flavor",
		}).Fatal("failed to save flavor")
	}
	return f
}

func newTestNetwork(context *lochness.Context) *lochness.Network {
	n := context.NewNetwork()
	if err := n.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    n.ID,
			"item":  "network",
		}).Fatal("failed to save network")
	}
	return n
}

func newTestFirewallGroup(context *lochness.Context) *lochness.FWGroup {
	fw := context.NewFWGroup()
	fw.Rules = append(fw.Rules, &lochness.FWRule{})
	if err := fw.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    fw.ID,
			"item":  "fwgroup",
		}).Fatal("failed to save fwgroup")
	}
	return fw
}

func newTestHypervisor(context *lochness.Context, mac string, ip, gateway, netmask net.IP, ifname string, s *lochness.Subnet) *lochness.Hypervisor {
	var err error
	h := context.NewHypervisor()
	h.IP = ip
	h.MAC, err = net.ParseMAC(mac)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "net.ParseMAC",
			"error": err,
			"id":    h.ID,
			"mac":   mac,
		}).Fatal("Could not parse hypervisor MAC")
	}
	h.Gateway = gateway
	h.Netmask = netmask
	if err = h.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "lochness.Hypervisor.Save",
			"error": err,
			"id":    h.ID,
		}).Fatal("Could not save hypervisor")
	}
	if err := h.AddSubnet(s, ifname); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddSubnet",
			"error": err,
			"id":    h.ID,
			"item":  "hypervisor",
		}).Fatal("failed to add subnet")
	}
	return h
}

func newTestGuest(context *lochness.Context, mac string, n *lochness.Network, s *lochness.Subnet, f *lochness.Flavor, fw *lochness.FWGroup, h *lochness.Hypervisor) *lochness.Guest {
	var err error
	g := context.NewGuest()
	g.MAC, err = net.ParseMAC(mac)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "net.ParseMAC",
			"error": err,
			"id":    g.ID,
			"mac":   mac,
		}).Fatal("Could not parse guest MAC")
	}
	g.NetworkID = n.ID
	g.SubnetID = s.ID
	g.FlavorID = f.ID
	g.FWGroupID = fw.ID
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
		}).Fatal("failed to reserve address")
	}

	if err := g.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    g.ID,
			"item":  "guest",
		}).Fatal("failed to save guest")
	}

	return g
}

func newTestSubnet(context *lochness.Context, cidr string, gateway, start, end net.IP, n *lochness.Network) *lochness.Subnet {
	var err error
	s := context.NewSubnet()
	_, s.CIDR, err = net.ParseCIDR(cidr)
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "net.ParseCIDR",
			"error": err,
			"id":    s.ID,
			"cidr":  cidr,
		}).Fatal("Could not parse subnet CIDR")
	}
	s.Gateway = gateway
	s.StartRange = start
	s.EndRange = end
	if err := s.Save(); err != nil {
		log.WithFields(log.Fields{
			"func":  "Save",
			"error": err,
			"id":    s.ID,
			"item":  "subnet",
		}).Fatal("Could not save subnet")
	}
	if err := n.AddSubnet(s); err != nil {
		log.WithFields(log.Fields{
			"func":  "AddSubnet",
			"error": err,
			"id":    n.ID,
			"item":  "network",
		}).Fatal("failed to add subnet to network")
	}
	return s
}

func fetchHypervisors(context *lochness.Context, etcdClient *etcd.Client) map[string]*lochness.Hypervisor {
	res, err := etcdClient.Get("lochness/hypervisors/", true, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Fatal("Could not retrieve hypervisors from etcd")
	}
	hypervisors := make(map[string]*lochness.Hypervisor)
	for _, node := range res.Node.Nodes {
		for _, hnode := range node.Nodes {
			if strings.Contains(hnode.Key, "metadata") {
				hv := context.NewHypervisor()
				hv.UnmarshalJSON([]byte(hnode.Value))
				hypervisors[hv.ID] = hv
			}
		}
	}
	log.WithFields(log.Fields{
		"hypervisors": hypervisors,
	}).Info("Fetched hypervisors metadata")
	return hypervisors
}

func fetchGuests(context *lochness.Context, etcdClient *etcd.Client) map[string]*lochness.Guest {
	res, err := etcdClient.Get("lochness/guests/", true, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Fatal("Could not retrieve guests from etcd")
	}
	guests := make(map[string]*lochness.Guest)
	for _, node := range res.Node.Nodes {
		for _, gnode := range node.Nodes {
			if strings.Contains(gnode.Key, "metadata") {
				g := context.NewGuest()
				g.UnmarshalJSON([]byte(gnode.Value))
				guests[g.ID] = g
			}
		}
	}
	log.WithFields(log.Fields{
		"guests": guests,
	}).Info("Fetched guests metadata")
	return guests
}

func fetchSubnets(context *lochness.Context, etcdClient *etcd.Client) map[string]*lochness.Subnet {
	res, err := etcdClient.Get("lochness/subnets/", true, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Fatal("Could not retrieve subnets from etcd")
	}
	subnets := make(map[string]*lochness.Subnet)
	for _, node := range res.Node.Nodes {
		for _, snode := range node.Nodes {
			if strings.Contains(snode.Key, "metadata") {
				s := context.NewSubnet()
				s.UnmarshalJSON([]byte(snode.Value))
				subnets[s.ID] = s
			}
		}
	}
	log.WithFields(log.Fields{
		"subnets": subnets,
	}).Info("Fetched subnets metadata")
	return subnets
}

func writeGuestsConfigFile(domain string, guests map[string]*lochness.Guest, subnets map[string]*lochness.Subnet) {
	fmt.Println("# Confd Generated")
	fmt.Println("")
	fmt.Println("group guests {")
	fmt.Println("    option domain-name " + domain + ";")
	fmt.Println("")
	for _, g := range guests {
		if g.HypervisorID == "" || g.SubnetID == "" {
			continue
		}
		s, ok := subnets[g.SubnetID]
		if !ok {
			continue
		}
		fmt.Println("    host " + g.ID + " {")
		fmt.Println("        hardware ethernet " + strings.ToUpper(g.MAC.String()) + ";")
		fmt.Println("        fixed-address " + g.IP.String() + ";")
		fmt.Println("        option routers " + s.Gateway.String() + ";")
		fmt.Println("        option subnet-mask " + s.CIDR.IP.String() + ";")
		fmt.Println("    }")
		fmt.Println("")
	}
	fmt.Println("}")
	fmt.Println("")
}

func writeHypervisorsConfigFile(domain string, hypervisors map[string]*lochness.Hypervisor) {
	fmt.Println("# Confd Generated")
	fmt.Println("")
	fmt.Println("group hypervisors {")
	fmt.Println("    option domain-name \"nodes." + domain + "\";")
	fmt.Println("    if exists user-class and option user-class = \"iPXE\" {")
	fmt.Println("        filename \"http://ipxe.services." + domain + ":8888/ipxe/${net0/ip}\";")
	fmt.Println("    } else {")
	fmt.Println("        next-server tftp.services." + domain + ";")
	fmt.Println("        filename \"undionly.kpxe\";")
	fmt.Println("    }")
	fmt.Println("")
	for _, hv := range hypervisors {
		if hv.Gateway != nil && hv.Netmask != nil {
			fmt.Println("    host " + hv.ID + " {")
			fmt.Println("        hardware ethernet   " + strings.ToUpper(hv.MAC.String()) + ";")
			fmt.Println("        fixed-address       " + hv.IP.String() + ";")
			fmt.Println("        option routers      " + hv.Gateway.String() + ";")
			fmt.Println("        option subnet-mask  " + hv.Netmask.String() + ";")
			fmt.Println("    }")
			fmt.Println("")
		}
	}
	fmt.Println("}")
	fmt.Println("")
}

func main() {

	// Command line options
	var doSetup bool
	var etcdAddress, domain, logLevel string
	flag.StringVarP(&etcdAddress, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.StringVarP(&domain, "domain", "d", "lochness.local", "domain for lochness")
	flag.StringVarP(&logLevel, "log-level", "l", "warning", "log level: debug/info/warning/error/critical/fatal")
	flag.BoolVarP(&doSetup, "setup", "s", false, "turn on to push a hypervisor into etcd first")
	flag.Parse()

	// Logging
	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "log.ParseLevel",
		}).Fatal("Could not parse log level")
	}
	log.SetLevel(level)

	// Lochness Context
	etcdClient := etcd.NewClient([]string{etcdAddress})
	context := lochness.NewContext(etcdClient)

	// Set up some hypervisors to work with
	if doSetup {
		doTestSetup(context, etcdClient)
	}

	// Fetch hypervisors, guests, and subnets
	hypervisors := fetchHypervisors(context, etcdClient)
	guests := fetchGuests(context, etcdClient)
	subnets := fetchSubnets(context, etcdClient)

	// Just print out the config for now
	writeGuestsConfigFile(domain, guests, subnets)
	writeHypervisorsConfigFile(domain, hypervisors)
}

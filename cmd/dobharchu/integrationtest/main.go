package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/dobharchu/testhelper"
	flag "github.com/spf13/pflag"
)

func finish(status int, e *etcd.Client, created map[string]string) {
	fmt.Print("\nExiting test...")
	var path string
	for t, id := range created {
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
			log.WithFields(log.Fields{
				"error": err,
				"func":  "etcd.Delete",
				"path":  path,
			}).Warning("Could not clear test-created data from etcd")
		}
	}
	fmt.Print("done.\n")
	os.Exit(status)
}

func main() {

	// Command line options
	var etcdAddress string
	flag.StringVarP(&etcdAddress, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.Parse()

	// Setup
	e := etcd.NewClient([]string{etcdAddress})
	c := lochness.NewContext(e)
	r := bufio.NewReader(os.Stdin)
	d := make(map[string]string)

	// Remind the user of what they need to do
	fmt.Print("Welcome to the Dobharchu Integration Test!\n")
	fmt.Print("\n")
	fmt.Print("You're using the etcd address " + etcdAddress + ". Is it up and running? (hit enter to continue)")
	_, _ = r.ReadString('\n')
	fmt.Print("Okay, fire up Dobharchu with that etcd address and tail its log. (hit enter when you're ready to continue)")
	_, _ = r.ReadString('\n')
	fmt.Print("\n")
	fmt.Print("Starting test...\n")
	fmt.Print("\n")

	// Add flavors, network, and firewall group
	fmt.Print("Creating two flavors, a network, and a firewall group for building the other objects...\n")
	f1, err := testhelper.NewTestFlavor(c, 4, 4096, 8192)
	if err != nil {
		finish(1, e, d)
	}
	d["flavor"] = f1.ID
	f2, err := testhelper.NewTestFlavor(c, 6, 8192, 1024)
	if err != nil {
		finish(1, e, d)
	}
	d["flavor"] = f2.ID
	n, err := testhelper.NewTestNetwork(c)
	if err != nil {
		finish(1, e, d)
	}
	d["network"] = n.ID
	fw, err := testhelper.NewTestFirewallGroup(c)
	if err != nil {
		finish(1, e, d)
	}
	d["fwgroup"] = fw.ID
	fmt.Print("Did Dobharchu touch the configs? It shouldn't have. (hit enter to continue)")
	_, _ = r.ReadString('\n')
	fmt.Print("\n")

	// Add subnet
	fmt.Print("Creating a new subnet...\n")
	s, err := testhelper.NewTestSubnet(c, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	if err != nil {
		finish(1, e, d)
	}
	fmt.Print("Did Dobharchu touch the configs? The mod date should be sooner, but no changes should appear. (hit enter to continue)")
	_, _ = r.ReadString('\n')
	fmt.Print("\n")

	// Add hypervisors
	fmt.Print("Creating two new hypervisors...\n")
	h1, err := testhelper.NewTestHypervisor(c, "fe:dc:ba:98:76:54", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	if err != nil {
		finish(1, e, d)
	}
	h2, err := testhelper.NewTestHypervisor(c, "dc:ba:98:76:54:32", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	if err != nil {
		finish(1, e, d)
	}
	fmt.Print("Did Dobharchu update the hypervisors config?\n")
	fmt.Print("You should see two new hosts with these IDs:\n")
	fmt.Print(h1.ID + "\n")
	fmt.Print(h2.ID + "\n")
	fmt.Print("Hit enter to continue...")
	_, _ = r.ReadString('\n')
	fmt.Print("\n")

	// Add guests
	fmt.Print("Creating four new guests...\n")
	g1, err := testhelper.NewTestGuest(c, "ba:98:76:54:32:10", n, s, f1, fw, h1)
	if err != nil {
		finish(1, e, d)
	}
	g2, err := testhelper.NewTestGuest(c, "98:76:54:32:10:fe", n, s, f2, fw, h1)
	if err != nil {
		finish(1, e, d)
	}
	g3, err := testhelper.NewTestGuest(c, "76:54:32:10:fe:dc", n, s, f1, fw, h2)
	if err != nil {
		finish(1, e, d)
	}
	g4, err := testhelper.NewTestGuest(c, "54:32:10:fe:dc:ba", n, s, f2, fw, h2)
	if err != nil {
		finish(1, e, d)
	}
	fmt.Print("Did Dobharchu update the guests config?\n")
	fmt.Print("You should see four new hosts with these IDs:\n")
	fmt.Print(g1.ID + "\n")
	fmt.Print(g2.ID + "\n")
	fmt.Print(g3.ID + "\n")
	fmt.Print(g4.ID + "\n")
	fmt.Print("Hit enter to continue...")
	_, _ = r.ReadString('\n')
	fmt.Print("\n")

	fmt.Print("Beginning cleanup. All the extra hosts should go away.")
	finish(0, e, d)
}

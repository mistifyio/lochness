package main

import (
	"fmt"
	"net"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
)

func doTestSetup(context *lochness.Context) {
	err := context.ForEachHypervisor(func(h *lochness.Hypervisor) error {
		return h.Destroy()
	})
	if err != nil {
		log.WithFields(log.Fields{
			"func":  "lochness.ForEachHypervisor",
			"error": err,
		}).Fatal("Could not destroy old hypervisors")
	}
	newTestHypervisor(context, "de:ad:be:ef:7f:21", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0))
	newTestHypervisor(context, "de:ad:be:ef:7f:23", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0))
}

func newTestHypervisor(context *lochness.Context, mac string, ip, gateway, netmask net.IP) {
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
				hv := context.BlankHypervisor("")
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

func writeConfigFile(domain string, hypervisors map[string]*lochness.Hypervisor) {
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
	log.AddHook(&logx.ErrorMessageHook{})
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
		doTestSetup(context)
	}

	// Fetch hypervisors
	hypervisors := fetchHypervisors(context, etcdClient)

	// Just print out the config for now
	writeConfigFile(domain, hypervisors)
}

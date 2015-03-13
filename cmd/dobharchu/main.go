package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/cmd/dobharchu/refresher"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
)

func main() {

	// Command line options
	var etcdAddress, domain, logLevel string
	flag.StringVarP(&etcdAddress, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.StringVarP(&domain, "domain", "d", "lochness.local", "domain for lochness")
	flag.StringVarP(&logLevel, "log-level", "l", "warning", "log level: debug/info/warning/error/critical/fatal")
	flag.Parse()

	// Logging
	err := logx.DefaultSetup(logLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
		}).Fatal("Could not set up logrus")
	}

	// Set up refresher
	r := refresher.NewRefresher(domain, etcdAddress)

	// Just print out the configs for now
	err = r.WriteHypervisorsConfigFile(os.Stdout) // eventual location: "/etc/dhcpd/hypervisors.conf"
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "refresher.WriteHypervisorsConfigFile",
		}).Fatal("Could not write /etc/dhcpd/hypervisors.conf")
	}
	err = r.WriteGuestsConfigFile(os.Stdout) // eventual location: "/etc/dhcpd/guests.conf"
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "refresher.WriteGuestsConfigFile",
		}).Fatal("Could not write /etc/dhcpd/guests.conf")
	}
}

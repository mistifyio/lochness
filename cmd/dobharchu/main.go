package main

import (
	"bufio"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/pkg/watcher"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
)

func updateConfigs(f *Fetcher, r *Refresher, hconfPath, gconfPath string) error {

	// Hypervisors
	hypervisors, err := f.GetHypervisors()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.GetHypervisors",
		}).Error("Could not fetch hypervisors")
		return err
	}
	f1, err := os.Create(hconfPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
			"path":  hconfPath,
		}).Error("Could not open hypervisors conf file")
		return err
	}
	w1 := bufio.NewWriter(f1)
	if err = r.WriteHypervisorsConfigFile(w1, hypervisors); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "Refresher.WriteHypervisorsConfigFile",
		}).Error("Could not refresh hypervisors conf file")
		return err
	}
	if err = w1.Flush(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "bufio.Writer.Flush",
		}).Error("Could not flush buffer for hypervisors conf file")
		return err
	}
	log.WithFields(log.Fields{
		"path": hconfPath,
	}).Info("Refreshed hypervisors conf file")

	// Guests
	guests, err := f.GetGuests()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.GetGuests",
		}).Error("Could not fetch guests")
		return err
	}
	subnets, err := f.GetSubnets()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.GetSubnets",
		}).Error("Could not fetch subnets")
		return err
	}
	f2, err := os.Create(gconfPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
			"path":  gconfPath,
		}).Error("Could not open guests conf file")
		return err
	}
	w2 := bufio.NewWriter(f2)
	if err = r.WriteGuestsConfigFile(w2, guests, subnets); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "Refresher.WriteGuestsConfigFile",
		}).Error("Could not refresh guests conf file")
		return err
	}
	if err = w2.Flush(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "bufio.Writer.Flush",
		}).Error("Could not flush buffer for guests conf file")
		return err
	}
	log.WithFields(log.Fields{
		"path": gconfPath,
	}).Info("Refreshed guests conf file")

	return nil
}

func main() {

	// Command line options
	var etcdAddress, domain, hconfPath, gconfPath, logLevel string
	flag.StringVarP(&domain, "domain", "d", "", "domain for lochness; required")
	flag.StringVarP(&etcdAddress, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.StringVarP(&hconfPath, "hypervisors-path", "", "/etc/dhcpd/hypervisors.conf", "alternative path to hypervisors.conf")
	flag.StringVarP(&gconfPath, "guests-path", "", "/etc/dhcpd/guests.conf", "alternative path to guests.conf")
	flag.StringVarP(&logLevel, "log-level", "l", "warning", "log level: debug/info/warning/error/critical/fatal")
	flag.Parse()

	// Domain is required
	if domain == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Logging
	if err := logx.SetLevel(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
		}).Fatal("Could not set up logrus")
	}

	// Set up fetcher and refresher
	f := NewFetcher(etcdAddress)
	r := NewRefresher(domain)
	err := f.FetchAll()
	if err != nil {
		os.Exit(1)
	}

	// Update at the start of each run
	err = updateConfigs(f, r, hconfPath, gconfPath)
	if err != nil {
		os.Exit(1)
	}

	// Create the watcher
	w, err := watcher.New(f.etcdClient)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "watcher.New",
		}).Fatal("Could not create watcher")
	}

	// Start watching the necessary etcd prefixs
	prefixes := [...]string{"/lochness/hypervisors", "/lochness/guests", "/lochness/subnets"}
	for _, prefix := range prefixes {
		if err := w.Add(prefix); err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"func":   "watcher.Add",
				"prefix": prefix,
			}).Fatal("Could not add watch prefix")
		}
	}

	// Channel for indicating work in progress
	// (to coordinate clean exiting between the consumer and the signal handler)
	ready := make(chan struct{}, 1)
	ready <- struct{}{}

	for w.Next() {
		// Remove item to indicate processing has begun
		done := <-ready

		// Integrate the response and update the configs if necessary
		err = f.IntegrateResponse(w.Response())
		if err == nil {
			_ = updateConfigs(f, r, hconfPath, gconfPath)
		}

		// Return item to indicate processing has completed
		ready <- done
	}
	if err := w.Err(); err != nil {
		log.WithField("error", err).Fatal("Watcher encountered an error")
	}

	// Handle signals for clean shutdown
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	s := <-sigs
	log.WithField("signal", s).Info("Signal received; waiting for current task to process")
	<-ready // wait until any current processing is finished
	_ = w.Close()
	log.Info("Exiting")
}

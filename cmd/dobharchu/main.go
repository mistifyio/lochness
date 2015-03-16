package main

import (
	"bufio"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/cmd/dobharchu/refresher"
	"github.com/mistifyio/lochness/pkg/watcher"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
)

func updateConfigs(r *refresher.Refresher, hconfPath, gconfPath string) error {
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
	err = r.WriteHypervisorsConfigFile(w1)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "refresher.WriteHypervisorsConfigFile",
		}).Error("Could not refresh hypervisors conf file")
		return err
	}
	w1.Flush()
	log.WithFields(log.Fields{
		"path": hconfPath,
	}).Info("Refreshed hypervisors conf file")

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
	err = r.WriteGuestsConfigFile(w2)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "refresher.WriteGuestsConfigFile",
		}).Error("Could not refresh guests conf file")
		return err
	}
	w2.Flush()
	log.WithFields(log.Fields{
		"path": gconfPath,
	}).Info("Refreshed guests conf file")

	return nil
}

func main() {

	// Command line options
	var etcdAddress, domain, hconfPath, gconfPath, logLevel string
	var force, testMode bool
	flag.StringVarP(&etcdAddress, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.StringVarP(&domain, "domain", "d", "lochness.local", "domain for lochness")
	flag.StringVarP(&hconfPath, "hypervisors-path", "", "/etc/dhcpd/hypervisors.conf", "alternative path to hypervisors.conf")
	flag.StringVarP(&gconfPath, "guests-path", "", "/etc/dhcpd/guests.conf", "alternative path to guests.conf")
	flag.BoolVarP(&force, "force", "f", false, "force an update right now")
	flag.BoolVarP(&testMode, "test-mode", "t", false, "run in test mode; do not require etcd.SyncCluster to work")
	flag.StringVarP(&logLevel, "log-level", "l", "warning", "log level: debug/info/warning/error/critical/fatal")
	flag.Parse()

	// Logging
	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
		}).Fatal("Could not set up logrus")
	}

	// Set up refresher
	r := refresher.NewRefresher(domain, etcdAddress)

	if !testMode {
		// Make sure we can actually connect to etcd
		if !r.EtcdClient.SyncCluster() {
			log.WithFields(log.Fields{
				"error":   err,
				"func":    "etcd.SyncCluster",
				"address": etcdAddress,
			}).Fatal("Could not connect to etcd cluster")
		}
	}

	// Is this a force-update run?
	if force {
		err := updateConfigs(r, hconfPath, gconfPath)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Create the watcher
	w, err := watcher.New(r.EtcdClient)
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

		// Just print out the configs for now
		updateConfigs(r, hconfPath, gconfPath)

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

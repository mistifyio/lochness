package main

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/pkg/watcher"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
)

func updateConfigs(f *Fetcher, r *Refresher, hconfPath, gconfPath string) (bool, error) {
	restart := false

	// Hypervisors
	hypervisors, err := f.Hypervisors()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.Hypervisors",
		}).Error("Could not fetch hypervisors")
		return restart, err
	}

	err = writeConfig("hypervisors", hconfPath, func(w io.Writer) error {
		err := r.genHypervisorsConf(w, hypervisors)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "Refresher.genHypervisorsConf",
				"type":  "hypervisors",
			}).Error("Could not generate configuration")
		}
		return err
	})
	if err == nil {
		restart = true
	}

	// Guests
	guests, err := f.Guests()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.Guests",
		}).Error("Could not fetch guests")
		return restart, err
	}
	subnets, err := f.Subnets()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.Subnets",
		}).Error("Could not fetch subnets")
		return restart, err
	}

	err = writeConfig("guests", gconfPath, func(w io.Writer) error {
		err := r.genGuestsConf(w, guests, subnets)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "Refresher.genGuestsConf",
				"type":  "guests",
			}).Error("Could not generate configuration")
		}
		return err
	})
	if err == nil {
		restart = true
	}

	return restart, nil
}

func writeConfig(confType, path string, generator func(io.Writer) error) error {
	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
			"path":  tmp,
			"type":  confType,
		}).Error("Could not create temporary conf file")
		return err
	}

	buff := bufio.NewWriter(file)
	if err = generator(buff); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  tmp,
			"type":  confType,
		}).Error("Could not generate temporary configuration")
		return err
	}

	if err = buff.Flush(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "buff.Flush",
			"path":  tmp,
			"type":  confType,
		}).Error("Could not flush buffer to temporary conf file")
		return err
	}

	if err = file.Close(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.File.Close",
			"path":  tmp,
			"type":  confType,
		}).Error("Could not close temporary conf file")
		return err
	}

	if err = os.Rename(tmp, path); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Rename",
			"from":  tmp,
			"to":    path,
			"type":  confType,
		}).Error("Could not rename temporary conf file")
		return err
	}

	log.WithFields(log.Fields{
		"path": path,
		"type": confType,
	}).Info("Replaced conf file")
	return nil
}

func restart_dhcpd() {
	cmd := exec.Command("systemctl", "restart", "dhcpd.service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "cmd.Run",
		}).Error("failed to restart dhcpd service")
	}
}

func main() {

	// Command line options
	var etcdAddress, domain, confPath, logLevel string
	flag.StringVarP(&domain, "domain", "d", "", "domain for lochness; required")
	flag.StringVarP(&etcdAddress, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.StringVarP(&confPath, "conf-dir", "c", "/etc/dhcp/", "dhcpd configuration directory")
	flag.StringVarP(&logLevel, "log-level", "l", "warning", "log level: debug/info/warning/error/critical/fatal")
	flag.Parse()

	// Domain is required
	if domain == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Logging
	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
		}).Fatal("Could not set up logrus")
	}

	hconfPath := path.Join(confPath, "hypervisors.conf")
	gconfPath := path.Join(confPath, "guests.conf")

	// Set up fetcher and refresher
	f := NewFetcher(etcdAddress)
	r := NewRefresher(domain)
	err := f.FetchAll()
	if err != nil {
		os.Exit(1)
	}

	// Update at the start of each run
	restart, err := updateConfigs(f, r, hconfPath, gconfPath)
	if restart {
		restart_dhcpd()
	}
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
		refresh, err := f.IntegrateResponse(w.Response())
		if err != nil {
			log.Info("Error on integration; re-fetching")
			err := f.FetchAll()
			if err != nil {
				os.Exit(1)
			}
			refresh = true
		}
		if refresh {
			restart, err := updateConfigs(f, r, hconfPath, gconfPath)
			if restart {
				restart_dhcpd()
			}
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"func":  "updateConfigs",
				}).Warn("Could not create watcher")
			}
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

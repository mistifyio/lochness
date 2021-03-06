package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
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

var hypervisorsHash []byte
var guestsHash []byte

func updateConfigs(f *Fetcher, r *Refresher, hconfPath, gconfPath string) (bool, error) {
	restart := false

	// Hypervisors
	hypervisors, err := f.Hypervisors()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.Hypervisors",
		}).Error("could not fetch hypervisors")
		return restart, err
	}

	checksum, err := writeConfig("hypervisors", hconfPath, hypervisorsHash, func(w io.Writer) error {
		err := r.genHypervisorsConf(w, hypervisors)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "Refresher.genHypervisorsConf",
				"type":  "hypervisors",
			}).Error("could not generate configuration")
		}
		return err
	})
	if err == nil && checksum != nil {
		hypervisorsHash = checksum
		restart = true
	}

	// Guests
	guests, err := f.Guests()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.Guests",
		}).Error("could not fetch guests")
		return restart, err
	}
	subnets, err := f.Subnets()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "fetcher.Subnets",
		}).Error("could not fetch subnets")
		return restart, err
	}

	checksum, err = writeConfig("guests", gconfPath, guestsHash, func(w io.Writer) error {
		err := r.genGuestsConf(w, guests, subnets)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "Refresher.genGuestsConf",
				"type":  "guests",
			}).Error("could not generate configuration")
		}
		return err
	})
	if err == nil && checksum != nil {
		guestsHash = checksum
		restart = true
	}

	return restart, nil
}

func writeConfig(confType, path string, checksum []byte, generator func(io.Writer) error) ([]byte, error) {
	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
			"path":  tmp,
			"type":  confType,
		}).Error("Could not create temporary conf file")
		return nil, err
	}

	hash := md5.New()
	buff := bufio.NewWriter(io.MultiWriter(file, hash))
	if err = generator(buff); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"path":  tmp,
			"type":  confType,
		}).Error("could not generate temporary configuration")
		return nil, err
	}

	if err = buff.Flush(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "buff.Flush",
			"path":  tmp,
			"type":  confType,
		}).Error("could not flush buffer to temporary conf file")
		return nil, err
	}

	if err = file.Close(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.File.Close",
			"path":  tmp,
			"type":  confType,
		}).Error("could not close temporary conf file")
		return nil, err
	}

	if bytes.Equal(checksum, hash.Sum(nil)) {
		log.Debug("no change to conf file")
		if err := os.Remove(tmp); err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"filepath": tmp,
			}).Error("failed to remove temp file")
		}
		return nil, nil
	}

	if err = os.Rename(tmp, path); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Rename",
			"from":  tmp,
			"to":    path,
			"type":  confType,
		}).Error("could not rename temporary conf file")
		return nil, err
	}

	log.WithFields(log.Fields{
		"path": path,
		"type": confType,
	}).Info("replaced conf file")

	return hash.Sum(nil), nil
}

func restartDhcpd() {
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
	var kvAddress, domain, confPath, logLevel string
	flag.StringVarP(&domain, "domain", "d", "", "domain for lochness; required")
	flag.StringVarP(&kvAddress, "kv", "k", "http://127.0.0.1:4001", "address of kv server")
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
		}).Fatal("could not set up logrus")
	}

	hconfPath := path.Join(confPath, "hypervisors.conf")
	gconfPath := path.Join(confPath, "guests.conf")

	// Set up fetcher and refresher
	f := NewFetcher(kvAddress)
	r := NewRefresher(domain)
	err := f.FetchAll()
	if err != nil {
		os.Exit(1)
	}

	// Update at the start of each run
	restart, err := updateConfigs(f, r, hconfPath, gconfPath)
	if restart {
		restartDhcpd()
	}
	if err != nil {
		os.Exit(1)
	}

	// Create the watcher
	w, err := watcher.New(f.kv)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "watcher.New",
		}).Fatal("could not create watcher")
	}

	// Start watching the necessary kv prefixes
	prefixes := []string{"/lochness/hypervisors", "/lochness/guests", "/lochness/subnets"}
	for _, prefix := range prefixes {
		if err := w.Add(prefix); err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"func":   "watcher.Add",
				"prefix": prefix,
			}).Fatal("could not add watch prefix")
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
		refresh, err := f.IntegrateResponse(w.Event())
		if err != nil {
			log.Info("error on integration; re-fetching")
			err := f.FetchAll()
			if err != nil {
				os.Exit(1)
			}
			refresh = true
		}
		if refresh {
			restart, err := updateConfigs(f, r, hconfPath, gconfPath)
			if restart {
				restartDhcpd()
			}
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"func":  "updateConfigs",
				}).Warn("could not create watcher")
			}
		}

		// Return item to indicate processing has completed
		ready <- done
	}
	if err := w.Err(); err != nil {
		log.WithField("error", err).Fatal("watcher encountered an error")
	}

	// Handle signals for clean shutdown
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	s := <-sigs
	log.WithField("signal", s).Info("signal received; waiting for current task to process")
	<-ready // wait until any current processing is finished
	_ = w.Close()
	log.Info("exiting")
}

package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	flag "github.com/ogier/pflag"
)

// loadConfig reads the config file and unmarshals it into a map containing
// keys to watch and whether to do so recursively. The config file should not
// be empty
func loadConfig(path string) (map[string]bool, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := make(map[string]bool)
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config) == 0 {
		return nil, errors.New("empty config")
	}

	return config, nil
}

// handleEvents handles notifications from etcd.Watch
func handleEvents(receiver chan *etcd.Response, done chan struct{}) {
	for {
		select {
		case <-done:
			log.Info("handler stopped")
			done <- struct{}{}
			return
		case resp := <-receiver:
			log.WithField("response", resp).Info("response received")
			// TODO: Run ansible, either a full run or specific tag
		}
	}
}

func main() {
	logLevel := flag.StringP("log-level", "l", "warn", "log level")
	eaddr := flag.StringP("etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	configPath := flag.StringP("config", "c", "", "path to config file with keys")
	flag.Parse()

	// Set up logging
	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logrus.ParseLevel",
			"level": *logLevel,
		}).Fatal("error parsing log level")
	}
	log.SetLevel(level)

	// Load config containing keys to watch
	config, err := loadConfig(*configPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"configPath": *configPath,
		}).Fatal("failed to load config")
	}

	log.WithField("config", config).Info("config loaded")

	// set up etcd connection
	log.WithField("address", *eaddr).Info("connection to etcd")
	etcdClient := etcd.NewClient([]string{*eaddr})
	// make sure we can actually connect to etcd
	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"error":   err,
			"address": *eaddr,
		}).Fatal("failed to connect to etcd cluster")
	}

	// to convey etcd events to the handler
	receiver := make(chan *etcd.Response)
	// to stop the handler and exit program
	done := make(chan struct{})

	// start waiting for events
	go handleEvents(receiver, done)

	// start watching etcd keys
	for key, recursive := range config {
		go etcdClient.Watch(key, 0, recursive, receiver, nil)
	}

	// handle signals for clean shutdown
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	s := <-sigs
	log.WithField("signal", s).Info("signal received")
	done <- struct{}{}
	<-done
	log.Info("exiting")
}

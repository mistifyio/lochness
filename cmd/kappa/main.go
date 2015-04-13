package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/watcher"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

type (
	// Tags is a list of ansible tags
	Tags []string

	// Config is a map of etcd prefixes to watch to ansible tags to run
	Config map[string]Tags
)

const eaddress = "http://127.0.0.1:4001"

var ansibleDir = "/tmp/lochness-ansible"

// loadConfig reads the config file and unmarshals it into a map containing
// prefixs to watch and ansible tags to run. An empty tag array means a full
// playbook run. The config file should not be empty
func loadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := make(Config)
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if len(config) == 0 {
		return nil, errors.New("empty config")
	}

	return config, nil
}

// getTags returns the ansible tags, if any, associated with a key
func getTags(config Config, key string) []string {
	// Check for exact match
	if tags, ok := config[key]; ok {
		return tags
	}

	// Find prefix
	for watchPrefix, tags := range config {
		if !strings.HasPrefix(key, watchPrefix) {
			continue
		}
		return tags
	}

	return nil
}

// runAnsible kicks off an ansible run
func runAnsible(config Config, etcdaddr string, keys ...string) {
	tagSet := map[string]struct{}{}
	for _, key := range keys {
		tags := getTags(config, key)
		if len(tags) == 0 {
			tagSet = map[string]struct{}{}
			break
		}
		for _, tag := range tags {
			tagSet[tag] = struct{}{}
		}
	}
	keyTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		keyTags = append(keyTags, tag)
	}
	sort.Strings(keyTags)

	args := make([]string, 0, 2+len(keyTags)*2)
	args = append(args, "--etcd", etcdaddr)
	for _, tag := range keyTags {
		args = append(args, "-t", tag)
	}
	cmd := exec.Command(path.Join(ansibleDir, "run"), args...)
	cmd.Dir = ansibleDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.WithFields(log.Fields{
			"keys":       keys,
			"ansibleDir": ansibleDir,
			"args":       args,
			"error":      err,
			"errorMsg":   err.Error(),
		}).Fatal("ansible run failed")
	}
}

// consumeResponses consumes etcd respones from a watcher and kicks off ansible
func consumeResponses(config Config, eaddr string, w *watcher.Watcher, ready chan struct{}) {
	key := make(chan string, 1)
	go func() {
		for w.Next() {
			resp := w.Response()
			log.WithField("response", resp).Info("response received")
			key <- resp.Node.Key
			log.WithField("response", resp).Info("response processed")
		}
		if err := w.Err(); err != nil {
			log.WithField("error", err).Fatal("watcher error")
		}
	}()

	keys := map[string]struct{}{}
	timer := time.NewTimer(100 * time.Millisecond)
	timer.Stop()
	max := time.NewTimer(1 * time.Second)
	max.Stop()
	maxStopped := true
	for {
		select {
		case k := <-key:
			timer.Reset(100 * time.Millisecond)
			if maxStopped {
				max.Reset(1 * time.Second)
				maxStopped = false
			}
			keys[k] = struct{}{}
			continue
		case <-max.C:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			if !max.Stop() {
				<-max.C
			}
		}
		maxStopped = true
		// remove item to indicate processing has begun
		done := <-ready
		aKeys := make([]string, 0, len(keys))
		for key := range keys {
			aKeys = append(aKeys, key)
		}
		runAnsible(config, eaddr, aKeys...)
		// return item to indicate processing has completed
		ready <- done
		keys = map[string]struct{}{}
	}
}

// watchKeys creates a new Watcher and adds all configured keys
func watchKeys(config Config, etcdClient *etcd.Client) *watcher.Watcher {
	w, err := watcher.New(etcdClient)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create watcher")
	}

	// start watching etcd prefixs
	for prefix := range config {
		if err := w.Add(prefix); err != nil {
			log.WithFields(log.Fields{
				"prefix":   prefix,
				"error":    err,
				"errorMsg": err.Error(),
			}).Fatal("failed to add watch prefix")
		}
	}

	return w
}

func main() {
	// environment can only override default address
	eaddr := os.Getenv("KAPPA_ETCD_ADDRESS")
	if eaddr == "" {
		eaddr = eaddress
	}

	logLevel := flag.StringP("log-level", "l", "warn", "log level")
	flag.StringVarP(&ansibleDir, "ansible", "a", ansibleDir, "directory containing the ansible run command")
	flag.StringP("etcd", "e", eaddress, "address of etcd server")
	configPath := flag.StringP("config", "c", "", "path to config file with prefixs")
	once := flag.BoolP("once", "o", false, "run only once and then exit")
	flag.Parse()
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "etcd" {
			eaddr = f.Value.String()
		}
	})

	// Set up logging
	if err := logx.DefaultSetup(*logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": *logLevel,
		}).Fatal("failed to set up logging")
	}

	// Load config containing prefixs to watch
	config, err := loadConfig(*configPath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"configPath": *configPath,
		}).Fatal("failed to load config")
	}

	log.WithField("config", config).Info("config loaded")

	// set up etcd connection
	log.WithField("address", eaddr).Info("connection to etcd")
	etcdClient := etcd.NewClient([]string{eaddr})
	// make sure we can actually connect to etcd
	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"error":   err,
			"address": eaddr,
		}).Fatal("failed to connect to etcd cluster")
	}

	// always run initially
	runAnsible(config, eaddr, "")
	if *once {
		return
	}

	// set up watcher
	w := watchKeys(config, etcdClient)

	// to coordinate clean exiting between the consumer and the signal handler
	ready := make(chan struct{}, 1)
	ready <- struct{}{}

	// handle events
	go consumeResponses(config, eaddr, w, ready)

	// handle signals for clean shutdown
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	s := <-sigs
	log.WithField("signal", s).Info("signal received. waiting for current task to process")
	// wait until any current processing is finished
	<-ready
	_ = w.Close()
	log.Info("exiting")
}

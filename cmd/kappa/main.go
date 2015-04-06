package main

import (
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/watcher"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type (
	// Tags is a list of ansible tags
	Tags []string

	// TagMap is a map of etcd prefixes to watch to ansible tags to run
	TagMap map[string]Tags
)

var tagMap TagMap

// getTags returns the ansible tags, if any, associated with a key
func getTags(key string) []string {
	// Check for exact match
	if tags, ok := tagMap[key]; ok {
		return tags
	}

	// Find prefix
	for watchPrefix, tags := range tagMap {
		if !strings.HasPrefix(key, watchPrefix) {
			continue
		}
		return tags
	}

	return nil
}

// runAnsible kicks off an ansible run
func runAnsible(key string) {
	keyTags := getTags(key)
	args := make([]string, 0, len(keyTags)*2)
	for _, tag := range keyTags {
		args = append(args, "-t", tag)
	}
	ansibleDir := viper.GetString("ansible")
	cmd := exec.Command(path.Join(ansibleDir, "run"), args...)
	cmd.Dir = ansibleDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.WithFields(log.Fields{
			"key":        key,
			"ansibleDir": ansibleDir,
			"args":       args,
			"error":      err,
			"errorMsg":   err.Error(),
		}).Fatal("ansible run failed")
	}
}

// consumeResponses consumes etcd respones from a watcher and kicks off ansible
func consumeResponses(w *watcher.Watcher, ready chan struct{}) {
	for w.Next() {
		// remove item to indicate processing has begun
		done := <-ready

		resp := w.Response()
		log.WithField("response", resp).Info("response received")
		runAnsible(resp.Node.Key)
		log.WithField("response", resp).Info("response processed")

		// return item to indicate processing has completed
		ready <- done
	}
	if err := w.Err(); err != nil {
		log.WithField("error", err).Fatal("watcher error")
	}
}

// watchKeys creates a new Watcher and adds all configured keys
func watchKeys(etcdClient *etcd.Client) *watcher.Watcher {
	w, err := watcher.New(etcdClient)
	if err != nil {
		log.WithField("error", err).Fatal("failed to create watcher")
	}

	// start watching etcd prefixs
	for prefix := range tagMap {
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

	// Flags for direct use
	var logLevel, configPath string
	flag.StringVarP(&logLevel, "log-level", "l", "warning", "log level: debug/info/warning/error/critical/fatal")
	flag.StringVarP(&configPath, "config", "c", "", "path to config file with prefixes")

	// Flags for use with viper
	_ = flag.StringP("etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	_ = flag.StringP("ansible", "a", "/root/lochness-ansible", "directory containing the ansible run command")
	flag.Parse()

	// Set up logging
	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": logLevel,
		}).Fatal("failed to set up logging")
	}

	// Read in the config file
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(log.Fields{
			"error":      err,
			"func":       "viper.ReadInConfig",
			"configPath": configPath,
		}).Fatal("failed to load config")
	}

	// Bind the flags to viper
	viper.BindPFlags(flag.CommandLine)
	if err := viper.ReadInConfig(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "viper.BindPFlags",
		}).Fatal("failed to bind flags")
	}

	// Parse the tag map
	if err := viper.MarshalKey("tag-map", &tagMap); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "viper.MarshalKey",
			"key":   "tag-map",
		}).Fatal("ansible tag map is improperly formed")
	}
	if len(tagMap) == 0 {
		log.Fatal("ansible tag map is empty")
	}
	log.WithField("tag-map", tagMap).Info("ansible tag map loaded")

	// Set up etcd connection
	eaddr := viper.GetString("etcd")
	log.WithField("address", eaddr).Info("connection to etcd")
	etcdClient := etcd.NewClient([]string{eaddr})

	// Make sure we can actually connect to etcd
	if !etcdClient.SyncCluster() {
		log.WithField("address", eaddr).Fatal("failed to connect to etcd cluster")
	}

	// Set up watcher
	w := watchKeys(etcdClient)

	// To coordinate clean exiting between the consumer and the signal handler
	ready := make(chan struct{}, 1)
	ready <- struct{}{}

	// Handle events
	go consumeResponses(w, ready)

	// Handle signals for clean shutdown
	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	s := <-sigs
	log.WithField("signal", s).Info("signal received. waiting for current task to process")
	// Wait until any current processing is finished
	<-ready
	_ = w.Close()
	log.Info("exiting")
}

package main

import (
	log "github.com/Sirupsen/logrus"
	kv "github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

const defaultKVAddr = "http://localhost:4001"

func main() {
	var port uint
	var kvAddr, logLevel string

	flag.UintVarP(&port, "port", "p", 19000, "listen port")
	flag.StringVarP(&kvAddr, "kv", "k", defaultKVAddr, "address of kv machine")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.Parse()

	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"level": logLevel,
		}).Fatal("failed to set up logging")
	}

	kvClient := kv.NewClient([]string{kvAddr})

	if !kvClient.SyncCluster() {
		log.WithFields(log.Fields{
			"addr": kvAddr,
		}).Fatal("unable to sync etcd cluster")
	}

	ctx := lochness.NewContext(kvClient)

	server := Run(port, ctx)
	// Block until the server is stopped
	<-server.StopChan()
}

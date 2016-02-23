package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

const defaultEtcdAddr = "http://localhost:4001"

func main() {
	var port uint
	var etcdAddr, logLevel string

	flag.UintVarP(&port, "port", "p", 17000, "listen port")
	flag.StringVarP(&etcdAddr, "etcd", "e", defaultEtcdAddr, "address of etcd machine")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.Parse()

	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": logLevel,
		}).Fatal("failed to set up logging")
	}

	etcdClient := etcd.NewClient([]string{etcdAddr})

	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"addr": etcdAddr,
			"func": "etcd.SyncCluster",
		}).Fatal("unable to sync etcd cluster")
	}

	ctx := lochness.NewContext(etcdClient)

	server := Run(port, ctx)
	// Block until the server is stopped
	<-server.StopChan()
}

package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	flag "github.com/ogier/pflag"
)

const defaultEtcdAddr = "http://localhost:4001"

func main() {
	var port uint
	var etcdAddr, logLevel string

	flag.UintVarP(&port, "port", "p", 18000, "listen port")
	flag.StringVarP(&etcdAddr, "etcd", "e", defaultEtcdAddr, "address of etcd machine")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.Parse()

	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "log.ParseLevel",
		}).Fatal(err)
	}
	log.SetLevel(level)

	etcdClient := etcd.NewClient([]string{etcdAddr})

	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.SyncCluster",
			"addr":  etcdAddr,
		}).Fatal("unable to sync etcd cluster")
	}

	ctx := lochness.NewContext(etcdClient)

	_ = Run(port, ctx)
}

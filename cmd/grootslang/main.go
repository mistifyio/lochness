package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	flag "github.com/ogier/pflag"
)

const defaultEtcdAddr = "http://localhost:4001"

func main() {
	var port uint
	var etcdAddr, logLevel string
	var h bool

	flag.BoolVarP(&h, "help", "h", false, "display the help")
	flag.UintVarP(&port, "port", "p", 17000, "listen port")
	flag.StringVarP(&etcdAddr, "etcd", "e", defaultEtcdAddr, "address of etcd machine")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.Parse()

	if h {
		flag.PrintDefaults()
		os.Exit(0)
	}

	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)

	etcdClient := etcd.NewClient([]string{etcdAddr})

	if !etcdClient.SyncCluster() {
		log.Fatal("unable to sync etcd at $s", etcdAddr)
	}

	ctx := lochness.NewContext(etcdClient)

	_ = Run(port, ctx)
}

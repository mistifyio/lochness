package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/mistifyio/lochness"
)

const defaultEtcdAddr = "http://localhost:4001"

func main() {
	var port uint
	var etcdAddr, logLevel string
	var h bool

	flag.BoolVar(&h, []string{"h", "#help", "-help"}, false, "display the help")
	flag.UintVar(&port, []string{"p", "#port", "-port"}, 15000, "listen port")
	flag.StringVar(&etcdAddr, []string{"e", "#etcd", "-etcd"}, defaultEtcdAddr, "address of etcd machine")
	flag.StringVar(&logLevel, []string{"l", "#log-level", "-log-level"}, "warn", "log level")
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
	ctx := lochness.NewContext(etcdClient)

	_ = Run(port, ctx)
}

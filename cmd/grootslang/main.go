package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/spf13/cobra"
)

var (
	port     uint = 17000
	etcdAddr      = "http://localhost:4001"
	logLevel      = "warn"
)

func run(cmd *cobra.Command, args []string) {
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

func main() {
	root := &cobra.Command{
		Use:  "grootslang",
		Long: "grootslang is an HTTP API for LochNess hypervisors",
		Run:  run,
	}
	root.Flags().UintVarP(&port, "port", "p", port, "listen port")
	root.Flags().StringVarP(&etcdAddr, "etcd", "e", etcdAddr, "address of etcd machine")
	root.Flags().StringVarP(&logLevel, "log-level", "l", logLevel, "log level")

	root.Execute()
}

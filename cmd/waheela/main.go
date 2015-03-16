package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-metrics"
	"github.com/bakins/go-metrics-map"
	"github.com/bakins/go-metrics-middleware"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

type metricsContext struct {
	sink    *mapsink.MapSink
	metrics *metrics.Metrics
	mmw     *mmw.Middleware
}

const defaultEtcdAddr = "http://localhost:4001"

func main() {
	var port uint
	var etcdAddr, logLevel, statsd string

	flag.UintVarP(&port, "port", "p", 18000, "listen port")
	flag.StringVarP(&etcdAddr, "etcd", "e", defaultEtcdAddr, "address of etcd machine")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.StringVarP(&statsd, "statsd", "s", "", "statsd address")
	flag.Parse()

	err := logx.DefaultSetup(logLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": logLevel,
		}).Fatal("unable to set up logrus")
	}

	etcdClient := etcd.NewClient([]string{etcdAddr})

	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.SyncCluster",
			"addr":  etcdAddr,
		}).Fatal("unable to sync etcd cluster")
	}

	// setup metrics
	sink := mapsink.New()
	fanout := metrics.FanoutSink{sink}

	if statsd != "" {
		ss, _ := metrics.NewStatsdSink(statsd)
		fanout = append(fanout, ss)
	}
	conf := metrics.DefaultConfig("waheela")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, fanout)

	mctx := &metricsContext{
		sink:    sink,
		metrics: m,
		mmw:     mmw.New(m),
	}
	ctx := lochness.NewContext(etcdClient)

	_ = Run(port, ctx, mctx)
}

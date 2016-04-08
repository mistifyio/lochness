package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-metrics"
	"github.com/bakins/go-metrics-map"
	"github.com/bakins/go-metrics-middleware"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/mistifyio/lochness/pkg/kv"
	_ "github.com/mistifyio/lochness/pkg/kv/consul"
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
	var kvAddr, bstalk, logLevel, statsd string

	flag.UintVarP(&port, "port", "p", 18000, "listen port")
	flag.StringVarP(&kvAddr, "kv", "k", defaultEtcdAddr, "address of kv machine")
	flag.StringVarP(&bstalk, "beanstalk", "b", "127.0.0.1:11300", "address of beanstalkd server")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.StringVarP(&statsd, "statsd", "s", "", "statsd address")
	flag.Parse()

	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": logLevel,
		}).Fatal("unable to set up logrus")
	}

	e, err := kv.New(kvAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"addr":  kvAddr,
			"error": err,
			"func":  "kv.New",
		}).Fatal("unable to connect to kv")
	}

	ctx := lochness.NewContext(e)

	log.WithField("address", bstalk).Info("connection to beanstalk")
	jobQueue, err := jobqueue.NewClient(bstalk, e)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": bstalk,
		}).Fatal("failed to create jobQueue client")
	}

	// setup metrics
	sink := mapsink.New()
	fanout := metrics.FanoutSink{sink}

	if statsd != "" {
		ss, _ := metrics.NewStatsdSink(statsd)
		fanout = append(fanout, ss)
	}
	conf := metrics.DefaultConfig("cguestd")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, fanout)

	mctx := &metricsContext{
		sink:    sink,
		metrics: m,
		mmw:     mmw.New(m),
	}

	server := Run(port, ctx, jobQueue, mctx)
	// Block until the server is stopped
	<-server.StopChan()
}

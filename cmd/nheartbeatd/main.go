package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/kv"
	_ "github.com/mistifyio/lochness/pkg/kv/etcd"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

func main() {
	interval := flag.DurationP("interval", "i", 0*time.Second, "update interval (default ttl/2)")
	ttl := flag.DurationP("ttl", "t", 120*time.Second, "heartbeat ttl (min: 10s)")
	kvAddr := flag.StringP("kv", "k", "http://localhost:4001", "address of kv machine")
	id := flag.StringP("id", "d", "", "hypervisor id")
	logLevel := flag.StringP("log-level", "l", "info", "log level")
	flag.Parse()

	var intervalSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "interval" {
			intervalSet = true
		}
	})

	if err := logx.DefaultSetup(*logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": logLevel,
		}).Fatal("failed to set up logging")
	}

	if !intervalSet {
		*interval = (*ttl) / 2
	}

	if *ttl < *interval {
		log.Fatal("ttl must be greater than interval")
	}
	if *ttl < 10*time.Second {
		log.Fatal("ttl must be at least 10s")
	}

	KV, err := kv.New(*kvAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "kvNew",
			"id":    id,
		}).Fatal("failed to connect to kv")
	}

	c := lochness.NewContext(KV)

	hn, err := lochness.SetHypervisorID(*id)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "lochness.SetHypervisorID",
			"id":    id,
		}).Fatal("failed to set hypervisor id")
	}

	hv, err := c.Hypervisor(hn)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "context.Hypervisor",
			"id":    hn,
		}).Fatal("failed to instantiate hypervisor")
	}

	for {
		if err = hv.UpdateResources(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "hv.UpdateResources",
			}).Fatal("failed to update hypervisor resources")
		}
		if err = hv.Heartbeat(*ttl); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "hv.Heartbeat",
				"ttl":   *ttl,
			}).Fatal("failed to beat heart")
		}
		time.Sleep(*interval)
	}
}

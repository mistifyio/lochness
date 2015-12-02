package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

func main() {
	interval := flag.IntP("interval", "i", 60, "update interval in seconds")
	ttl := flag.IntP("ttl", "t", 0, "heartbeat ttl in seconds")
	eaddr := flag.StringP("etcd", "e", "http://localhost:4001", "address of etcd machine")
	id := flag.StringP("id", "d", "", "hypervisor id")
	logLevel := flag.StringP("log-level", "l", "info", "log level")
	flag.Parse()

	if err := logx.DefaultSetup(*logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": logLevel,
		}).Fatal("failed to set up logging")
	}

	if *ttl == 0 {
		*ttl = 2 * (*interval)
	}

	if *ttl < *interval {
		log.Fatal("ttl must be greater than interval")
	}

	e := etcd.NewClient([]string{*eaddr})
	c := lochness.NewContext(e)

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
		if err = hv.Heartbeat(time.Duration(*ttl) * time.Second); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "hv.Heartbeat",
				"ttl":   *ttl,
			}).Fatal("failed to beat heart")
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

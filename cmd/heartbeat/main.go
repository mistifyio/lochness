package main

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	flag "github.com/ogier/pflag"
)

func main() {
	interval := flag.IntP("interval", "i", 60, "update interval in seconds")
	ttl := flag.IntP("ttl", "t", 0, "heartbeat ttl in seconds")
	eaddr := flag.StringP("etcd", "e", "http://localhost:4001", "address of etcd machine")
	id := flag.StringP("id", "d", "", "hypervisor id")
	flag.Parse()

	if *ttl == 0 {
		*ttl = 2 * (*interval)
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
			}).Error("failed to update hypervisor resources")
		}
		if err = hv.Heartbeat(time.Duration(*ttl)); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "hv.Heartbeat",
				"ttl":   *ttl,
			}).Error("failed to beat heart")
		}
		if _, err := os.Stdout.WriteString("♥ "); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "os.Stdout.WriteString",
			}).Error("failed to write heartbeat to stdout")
		}
		if err := os.Stdout.Sync(); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "os.Stdout.Sync",
			}).Error("failed to sync stdout")
		}
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

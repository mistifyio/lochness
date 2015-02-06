package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

func main() {
	interval := flag.Int("interval", 60, "update interval in seconds")
	ttl := flag.Int("ttl", 2*(*interval), "heartbeat ttl in seconds")
	eaddr := flag.String("etcd", "http://localhost:4001", "address of etcd machine")
	hid := flag.String("hid", "", "hypervisor id")
	flag.Parse()

	e := etcd.NewClient([]string{*eaddr})
	c := lochness.NewContext(e)

	hn, err := lochness.SetHypervisorID(*hid)
	if err != err {
		log.Fatal(err)
	}

	hv, err := c.Hypervisor(hn)
	if err != nil {
		log.Fatal(err)
	}

	for {
		if err = hv.UpdateResources(); err != nil {
			log.Println(err)
		}
		if err = hv.Heartbeat(time.Duration(*ttl)); err != nil {
			log.Println(err)
		}
		os.Stdout.WriteString("♥ ")
		os.Stdout.Sync()
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

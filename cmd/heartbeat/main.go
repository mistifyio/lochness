package main

import (
	"log"
	"os"
	"time"

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

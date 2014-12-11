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
	var interval = flag.Int("interval", 60, "resource update interval in seconds")
	flag.Parse()

	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	c := lochness.NewContext(e)

	hn, err := os.Hostname()
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
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

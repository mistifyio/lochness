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
	var interval = flag.Int("interval", 60, "heartbeat/resource update interval in seconds")
	flag.Parse()

	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	c := lochness.NewContext(e)

	hn := os.Getenv("TEST_HOSTNAME")
	if hn == "" {
		var err error
		hn, err = os.Hostname()
		if err != err {
			log.Fatal(err)
		}
	}

	hv, err := c.Hypervisor(hn)
	if err != nil {
		log.Fatal(err)
	}

	for {
		if err = hv.UpdateResources(); err != nil {
			log.Println(err)
		}
		if err = hv.Heartbeat(time.Duration(*interval * 2)); err != nil {
			log.Println(err)
		}
		os.Stdout.WriteString("♥ ")
		os.Stdout.Sync()
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

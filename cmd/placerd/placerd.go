package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/queue"
)

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	interval := flag.Duration("interval", 30, "Interval in seconds to refresh lock")
	ttl := flag.Uint64("ttl", 0, "TTL for key in seconds, leave 0 to keep default (2 * interval)")
	dir := flag.String("queue", "/queue", "etcd directory to use for queue")
	eaddr := flag.String("etcd", "http://localhost:4001", "address of etcd machine")
	flag.Parse()

	if *ttl == 0 {
		*ttl = uint64(2 * (*interval))
	}

	e := etcd.NewClient([]string{*eaddr})

	hn := os.Getenv("TEST_HOSTNAME")
	if hn == "" {
		var err error
		hn, err = os.Hostname()
		if err != err {
			log.Fatal("unable to get hostname:", err)
		}
	}

	stop := make(chan bool)
	q, err := queue.Open(e, *dir, stop)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("received signal:", sig)
		stop <- true
	}()

	for job := range q.C {
		log.Println("got new job:", job)
	}
}

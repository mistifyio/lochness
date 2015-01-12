package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/queue"
)

type Job struct {
	Action string `json:"action"`
	Guest  string `json:"guest"`
}

type Agenter interface {
	Create(guestID string) error
	Delete(guestID string) error
	Start(guestID string) error
	Stop(guestID string) error
}

type FakeAgent struct{}

func (a *FakeAgent) Create(id string) error { return nil }
func (a *FakeAgent) Delete(id string) error { return nil }
func (a *FakeAgent) Start(id string) error  { return nil }
func (a *FakeAgent) Stop(id string) error   { return nil }

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

	log.Println("waiting for jobs")
	var agent Agenter = &FakeAgent{}
	for value := range q.C {
		job := Job{}
		err := json.Unmarshal([]byte(value), &job)
		if err != nil {
			log.Println(err)
			continue
		}
		log.Println("got new job:", job)
		switch job.Action {
		case "create":
			agent.Create(job.Guest)
		case "delete":
			agent.Delete(job.Guest)
		case "start":
			agent.Start(job.Guest)
		case "stop":
			agent.Stop(job.Guest)
		default:
			log.Println("invalid job", job)
		}
	}
}

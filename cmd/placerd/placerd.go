package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/queue"
)

type job struct {
	Action string `json:"action"`
	Guest  string `json:"guest"`
}

func create(agent lochness.Agenter, id string) error {
	_, err := agent.CreateGuest(id)
	return err
}

func del(agent lochness.Agenter, id string) error {
	_, err := agent.DeleteGuest(id)
	return err
}

func start(agent lochness.Agenter, id string) error {
	_, err := agent.GuestAction(id, "start")
	return err
}

func stop(agent lochness.Agenter, id string) error {
	_, err := agent.GuestAction(id, "shutdown")
	return err
}

func main() {
	log.SetFlags(log.Lshortfile | log.Ltime)

	interval := flag.Duration("interval", 30, "Interval in seconds to refresh lock")
	ttl := flag.Uint64("ttl", 0, "TTL for key in seconds, leave 0 to keep default (2 * interval)")
	prefix := flag.String("queue", "/queue", "etcd prefix to use as the queue")
	eaddr := flag.String("etcd", "http://localhost:4001", "address of etcd machine")
	mock := flag.Bool("mocked", true, "use fake agenter")
	failrate := flag.Int("fail-rate", 0, "percentage of api calls to artificially fail, ignored if mock==false")
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

	queueStop := make(chan bool)
	q, err := queue.Open(e, *prefix, queueStop)
	if err != nil {
		panic(err)
	}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGQUIT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Println("received signal:", sig)
		queueStop <- true
	}()

	ctx := lochness.NewContext(e)
	var agent lochness.Agenter
	if *mock == true {
		agent = ctx.NewAgentStubs(*failrate)
	} else {
		log.Fatal("no non-mocked agenters yet")
	}

	for queued := range q.Requests {
		job := job{}
		err := json.Unmarshal([]byte(queued.Request), &job)
		if err != nil {
			log.Println(err)
			continue
		}
		switch job.Action {
		case "create":
			err = create(agent, job.Guest)
		case "delete":
			err = del(agent, job.Guest)
		case "start":
			err = start(agent, job.Guest)
		case "stop":
			err = stop(agent, job.Guest)
		default:
			err = errors.New("unknown job type")
		}
		resp := "ok"
		if err != nil {
			resp = err.Error()
		}
		queued.Response = resp
		q.Responses <- queued
	}
}

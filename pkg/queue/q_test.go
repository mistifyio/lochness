package queue

// Not really any tests, just broke out my dirty hacking into lib and extra

import (
	"encoding/json"
	"log"
	"testing"

	"code.google.com/p/go-uuid/uuid"

	"github.com/coreos/go-etcd/etcd"
)

var c = etcd.NewClient([]string{"http://localhost:4001"})

func init() {
	log.SetFlags(log.Lshortfile | log.Ldate)
}

type q struct {
	*Q
	k string
}

func newQNamed(t *testing.T, name string, stop chan bool) q {
	_q, err := Open(c, name, stop)
	if err != nil && err != etcd.ErrWatchStoppedByUser {
		t.Fatal(err)
	}

	return q{k: name, Q: _q}
}

func newQ(t *testing.T, stop chan bool) q {
	return newQNamed(t, "/test/"+uuid.New(), stop)
}

func delQ(t *testing.T, q q, recursive bool) {
	if recursive {
		if _, err := c.Delete(q.k, recursive); err != nil {
			t.Fatal(err)
		}
	} else {
		if _, err := c.DeleteDir(q.k); err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenClose(t *testing.T) {
	q := newQ(t, nil)
	defer delQ(t, q, false)

	_, err := c.CreateDir(q.k, 0)
	if err == nil || !isKeyExists(err) {
		t.Fatal("expected a KeyExists error, got:", err)
	}

	_, err = q.c.Get(q.k, true, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestStopPrePoll(t *testing.T) {
	stop := make(chan bool, 1)
	stop <- true
	defer close(stop)

	q := newQ(t, stop)
	defer delQ(t, q, false)

	v, ok := <-q.Requests
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}
}

func handler(reqs <-chan Job, resps chan<- Job) {
	for req := range reqs {
		req.Response = req.Request + req.Request
		resps <- req
	}
}

func TestPut(t *testing.T) {
	q := newQ(t, nil)
	defer delQ(t, q, true)

	go handler(q.Requests, q.Responses)

	conn := Connect(c, q.k)

	vals := []string{"1", "2"}
	for i := range vals {
		resp, err := conn.Put(vals[i])
		if err != nil {
			t.Fatal(err)
		}
		if resp != vals[i]+vals[i] {
			t.Fatal("wanted:", vals[i]+vals[i], "got:", resp)
		}
	}

	// queue should be drained
	resp, err := c.Get(q.k, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Node.Nodes) != 0 {
		t.Fatal("queue should be drained, found", len(resp.Node.Nodes), "nodes")
	}
}

func TestPoll(t *testing.T) {
	n := "/test/" + uuid.New()
	vals := map[string]struct{}{
		"1": struct{}{},
		"2": struct{}{},
		"3": struct{}{},
		"4": struct{}{},
		"5": struct{}{},
	}
	for k := range vals {
		req := Job{Request: k}
		data, err := json.Marshal(&req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.CreateInOrder(n, string(data), 0); err != nil {
			t.Fatal(err)
		}
	}

	stop := make(chan bool)
	q := newQNamed(t, n, stop)
	defer delQ(t, q, true)

	i := 0
	for resp := range q.Requests {
		i++
		if _, ok := vals[resp.Request]; !ok {
			t.Fatal("unknown request:", resp.Request)
		}
		if i == len(vals) {
			stop <- true
		}
	}
	v, ok := <-q.Requests
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}
}

func TestStopMidPoll(t *testing.T) {
	n := "/test/" + uuid.New()
	vals := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	for i := range vals {
		req := Job{Request: vals[i]}
		data, err := json.Marshal(&req)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.CreateInOrder(n, string(data), 0); err != nil {
			t.Fatal(err)
		}
	}

	stop := make(chan bool)
	q := newQNamed(t, n, stop)
	defer delQ(t, q, true)

	i := 0
	for v := range q.Requests {
		if vals[i] != v.Request {
			t.Fatal("wanted:", vals[i], "v:", v.Request)
		}
		if i == 0 {
			// done here so we are sure to not go further than poll
			go handler(q.Requests, q.Responses)
			stop <- true
		}
		i++
		v.Response = v.Request
		q.Responses <- v
	}
	if i == len(vals) {
		t.Fatal("stopped after poll finished")
	}

	v, ok := <-q.Requests
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}
}

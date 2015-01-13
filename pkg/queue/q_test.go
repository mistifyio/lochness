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
	if err != nil && err != ErrStopped {
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
	stop := make(chan bool)
	close(stop)

	q := newQ(t, stop)
	defer delQ(t, q, false)

	v, ok := <-q.C
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}
}

func TestPut(t *testing.T) {
	q := newQ(t, nil)
	defer delQ(t, q, true)

	go func() {
		for v := range q.C {
			q.C <- v + v
		}
	}()

	conn := Connect(c, q.k)

	vals := []string{"1", "2"}
	for i := range vals {
		_, err := conn.Put(vals[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	resp, err := c.Get(q.k, true, true)
	if err != nil {
		t.Fatal(err)
	}
	i := 0
	for _, node := range resp.Node.Nodes {
		if node.Value != vals[i] {
			t.Fatal("wanted:", vals[i], "got:", node.Value)
		}
		i++
	}
}

func TestPoll(t *testing.T) {
	n := "/test/" + uuid.New()
	vals := []string{"1", "2", "3", "4", "5"}
	for i := range vals {
		req := queued{Request: vals[i]}
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

	for i := 0; i < len(vals); i++ {
		got := <-q.C
		if vals[i] != got {
			t.Fatal("wanted:", vals[i], "got:", got)
		}
		q.C <- got
	}
	close(stop)
	v, ok := <-q.C
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}
}

func TestStopMidPoll(t *testing.T) {
	n := "/test/" + uuid.New()
	vals := []string{"1", "2", "3", "4", "5"}
	for i := range vals {
		req := queued{Request: vals[i]}
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
	for v := range q.C {
		if vals[i] != v {
			t.Fatal("wanted:", vals[i], "v:", v)
		}
		i++
		q.C <- v
		close(stop)
	}
	if i != 1 {
		t.Fatal("wanted 1 value, got:", i)
	}

	v, ok := <-q.C
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}
}

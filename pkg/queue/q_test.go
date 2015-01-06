package queue

// Not really any tests, just broke out my dirty hacking into lib and extra

import (
	"testing"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/lock"
)

var c = etcd.NewClient([]string{"http://localhost:4001"})

type q struct {
	*Q
	k string
}

func newQNamed(t *testing.T, name string, stop chan bool) *q {
	_q, err := Open(c, name, stop)
	if err != nil && err != ErrStopped {
		t.Fatal(err)
	}

	return &q{k: name, Q: _q}
}

func newQ(t *testing.T, stop chan bool) *q {
	return newQNamed(t, "/test/"+uuid.New(), stop)
}

func delQ(t *testing.T, q *q, recursive bool) {
	if err := q.Close(); err != nil && err != lock.ErrLockNotHeld {
		t.Fatal(err)
	}
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

	_, err := c.Get(q.k+"/lock", false, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := q.Close(); err != nil {
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

	vals := []string{"1", "2"}
	for i := range vals {
		if err := q.Put(vals[i]); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := c.Get(q.k, true, true)
	if err != nil {
		t.Fatal(err)
	}
	i := 0
	for _, node := range resp.Node.Nodes {
		if node.Key == q.k+"/lock" {
			continue
		}
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
		if _, err := c.CreateInOrder(n, vals[i], 0); err != nil {
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
	}
	close(stop)
	v, ok := <-q.C
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}

	// allow the last delete to work, 3ms seems to be the edge
	time.Sleep(6 * time.Millisecond)
	resp, err := c.Get(q.k, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range resp.Node.Nodes {
		if node.Key == q.k+"/lock" {
			continue
		}
		t.Fatal("unexpected job (extra or not deleted):", node.Value)
	}
}

func TestWatch(t *testing.T) {
	stop := make(chan bool)
	q := newQ(t, stop)
	defer delQ(t, q, true)

	vals := []string{"1", "2", "3", "4", "5"}
	for i := range vals {
		q.Put(vals[i])
	}

	for i := 0; i < len(vals); i++ {
		got := <-q.C
		if vals[i] != got {
			t.Fatal("wanted:", vals[i], "got:", got)
		}
	}
	close(stop)
	v, ok := <-q.C
	if ok {
		t.Fatal("unexpected chan receive:", v, ok)
	}

	// allow the last delete to work, 3ms seems to be the edge
	time.Sleep(6 * time.Millisecond)
	resp, err := c.Get(q.k, true, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range resp.Node.Nodes {
		if node.Key == q.k+"/lock" {
			continue
		}
		t.Fatal("unexpected job (extra or not deleted):", node.Value)
	}
}

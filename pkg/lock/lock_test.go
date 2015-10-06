package lock

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/pborman/uuid"

	"github.com/coreos/go-etcd/etcd"
)

func newClient(t *testing.T) *etcd.Client {
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	if !e.SyncCluster() {
		t.Fatal("cannot sync cluster. make sure etcd is running at http://127.0.0.1:4001")
	}
	return e
}

func TestAcquire(t *testing.T) {
	t.Parallel()

	key := "lock-TestAcquire"
	id := uuid.New()
	c := newClient(t)
	defer cleanup(c, key)

	_, err := Acquire(c, key, id, 60, false)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Get(key, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Node.Value != id {
		t.Fatalf("wanted: %s, got: %s\n", id, resp.Node.Value)
	}
}

func TestAcquireExists(t *testing.T) {
	t.Parallel()

	key := "lock-TestAcquireExists"
	id := uuid.New()
	c := newClient(t)
	defer cleanup(c, key)

	_, err := Acquire(c, key, id, 60, false)
	if err != nil {
		t.Fatal(err)
	}

	l, err := Acquire(c, key, id, 60, false)
	if err == nil {
		t.Fatal("expected a non-nil error, got:", err, l)
	}
}

func TestAcquireExistsWait(t *testing.T) {
	t.Parallel()

	key := "lock-TestAcquireExistsWait"
	id1 := uuid.New()
	id2 := uuid.New()
	ttl := uint64(2)
	c := newClient(t)
	defer cleanup(c, key)

	_, err := Acquire(c, key, id1, ttl, false)
	if err != nil {
		t.Fatal(err)
	}

	tstart := time.Now().Unix()
	_, err = Acquire(c, key, id2, ttl, true)
	if err != nil {
		t.Fatal(err)
	}
	tstop := time.Now().Unix()
	if uint64(tstop-tstart) < ttl-1 {
		t.Fatalf("expected atleast %ds(ttl-1)  wait time, got: %d\n", ttl-1, tstop-tstart)
	}

	resp, err := c.Get(key, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Node.Value != id2 {
		t.Fatalf("incorrect data in lock, wanted: %s, got: %s\n", id2, resp.Node.Value)
	}
}

func TestRefresh(t *testing.T) {
	t.Parallel()

	key := "lock-TestRefresh"
	id := uuid.New()
	ttl := uint64(2)
	c := newClient(t)
	defer cleanup(c, key)

	l, err := Acquire(c, key, id, ttl, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := l.Refresh(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Duration(ttl) * time.Second)
	if err := l.Refresh(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Duration(ttl+1) * time.Second)
	if err := l.Refresh(); err != ErrKeyNotFound {
		t.Fatalf("wanted: %v, got: %v\n", ErrKeyNotFound, err)
	}

	if err := l.Refresh(); err != ErrLockNotHeld {
		t.Fatalf("wanted: %v, got: %v\n", ErrLockNotHeld, err)
	}
}

func TestRelease(t *testing.T) {
	t.Parallel()

	key := "lock-TestRelease"
	id := uuid.New()
	ttl := uint64(2)
	c := newClient(t)
	defer cleanup(c, key)

	l := &Lock{}
	if err := l.Release(); err != ErrLockNotHeld {
		t.Fatalf("wanted: %v, got: %v\n", ErrLockNotHeld, err)
	}

	l, err := Acquire(c, key, id, ttl, false)
	if err != nil {
		t.Fatal(err)
	}

	if err := l.Release(); err != nil {
		t.Fatal(err)
	}

	l, err = Acquire(c, key, id, ttl, false)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Duration(ttl+1) * time.Second)
	if err := l.Release(); err != ErrKeyNotFound {
		t.Fatalf("wanted: %v, got: %v\n", ErrKeyNotFound, err)
	}

	if err := l.Release(); err != ErrLockNotHeld {
		t.Fatalf("wanted: %v, got: %v\n", ErrLockNotHeld, err)
	}
}

func TestJSON(t *testing.T) {
	t.Parallel()

	key := "lock-TestJSON"
	id := uuid.New()
	c := newClient(t)
	defer cleanup(c, key)

	l, err := Acquire(c, key, id, 60, false)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(&l)
	if err != nil {
		t.Fatal(err)
	}

	l2 := &Lock{}
	err = json.Unmarshal(data, l2)
	if err != nil {
		t.Fatal(err)
	}

	if l2.c == nil {
		t.Fatal("l2.c should not be nil")
	}

	match := false
LOOP:
	for _, h := range l.c.GetCluster() {
		for _, h2 := range l2.c.GetCluster() {
			if h == h2 {
				match = true
				break LOOP
			}
		}
	}
	if !match {
		t.Fatalf("could not find a matching host in clusters, wanted: %v, got: %v\n",
			l.c.GetCluster, l2.c.GetCluster())
	}
	l2.c = l.c
	if !reflect.DeepEqual(*l, *l2) {
		t.Fatalf("lock mismatch\nwanted: %#v\n   got: %#v\n", l, l2)
	}
}

func cleanup(c *etcd.Client, key string) {
	if _, err := c.Delete(key, true); err != nil && !isKeyNotFound(err) {
		fmt.Printf("Unable to delete key '%s': %+v", key, err)
	}
}

package watcher

import (
	"strconv"
	"testing"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

func newClient(t *testing.T) *etcd.Client {
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	if !e.SyncCluster() {
		t.Fatal("cannot sync cluster. make sure etcd is running at http://127.0.0.1:4001")
	}
	return e
}

func populate(t *testing.T, w *Watcher, prefixes ...string) {
	for _, prefix := range prefixes {
		err := w.Add(prefix)
		if err != nil {
			t.Fatalf("error adding prefix: %s, err: %s", prefix, err)
		}
	}
}

func clean(t *testing.T, w *Watcher, prefixes ...string) {
	for _, prefix := range prefixes {
		err := w.Remove(prefix)
		if err != nil && err != ErrPrefixNotWatched {
			t.Fatalf("error removing prefix: %s, err: %s", prefix, err)
		}
	}
}

func TestAdd(t *testing.T) {
	e := newClient(t)
	w, err := New(e)
	if err != nil {
		t.Fatal(err)
	}
	defer clean(t, w, "/prefix", "/another-prefix", "/-prefix")

	var prefixes = []struct {
		prefix string
		count  int
	}{
		{"/prefix", 1},
		{"/prefix", 1},
		{"/another-prefix", 2},
		{"/another-prefix", 2},
		{"/another-prefix", 2},
		{"/-prefix", 3},
	}

	for _, test := range prefixes {
		err = w.Add(test.prefix)
		if err != nil {
			t.Fatalf("error adding prefix: %s, err: %s", test.prefix, err)
		}
		if len(w.prefixes) != test.count {
			t.Fatalf("unexpected number of watched prefixes, got:%d, wanted:%d",
				len(w.prefixes), test.count)
		}
	}

	if err = w.Close(); err != nil {
		t.Fatal("unexepected error:", err)
	}

	if err = w.Add("/prefix2"); err != ErrStopped {
		t.Fatalf("unexpected error, wanted: %v, got: %v", ErrStopped, err)
	}
}

func TestRemove(t *testing.T) {
	e := newClient(t)
	w, err := New(e)
	if err != nil {
		t.Fatal(err)
	}
	populate(t, w, "/prefix", "/another-prefix", "/-prefix")
	defer clean(t, w, "/prefix", "/another-prefix", "/-prefix")

	var prefixes = []struct {
		prefix string
		err    error
		count  int
	}{
		{"some-non-existant-prefix", ErrPrefixNotWatched, 3},
		{"/-prefix", nil, 2},
		{"/another-prefix", nil, 1},
		{"/another-prefix", ErrPrefixNotWatched, 1},
		{"/another-prefix", ErrPrefixNotWatched, 1},
		{"/prefix", nil, 0},
		{"/prefix", ErrPrefixNotWatched, 0},
	}

	for _, test := range prefixes {
		err := w.Remove(test.prefix)
		if err != test.err {
			t.Fatalf("unexpected error, wanted: %v, got: %v", test.err, err)
		}
		if len(w.prefixes) != test.count {
			t.Fatalf("unexpected number of watched prefixes, wanted: %d, got: %d",
				test.count, len(w.prefixes))
		}
	}
}

func TestClose(t *testing.T) {
	e := newClient(t)
	w, err := New(e)
	if err != nil {
		t.Fatal(err)
	}
	populate(t, w, "/prefix", "/another-prefix", "/-prefix")
	defer clean(t, w, "/prefix", "/another-prefix", "/-prefix")

	want := 3
	if len(w.prefixes) != want {
		t.Fatalf("unexpected number of watched prefixes, wanted: %d, got: %d",
			want, len(w.prefixes))
	}

	if w.isClosed {
		t.Fatal("isClosed should be false")
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	want = 0
	if len(w.prefixes) != want {
		t.Fatalf("unexpected number of watched prefixes, wanted: %d, got: %d",
			want, len(w.prefixes))
	}

	if !w.isClosed {
		t.Fatal("isClosed should be true")
	}
}

func TestNext(t *testing.T) {
	e := newClient(t)
	w, err := New(e)
	if err != nil {
		t.Fatal(err)
	}
	prefixes := []string{"/prefix", "/another-prefix", "/-prefix"}
	populate(t, w, prefixes...)
	defer clean(t, w, prefixes...)

	for _, prefix := range prefixes {
		err := w.Add(prefix)
		if err != nil {
			t.Fatal(err)
		}
	}

	time.Sleep(200 * time.Microsecond)
	go func() {
		for _, prefix := range prefixes {
			go func(prefix string) {
				_, err := e.Set(prefix+"/blah", "blah", 0)
				if err != nil {
					t.Fatal(err)
				}
			}(prefix)
			go func(prefix string) {
				_, err := e.Set(prefix+"/blah", "blah2", 0)
				if err != nil {
					t.Fatal(err)
				}
			}(prefix)
		}
	}()

	i := 0
	for w.Next() {
		i++
		if i == len(prefixes)*2 {
			break
		}
	}
}

func TestResponse(t *testing.T) {
	e := newClient(t)
	w, err := New(e)
	if err != nil {
		t.Fatal(err)
	}
	prefixes := []string{"/prefix", "/another-prefix", "/-prefix"}
	populate(t, w, prefixes...)
	defer clean(t, w, prefixes...)

	for _, prefix := range prefixes {
		err := w.Add(prefix)
		if err != nil {
			t.Fatal(err)
		}
	}

	num := 1024
	time.Sleep(200 * time.Microsecond)
	go func() {
		for i := 0; i < num; i++ {
			node := strconv.Itoa(i % 10)
			val := strconv.Itoa(i)
			_, err := e.Set("/prefix/"+node, val, 0)
			if err != nil {
				t.Fatal(err)
			}
		}
	}()

	i := 0
	for w.Next() {
		resp := w.Response()
		if resp.Node.Value != strconv.Itoa(i) {
			t.Fatal("unexpected value, want: %d, got: %s", i, resp.Node.Value)
		}
		i++
		if i == num {
			break
		}
	}
}

package watcher

import (
	"errors"
	"sync"

	"github.com/coreos/go-etcd/etcd"
)

var ErrPrefixNotWatched = errors.New("prefix is not being watched")
var ErrStopped = errors.New("watcher has been stopped")

type Watcher struct {
	c         *etcd.Client
	responses chan *etcd.Response
	errors    chan error
	err       error
	response  *etcd.Response

	mu       sync.Mutex // mu protects the following two vars
	isClosed bool
	prefixes map[string]chan bool
}

func New(c *etcd.Client) (*Watcher, error) {
	w := &Watcher{
		responses: make(chan *etcd.Response),
		errors:    make(chan error),
		c:         c,
		prefixes:  map[string]chan bool{},
	}
	return w, nil
}

func (w *Watcher) Add(prefix string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isClosed {
		return ErrStopped
	}

	_, ok := w.prefixes[prefix]
	if ok {
		return nil
	}

	ch := make(chan bool)
	w.prefixes[prefix] = ch
	go w.watch(prefix, ch)
	return nil
}

func (w *Watcher) Next() bool {
	select {
	case resp := <-w.responses:
		w.response = resp
		return true
	case err := <-w.errors:
		w.err = err
		return false
	}
}

func (w *Watcher) Response() *etcd.Response {
	return w.response
}

func (w *Watcher) Err() error {
	return w.err
}

func (w *Watcher) Remove(prefix string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	ch, ok := w.prefixes[prefix]
	if !ok {
		return ErrPrefixNotWatched
	}

	close(ch)
	delete(w.prefixes, prefix)
	return nil
}

func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isClosed = true

	for prefix, ch := range w.prefixes {
		close(ch)
		delete(w.prefixes, prefix)
	}

	return nil
}

func (w *Watcher) watch(prefix string, stop chan bool) {
	for {
		responses := make(chan *etcd.Response)
		go func() {
			for resp := range responses {
				w.responses <- resp
			}
		}()

		_, err := w.c.Watch(prefix, 0, true, responses, stop)
		if err == nil || err == etcd.ErrWatchStoppedByUser {
			break
		}
		if err != nil {
			w.errors <- err
		}
	}
}

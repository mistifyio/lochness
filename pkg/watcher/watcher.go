package watcher

import (
	"errors"
	"sync"

	"github.com/coreos/go-etcd/etcd"
)

type Watcher struct {
	c        *etcd.Client
	events   chan *etcd.Response
	errors   chan error
	err      error
	response *etcd.Response

	mu       *sync.Mutex // mu protects the following two vars
	isClosed bool
	prefixes map[string]chan bool
}

func New(c *etcd.Client) (*Watcher, error) {
	w := &Watcher{
		events:   make(chan *etcd.Response),
		errors:   make(chan error),
		c:        c,
		mu:       &sync.Mutex{},
		prefixes: map[string]chan bool{},
	}
	return w, nil
}

func (w *Watcher) Add(prefix string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isClosed {
		return errors.New("watcher has been shutdown")
	}

	_, ok := w.prefixes[prefix]
	if ok {
		return nil
	}

	ch := make(chan bool)
	go w.watch(prefix, ch)
	return nil
}

func (w *Watcher) Next() bool {
	select {
	case resp := <-w.events:
		w.response = resp
		return true
	default:
		select {
		case resp := <-w.events:
			w.response = resp
			return true
		case err := <-w.errors:
			w.err = err
			return false
		}
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
		return errors.New("prefix is not being watched")
	}

	close(ch)
	return nil
}

func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isClosed = true

	for _, ch := range w.prefixes {
		close(ch)
	}

	return nil
}

func (w *Watcher) watch(prefix string, stop chan bool) {
	for {
		responses := make(chan *etcd.Response)
		go func() {
			for resp := range responses {
				w.events <- resp
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

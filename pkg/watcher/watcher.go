// Package watcher provides etcd prefix watching capabilities.
package watcher

import (
	"errors"
	"sync"

	"github.com/coreos/go-etcd/etcd"
)

// ErrPrefixNotWatched is an error for attempting to remove an unwatched prefix
var ErrPrefixNotWatched = errors.New("prefix is not being watched")

// ErrStopped is an error for attempting to add a prefix to a stopped watcher
var ErrStopped = errors.New("watcher has been stopped")

// Watcher monitors etcd prefixes and notifies on change
type Watcher struct {
	c         *etcd.Client
	responses chan *etcd.Response
	errors    chan *Error
	err       *Error
	response  *etcd.Response

	mu       sync.Mutex // mu protects the following two vars
	isClosed bool
	prefixes map[string]chan bool
}

type Error struct {
	Prefix string
	Err    error
}

func (e *Error) Error() string {
	return e.Err.Error()
}

// New creates a new Watcher
func New(c *etcd.Client) (*Watcher, error) {
	if c == nil {
		return nil, errors.New("missing etcd client")
	}
	w := &Watcher{
		responses: make(chan *etcd.Response),
		errors:    make(chan *Error),
		c:         c,
		prefixes:  map[string]chan bool{},
	}
	return w, nil
}

// Add will add prefix to the watch list, there still may be a short time (<500us)
// after Add returns when an event on prefix may be missed.
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

// Next blocks until an event has been received by any of the wathed prefixes.
// The event it self may be accesed via the Response method. False will be
// returned upon an error, the error can be retrieved via the Err method.
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

// Response returns the response received that caused Next to return.
func (w *Watcher) Response() *etcd.Response {
	return w.response
}

// Err returns the last error received
func (w *Watcher) Err() *Error {
	return w.err
}

// Remove will remove said prefix from the watch list, it will return an error
// if the prefix is not being watched.
func (w *Watcher) Remove(prefix string) *Error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.removeLocked(prefix)
}

func (w *Watcher) removeLocked(prefix string) *Error {
	ch, ok := w.prefixes[prefix]
	if !ok {
		return &Error{Prefix: prefix, Err: ErrPrefixNotWatched}
	}

	// Stop the etcd prefix watch
	close(ch)

	// Remove the prefix
	delete(w.prefixes, prefix)
	return nil
}

// Close will stop all watches and disable any new watches from being started.
// Close may be called multiple times in case there is a transient error.
func (w *Watcher) Close() *Error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isClosed = true

	for prefix := range w.prefixes {
		if err := w.removeLocked(prefix); err != nil {
			return err
		}
	}

	return nil
}

func (w *Watcher) watch(prefix string, stop chan bool) {
	defer w.Remove(prefix)

	// Get the index to start watching from.
	// This minimizes the window between calling etcd.Watch() and the watch
	// actually starting where changes can be missed.  Since etcd.Watch()
	// itself blocks, we have no direct way of knowing when the watch actually
	// starts, so this is the best we can do.
	waitIndex := uint64(0)
	resp, err := w.c.Get("/", false, false)
	if err == nil {
		waitIndex = resp.EtcdIndex
	}

	responses := make(chan *etcd.Response)
	go func() {
		for resp := range responses {
			w.responses <- resp
		}
	}()

	_, err = w.c.Watch(prefix, waitIndex, true, responses, stop)
	if err == nil || err == etcd.ErrWatchStoppedByUser {
		return
	}
	w.errors <- &Error{Prefix: prefix, Err: err}
}

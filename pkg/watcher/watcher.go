// Package watcher provides kv prefix watching capabilities.
package watcher

import (
	"errors"
	"strings"
	"sync"

	"github.com/mistifyio/lochness/pkg/kv"
)

// ErrPrefixNotWatched is an error for attempting to remove an unwatched prefix
var ErrPrefixNotWatched = errors.New("prefix is not being watched")

// ErrStopped is an error for attempting to add a prefix to a stopped watcher
var ErrStopped = errors.New("watcher has been stopped")

// Watcher monitors kv prefixes and notifies on change
type Watcher struct {
	kv     kv.KV
	events chan kv.Event
	errors chan *Error
	err    *Error
	event  kv.Event

	mu       sync.Mutex // mu protects the following two vars
	isClosed bool
	prefixes map[string]chan struct{}
}

// Error contains both the watched prefix and the error.
type Error struct {
	Prefix string
	Err    error
}

func (e *Error) Error() string {
	return e.Err.Error()
}

// New creates a new Watcher
func New(KV kv.KV) (*Watcher, error) {
	if KV == nil {
		return nil, errors.New("kv instance must be non-nil")
	}
	w := &Watcher{
		events:   make(chan kv.Event),
		errors:   make(chan *Error),
		kv:       KV,
		prefixes: map[string]chan struct{}{},
	}
	return w, nil
}

// Add will add prefix to the watch list, there may still be a short time (<500us) after Add returns when an event on prefix may be missed.
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

	ch := make(chan struct{})
	w.prefixes[prefix] = ch
	go w.watch(prefix, ch)
	return nil
}

// Next blocks until an event has been received by any of the watched prefixes.
// The event itself may be accessed via the Response method.
// If an error is encountered false will be returned, the error can be retrieved via the Err method.
func (w *Watcher) Next() bool {
	select {
	case event := <-w.events:
		w.event = event
		return true
	case err := <-w.errors:
		w.err = err
		return false
	}
}

// Event returns the event received that caused Next to return.
func (w *Watcher) Event() kv.Event {
	return w.event
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

	// Stop the kv prefix watch
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

func getLatestIndex(kv kv.KV, prefix string) uint64 {
	resp, err := kv.Get(prefix)
	if err == nil {
		return resp.Index
	} else if !strings.Contains(err.Error(), "directory") {
		return 0
	}

	keys, err := kv.Keys(prefix)
	if err != nil || len(keys) == 0 {
		return 0
	}

	latest := uint64(0)
	for _, key := range keys {
		resp, err := kv.Get(key)
		if err != nil {
			continue
		}
		if resp.Index > latest {
			latest = resp.Index
		}
	}
	return latest
}

func (w *Watcher) watch(prefix string, stop chan struct{}) {
	defer func() {
		_ = w.Remove(prefix)
	}()

	// Get the index to start watching from.
	// This minimizes the window between calling kv.Watch() and the watch actually starting where changes can be missed.
	// Since kv.Watch() itself blocks, we have no direct way of knowing when the watch actually starts, so this is the best we can do.
	waitIndex := getLatestIndex(w.kv, prefix)

	events, errors, err := w.kv.Watch(prefix, waitIndex, stop)
	if err != nil {
		w.errors <- &Error{Prefix: prefix, Err: err}
		return
	}

LOOP:
	for {
		select {
		case event, ok := <-events:
			if !ok {
				close(stop)
				break LOOP
			}
			w.events <- event
		case err, ok := <-errors:
			if !ok {
				close(stop)
				break LOOP
			}
			w.errors <- &Error{Prefix: prefix, Err: err}
		case <-stop:
			break LOOP
		}
	}

}

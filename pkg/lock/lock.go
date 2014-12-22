// Package lock implements a lock in an etcd cluster using CAS semantics.
package lock

import (
	"errors"

	etcdErr "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

var (
	// ErrKeyNotFound signifies an attempt to operate on a non-existent lock
	ErrKeyNotFound = errors.New("Key not found")
	// ErrLockNotHeld signifies an attempt to operate on a released/lost lock
	ErrLockNotHeld = errors.New("Lock not held")
)

// Lock is a lock in etcd
type Lock struct {
	c     *etcd.Client
	key   string
	value string
	ttl   uint64
	index uint64
	held  bool
}

func acquire(c *etcd.Client, key, value string, ttl uint64) (uint64, error) {
	resp, err := c.Create(key, value, ttl)
	if err != nil {
		return 0, err
	}
	return resp.EtcdIndex, nil
}

//Acquire will attempt to acquire the lock, if blocking is set to true it will wait forever to do so.
//Setting blocking to false would be the equivalent of a fictional TryAcquire, an immediate return
//if locking fails.
func Acquire(c *etcd.Client, key, value string, ttl uint64, blocking bool) (*Lock, error) {
	index := uint64(0)
	var err error
	tryAcquire := true
LOOP:
	for {
		if tryAcquire {
			index, err = acquire(c, key, value, ttl)
			if err == nil {
				break LOOP
			}
			if !blocking {
				return nil, err
			}
			tryAcquire = false
		}
		resp, err := c.Watch(key, 0, false, nil, nil)
		if err != nil {
			return nil, err
		}
		if resp.Action != "compareAndSwap" {
			tryAcquire = true
		}
	}
	return &Lock{
		c:     c,
		key:   key,
		value: value,
		ttl:   ttl,
		index: index,
		held:  true,
	}, nil
}

//Refresh will refresh the lock. An error is returned if the lock was lost, likely due ttl expiration
func (l *Lock) Refresh() error {
	if !l.held {
		return ErrLockNotHeld
	}

	resp, err := l.c.CompareAndSwap(l.key, l.value, l.ttl, l.value, l.index)
	if err != nil {
		if isKeyNotFound(err) {
			err = ErrKeyNotFound
		}
		l.held = false
		return err
	}
	l.index = resp.EtcdIndex
	return nil
}

//Release will release the lock and delete the key
func (l *Lock) Release() error {
	if !l.held {
		return ErrLockNotHeld
	}
	_, err := l.c.CompareAndDelete(l.key, l.value, l.index)
	if err != nil && isKeyNotFound(err) {
		err = ErrKeyNotFound
	}
	l.held = false
	return err
}

func isKeyNotFound(err error) bool {
	e, ok := err.(*etcd.EtcdError)
	return ok && e.ErrorCode == etcdErr.EcodeKeyNotFound
}

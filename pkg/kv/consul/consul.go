package consul

import (
	"errors"
	"net/url"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
	"github.com/mistifyio/lochness/pkg/kv"
)

var err404 = errors.New("key not found")

func init() {
	kv.Register("consul", New)
}

type ckv struct {
	c      *consul.KV
	client *consul.Client
	config *consul.Config
}

// New instantiates a consul kv implementation.
// The parameter addr may be the empty string or a valid URL.
// If addr is not empty it must be a valid URL with schemes http, https or consul; consul is synonymous with http.
// If addr is the empty string the consul client will connect to the default address, which may be influenced by the environment.
func New(addr string) (kv.KV, error) {
	config := consul.DefaultConfig()
	if addr == "" {
		addr = config.Scheme + "://" + config.Address
	} else {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}

		if u.Scheme != "consul" {
			config.Scheme = u.Scheme
		}
		config.Address = u.Host
	}

	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &ckv{c: client.KV(), client: client, config: config}, nil
}

func (c *ckv) Delete(key string, recurse bool) error {
	var err error
	if recurse {
		_, err = c.c.DeleteTree(key, nil)
	} else {
		_, err = c.c.Delete(key, nil)
	}
	return err
}

func (c *ckv) Get(key string) (kv.Value, error) {
	kvp, _, err := c.c.Get(key, nil)
	if err != nil {
		return kv.Value{nil, 0}, err
	}
	if kvp == nil || kvp.Value == nil {
		return kv.Value{nil, 0}, err404
	}
	return kv.Value{Data: kvp.Value, Index: kvp.ModifyIndex}, nil
}

func (c *ckv) GetAll(prefix string) (map[string]kv.Value, error) {
	pairs, _, err := c.c.List(prefix, nil)
	if err != nil {
		return nil, err
	}
	many := make(map[string]kv.Value, len(pairs))
	for _, kvp := range pairs {
		many[kvp.Key] = kv.Value{Data: kvp.Value, Index: kvp.ModifyIndex}
	}
	return many, nil
}

func (c *ckv) Keys(key string) ([]string, error) {
	keys, _, err := c.c.Keys(key, "/", nil)
	return keys, err
}

func (c *ckv) Set(key, value string) error {
	_, err := c.c.Put(&consul.KVPair{Key: key, Value: []byte(value)}, nil)
	return err
}

func (c *ckv) cas(key string, value kv.Value) error {
	kvp := consul.KVPair{
		Key:         key,
		Value:       value.Data,
		ModifyIndex: value.Index,
	}

	valid, _, err := c.c.CAS(&kvp, nil)
	if err != nil {
		return err
	}

	if !valid {
		// TODO(mm) better error
		return errors.New("CAS failed")
	}

	return nil
}

// Update is racy with other modifiers since the consul KV API does not return the new modified index.
// See https://github.com/hashicorp/consul/issues/304
func (c *ckv) Update(key string, value kv.Value) (uint64, error) {
	// TODO(mmlb): setup a key watch before this update to get the new modified index
	// can be if cas works then the watched index returned is valid
	err := c.cas(key, value)
	if err != nil {
		return 0, err
	}

	v, err := c.Get(key)
	return v.Index, err
}

func (c *ckv) Remove(key string, index uint64) error {
	ok, _, err := c.c.DeleteCAS(&consul.KVPair{Key: key, ModifyIndex: index}, nil)
	if err != nil {
		return err
	}

	if !ok {
		err = errors.New("failed to delete atomically")
	}

	return err
}

func (c *ckv) IsKeyNotFound(err error) bool {
	return err == err404
}

func (c *ckv) Watch(prefix string, index uint64, stop chan struct{}) (chan kv.Event, chan error, error) {
	wp, err := watch.Parse(map[string]interface{}{
		"type":   "keyprefix",
		"prefix": prefix,
	})
	if err != nil {
		return nil, nil, err
	}

	events := make(chan kv.Event)
	errs := make(chan error)

	saved := map[string]uint64{}
	wp.Handler = func(index uint64, data interface{}) {
		new := map[string]uint64{}

		for _, kvp := range data.(consul.KVPairs) {
			new[kvp.Key] = kvp.ModifyIndex

			event := kv.Event{
				Key: kvp.Key,
				Value: kv.Value{
					Data:  kvp.Value,
					Index: kvp.ModifyIndex,
				},
			}

			old, ok := saved[kvp.Key]
			switch {
			case !ok:
				// doesn't exist in saved so must be created
				event.Type = kv.Create
			case old != kvp.ModifyIndex:
				// mod indexes differ so must be changed
				event.Type = kv.Update
			}
			events <- event

			// saved[kvp.Key] won't exist for "create" events, but
			// overhead is probably negligible and ignoring it makes
			// the code easier to read
			delete(saved, kvp.Key)
		}

		// anything left over in "saved" has not been found in "new"
		// so it must have been deleted
		for key, index := range saved {
			events <- kv.Event{
				Key:  key,
				Type: kv.Update,
				Value: kv.Value{
					Index: index,
				},
			}
		}

		saved = new
	}

	go func() {
		<-stop
		wp.Stop()
	}()
	go func() {
		err = wp.Run(c.config.Address)
		if err != nil {
			errs <- err
		}
	}()

	return events, errs, nil
}

type lock struct {
	sessions *consul.Session
	kv       *consul.KV
	session  string
	key      string
}

func (c *ckv) lock(key string, ttl time.Duration, behavior string) (string, error) {
	sEntry := &consul.SessionEntry{
		TTL:      ttl.String(),
		Behavior: behavior,
	}

	session, _, err := c.client.Session().Create(sEntry, nil)
	if err != nil {
		return "", err
	}

	ok, _, err := c.c.Acquire(&consul.KVPair{Key: key, Session: session}, nil)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("lock held by another client")
	}
	return session, nil
}

func (c *ckv) Lock(key string, ttl time.Duration) (kv.Lock, error) {
	session, err := c.lock(key, ttl, consul.SessionBehaviorRelease)
	if err != nil {
		return nil, err
	}

	l := &lock{sessions: c.client.Session(), session: session, kv: c.c, key: key}
	if err != nil {
		return nil, err
	}
	return l, nil
}

func (c *ckv) EphemeralKey(key string, ttl time.Duration) (kv.EphemeralKey, error) {
	session, err := c.lock(key, ttl, consul.SessionBehaviorDelete)
	if err != nil {
		return nil, err
	}

	l := &ekey{kv: c, lock: lock{sessions: c.client.Session(), session: session, key: key}}
	return l, err
}

func (l *lock) Renew() error {
	entry, _, err := l.sessions.Renew(l.session, nil)
	if err != nil {
		return err
	}
	if entry == nil {
		return errors.New("lock not held")
	}
	return nil
}

func (l *lock) unlock() error {
	err := l.Renew()
	if err != nil {
		return err
	}
	ok, _, err := l.kv.Release(&consul.KVPair{Key: l.key, Session: l.session}, nil)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("lock not held")
	}
	return nil
}

func (l *lock) Unlock() error {
	return l.unlock()
}

// Ping verifies communication with the cluster
func (c *ckv) Ping() error {
	_, err := c.client.Agent().NodeName()
	return err
}

type ekey struct {
	kv *ckv
	lock
}

func (e *ekey) Set(value string) error {
	err := e.Renew()
	if err != nil {
		return err
	}
	return e.kv.Set(e.key, value)
}

func (e *ekey) Destroy() error {
	return e.unlock()
}

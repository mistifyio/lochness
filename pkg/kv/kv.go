package kv

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)

type Value struct {
	Data  []byte
	Index uint64
}

type EventType int

const (
	None EventType = iota
	Get
	Create
	Delete
	Update
)

var types = map[EventType]string{
	None:   "None",
	Get:    "Get",
	Create: "Create",
	Delete: "Delete",
	Update: "Update",
}

type Event struct {
	Key  string
	Type EventType
	Value
}

func (e Event) GoString() string {
	return fmt.Sprintf("{Key:%s, Type:%s, Index: %d, Value: %s}", e.Key, types[e.Type], e.Index, string(e.Data))
}

var register = struct {
	sync.RWMutex
	kvs map[string]func(string) (KV, error)
}{
	kvs: map[string]func(string) (KV, error){},
}

// Register is called by KV implementors to register their scheme to be used
// with New
func Register(name string, fn func(string) (KV, error)) {
	register.Lock()
	defer register.Unlock()

	if _, dup := register.kvs[name]; dup {
		panic("kv: Register called twice for " + name)
	}
	register.kvs[name] = fn
}

// New will return a KV implementation according to the connection string addr.
// addr is a URL where the scheme is used to determine which kv implementation to return.
// The special `http` and `https` schemes are deemed generic, the first implementation that supports it will be returned.
func New(addr string) (KV, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	register.RLock()
	defer register.RUnlock()

	fn := register.kvs[u.Scheme]
	if fn != nil {
		return fn(addr)
	} else if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unknown kv store %s (forgotten import?)", u.Scheme)
	}

	for _, constructor := range register.kvs {
		kv, err := constructor(addr)
		if err != nil {
			return nil, err
		}
		if kv != nil {
			return kv, nil
		}
	}
	return nil, fmt.Errorf("unknown kv store")
}

// KV is the interface for distributed key value store interaction
type KV interface {
	Delete(string, bool) error
	Get(string) (Value, error)
	GetAll(string) (map[string]Value, error)
	Keys(string) ([]string, error)
	Set(string, string) error

	// Atomic operations
	// Update will set key=value while ensuring that newer values are not clobbered
	Update(string, Value) (uint64, error)
	// Remove will delete key only if it has not been modified since index
	Remove(string, uint64) error

	// IsKeyNotFound is a helper to determine if the error is a key not found error
	IsKeyNotFound(error) bool

	Watch(string, uint64, chan struct{}) (chan Event, chan error, error)

	TTL(string, time.Duration) error
}

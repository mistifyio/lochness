# kv

[![kv](https://godoc.org/github.com/mistifyio/lochness/pkg/kv?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/kv)

Package kv abstracts a distributed/clusted kv store for use with lochness kv
does not aim to be a full-featured generic kv abstraction, but can be useful
anyway. Only implementors imported by users will be available at runtime. See
documentation of KV for handled operations.

## Usage

#### func  Register

```go
func Register(name string, fn func(string) (KV, error))
```
Register is called by KV implementors to register their scheme to be used with
New

#### type EphemeralKey

```go
type EphemeralKey interface {
	// Set will first renew the tll then set the value of key, it is an error if the ttl has expired since last renewal
	Set(value string) error
	// Renew renews the key tll
	Renew() error
	// Destroy will delete the key without having to wait for expiration via TTL
	Destroy() error
}
```


#### type Event

```go
type Event struct {
	Key  string
	Type EventType
	Value
}
```

Event represents an action occurring to a watched key or prefix

#### type EventType

```go
type EventType int
```

EventType is used to describe actions on watch events

```go
const (
	// None indicates no event, should induce a panic if ever seen
	None EventType = iota
	// Create indicates a new key being added
	Create
	// Delete indicates a key being deleted
	Delete
	// Update indicates a key being modified, the contents of the key are not taken into account
	Update
)
```

#### type KV

```go
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

	// Watch returns channels for watching prefixes.
	// stop *must* always be closed by callers
	Watch(string, uint64, chan struct{}) (chan Event, chan error, error)

	// EphemeralKey creates a key that will be deleted if the ttl expires
	EphemeralKey(string, time.Duration) (EphemeralKey, error)

	// Lock creates a new lock, it blocks until the lock is acquired.
	Lock(string, time.Duration) (Lock, error)

	// Ping verifies communication with the cluster
	Ping() error
}
```

KV is the interface for distributed key value store interaction

#### func  New

```go
func New(addr string) (KV, error)
```
New will return a KV implementation according to the connection string addr. The
parameter addr may be the empty string or a valid URL. The special `http` and
`https` schemes are deemed generic, the first implementation that supports it
will be used. Otherwise the scheme portion of the URL will be used to select the
exact implementation to instantiate.

#### type Lock

```go
type Lock interface {
	// Renew renews the lock, it should be called before attempting any operation on whatever is being protected
	Renew() error
	// Unlock unlocks and invalidates the lock
	Unlock() error
}
```

Lock represents a locked key in the distributed key value store. The value
stored in key is managed by lock and may contain private implementation data and
should not be fetched out-of-band

#### type Value

```go
type Value struct {
	Data  []byte
	Index uint64
}
```

Value represents the value stored in a key, including the last modification
index of the key

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

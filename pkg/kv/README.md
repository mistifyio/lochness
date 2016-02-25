# kv

[![kv](https://godoc.org/github.com/mistifyio/lochness/pkg/kv?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/kv)



## Usage

#### func  Register

```go
func Register(name string, fn func(string) (KV, error))
```
Register is called by KV implementors to register their scheme to be used with
New

#### type Event

```go
type Event struct {
	Key  string
	Type EventType
	Value
}
```


#### func (Event) GoString

```go
func (e Event) GoString() string
```

#### type EventType

```go
type EventType int
```


```go
const (
	None EventType = iota
	Get
	Create
	Delete
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

	Watch(string, uint64, chan struct{}) (chan Event, chan error, error)

	TTL(string, time.Duration) error
}
```

KV is the interface for distributed key value store interaction

#### func  New

```go
func New(addr string) (KV, error)
```
New will return a KV implementation according to the connection string addr.
addr is a URL where the scheme is used to determine which kv implementation to
return. The special `http` and `https` schemes are deemed generic, the first
implementation that supports it will be returned.

#### type Value

```go
type Value struct {
	Data  []byte
	Index uint64
}
```

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

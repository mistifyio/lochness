# lock

[![lock](https://godoc.org/github.com/mistifyio/lochness/pkg/lock?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/lock)

Package lock implements a lock in an etcd cluster using CAS semantics.

## Usage

```go
var (
	// ErrKeyNotFound signifies an attempt to operate on a non-existent lock
	ErrKeyNotFound = errors.New("Key not found")
	// ErrLockNotHeld signifies an attempt to operate on a released/lost lock
	ErrLockNotHeld = errors.New("Lock not held")
)
```

#### type Lock

```go
type Lock struct {
}
```

Lock is a lock in etcd

#### func  Acquire

```go
func Acquire(c *etcd.Client, key, value string, ttl uint64, blocking bool) (*Lock, error)
```
Acquire will attempt to acquire the lock, if blocking is set to true it will
wait forever to do so. Setting blocking to false would be the equivalent of a
fictional TryAcquire, an immediate return if locking fails.

#### func (*Lock) MarshalJSON

```go
func (l *Lock) MarshalJSON() ([]byte, error)
```
MarshalJSON creates a json string from the Lock

#### func (*Lock) Refresh

```go
func (l *Lock) Refresh() error
```
Refresh will refresh the lock. An error is returned if the lock was lost, likely
due ttl expiration

#### func (*Lock) Release

```go
func (l *Lock) Release() error
```
Release will release the lock and delete the key

#### func (*Lock) UnmarshalJSON

```go
func (l *Lock) UnmarshalJSON(data []byte) error
```
UnmarshalJSON populates a Lock from a json string

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

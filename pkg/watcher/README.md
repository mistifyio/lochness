# watcher

[![watcher](https://godoc.org/github.com/mistifyio/lochness/pkg/watcher?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/watcher)

Package watcher provides etcd prefix watching capabilities.

## Usage

```go
var ErrPrefixNotWatched = errors.New("prefix is not being watched")
```
ErrPrefixNotWatched is an error for attempting to remove an unwatched prefix

```go
var ErrStopped = errors.New("watcher has been stopped")
```
ErrStopped is an error for attempting to add a prefix to a stopped watcher

#### type Error

```go
type Error struct {
	Prefix string
	Err    error
}
```

Error contains both the watched prefix and the error.

#### func (*Error) Error

```go
func (e *Error) Error() string
```

#### type Watcher

```go
type Watcher struct {
}
```

Watcher monitors etcd prefixes and notifies on change

#### func  New

```go
func New(c *etcd.Client) (*Watcher, error)
```
New creates a new Watcher

#### func (*Watcher) Add

```go
func (w *Watcher) Add(prefix string) error
```
Add will add prefix to the watch list, there still may be a short time (<500us)
after Add returns when an event on prefix may be missed.

#### func (*Watcher) Close

```go
func (w *Watcher) Close() *Error
```
Close will stop all watches and disable any new watches from being started.
Close may be called multiple times in case there is a transient error.

#### func (*Watcher) Err

```go
func (w *Watcher) Err() *Error
```
Err returns the last error received

#### func (*Watcher) Next

```go
func (w *Watcher) Next() bool
```
Next blocks until an event has been received by any of the wathed prefixes. The
event it self may be accesed via the Response method. False will be returned
upon an error, the error can be retrieved via the Err method.

#### func (*Watcher) Remove

```go
func (w *Watcher) Remove(prefix string) *Error
```
Remove will remove said prefix from the watch list, it will return an error if
the prefix is not being watched.

#### func (*Watcher) Response

```go
func (w *Watcher) Response() *etcd.Response
```
Response returns the response received that caused Next to return.

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

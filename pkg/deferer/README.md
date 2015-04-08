# deferer

[![deferer](https://godoc.org/github.com/mistifyio/lochness/pkg/deferer?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/deferer)

Package deferer provides a way to use defer calls with log.Fatal. Using
log.Fatal() is effecively the same as calling fmt.Println() followed by
os.Exit(1). The normal defer methods are not run when os.Exit() is called but
sometimes it is necessary (e.g. release a lock)
## Usage

#### type Deferer

```go
type Deferer struct {
}
```

Deferer holds a slice of deferred functions and an optional pointer to the
caller's Deferrer

#### func  NewDeferer

```go
func NewDeferer(d *Deferer) *Deferer
```
NewDeferer returns a pointer to a new Deferer instance with the function slice
initialized and the optional caller set

#### func (*Deferer) Defer

```go
func (d *Deferer) Defer(f func())
```
Defer adds to the array of defered function calls

#### func (*Deferer) Fatal

```go
func (d *Deferer) Fatal(v ...interface{})
```
Fatal runs each set of deferred functions, walking up the call change if the
parent property is set, finishing with a call to log.Fatal()

#### func (*Deferer) FatalWithFields

```go
func (d *Deferer) FatalWithFields(fields logrus.Fields, v ...interface{})
```
FatalWithFields is a mash up of Fatal and logrus

#### func (*Deferer) Run

```go
func (d *Deferer) Run()
```
Run calls each function in the defered array in reverse order. Common usage is
to call `defer d.Run()` after creating the Deferer instance

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

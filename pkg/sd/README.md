# sd

[![sd](https://godoc.org/github.com/mistifyio/lochness/pkg/sd?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/sd)

Package sd implements some systemd interaction, namely the equivalent of
sd_notify and sd_watchdog_enabled.

## Usage

```go
var ErrNotifyNoSocket = errors.New("No socket")
```
ErrNotifyNoSocket is an error for when a valid notify socket name isn't prvided

#### func  Notify

```go
func Notify(state string) error
```
Notify sends a message to the init daemon. It is common to ignore the error.

#### func  WatchdogEnabled

```go
func WatchdogEnabled() (time.Duration, error)
```
WatchdogEnabled checks whether the service manager expects watchdog keep-alive
notifications and returns the timeout value in µs. A timeout value of 0
signifies no notifications are expected.
http://www.freedesktop.org/software/systemd/man/sd_watchdog_enabled.html

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

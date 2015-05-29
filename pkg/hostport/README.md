# hostport

[![hostport](https://godoc.org/github.com/mistifyio/lochness/pkg/hostport?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/hostport)

Package hostport provides more sane and expected behavior for splitting a
networks address into host and port parts than net.SplitHostPort

## Usage

#### func  Split

```go
func Split(hostport string) (host string, port string, err error)
```
Split splits a network address of the form "host", "host:port", "[host]",
"[host]:port", ""[ipv6-host%zone]", or "[ipv6-host%zone]:port" into host or
ipv6-host%zone and port. Port will be an empty string if not supplied.

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

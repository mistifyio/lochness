# dobharchu

[![dobharchu](https://godoc.org/github.com/mistifyio/lochness/cmd/dobharchu?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/dobharchu)

dobharchu is a service to monitor etcd for changes to hyperviors and guests and
rebuild the DHCP config files as needed

### Command Usage

    $ dobharchu -h
    Usage of dobharchu:
    -d, --domain="": domain for lochness; required
    -e, --etcd="http://127.0.0.1:4001": address of etcd server
    	--guests-path="/etc/dhcpd/guests.conf": alternative path to guests.conf
    	--hypervisors-path="/etc/dhcpd/hypervisors.conf": alternative path to hypervisors.conf
    -l, --log-level="warning": log level: debug/info/warning/error/critical/fatal


### Watching

Dobharchu watches for changes to the following prefixes in etcd:

    /lochness/hypervisors
    /lochness/guests
    /lochness/subnets
## Usage

#### type Fetcher

```go
type Fetcher struct {
}
```

Fetcher grabs keys from etcd and maintains lists of hypervisors, guests, and
subnets

#### func  NewFetcher

```go
func NewFetcher(etcdAddress string) *Fetcher
```
NewFetcher creates a new fetcher

#### func (*Fetcher) FetchAll

```go
func (f *Fetcher) FetchAll() error
```
FetchAll pulls the hypervisors, guests, and subnets from etcd

#### func (*Fetcher) Guests

```go
func (f *Fetcher) Guests() (map[string]*lochness.Guest, error)
```
Guests retrieves the stored guests, or fetches them if they aren't stored yet

#### func (*Fetcher) Hypervisors

```go
func (f *Fetcher) Hypervisors() (map[string]*lochness.Hypervisor, error)
```
Hypervisors retrieves the stored hypervisors, or fetches them if they aren't
stored yet

#### func (*Fetcher) IntegrateResponse

```go
func (f *Fetcher) IntegrateResponse(r *etcd.Response) (bool, error)
```
IntegrateResponse takes an etcd reponse and updates our list of hypervisors,
subnets, or guests, then returns whether a refresh should happen

#### func (*Fetcher) Subnets

```go
func (f *Fetcher) Subnets() (map[string]*lochness.Subnet, error)
```
Subnets retrieves the stored subnets, or fetches them if they aren't stored yet

#### type Refresher

```go
type Refresher struct {
	Domain string
}
```

Refresher writes out the dhcp configuration files hypervisors.conf and
guests.conf, given a fetcher

#### func  NewRefresher

```go
func NewRefresher(domain string) *Refresher
```
NewRefresher creates a new refresher

#### func (*Refresher) WriteGuestsConfigFile

```go
func (r *Refresher) WriteGuestsConfigFile(w io.Writer, guests map[string]*lochness.Guest, subnets map[string]*lochness.Subnet) error
```
WriteGuestsConfigFile writes out the guests config file using the given writer

#### func (*Refresher) WriteHypervisorsConfigFile

```go
func (r *Refresher) WriteHypervisorsConfigFile(w io.Writer, hypervisors map[string]*lochness.Hypervisor) error
```
WriteHypervisorsConfigFile writes out the hypervisors config file using the
given writer

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

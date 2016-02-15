# ct

[![ct](https://godoc.org/github.com/mistifyio/lochness/internal/tests/common?status.png)](https://godoc.org/github.com/mistifyio/lochness/internal/tests/common)

ct contains common utilities and suites to be used in other tests

## Usage

#### func  Build

```go
func Build() error
```

#### func  ExitStatus

```go
func ExitStatus(err error) int
```

#### func  TestMsgFunc

```go
func TestMsgFunc(prefix string) func(...interface{}) string
```

#### type Cmd

```go
type Cmd struct {
	Cmd *exec.Cmd
	Out *bytes.Buffer
}
```


#### func  Exec

```go
func Exec(cmdName string, args ...string) (*Cmd, error)
```

#### func  ExecSync

```go
func ExecSync(cmdName string, args ...string) (*Cmd, error)
```

#### func (*Cmd) Alive

```go
func (c *Cmd) Alive() bool
```

#### func (*Cmd) ExitStatus

```go
func (c *Cmd) ExitStatus() (int, error)
```

#### func (*Cmd) Stop

```go
func (c *Cmd) Stop() error
```

#### func (*Cmd) Wait

```go
func (c *Cmd) Wait() error
```

#### type CommonTestSuite

```go
type CommonTestSuite struct {
	suite.Suite
	EtcdDir    string
	EtcdPrefix string
	EtcdClient *etcd.Client
	EtcdCmd    *exec.Cmd
	Context    *lochness.Context
}
```


#### func (*CommonTestSuite) NewFWGroup

```go
func (s *CommonTestSuite) NewFWGroup() *lochness.FWGroup
```

#### func (*CommonTestSuite) NewFlavor

```go
func (s *CommonTestSuite) NewFlavor() *lochness.Flavor
```

#### func (*CommonTestSuite) NewGuest

```go
func (s *CommonTestSuite) NewGuest() *lochness.Guest
```

#### func (*CommonTestSuite) NewHypervisor

```go
func (s *CommonTestSuite) NewHypervisor() *lochness.Hypervisor
```

#### func (*CommonTestSuite) NewHypervisorWithGuest

```go
func (s *CommonTestSuite) NewHypervisorWithGuest() (*lochness.Hypervisor, *lochness.Guest)
```

#### func (*CommonTestSuite) NewNetwork

```go
func (s *CommonTestSuite) NewNetwork() *lochness.Network
```

#### func (*CommonTestSuite) NewSubnet

```go
func (s *CommonTestSuite) NewSubnet() *lochness.Subnet
```

#### func (*CommonTestSuite) NewVLAN

```go
func (s *CommonTestSuite) NewVLAN() *lochness.VLAN
```

#### func (*CommonTestSuite) NewVLANGroup

```go
func (s *CommonTestSuite) NewVLANGroup() *lochness.VLANGroup
```

#### func (*CommonTestSuite) PrefixKey

```go
func (s *CommonTestSuite) PrefixKey(key string) string
```

#### func (*CommonTestSuite) SetupSuite

```go
func (s *CommonTestSuite) SetupSuite()
```

#### func (*CommonTestSuite) SetupTest

```go
func (s *CommonTestSuite) SetupTest()
```

#### func (*CommonTestSuite) TearDownSuite

```go
func (s *CommonTestSuite) TearDownSuite()
```

#### func (*CommonTestSuite) TearDownTest

```go
func (s *CommonTestSuite) TearDownTest()
```

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

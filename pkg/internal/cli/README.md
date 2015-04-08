# cli

[![cli](https://godoc.org/github.com/mistifyio/lochness/pkg/internal/cli?status.png)](https://godoc.org/github.com/mistifyio/lochness/pkg/internal/cli)

Package cli provides a client and utilities for lochness cli applications to
interact with agents
## Usage

#### func  AssertID

```go
func AssertID(id string)
```
AssertID checks whether a string is a valid id

#### func  AssertSpec

```go
func AssertSpec(spec string)
```
AssertSpec checks whether a json string parses as expected

#### func  Read

```go
func Read(r io.Reader) []string
```
Read parses cli args into an array of strings

#### type Client

```go
type Client struct {
}
```

Client interacts with an http api

#### func  NewClient

```go
func NewClient(address string) *Client
```
NewClient creates a new Client

#### func (*Client) Delete

```go
func (c *Client) Delete(title, endpoint string) map[string]interface{}
```
Delete DELETEs a resource

#### func (*Client) Get

```go
func (c *Client) Get(title, endpoint string) map[string]interface{}
```
Get GETs a single resource

#### func (*Client) GetList

```go
func (c *Client) GetList(title, endpoint string) []string
```
GetList GETs an array of string (e.g. IDs)

#### func (*Client) GetMany

```go
func (c *Client) GetMany(title, endpoint string) []map[string]interface{}
```
GetMany GETs a set of resources

#### func (*Client) Patch

```go
func (c *Client) Patch(title, endpoint, body string) map[string]interface{}
```
Patch PATCHes a resource

#### func (*Client) Post

```go
func (c *Client) Post(title, endpoint, body string) map[string]interface{}
```
Post POSTs a body

#### func (*Client) URLString

```go
func (c *Client) URLString(endpoint string) string
```
URLString generates the full url given an endpoint path

#### type JMap

```go
type JMap map[string]interface{}
```

JMap is a generic resource

#### func (JMap) ID

```go
func (j JMap) ID() string
```
ID returns the id value

#### func (JMap) Print

```go
func (j JMap) Print(json bool)
```
Print prints either the json string or just the id

#### func (JMap) String

```go
func (j JMap) String() string
```
String marshals into a json string

#### type JMapSlice

```go
type JMapSlice []JMap
```

JMapSlice is an array of generic resources

#### func (JMapSlice) Len

```go
func (js JMapSlice) Len() int
```
Len returns the length of the array

#### func (JMapSlice) Less

```go
func (js JMapSlice) Less(i, j int) bool
```
Less returns the comparsion of two elements

#### func (JMapSlice) Swap

```go
func (js JMapSlice) Swap(i, j int)
```
Swap swaps two elements

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

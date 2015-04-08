# waheela

[![waheela](https://godoc.org/github.com/mistifyio/lochness/cmd/waheela?status.png)](https://godoc.org/github.com/mistifyio/lochness/cmd/waheela)

waheela is the guest management service. It exposes functionality over an HTTP
API with JSON formatting.

### Command Usage

    $ waheela -h
    Usage of waheela:
    -e, --etcd="http://localhost:4001": address of etcd machine
    -l, --log-level="warn": log level
    -p, --port=18000: listen port
    -s, --statsd="": statsd address

### HTTP API Endpoints

    /guests
    	* GET - Retrieve a list of guests
    	* POST - Create a new guest
    /guests/{guestID}
    	* GET    - Retrieve information about a guest
    	* PATCH  - Update information for a guest
    	* DELETE - Delete a guest


### Example Structs

Guest - lochness.Guest

    {
    	"id": "f2011319-ad59-42fb-9bad-92e261f0651c",
    	"metadata": {},
    	"type": "",
    	"flavor": "fe6de923-7230-416e-89d7-374b4b7b9362",
    	"hypervisor": "e88a75a6-7ae6-487c-9634-6553d3793437",
    	"network": "ac258bc2-4fc4-4713-a6fd-fc1afb65cd32",
    	"subnet": "c6430cba-648a-41aa-aee4-b59dacfc790d",
    	"fwgroup": "ecf5f19a-83e3-4dff-8f03-871d0d13ae65",
    	"mac": "01:23:45:67:89:ac",
    	"ip": "10.10.10.28",
    	"bridge": "br0"
    }


### Example Requests

GET /guests

    $ curl http://localhost:18000/guests

    [{"id":"f2011319-ad59-42fb-9bad-92e261f0651c","metadata":{},"type":"","flavor":"fe6de923-7230-416e-89d7-374b4b7b9362","hypervisor":"e88a75a6-7ae6-487c-9634-6553d3793437","network":"ac258bc2-4fc4-4713-a6fd-fc1afb65cd32","subnet":"c6430cba-648a-41aa-aee4-b59dacfc790d","fwgroup":"ecf5f19a-83e3-4dff-8f03-871d0d13ae65","mac":"01:23:45:67:89:ac","ip":"10.10.10.28","bridge":"br0"},{"id":"ad762efc-3c23-402b-8e1f-a248a005efb9","metadata":{},"type":"","flavor":"1f5acce3-96b4-4ccb-865f-e6c44f68900d","hypervisor":"e88a75a6-7ae6-487c-9634-6553d3793437","network":"ac258bc2-4fc4-4713-a6fd-fc1afb65cd32","subnet":"c6430cba-648a-41aa-aee4-b59dacfc790d","fwgroup":"9b2342a9-c1c1-4410-9b25-5984485cd247","mac":"01:23:45:67:89:ab","ip":"10.10.10.231","bridge":"br0"}]

POST /guests

    $ curl -XPOST http://localhost:18000/guests --data-binary '{"bridge":"br0", "flavor":"1","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234", "ip":"10.100.101.66", "mac":"A4-75-C1-6B-E3-49", "network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","type":"foo"}'

    {"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}

GET /guests/{guestID}

    $ curl http://localhost:18000/guests/94ea0ba1-5ec2-460e-9c2e-8269593cdad3

    {"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}

PATCH /guests/{guestID}

    $ curl -X PATCH http://localhost:18000/guests/94ea0ba1-5ec2-460e-9c2e-8269593cdad3 --data-binary '{"metadata":{"foo":"bar"}}'

    {"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{"foo":"bar"},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}

DELETE /guests/{guestID}

    $ curl -X DELETE http://localhost:18000/guests/94ea0ba1-5ec2-460e-9c2e-8269593cdad3

    {"id":"94ea0ba1-5ec2-460e-9c2e-8269593cdad3","metadata":{"foo":"bar"},"type":"foo","flavor":"1","hypervisor":"","network":"1234asdf-1234-asdf-1234-asdf1234asdf1234","subnet":"1234asdf-1234-asdf-1234-asdf1234asdf1234","fwgroup":"1234asdf-1234-asdf-1234-asdf1234asdf1234","mac":"a4:75:c1:6b:e3:49","ip":"10.100.101.66","bridge":"br0"}
## Usage

#### func  CreateGuest

```go
func CreateGuest(w http.ResponseWriter, r *http.Request)
```
CreateGuest creates a new guest

#### func  DestroyGuest

```go
func DestroyGuest(w http.ResponseWriter, r *http.Request)
```
DestroyGuest removes a guest and frees its IP

#### func  GetContext

```go
func GetContext(r *http.Request) *lochness.Context
```
GetContext retrieves a lochness.Context value for a request

#### func  GetGuest

```go
func GetGuest(w http.ResponseWriter, r *http.Request)
```
GetGuest gets a particular guest

#### func  GetRequestGuest

```go
func GetRequestGuest(r *http.Request) *lochness.Guest
```
GetRequestGuest retrieves the guest from teh request context

#### func  ListGuests

```go
func ListGuests(w http.ResponseWriter, r *http.Request)
```
ListGuests gets a list of all guests

#### func  RegisterGuestRoutes

```go
func RegisterGuestRoutes(prefix string, router *mux.Router, m *metricsContext)
```
RegisterGuestRoutes registers the guest routes and handlers

#### func  Run

```go
func Run(port uint, ctx *lochness.Context, m *metricsContext) error
```
Run starts the server

#### func  SetContext

```go
func SetContext(r *http.Request, ctx *lochness.Context)
```
SetContext sets a lochness.Context value for a request

#### func  SetRequestGuest

```go
func SetRequestGuest(r *http.Request, g *lochness.Guest)
```
SetRequestGuest saves the guest to the request context

#### func  UpdateGuest

```go
func UpdateGuest(w http.ResponseWriter, r *http.Request)
```
UpdateGuest updates an existing guest

#### type HTTPError

```go
type HTTPError struct {
	Message string   `json:"message"`
	Code    int      `json:"code"`
	Stack   []string `json:"stack"`
}
```

HTTPError contains information for http error responses

#### type HTTPResponse

```go
type HTTPResponse struct {
	http.ResponseWriter
}
```

HTTPResponse is a wrapper for http.ResponseWriter which provides access to
several convenience methods

#### func (*HTTPResponse) JSON

```go
func (hr *HTTPResponse) JSON(code int, obj interface{})
```
JSON writes appropriate headers and JSON body to the http response

#### func (*HTTPResponse) JSONError

```go
func (hr *HTTPResponse) JSONError(code int, err error)
```
JSONError prepares an HTTPError with a stack trace and writes it with
HTTPResponse.JSON

#### func (*HTTPResponse) JSONMsg

```go
func (hr *HTTPResponse) JSONMsg(code int, msg string)
```
JSONMsg is a convenience method to write a JSON response with just a message
string

--
*Generated with [godocdown](https://github.com/robertkrimen/godocdown)*

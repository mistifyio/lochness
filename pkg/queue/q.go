// Package queue implements a FIFO queue using etcd
package queue

import (
	"encoding/json"
	"errors"
	"log"

	etcdErr "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
)

var ErrStopped = errors.New("stopped by the user via stop channel")

// TODO unique error with json error return?
var jsonMarshalError = []byte(`{"response":"internal error unmarshalling response"}`)

type queued struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

type Conn struct {
	dir string
	c   *etcd.Client
}

// Connect returns a new connection to the queue
func Connect(c *etcd.Client, dir string) *Conn {
	return &Conn{dir: dir, c: c}
}

// Put enqueues the value
func (conn *Conn) Put(value string) (string, error) {
	Req := queued{Request: value}
	data, err := json.Marshal(&Req)
	if err != nil {
		return "", err
	}

	resp, err := conn.c.CreateInOrder(conn.dir, string(data), 0)
	if err != nil {
		return "", err
	}

	resp, err = conn.c.Watch(resp.Node.Key, resp.Node.CreatedIndex+1, false, nil, nil)
	if err != nil {
		return "", err
	}

	Resp := queued{}
	if err := json.Unmarshal([]byte(resp.Node.Value), &Resp); err != nil {
		return "", err
	}

	_, err = conn.c.Delete(resp.Node.Key, false)
	return Resp.Response, err
}

func isKeyExists(err error) bool {
	e, ok := err.(*etcd.EtcdError)
	return ok && e.ErrorCode == etcdErr.EcodeNodeExist
}

// Q represents the opened queue
type Q struct {
	// C is a blocking chan used to deliver values as they are inserted into
	// the queue. Once the value is delivered the object is deleted from
	// the etcd directory.
	C   chan string
	dir string
	c   *etcd.Client
}

// Open will use the dir argument as a queue. Only one queue may be opened per
// directory, bad things happen if not ensured, it is up to the caller to
// ensure.
func Open(c *etcd.Client, dir string, stop chan bool) (*Q, error) {
	_, err := c.CreateDir(dir, 0)
	if err != nil && !isKeyExists(err) {
		return nil, err
	}

	keys := make(chan string)
	q := &Q{C: keys, dir: dir, c: c}
	go func() {
		defer close(keys)

		index, err := poll(c, dir, keys, stop)
		if err != nil {
			return
		}

		watch(c, dir, index, keys, stop)
	}()

	return q, nil
}

// passMessage will Marshal the job, post it, wait for a response (forever),
// unmarshal the resopnse and then return the data.
func passMessage(c *etcd.Client, values chan string, key string) error {
	resp, err := c.Get(key, false, false)
	if err != nil {
		return err
	}

	data := queued{}
	if err := json.Unmarshal([]byte(resp.Node.Value), &data); err != nil {
		return err
	}

	if data.Request == "" {
		return errors.New("empty request")
	}

	values <- data.Request
	data = queued{Response: <-values}

	buf, err := json.Marshal(&data)
	if err != nil {
		log.Println(err)
		buf = jsonMarshalError
	}

	_, err = c.Update(key, string(buf), 0)
	if err != nil {
		// retry?
		return err
	}
	return nil
}

func poll(c *etcd.Client, dir string, keys chan string, stop chan bool) (uint64, error) {
	resp, err := c.Get(dir, true, true)
	if err != nil {
		return 0, err
	}
	if !resp.Node.Dir {
		return 0, errors.New("node is not a dir")
	}
	index := uint64(0)
	for _, node := range resp.Node.Nodes {
		select {
		case <-stop:
			return 0, ErrStopped
		default:
			index = node.ModifiedIndex + 1
			err := passMessage(c, keys, node.Key)
			if err != nil {
				log.Println(err)
			}
		}
	}
	return index, nil
}

func watch(c *etcd.Client, dir string, index uint64, keys chan string, stop chan bool) {
	resps := make(chan *etcd.Response)
	go func() {
		for resp := range resps {
			switch resp.Action {
			case "create", "set":
			default:
				continue
			}
			err := passMessage(c, keys, resp.Node.Key)
			if err != nil {
				log.Println(err)
			}
		}
	}()
	_, err := c.Watch(dir, index, true, resps, stop)
	if err != nil {
		log.Println(err)
	}
}

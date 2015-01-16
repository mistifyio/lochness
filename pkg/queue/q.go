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

type Job struct {
	key      string
	Request  string `json:"request"`
	Response string `json:"response"`
}

type Conn struct {
	dir string
	c   *etcd.Client
}

// Connect returns a new connection to the queue
func Connect(c *etcd.Client, dir string) *Conn {
	return &Conn{dir: dir, c: c}
}

// Put enqueues the value, and wait for the response. Once the response has been
// received, the node is deleted.
func (conn *Conn) Put(value string) (string, error) {
	Req := Job{Request: value}
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

	Resp := Job{}
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

func isEventIndexCleared(err error) bool {
	e, ok := err.(*etcd.EtcdError)
	return ok && e.ErrorCode == etcdErr.EcodeEventIndexCleared
}

// Q represents the opened queue
type Q struct {
	// Requests and Responses are both blocking channels
	Requests  <-chan Job
	Responses chan<- Job
	dir       string
	c         *etcd.Client
}

// Open will use the dir argument as a queue. Only one queue may be opened per
// directory, bad things happen if not ensured, it is up to the caller to
// ensure.
func Open(c *etcd.Client, dir string, stop chan bool) (*Q, error) {
	_, err := c.CreateDir(dir, 0)
	if err != nil && !isKeyExists(err) {
		return nil, err
	}

	reqs := make(chan Job)
	resps := make(chan Job)
	q := &Q{Requests: reqs, Responses: resps, dir: dir, c: c}
	go func() {
		for resp := range resps {
			receiveMessage(c, resp)
		}
	}()
	go func() {
		defer close(reqs)
		// closing responses could cause panics when shutting down!

		index, err := poll(c, dir, reqs, stop)
		if err != nil {
			return
		}

		watch(c, dir, index, reqs, stop)
	}()

	return q, nil
}

func sendMessage(c *etcd.Client, reqs chan Job, key string) {
	resp, err := c.Get(key, false, false)
	if err != nil {
		log.Println(err)
		return
	}

	req := Job{}
	if err := json.Unmarshal([]byte(resp.Node.Value), &req); err != nil {
		log.Println(err)
		return
	}

	if req.Request == "" {
		log.Println("empty request")
		return
	}

	if req.Response != "" {
		log.Println("message already handled")
		return
	}

	req.key = key
	reqs <- req
}

func receiveMessage(c *etcd.Client, j Job) {
	buf, err := json.Marshal(&j)
	if err != nil {
		log.Println(err)
		buf = jsonMarshalError
	}

	_, err = c.Update(j.key, string(buf), 0)
	if err != nil {
		log.Println(err)
	}
}

func poll(c *etcd.Client, dir string, reqs chan Job, stop chan bool) (uint64, error) {
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
			sendMessage(c, reqs, node.Key)
		}
	}
	return index, nil
}

func watch(c *etcd.Client, dir string, index uint64, reqs chan Job, stop chan bool) {
	for {
		resps := make(chan *etcd.Response)
		go func() {
			for resp := range resps {
				switch resp.Action {
				case "create", "set":
				default:
					continue
				}
				sendMessage(c, reqs, resp.Node.Key)
			}
		}()

		_, err := c.Watch(dir, index, true, resps, stop)
		if err == nil || err == etcd.ErrWatchStoppedByUser {
			break
		}
		if !isEventIndexCleared(err) {
			log.Printf("%#v\n", err)
			break
		}
		index++
		continue
	}
}

package etcd

import (
	"errors"
	"time"

	etcdErr "github.com/coreos/etcd/error"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/kv"
)

func init() {
	kv.Register("etcd", New)
}

type ekv struct {
	e *etcd.Client
}

func New(addr string) (kv.KV, error) {
	return &ekv{e: etcd.NewClient([]string{addr})}, nil
}

func (e *ekv) Delete(key string, recurse bool) error {
	_, err := e.e.Delete(key, recurse)
	return err
}

func (e ekv) Get(key string) (kv.Value, error) {
	resp, err := e.e.Get(key, false, false)
	if err != nil {
		return kv.Value{}, err
	}

	if resp.Node.Dir {
		return kv.Value{}, errors.New("key is a directory")
	}

	return kv.Value{Data: []byte(resp.Node.Value), Index: resp.Node.ModifiedIndex}, nil
}

func (e ekv) GetAll(prefix string) (map[string]kv.Value, error) {
	resp, err := e.e.Get(prefix, false, true)
	if err != nil {
		return nil, err
	}

	if !resp.Node.Dir {
		return map[string]kv.Value{
			resp.Node.Key: kv.Value{Data: []byte(resp.Node.Value), Index: resp.Node.ModifiedIndex},
		}, nil
	}

	many := map[string]kv.Value{}
	var recursive func(etcd.Nodes)
	recursive = func(nodes etcd.Nodes) {
		for _, node := range nodes {
			if node.Dir {
				recursive(node.Nodes)
			} else {
				many[node.Key] = kv.Value{Data: []byte(node.Value), Index: node.ModifiedIndex}
			}
		}
	}
	recursive(resp.Node.Nodes)

	return many, nil
}

func (e *ekv) Keys(key string) ([]string, error) {
	resp, err := e.e.Get(key, true, false)
	if err != nil {
		return nil, err
	}

	if !resp.Node.Dir {
		return nil, errors.New("key is not a directory")
	}

	nodes := resp.Node.Nodes
	keys := make([]string, len(nodes))
	for i := range nodes {
		keys[i] = nodes[i].Key
	}

	return keys, err
}

func (e ekv) Set(key, value string) error {
	_, err := e.e.Set(key, value, 0)
	return err
}

func (e ekv) Update(key string, value kv.Value) (uint64, error) {
	var err error
	var resp *etcd.Response
	if value.Index == 0 {
		resp, err = e.e.Create(key, string(value.Data), 0)
	} else {
		resp, err = e.e.CompareAndSwap(key, string(value.Data), 0, "", value.Index)
	}
	if err != nil {
		return 0, err
	}
	return resp.Node.ModifiedIndex, nil
}

func (e *ekv) Remove(key string, index uint64) error {
	_, err := e.e.CompareAndDelete(key, "", index)
	return err
}

func (e *ekv) IsKeyNotFound(err error) bool {
	eErr, ok := err.(*etcd.EtcdError)
	return ok && eErr.ErrorCode == etcdErr.EcodeKeyNotFound
}

var typeE2KV = map[string]kv.EventType{
	"compareAndSwap": kv.Update,
	"create":         kv.Create,
	"delete":         kv.Delete,
	"get":            kv.Get,
	"set":            kv.Update,
}

func (e *ekv) Watch(prefix string, index uint64, stop chan struct{}) (chan kv.Event, chan error, error) {
	bStop := make(chan bool)
	go func() {
		<-stop
		bStop <- true
	}()

	responses := make(chan *etcd.Response)
	events := make(chan kv.Event)
	go func() {
		for resp := range responses {
			events <- kv.Event{
				Type: typeE2KV[resp.Action],
				Key:  resp.Node.Key,
				Value: kv.Value{
					Data:  []byte(resp.Node.Value),
					Index: resp.Node.ModifiedIndex,
				},
			}
		}
	}()

	errors := make(chan error)
	go func() {
		_, err := e.e.Watch(prefix, index, true, responses, bStop)
		if err != nil && err != etcd.ErrWatchStoppedByUser {
			errors <- err
		}
	}()

	return events, errors, nil
}

func (e *ekv) TTL(key string, ttl time.Duration) error {
	_, err := e.e.Set(key, time.Now().String(), uint64(ttl.Seconds()))
	return err
}

func (e ekv) SyncCluster() bool {
	return e.e.SyncCluster()
}

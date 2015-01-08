package queue

import (
	"errors"
	"log"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/pkg/lock"
)

var ErrStopped = errors.New("stopped by the user via stop channel")

type Q struct {
	C   chan string
	dir string
	c   *etcd.Client
	l   *lock.Lock
}

func Open(c *etcd.Client, dir string, stop chan bool) (*Q, error) {
	l, err := lock.Acquire(c, dir+"/lock", "", 20, true)
	if err != nil {
		return nil, err
	}
	go refresh(l)

	keys := make(chan string, 10)
	index, err := poll(c, dir, keys, stop)
	if err != nil && err != ErrStopped {
		close(keys)
		return nil, err
	}
	go watch(c, dir, index, keys, stop)

	return &Q{C: keys, dir: dir, c: c, l: l}, nil
}

func (q *Q) Close() error {
	return q.l.Release()
}

func (q *Q) Put(value string) error {
	_, err := q.c.CreateInOrder(q.dir, value, 0)
	return err
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
			if node.Key == dir+"/lock" {
				log.Println("p: lock, skipping")
				continue
			}

			resp, err := c.Get(node.Key, false, false)
			if err != nil {
				return 0, err
			}
			log.Println(resp)
			keys <- resp.Node.Value
			_, err = c.Delete(node.Key, false)
			if err != nil {
				return 0, err
			}
			index = node.ModifiedIndex + 1
		}
	}
	log.Println("p: done polling")
	return index, nil
}

func watch(c *etcd.Client, dir string, index uint64, keys chan string, stop chan bool) {
	resps := make(chan *etcd.Response)
	go func() {
		for resp := range resps {
			if resp.Action != "create" {
				log.Println("w: skipping", resp.Node.Key)
				continue
			}
			if resp.Node.Key == dir+"/lock" {
				log.Println("w: lock", resp.Node.Key)
				continue
			}
			log.Println("w:", resp.Node.Key, resp.Action)

			r, err := c.Get(resp.Node.Key, false, false)
			if err != nil {
				break
			}
			keys <- r.Node.Value
			_, err = c.Delete(r.Node.Key, false)
			if err != nil {
				break
			}
		}
		log.Println("w: done")
		close(keys)
	}()
	log.Println("w: watching")
	_, err := c.Watch(dir, index, true, resps, stop)
	if err != nil {
		log.Println("w:", err)
	}
	log.Println("w: watch done")
}

func refresh(l *lock.Lock) {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		log.Println("r: pre-refresh")
		if err := l.Refresh(); err != nil {
			panic(err)
		}
		log.Println("r: post-refresh")
	}
}

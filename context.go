package lochness

import "github.com/coreos/go-etcd/etcd"

// Context carries around data/structs needed for operations
type Context struct {
	etcd *etcd.Client
}

func NewContext(e *etcd.Client) *Context {
	return &Context{
		etcd: e,
	}
}

package lochness

import (
	etcdErr "github.com/coreos/etcd/error"
	kv "github.com/coreos/go-etcd/etcd"
)

// Context carries around data/structs needed for operations
type Context struct {
	kv *kv.Client
}

// NewContext creates a new context
func NewContext(e *kv.Client) *Context {
	return &Context{
		kv: e,
	}
}

// IsKeyNotFound is a helper to determine if the error is a key not found error
func IsKeyNotFound(err error) bool {
	e, ok := err.(*kv.EtcdError)
	return ok && e.ErrorCode == etcdErr.EcodeKeyNotFound
}

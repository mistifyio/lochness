package lochness

import (
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

// Context carries around data/structs needed for operations
type Context struct {
	etcd *etcd.Client
}

func NewContext(e *etcd.Client) *Context {
	return &Context{
		etcd: e,
	}
}

func IsKeyNotFound(err error) bool {
	return strings.Contains(err.Error(), "Key not found")
}

package lochness

import (
	"github.com/mistifyio/lochness/pkg/kv"
)

// Context carries around data/structs needed for operations
type Context struct {
	kv kv.KV
}

// NewContext creates a new context
func NewContext(kv kv.KV) *Context {
	return &Context{
		kv: kv,
	}
}

// IsKeyNotFound is a helper to determine if the error is a key not found error
func (c *Context) IsKeyNotFound(err error) bool {
	return c.kv.IsKeyNotFound(err)
}

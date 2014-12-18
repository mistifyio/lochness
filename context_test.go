package lochness_test

import (
	"testing"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

func newContext(t *testing.T) *lochness.Context {
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	if !e.SyncCluster() {
		t.Fatal("cannot sync cluster. make sure etcd is running at http://127.0.0.1:4001")
	}

	c := lochness.NewContext(e)

	return c
}

func TestNewContext(t *testing.T) {
	_ = newContext(t)
}

package lochness_test

import (
	"errors"
	"testing"

	h "github.com/bakins/test-helpers"
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

func contextCleanup(t *testing.T) {
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	if !e.SyncCluster() {
		t.Fatal("cannot sync cluster. make sure etcd is running at http://127.0.0.1:4001")
	}

	_, err := e.Delete("lochness", true)
	h.Ok(t, err)
}

func TestIsKeyNotFound(t *testing.T) {
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	newContext(t)
	defer contextCleanup(t)

	_, err := e.Get("lochness/some-randon-non-existent-key", false, false)

	if !lochness.IsKeyNotFound(err) {
		t.Fatalf("was expecting a KeyNotFound error, got: %#v\n", err)
	}

	err = errors.New("lochness/some-random-non-key-not-found-error")
	if lochness.IsKeyNotFound(err) {
		t.Fatal("got unexpected positive KeyNotFound error for err:", err)
	}
}

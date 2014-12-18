package lochness_test

import (
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
)

func newGuest(t *testing.T) *lochness.Guest {
	c := newContext(t)
	g := c.NewGuest()

	return g
}

func TestNewGuest(t *testing.T) {
	g := newGuest(t)
	h.Equals(t, 36, len(g.ID))
}

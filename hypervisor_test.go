package lochness_test

import (
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
)

func newHypervisor(t *testing.T) *lochness.Hypervisor {
	c := newContext(t)
	hv := c.NewHypervisor()

	err := hv.Save()
	h.Ok(t, err)

	return hv
}

func TestNewHypervisor(t *testing.T) {
	hv := newHypervisor(t)
	h.Equals(t, 36, len(hv.ID))
}

func TestHypervisor(t *testing.T) {
	c := newContext(t)
	hv := newHypervisor(t)
	id := hv.ID
	hv, err := c.Hypervisor(id)
	h.Ok(t, err)
	h.Equals(t, id, hv.ID)
}

func TestHypervisorIsAlive(t *testing.T) {
	hv := newHypervisor(t)
	alive, err := hv.IsAlive()
	h.Ok(t, err)
	h.Equals(t, false, alive)
}

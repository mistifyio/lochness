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
	defer contextCleanup(t)
	h.Equals(t, 36, len(hv.ID))
}

func TestHypervisor(t *testing.T) {
	c := newContext(t)
	defer contextCleanup(t)
	hv := newHypervisor(t)
	id := hv.ID
	hv, err := c.Hypervisor(id)
	h.Ok(t, err)
	h.Equals(t, id, hv.ID)
}

func TestHypervisorIsAlive(t *testing.T) {
	hv := newHypervisor(t)
	defer contextCleanup(t)
	h.Equals(t, false, hv.IsAlive())
}

func TestHypervisorsAlias(t *testing.T) {
	_ = lochness.Hypervisors([]*lochness.Hypervisor{})
}

func TestHypervisorSetConfig(t *testing.T) {
	hv := newHypervisor(t)
	defer contextCleanup(t)

	h.Ok(t, hv.SetConfig("foo", "bar"))

	h.Equals(t, "bar", hv.Config["foo"])

	h.Ok(t, hv.Refresh())

	h.Equals(t, "bar", hv.Config["foo"])

	h.Ok(t, hv.SetConfig("foo", ""))

	_, ok := hv.Config["foo"]
	h.Equals(t, ok, false)
}

func TestFirstHypervisor(t *testing.T) {
	c := newContext(t)
	newHypervisor(t)
	newHypervisor(t)
	hv := newHypervisor(t)
	defer contextCleanup(t)

	found, err := c.FirstHypervisor(func(h *lochness.Hypervisor) bool {
		return h.ID == "foo"
	})

	h.Ok(t, err)

	h.Assert(t, found == nil, "unexpected value")

	found, err = c.FirstHypervisor(func(h *lochness.Hypervisor) bool {
		return h.ID == hv.ID
	})

	h.Ok(t, err)

	h.Assert(t, found != nil, "unexpected nil")

}

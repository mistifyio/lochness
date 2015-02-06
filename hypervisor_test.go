package lochness_test

import (
	"os"
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

func TestHypervisorDestroy(t *testing.T) {
	defer contextCleanup(t)
	hv := newHypervisor(t)

	err := hv.Destroy()
	h.Ok(t, err)
	// need a test with a guest
}

func TestSetHypervisorID(t *testing.T) {
	// passing test with uuid
	uuid := "d3cac004-4d89-4f26-9776-97df74a41417"
	id, err := lochness.SetHypervisorID(uuid)
	h.Ok(t, err)
	h.Equals(t, uuid, id)

	id, err = lochness.SetHypervisorID("foo")
	h.Assert(t, err != nil, "should have got an error")
	h.Equals(t, "", id)

	// set with ENV
	uuid = "3e0f2128-0342-49f6-8e5f-ecd401bae99e"
	os.Setenv("HYPERVISOR_ID", uuid)
	id, err = lochness.SetHypervisorID("")
	h.Ok(t, err)
	h.Equals(t, uuid, id)

}

func TestGetHypervisorID(t *testing.T) {
	uuid := "d3cac004-4d89-4f26-9776-97df74a41417"
	id, err := lochness.SetHypervisorID(uuid)
	h.Ok(t, err)
	h.Equals(t, uuid, id)

	id = lochness.GetHypervisorID()
	h.Equals(t, uuid, id)

}

func TestVerifyOnHV(t *testing.T) {
	defer contextCleanup(t)
	hv := newHypervisor(t)

	// failing test
	uuid := "d3cac004-4d89-4f26-9776-97df74a41417"
	id, err := lochness.SetHypervisorID(uuid)
	h.Ok(t, err)
	h.Equals(t, uuid, id)

	err = hv.VerifyOnHV()
	h.Assert(t, err != nil, "should have got an error")

	// passing
	id, err = lochness.SetHypervisorID(hv.ID)
	h.Ok(t, err)
	h.Equals(t, hv.ID, id)

	err = hv.VerifyOnHV()
	h.Ok(t, err)
}

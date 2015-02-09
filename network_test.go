package lochness_test

import (
	"strings"
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
)

func newNetwork(t *testing.T) *lochness.Network {
	c := newContext(t)
	n := c.NewNetwork()

	return n
}

func TestNewNetwork(t *testing.T) {
	defer contextCleanup(t)
	n := newNetwork(t)
	h.Equals(t, 36, len(n.ID))
	err := n.Save()
	h.Ok(t, err)
}

func TestNetworkAddSubnet(t *testing.T) {
	defer contextCleanup(t)
	n := newNetwork(t)
	h.Equals(t, 36, len(n.ID))
	err := n.Save()
	h.Ok(t, err)

	s := newSubnet(t)

	err = n.AddSubnet(s)
	h.Ok(t, err)

	h.Equals(t, 1, len(n.Subnets()))
}

func TestNetworkAlias(t *testing.T) {
	_ = lochness.Networks([]*lochness.Network{})
}

func TestNetworkWithBadID(t *testing.T) {
	c := newContext(t)
	_, err := c.Network("")
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "invalid UUID"), "unexpected error")

	_, err = c.Network("foo")
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "invalid UUID"), "unexpected error")

}

package lochness_test

import (
	"net"
	"strings"
	"testing"
	"time"

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

// TODO: cleanup after test

func TestGuestCandidates(t *testing.T) {
	c := newContext(t)
	defer contextCleanup(t)

	s := newSubnet(t)
	hv := newHypervisor(t)

	n := c.NewNetwork()
	err := n.Save()
	h.Ok(t, err)
	err = n.AddSubnet(s)
	h.Ok(t, err)

	err = hv.AddSubnet(s, "br0")
	h.Ok(t, err)

	f := c.NewFlavor()
	f.Resources.Memory = 1024
	f.Resources.CPU = 2
	f.Resources.Disk = 8192
	err = f.Save()
	h.Ok(t, err)

	// horrible hack for testing
	hv, err = c.Hypervisor(hv.ID)
	h.Ok(t, err)

	hv.AvailableResources = lochness.Resources{
		Memory: 8192,
		CPU:    4,
		Disk:   65536,
	}

	hv.TotalResources = hv.AvailableResources
	err = hv.Save()
	h.Ok(t, err)

	// cheesy
	_, err = lochness.SetHypervisorID(hv.ID)
	h.Ok(t, err)

	err = hv.Heartbeat(9999 * time.Second)
	h.Ok(t, err)

	g := c.NewGuest()
	g.FlavorID = f.ID
	g.NetworkID = n.ID

	err = g.Save()
	h.Ok(t, err)

	candidates, err := g.Candidates(lochness.DefaultCandidateFunctions...)
	h.Ok(t, err)

	h.Equals(t, 1, len(candidates))

	h.Equals(t, hv.ID, candidates[0].ID)

	// umm what about IP?? we need to reserve an ip on this hv in proper subnet

	err = hv.AddGuest(g)
	h.Ok(t, err)
}

func TestGuestsAlias(t *testing.T) {
	_ = lochness.Guests([]*lochness.Guest{})
}

func TestFirstGuest(t *testing.T) {
	c := newContext(t)
	defer contextCleanup(t)

	f := c.NewFlavor()
	f.Resources.Memory = 1024
	f.Resources.CPU = 2
	f.Resources.Disk = 8192
	h.Ok(t, f.Save())

	g := newGuest(t)
	g.MAC, _ = net.ParseMAC("72:00:04:30:c9:e0")
	g.FlavorID = f.ID
	h.Ok(t, g.Save())

	g = newGuest(t)
	g.MAC, _ = net.ParseMAC("72:00:04:30:c9:e1")
	g.FlavorID = f.ID
	h.Ok(t, g.Save())

	g = newGuest(t)
	g.MAC, _ = net.ParseMAC("72:00:04:30:c9:e2")
	g.FlavorID = f.ID
	h.Ok(t, g.Save())

	found, err := c.FirstGuest(func(g *lochness.Guest) bool {
		return g.ID == "foo"
	})

	h.Ok(t, err)

	h.Assert(t, found == nil, "unexpected value")

	found, err = c.FirstGuest(func(g2 *lochness.Guest) bool {
		return g.ID == g2.ID
	})

	h.Ok(t, err)

	h.Assert(t, found != nil, "unexpected nil")

}

func TestGuestWithBadID(t *testing.T) {
	c := newContext(t)
	_, err := c.Guest("")
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "invalid UUID"), "unexpected error")

	_, err = c.Guest("foo")
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "invalid UUID"), "unexpected error")

}

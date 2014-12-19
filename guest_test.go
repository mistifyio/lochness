package lochness_test

import (
	"os"
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

	hv.Resources["available"] = lochness.Resources{
		Memory: 8192,
		CPU:    4,
		Disk:   65536,
	}

	hv.Resources["total"] = hv.Resources["available"]
	err = hv.Save()
	h.Ok(t, err)

	os.Setenv("TEST_HOSTNAME", hv.ID)
	err = hv.Heartbeat(9999 * time.Second)
	h.Ok(t, err)

	g := c.NewGuest()
	g.FlavorID = f.ID
	g.NetworkID = n.ID

	err = g.Save()
	h.Ok(t, err)

	candidates, err := g.Candidates()
	h.Ok(t, err)

	h.Equals(t, 1, len(candidates))

	h.Equals(t, hv.ID, candidates[0].ID)

	// umm what about IP?? we need to reserve an ip on this hv in proper subnet

	err = hv.AddGuest(g)
	h.Ok(t, err)
}

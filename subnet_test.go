package lochness_test

import (
	"net"
	"strings"
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
)

func TestNewSubnet(t *testing.T) {
	c := newContext(t)
	s := c.NewSubnet()
	h.Equals(t, 36, len(s.ID))
}

func newSubnet(t *testing.T) *lochness.Subnet {
	c := newContext(t)
	s := c.NewSubnet()

	var err error
	_, s.CIDR, err = net.ParseCIDR("10.10.10.0/24")
	h.Ok(t, err)

	s.StartRange = net.IPv4(10, 10, 10, 10)
	s.EndRange = net.IPv4(10, 10, 10, 100)

	err = s.Save()
	h.Ok(t, err)

	return s
}

func removeSubnet(t *testing.T, s *lochness.Subnet) {
	err := s.Delete()
	h.Ok(t, err)
}

func TestSubnetSaveFail(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)

	s.CIDR = nil
	err := s.Save()
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "CIDR cannot be nil"), "unexpected error message")
}

func TestSubnetSave(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)

	s.Metadata["foo"] = "bar"
	err := s.Save()
	h.Ok(t, err)
}

func TestSubnetSaveInvalidRange(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)

	s.StartRange = net.IPv4(10, 10, 11, 10)
	err := s.Save()

	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "does not contain"), "unexpected error message")

}

func TestSubnetAddresses(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)

	addresses, err := s.Addresses()
	h.Ok(t, err)
	h.Equals(t, 0, len(addresses))
}

func TestSubnetAvailibleAddresses(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)
	avail := s.AvailibleAddresses()
	h.Equals(t, 91, len(avail))
}

func reserveAddress(t *testing.T, s *lochness.Subnet) net.IP {
	ip, err := s.ReserveAddress("fake")
	h.Ok(t, err)

	h.Assert(t, strings.Contains(ip.String(), "10.10.10."), "unexpected ip address")

	avail := s.AvailibleAddresses()
	h.Equals(t, 90, len(avail))

	addresses, err := s.Addresses()
	h.Ok(t, err)
	h.Equals(t, 1, len(addresses))

	// make sure change persists
	err = s.Refresh()
	h.Ok(t, err)

	avail = s.AvailibleAddresses()
	h.Equals(t, 90, len(avail))

	return ip

}
func TestSubnetReserveAddress(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)

	_ = reserveAddress(t, s)
}

func TestSubnetReleaseAddress(t *testing.T) {
	s := newSubnet(t)
	defer removeSubnet(t, s)

	ip := reserveAddress(t, s)
	err := s.ReleaseAddress(ip)
	h.Ok(t, err)

	avail := s.AvailibleAddresses()
	h.Equals(t, 91, len(avail))

	// make sure change persists
	err = s.Refresh()
	h.Ok(t, err)

	avail = s.AvailibleAddresses()
	h.Equals(t, 91, len(avail))

	addresses, err := s.Addresses()
	h.Ok(t, err)
	h.Equals(t, 0, len(addresses))

}

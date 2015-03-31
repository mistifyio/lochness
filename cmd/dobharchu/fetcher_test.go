package main

import (
	"encoding/json"
	"net"
	"path/filepath"
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/testhelper"
)

func TestFetchHypervisors(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)

	// Create supporting objects
	n, err := testhelper.NewNetwork(f.context)
	h.Ok(t, err)
	s, err := testhelper.NewSubnet(f.context, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	h.Ok(t, err)

	// Create two hypervisors
	h1, err := testhelper.NewHypervisor(f.context, "de:ad:be:ef:7f:21", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	h.Ok(t, err)
	h2, err := testhelper.NewHypervisor(f.context, "de:ad:be:ef:7f:23", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	h.Ok(t, err)

	// Fetch and make sure they're present
	hvs, err := f.Hypervisors()
	h.Ok(t, err)
	if _, ok := hvs[h1.ID]; !ok {
		t.Error("Hypervisor #1 is missing from list")
	}
	h.Equals(t, hvs[h1.ID].MAC.String(), "de:ad:be:ef:7f:21")
	if _, ok := hvs[h2.ID]; !ok {
		t.Error("Hypervisor #2 is missing from list")
	}
	h.Equals(t, hvs[h2.ID].MAC.String(), "de:ad:be:ef:7f:23")

}

func TestFetchGuests(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)

	// Create supporting objects
	f1, err := testhelper.NewFlavor(f.context, 4, 4096, 8192)
	h.Ok(t, err)
	n, err := testhelper.NewNetwork(f.context)
	h.Ok(t, err)
	fw, err := testhelper.NewFirewallGroup(f.context)
	h.Ok(t, err)
	s, err := testhelper.NewSubnet(f.context, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	h.Ok(t, err)
	h1, err := testhelper.NewHypervisor(f.context, "de:ad:be:ef:7f:21", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	h.Ok(t, err)

	// Create two guests
	g1, err := testhelper.NewGuest(f.context, "01:23:45:67:89:ab", n, s, f1, fw, h1)
	h.Ok(t, err)
	g2, err := testhelper.NewGuest(f.context, "23:45:67:89:ab:cd", n, s, f1, fw, h1)
	h.Ok(t, err)

	// Fetch and make sure they're present
	gs, err := f.Guests()
	h.Ok(t, err)
	if _, ok := gs[g1.ID]; !ok {
		t.Error("Guest #1 is missing from list")
	}
	h.Equals(t, gs[g1.ID].MAC.String(), "01:23:45:67:89:ab")
	if _, ok := gs[g2.ID]; !ok {
		t.Error("Guest #2 is missing from list")
	}
	h.Equals(t, gs[g2.ID].MAC.String(), "23:45:67:89:ab:cd")

}

func TestFetchSubnets(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)

	// Create supporting object
	n, err := testhelper.NewNetwork(f.context)
	h.Ok(t, err)

	// Create two subnets
	s1, err := testhelper.NewSubnet(f.context, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	h.Ok(t, err)
	s2, err := testhelper.NewSubnet(f.context, "12.12.12.0/24", net.IPv4(12, 12, 12, 1), net.IPv4(12, 12, 12, 12), net.IPv4(12, 12, 12, 250), n)
	h.Ok(t, err)

	// Fetch and make sure they're present
	ss, err := f.Subnets()
	h.Ok(t, err)
	if _, ok := ss[s1.ID]; !ok {
		t.Error("Subnet #1 is missing from list")
	}
	h.Equals(t, ss[s1.ID].CIDR.String(), "10.10.10.0/24")
	if _, ok := ss[s2.ID]; !ok {
		t.Error("Subnet #2 is missing from list")
	}
	h.Equals(t, ss[s2.ID].CIDR.String(), "12.12.12.0/24")

}

func TestFetchAll(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)

	// Create objects
	f1, err := testhelper.NewFlavor(f.context, 4, 4096, 8192)
	h.Ok(t, err)
	f2, err := testhelper.NewFlavor(f.context, 6, 8192, 1024)
	h.Ok(t, err)
	n, err := testhelper.NewNetwork(f.context)
	h.Ok(t, err)
	fw, err := testhelper.NewFirewallGroup(f.context)
	h.Ok(t, err)
	s, err := testhelper.NewSubnet(f.context, "10.10.10.0/24", net.IPv4(10, 10, 10, 1), net.IPv4(10, 10, 10, 10), net.IPv4(10, 10, 10, 250), n)
	h.Ok(t, err)
	h1, err := testhelper.NewHypervisor(f.context, "de:ad:be:ef:7f:21", net.IPv4(192, 168, 100, 200), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	h.Ok(t, err)
	h2, err := testhelper.NewHypervisor(f.context, "de:ad:be:ef:7f:23", net.IPv4(192, 168, 100, 203), net.IPv4(192, 168, 100, 1), net.IPv4(255, 255, 255, 0), "br0", s)
	h.Ok(t, err)
	g1, err := testhelper.NewGuest(f.context, "01:23:45:67:89:ab", n, s, f1, fw, h1)
	h.Ok(t, err)
	g2, err := testhelper.NewGuest(f.context, "23:45:67:89:ab:cd", n, s, f1, fw, h1)
	h.Ok(t, err)
	g3, err := testhelper.NewGuest(f.context, "45:67:89:ab:cd:ef", n, s, f1, fw, h2)
	h.Ok(t, err)
	g4, err := testhelper.NewGuest(f.context, "67:89:ab:cd:ef:01", n, s, f2, fw, h2)
	h.Ok(t, err)

	// Fetch and make sure everything expected is present
	err = f.FetchAll()
	h.Ok(t, err)

	// Check hypervisors
	hvs, err := f.Hypervisors()
	h.Ok(t, err)
	if _, ok := hvs[h1.ID]; !ok {
		t.Error("Hypervisor #1 is missing from list")
	}
	h.Equals(t, hvs[h1.ID].MAC.String(), "de:ad:be:ef:7f:21")
	if _, ok := hvs[h2.ID]; !ok {
		t.Error("Hypervisor #2 is missing from list")
	}
	h.Equals(t, hvs[h2.ID].MAC.String(), "de:ad:be:ef:7f:23")

	// Check guests
	gs, err := f.Guests()
	if _, ok := gs[g1.ID]; !ok {
		t.Error("Guest #1 is missing from list")
	}
	h.Equals(t, gs[g1.ID].MAC.String(), "01:23:45:67:89:ab")
	if _, ok := gs[g2.ID]; !ok {
		t.Error("Guest #2 is missing from list")
	}
	h.Equals(t, gs[g2.ID].MAC.String(), "23:45:67:89:ab:cd")
	if _, ok := gs[g3.ID]; !ok {
		t.Error("Guest #3 is missing from list")
	}
	h.Equals(t, gs[g3.ID].MAC.String(), "45:67:89:ab:cd:ef")
	if _, ok := gs[g4.ID]; !ok {
		t.Error("Guest #4 is missing from list")
	}
	h.Equals(t, gs[g4.ID].MAC.String(), "67:89:ab:cd:ef:01")

	// Check subnet
	ss, err := f.Subnets()
	h.Ok(t, err)
	if _, ok := ss[s.ID]; !ok {
		t.Error("Subnet is missing from list")
	}
	h.Equals(t, ss[s.ID].CIDR.String(), "10.10.10.0/24")

}

func TestIntegrateHypervisorResponses(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)
	_, err := f.Hypervisors()
	h.Ok(t, err)

	// Create-hypervisor integration
	hv := f.context.NewHypervisor()
	mac := "55:55:55:55:55:55"
	hv.MAC, err = net.ParseMAC(mac)
	if err != nil {
		t.Error("Could not parse MAC '" + mac + "': " + err.Error())
	}
	hj, err := json.Marshal(hv)
	h.Ok(t, err)
	key := filepath.Join(lochness.HypervisorPath, hv.ID, "metadata")
	resp, err := f.etcdClient.Create(key, string(hj), 0)
	h.Ok(t, err)
	refresh, err := f.IntegrateResponse(resp)
	h.Ok(t, err)
	h.Equals(t, refresh, true)
	hvs, err := f.Hypervisors()
	h.Ok(t, err)
	if _, ok := hvs[hv.ID]; !ok {
		t.Error("Newly integrated hypervisor is missing from list")
	}
	h.Equals(t, hvs[hv.ID].MAC.String(), mac)

	// Delete-hypervisor integration (update requires modifiedIndex, which is not exported)
	resp, err = f.etcdClient.Delete(filepath.Join(lochness.HypervisorPath, hv.ID), true)
	h.Ok(t, err)
	refresh, err = f.IntegrateResponse(resp)
	h.Ok(t, err)
	h.Equals(t, refresh, true)
	hvs, err = f.Hypervisors()
	h.Ok(t, err)
	if _, ok := hvs[hv.ID]; ok {
		t.Error("Newly deleted hypervisor is present in list")
	}
}

func TestIntegrateGuestResponses(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)
	_, err := f.Guests()
	h.Ok(t, err)

	// Create-guest integration
	g := f.context.NewGuest()
	mac := "66:66:66:66:66:66"
	g.MAC, err = net.ParseMAC(mac)
	if err != nil {
		t.Error("Could not parse MAC '" + mac + "': " + err.Error())
	}
	gj, err := json.Marshal(g)
	h.Ok(t, err)
	key := filepath.Join(lochness.GuestPath, g.ID, "metadata")
	resp, err := f.etcdClient.Create(key, string(gj), 0)
	h.Ok(t, err)
	refresh, err := f.IntegrateResponse(resp)
	h.Ok(t, err)
	h.Equals(t, refresh, true)
	gs, err := f.Guests()
	h.Ok(t, err)
	if _, ok := gs[g.ID]; !ok {
		t.Error("Newly integrated guest is missing from list")
	}
	h.Equals(t, gs[g.ID].MAC.String(), mac)

	// Delete-guest integration
	resp, err = f.etcdClient.Delete(filepath.Join(lochness.GuestPath, g.ID), true)
	h.Ok(t, err)
	refresh, err = f.IntegrateResponse(resp)
	h.Ok(t, err)
	h.Equals(t, refresh, true)
	gs, err = f.Guests()
	h.Ok(t, err)
	if _, ok := gs[g.ID]; ok {
		t.Error("Newly deleted guest is present in list")
	}
}

func TestIntegrateSubnetResponses(t *testing.T) {

	// Setup
	f := NewFetcher("http://127.0.0.1:4001")
	defer testhelper.Cleanup(f.etcdClient)
	_, err := f.Subnets()
	h.Ok(t, err)

	// Create-subnet integration
	s := f.context.NewSubnet()
	cidr := "77.77.77.0/24"
	_, s.CIDR, err = net.ParseCIDR(cidr)
	if err != nil {
		t.Error("Could not parse CIDR '" + cidr + "': " + err.Error())
	}
	sj, err := json.Marshal(s)
	h.Ok(t, err)
	key := filepath.Join(lochness.SubnetPath, s.ID, "metadata")
	resp, err := f.etcdClient.Create(key, string(sj), 0)
	h.Ok(t, err)
	refresh, err := f.IntegrateResponse(resp)
	h.Ok(t, err)
	h.Equals(t, refresh, true)
	h.Ok(t, err)
	ss, err := f.Subnets()
	h.Ok(t, err)
	if _, ok := ss[s.ID]; !ok {
		t.Error("Newly integrated subnet is missing from list")
	}
	h.Equals(t, ss[s.ID].CIDR.String(), cidr)

	// Delete-subnet integration
	resp, err = f.etcdClient.Delete(filepath.Join(lochness.SubnetPath, s.ID), true)
	h.Ok(t, err)
	refresh, err = f.IntegrateResponse(resp)
	h.Ok(t, err)
	h.Equals(t, refresh, true)
	ss, err = f.Subnets()
	h.Ok(t, err)
	if _, ok := ss[s.ID]; ok {
		t.Error("Newly deleted subnet is present in list")
	}
}

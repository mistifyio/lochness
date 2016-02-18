package main_test

import (
	"encoding/json"
	"fmt"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness/cmd/cdhcpd"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/suite"
)

type FetcherSuite struct {
	common.Suite
	Fetcher *main.Fetcher
}

func (s *FetcherSuite) SetupSuite() {
	s.Suite.SetupSuite()
	log.SetLevel(log.ErrorLevel)
}

func (s *FetcherSuite) SetupTest() {
	s.Suite.SetupTest()
	s.Fetcher = main.NewFetcher(s.EtcdURL)

	log.SetLevel(log.FatalLevel)
}

func TestFetcher(t *testing.T) {
	suite.Run(t, new(FetcherSuite))
}

func (s *FetcherSuite) TestHypervisors() {
	hypervisor, _ := s.NewHypervisorWithGuest()
	hypervisors, err := s.Fetcher.Hypervisors()
	s.NoError(err)

	h, ok := hypervisors[hypervisor.ID]
	if !s.True(ok) {
		return
	}
	s.Equal(hypervisor.MAC, h.MAC)
}

func (s *FetcherSuite) TestGuests() {
	_, guest := s.NewHypervisorWithGuest()
	guests, err := s.Fetcher.Guests()
	s.NoError(err)
	g, ok := guests[guest.ID]
	if !s.True(ok) {
		return
	}
	s.Equal(guest.MAC, g.MAC)
}

func (s *FetcherSuite) TestSubnets() {
	subnet := s.NewSubnet()
	network := s.NewNetwork()
	_ = network.AddSubnet(subnet)

	subnets, err := s.Fetcher.Subnets()
	s.NoError(err)
	sub, ok := subnets[subnet.ID]
	if !s.True(ok) {
		return
	}
	s.Equal(subnet.StartRange, sub.StartRange)
}

func (s *FetcherSuite) TestFetchAll() {
	s.NoError(s.Fetcher.FetchAll())
	hypervisor, guest := s.NewHypervisorWithGuest()
	s.NoError(s.Fetcher.FetchAll())

	hypervisors, err := s.Fetcher.Hypervisors()
	s.NoError(err)
	_, ok := hypervisors[hypervisor.ID]
	s.True(ok)

	guests, err := s.Fetcher.Guests()
	s.NoError(err)
	_, ok = guests[guest.ID]
	s.True(ok)

	subnets, err := s.Fetcher.Subnets()
	s.NoError(err)
	_, ok = subnets[guest.SubnetID]
	s.True(ok)
}

func getResp(resp *etcd.Response, err error) *etcd.Response { return resp }

func (s *FetcherSuite) TestIntegrateResponse() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	subnet, _ := s.Context.Subnet(guest.SubnetID)

	hJSON, _ := json.Marshal(hypervisor)
	gJSON, _ := json.Marshal(guest)
	sJSON, _ := json.Marshal(subnet)

	hPath := s.EtcdPrefix + "/hypervisors/%s/metadata"
	sPath := s.EtcdPrefix + "/subnets/%s/metadata"
	gPath := s.EtcdPrefix + "/guests/%s/metadata"

	// Should fail before first fetch
	resp, err := s.EtcdClient.Get(fmt.Sprintf(hPath, hypervisor.ID), false, false)
	refresh, err := s.Fetcher.IntegrateResponse(resp)
	s.False(refresh)
	s.Error(err)

	_ = s.Fetcher.FetchAll()

	tests := []struct {
		description string
		resp        *etcd.Response
		refresh     bool
		expectedErr bool
	}{
		{
			"create wrong key",
			getResp(s.EtcdClient.Create("/foobar", "baz", 0)),
			false, true,
		},
		{
			"get hypervisor",
			getResp(s.EtcdClient.Get(fmt.Sprintf(hPath, hypervisor.ID), false, false)),
			false, false,
		},
		{
			"set hypervisor",
			getResp(s.EtcdClient.Set(fmt.Sprintf(hPath, hypervisor.ID), string(hJSON), 0)),
			true, false,
		},
		{
			"set guest",
			getResp(s.EtcdClient.Set(fmt.Sprintf(gPath, guest.ID), string(gJSON), 0)),
			true, false,
		},
		{
			"set subnet",
			getResp(s.EtcdClient.Set(fmt.Sprintf(sPath, subnet.ID), string(sJSON), 0)),
			true, false,
		},
		{
			"delete guest",
			getResp(s.EtcdClient.Delete(fmt.Sprintf(gPath, guest.ID), false)),
			true, false,
		},
		{
			"delete subnet",
			getResp(s.EtcdClient.Delete(fmt.Sprintf(sPath, subnet.ID), false)),
			true, false,
		},
		{
			"delete hypervisor",
			getResp(s.EtcdClient.Delete(fmt.Sprintf(hPath, hypervisor.ID), false)),
			true, false,
		},
		{
			"create hypervisor",
			getResp(s.EtcdClient.Create(fmt.Sprintf(hPath, hypervisor.ID), string(hJSON), 0)),
			true, false,
		},
	}

	for _, test := range tests {
		msg := common.TestMsgFunc(test.description)
		refresh, err := s.Fetcher.IntegrateResponse(test.resp)

		s.Equal(test.refresh, refresh, msg("wrong refresh conclusion"))
		if test.expectedErr {
			s.Error(err, msg("should have errored"))
		} else {
			s.NoError(err, msg("should have succeeded"))
		}
	}
}

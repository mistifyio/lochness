package main_test

import (
	"encoding/json"
	"fmt"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/cmd/cdhcpd"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/mistifyio/lochness/pkg/kv"
	"github.com/stretchr/testify/suite"
)

func TestFetcher(t *testing.T) {
	suite.Run(t, new(FetcherSuite))
}

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
	s.Fetcher = main.NewFetcher(s.KVURL)

	log.SetLevel(log.FatalLevel)
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

func (s *FetcherSuite) TestIntegrateResponse() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	subnet, _ := s.Context.Subnet(guest.SubnetID)

	hJSON, _ := json.Marshal(hypervisor)
	gJSON, _ := json.Marshal(guest)
	sJSON, _ := json.Marshal(subnet)

	hPath := s.KVPrefix + "/hypervisors/%s/metadata"
	sPath := s.KVPrefix + "/subnets/%s/metadata"
	gPath := s.KVPrefix + "/guests/%s/metadata"

	// Should fail before first fetch
	refresh, err := s.Fetcher.IntegrateResponse(kv.Event{
		Type: kv.Create,
		Key:  fmt.Sprintf(hPath, hypervisor.ID),
	})
	s.Error(err)
	s.False(refresh)

	_ = s.Fetcher.FetchAll()

	tests := []struct {
		description string
		resp        kv.Event
		refresh     bool
		expectedErr bool
	}{
		{"create wrong key",
			kv.Event{
				Type: kv.Create,
				Key:  "/foobar/baz",
			}, false, true,
		},
		{"set hypervisor",
			kv.Event{
				Type:  kv.Update,
				Key:   fmt.Sprintf(hPath, hypervisor.ID),
				Value: kv.Value{Data: hJSON},
			}, true, false,
		},
		{"set guest",
			kv.Event{
				Type:  kv.Update,
				Key:   fmt.Sprintf(gPath, guest.ID),
				Value: kv.Value{Data: gJSON},
			}, true, false,
		},
		{"set subnet",
			kv.Event{
				Type:  kv.Update,
				Key:   fmt.Sprintf(sPath, subnet.ID),
				Value: kv.Value{Data: sJSON},
			}, true, false,
		},
		{"delete guest",
			kv.Event{
				Type: kv.Delete,
				Key:  fmt.Sprintf(gPath, guest.ID),
			}, true, false,
		},
		{"delete subnet",
			kv.Event{
				Type: kv.Delete,
				Key:  fmt.Sprintf(sPath, subnet.ID),
			}, true, false,
		},
		{"delete hypervisor",
			kv.Event{
				Type: kv.Delete,
				Key:  fmt.Sprintf(hPath, hypervisor.ID),
			}, true, false,
		},
		{"create hypervisor",
			kv.Event{
				Type:  kv.Create,
				Key:   fmt.Sprintf(hPath, hypervisor.ID),
				Value: kv.Value{Data: hJSON},
			}, true, false,
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

package lochness_test

import (
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	common.Suite
}

func TestNetworkTestSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func (s *NetworkTestSuite) TestNewNetwork() {
	network := s.Context.NewNetwork()
	s.NotNil(uuid.Parse(network.ID))
}

func (s *NetworkTestSuite) TestNework() {
	network := s.NewNetwork()

	tests := []struct {
		description string
		ID          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid ID", "adf", true},
		{"nonexistant ID", uuid.New(), true},
		{"real ID", network.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		n, err := s.Context.Network(test.ID)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(n, msg("failure shouldn't return a network"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(network, n), msg("success should return correct data"))
		}
	}
}

func (s *NetworkTestSuite) TestRefresh() {
	network := s.NewNetwork()
	networkCopy := &lochness.Network{}
	*networkCopy = *network
	_ = network.AddSubnet(s.NewSubnet())

	_ = network.Save()
	s.NoError(networkCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(network, networkCopy), "refresh should pull new data")

	NewNetwork := s.Context.NewNetwork()
	s.Error(NewNetwork.Refresh(), "unsaved network refresh should fail")
}

func (s *NetworkTestSuite) TestValidate() {
	tests := []struct {
		description string
		ID          string
		expectedErr bool
	}{
		{"missing ID", "", true},
		{"non uuid ID", "asdf", true},
		{"uuid ID", uuid.New(), false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		n := &lochness.Network{ID: test.ID}
		err := n.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *NetworkTestSuite) TestSave() {
	goodNetwork := s.Context.NewNetwork()

	clobberNetwork := *goodNetwork

	tests := []struct {
		description string
		network     *lochness.Network
		expectedErr bool
	}{
		{"invalid network", &lochness.Network{}, true},
		{"valid network", goodNetwork, false},
		{"existing network", goodNetwork, false},
		{"existing network clobber", &clobberNetwork, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.network.Save()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

func (s *NetworkTestSuite) TestAddSubnet() {
	tests := []struct {
		description string
		network     *lochness.Network
		subnet      *lochness.Subnet
		expectedErr bool
	}{
		{"nonexisting network, nonexisting subnet", s.Context.NewNetwork(), s.Context.NewSubnet(), true},
		{"existing network, nonexisting subnet", s.NewNetwork(), s.Context.NewSubnet(), true},
		{"nonexisting network, existing subnet", s.Context.NewNetwork(), s.NewSubnet(), true},
		{"existing network and subnet", s.NewNetwork(), s.NewSubnet(), false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.network.AddSubnet(test.subnet)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Len(test.network.Subnets(), 0, msg("fail should not add subnet to network"))
			s.Empty(test.subnet.NetworkID, msg("fail should not add network to subnet"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Len(test.network.Subnets(), 1, msg("fail should add subnet to network"))
			s.NotNil(uuid.Parse(test.subnet.NetworkID), msg("fail should add network to subnet"))
		}
	}
}

func (s *NetworkTestSuite) TestSubnets() {
	network := s.NewNetwork()
	_ = network.AddSubnet(s.NewSubnet())

	s.Len(network.Subnets(), 1)
}

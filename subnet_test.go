package lochness_test

import (
	"encoding/json"
	"errors"
	"net"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestSubnet(t *testing.T) {
	suite.Run(t, new(SubnetSuite))
}

type SubnetSuite struct {
	common.Suite
}

func (s *SubnetSuite) TestNewSubnet() {
	subnet := s.Context.NewSubnet()
	s.NotNil(uuid.Parse(subnet.ID))
}

func (s *SubnetSuite) TestSubnet() {
	subnet := s.NewSubnet()

	tests := []struct {
		description string
		ID          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid ID", "adf", true},
		{"nonexistant ID", uuid.New(), true},
		{"real ID", subnet.ID, false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		n, err := s.Context.Subnet(test.ID)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(n, msg("failure shouldn't return a subnet"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(subnet, n), msg("success should return correct data"))
		}
	}
}

func (s *SubnetSuite) TestRefresh() {
	subnet := s.NewSubnet()
	subnetCopy := &lochness.Subnet{}
	*subnetCopy = *subnet

	network := s.NewNetwork()
	_ = network.AddSubnet(subnet)
	_, _ = subnet.ReserveAddress("foobar")

	_ = subnet.Save()
	s.NoError(subnetCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(subnet, subnetCopy), "refresh should pull new data")

	NewSubnet := s.Context.NewSubnet()
	s.Error(NewSubnet.Refresh(), "unsaved subnet refresh should fail")
}

func (s *SubnetSuite) TestJSON() {
	subnet := s.NewSubnet()

	subnetBytes, err := json.Marshal(subnet)
	s.NoError(err)

	subnetFromJSON := &lochness.Subnet{}
	s.NoError(json.Unmarshal(subnetBytes, subnetFromJSON))
	s.Equal(subnet.ID, subnetFromJSON.ID)
	s.Equal(subnet.CIDR, subnetFromJSON.CIDR)
	s.Equal(subnet.StartRange, subnetFromJSON.StartRange)
}

func (s *SubnetSuite) TestValidate() {
	tests := []struct {
		description string
		id          string
		cidr        string
		start       string
		end         string
		expectedErr bool
	}{
		{"missing ID", "", "192.168.100.1/24", "192.168.100.2", "192.168.100.3", true},
		{"invalid ID", "asdf", "192.168.100.1/24", "192.168.100.2", "192.168.100.3", true},
		{"missing cidr ", uuid.New(), "", "192.168.100.2", "192.168.100.3", true},
		{"missing start", uuid.New(), "192.168.100.1/24", "", "192.168.100.3", true},
		{"outside range start", uuid.New(), "192.168.100.1/24", "192.168.200.2", "192.168.100.3", true},
		{"missing end", uuid.New(), "192.168.100.1/24", "192.168.100.2", "", true},
		{"outside range end", uuid.New(), "192.168.100.1/24", "192.168.100.2", "192.168.200.3", true},
		{"end before start", uuid.New(), "192.168.100.1/24", "192.168.100.3", "192.168.100.2", true},
		{"all fields", uuid.New(), "192.168.100.1/24", "192.168.100.2", "192.168.100.3", false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		sub := &lochness.Subnet{
			ID:         test.id,
			StartRange: net.ParseIP(test.start),
			EndRange:   net.ParseIP(test.end),
		}
		_, sub.CIDR, _ = net.ParseCIDR(test.cidr)

		err := sub.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *SubnetSuite) TestSave() {
	subnet := s.NewSubnet()
	subnetCopy := &lochness.Subnet{}
	*subnetCopy = *subnet
	network := s.NewNetwork()
	_ = network.AddSubnet(subnet)

	_ = subnet.Save()
	s.NoError(subnetCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(subnet, subnetCopy), "refresh should pull new data")

	NewSubnet := s.Context.NewSubnet()
	s.Error(NewSubnet.Refresh(), "unsaved subnet refresh should fail")
}

func (s *SubnetSuite) TestDelete() {
	subnet := s.NewSubnet()
	network := s.NewNetwork()
	_ = network.AddSubnet(subnet)

	invalidSub := s.Context.NewSubnet()
	invalidSub.ID = "asdf"

	tests := []struct {
		description string
		sub         *lochness.Subnet
		expectedErr bool
	}{
		{"invalid subnet", invalidSub, true},
		{"existing subnet", subnet, false},
		{"nonexistant subnet", s.Context.NewSubnet(), true},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		err := test.sub.Delete()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
			_ = network.Refresh()
			s.Len(network.Subnets(), 0, msg("should remove subnet link"))
		}
	}
}

func (s *SubnetSuite) TestAvailableAddresses() {
	subnet := s.NewSubnet()
	addresses := subnet.AvailableAddresses()
	s.Len(addresses, 9, "all addresses should be available")
	s.Equal(subnet.StartRange, addresses[0], "should start at the beginning of the range")
	s.Equal(subnet.EndRange, addresses[len(addresses)-1], "should end at the end of the array")
}

func (s *SubnetSuite) TestReserveAddress() {
	subnet := s.NewSubnet()
	n := len(subnet.AvailableAddresses())
	for i := 0; i <= n; i++ {
		msg := s.Messager("attempt " + string(i))
		ip, err := subnet.ReserveAddress("foo")
		if i < n {
			s.NoError(err, msg("should succeed when addresses available"))
			s.NotNil(ip, msg("should return ip when addresses available"))
			s.Len(subnet.AvailableAddresses(), n-i-1, msg("should update available addresses"))
		} else {
			s.Error(err, msg("should fail when no addresses available"))
			s.Nil(ip, msg("should not return ip when no addresses available"))
			s.Len(subnet.AvailableAddresses(), 0, msg("should have no available addresses"))
		}
	}
}

func (s *SubnetSuite) TestReleaseAddress() {
	subnet := s.NewSubnet()
	ip, _ := subnet.ReserveAddress("foobar")
	n := len(subnet.AvailableAddresses())

	s.Error(subnet.ReleaseAddress(net.ParseIP("192.168.0.1")))
	s.Len(subnet.AvailableAddresses(), n)

	s.NoError(subnet.ReleaseAddress(ip))
	s.Len(subnet.AvailableAddresses(), n+1)
}

func (s *SubnetSuite) TestAddresses() {
	subnet := s.NewSubnet()
	addresses := subnet.AvailableAddresses()

	ip, _ := subnet.ReserveAddress("foobar")

	addressMap := subnet.Addresses()
	for _, address := range addresses {
		val, ok := addressMap[address.String()]
		if ip.Equal(address) {
			s.True(ok)
			s.Equal("foobar", val)
		} else {
			s.False(ok)
		}
	}
}

func (s *SubnetSuite) TestForEachSubnet() {
	subnet := s.NewSubnet()
	subnet2 := s.NewSubnet()
	expectedFound := map[string]bool{
		subnet.ID:  true,
		subnet2.ID: true,
	}

	resultFound := make(map[string]bool)

	err := s.Context.ForEachSubnet(func(sub *lochness.Subnet) error {
		resultFound[sub.ID] = true
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(expectedFound, resultFound))

	returnErr := errors.New("an error")
	err = s.Context.ForEachSubnet(func(sub *lochness.Subnet) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

package lochness_test

import (
	"errors"
	"net"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SubnetTestSuite struct {
	ContextTestSuite
}

func TestSubnetTestSuite(t *testing.T) {
	suite.Run(t, new(SubnetTestSuite))
}

func (s *SubnetTestSuite) TestNewSubnet() {
	subnet := s.Context.NewSubnet()
	s.NotNil(uuid.Parse(subnet.ID))
}

func (s *SubnetTestSuite) TestSubnet() {
	subnet := s.newSubnet()

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
		msg := testMsgFunc(test.description)
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

func (s *SubnetTestSuite) TestRefresh() {
	subnet := s.newSubnet()
	subnetCopy := &lochness.Subnet{}
	*subnetCopy = *subnet

	network := s.newNetwork()
	network.AddSubnet(subnet)
	_, _ = subnet.ReserveAddress("foobar")

	_ = subnet.Save()
	s.NoError(subnetCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(subnet, subnetCopy), "refresh should pull new data")

	newSubnet := s.Context.NewSubnet()
	s.Error(newSubnet.Refresh(), "unsaved subnet refresh should fail")
}

func (s *SubnetTestSuite) TestJSON() {
}

func (s *SubnetTestSuite) TestValidate() {
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
		msg := testMsgFunc(test.description)
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

func (s *SubnetTestSuite) TestSave() {
	subnet := s.newSubnet()
	subnetCopy := &lochness.Subnet{}
	*subnetCopy = *subnet
	network := s.newNetwork()
	network.AddSubnet(subnet)

	_ = subnet.Save()
	s.NoError(subnetCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(subnet, subnetCopy), "refresh should pull new data")

	newSubnet := s.Context.NewSubnet()
	s.Error(newSubnet.Refresh(), "unsaved subnet refresh should fail")
}

func (s *SubnetTestSuite) TestDelete() {
	subnet := s.newSubnet()
	network := s.newNetwork()
	network.AddSubnet(subnet)

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
		msg := testMsgFunc(test.description)
		err := test.sub.Delete()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
			network.Refresh()
			s.Len(network.Subnets(), 0, msg("should remove subnet link"))
		}
	}
}

func (s *SubnetTestSuite) TestAvailibleAddresses() {
	subnet := s.newSubnet()
	addresses := subnet.AvailibleAddresses()
	s.Len(addresses, 9, "all addresses should be available")
	s.Equal(subnet.StartRange, addresses[0], "should start at the beginning of the range")
	s.Equal(subnet.EndRange, addresses[len(addresses)-1], "should end at the end of the array")
}

func (s *SubnetTestSuite) TestReserveAddress() {
	subnet := s.newSubnet()
	n := len(subnet.AvailibleAddresses())
	for i := 0; i <= n; i++ {
		msg := testMsgFunc("attempt " + string(i))
		ip, err := subnet.ReserveAddress("foo")
		if i < n {
			s.NoError(err, msg("should succeed when addresses availible"))
			s.NotNil(ip, msg("should return ip when addresses availible"))
			s.Len(subnet.AvailibleAddresses(), n-i-1, msg("should update availible addresses"))
		} else {
			s.Error(err, msg("should fail when no addresses availible"))
			s.Nil(ip, msg("should not return ip when no addresses availible"))
			s.Len(subnet.AvailibleAddresses(), 0, msg("should have no available addresses"))
		}
	}
}

func (s *SubnetTestSuite) TestReleaseAddress() {
	subnet := s.newSubnet()
	ip, _ := subnet.ReserveAddress("foobar")
	n := len(subnet.AvailibleAddresses())

	s.Error(subnet.ReleaseAddress(net.ParseIP("192.168.0.1")))
	s.Len(subnet.AvailibleAddresses(), n)

	s.NoError(subnet.ReleaseAddress(ip))
	s.Len(subnet.AvailibleAddresses(), n+1)
}

func (s *SubnetTestSuite) TestAddresses() {
	subnet := s.newSubnet()
	addresses := subnet.AvailibleAddresses()

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

func (s *SubnetTestSuite) TestForEachSubnet() {
	subnet := s.newSubnet()
	subnet2 := s.newSubnet()
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

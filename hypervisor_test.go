package lochness_test

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/common_test"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HypervisorTestSuite struct {
	ct.CommonTestSuite
}

func TestHypervisorTestSuite(t *testing.T) {
	suite.Run(t, new(HypervisorTestSuite))
}

func (s *HypervisorTestSuite) TestJSON() {
	hypervisor, _ := s.NewHypervisorWithGuest()

	hypervisorBytes, err := json.Marshal(hypervisor)
	s.NoError(err)

	hypervisorFromJSON := &lochness.Hypervisor{}
	s.NoError(json.Unmarshal(hypervisorBytes, hypervisorFromJSON))
	s.Equal(hypervisor.ID, hypervisorFromJSON.ID)
	s.Equal(hypervisor.Metadata, hypervisorFromJSON.Metadata)
	s.Equal(hypervisor.IP, hypervisorFromJSON.IP)
	s.Equal(hypervisor.Netmask, hypervisorFromJSON.Netmask)
	s.Equal(hypervisor.Gateway, hypervisorFromJSON.Gateway)
	s.Equal(hypervisor.MAC, hypervisorFromJSON.MAC)
	s.Equal(hypervisor.TotalResources, hypervisorFromJSON.TotalResources)
	s.Equal(hypervisor.AvailableResources, hypervisorFromJSON.AvailableResources)
	s.Equal(hypervisor.Config, hypervisorFromJSON.Config)
}

func (s *HypervisorTestSuite) TestNewHypervisor() {
	hypervisor := s.Context.NewHypervisor()
	s.NotNil(uuid.Parse(hypervisor.ID))
}

func (s *HypervisorTestSuite) TestHypervisor() {
	hypervisor := s.NewHypervisor()

	tests := []struct {
		description string
		ID          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid ID", "adf", true},
		{"nonexistant ID", uuid.New(), true},
		{"real ID", hypervisor.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		h, err := s.Context.Hypervisor(test.ID)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(h, msg("failure shouldn't return a hypervisor"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(hypervisor, h), msg("success should return correct data"))
		}
	}
}

func (s *HypervisorTestSuite) TestRefresh() {
	hypervisor, _ := s.NewHypervisorWithGuest()
	hypervisorCopy := &lochness.Hypervisor{}
	*hypervisorCopy = *hypervisor
	_, _ = lochness.SetHypervisorID(hypervisor.ID)
	_ = hypervisor.Heartbeat(60 * time.Second)

	_ = hypervisor.Save()
	s.NoError(hypervisorCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(hypervisor, hypervisorCopy), "refresh should pull new data")

	NewHypervisor := s.Context.NewHypervisor()
	s.Error(NewHypervisor.Refresh(), "unsaved hypervisor refresh should fail")
}

func (s *HypervisorTestSuite) TestGetAndSetHypervisorID() {
	// hostname only case will only pass if it's a uuid. Determine if this
	// machine's hostname will pass.
	hostname, _ := os.Hostname()
	var hostnameExpectedErr bool
	if uuid.Parse(hostname) == nil {
		hostnameExpectedErr = true
		hostname = ""
	}

	hypervisorID := lochness.GetHypervisorID()

	tests := []struct {
		description string
		id          string
		env         string
		expectedID  string
		expectedErr bool
	}{
		{"hostname only", "", "", hostname, hostnameExpectedErr},
		{"non-uuid id, no env", "asdf", "", "", true},
		{"uuid id, no env", "4f2b89d8-79e1-4ee6-9ca6-4d41856b507b", "", "4f2b89d8-79e1-4ee6-9ca6-4d41856b507b", false},
		{"no id, non-uuid env", "", "asdf", "", true},
		{"no id, uuid id", "", "717265be-edcd-4131-9adf-1aae1852c9bd", "717265be-edcd-4131-9adf-1aae1852c9bd", false},
		{"non-uuid id, non-uuid env", "asdf", "asdf", "", true},
		{"uuid id, non-uuid env", "62d6b8ea-66de-4061-99ff-aa3792968c0d", "asdf", "62d6b8ea-66de-4061-99ff-aa3792968c0d", false},
		{"non-uuid id, uuid env", "asdf", "ad126074-69c6-41a3-8cb6-d6b71947c74c", "", true},
		{"uuid id, uuid env", "08270977-3cf2-4fb9-89e4-a7ca7435aa8c", "769bf032-2cfb-47ee-8e28-9d6aa856eca6", "08270977-3cf2-4fb9-89e4-a7ca7435aa8c", false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		_ = os.Setenv("HYPERVISOR_ID", test.env)
		id, err := lochness.SetHypervisorID(test.id)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Empty(id, msg("should return empty id"))
			s.Equal(hypervisorID, lochness.GetHypervisorID(), msg("should not update the hypervisorID"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Equal(test.expectedID, id, msg("should return expected id"))
			hypervisorID = lochness.GetHypervisorID()
			s.Equal(test.expectedID, hypervisorID, msg("should update the hypervisorID"))
		}
	}
}

func (s *HypervisorTestSuite) TestVerifyOnHV() {
	hypervisor := s.NewHypervisor()

	s.Error(hypervisor.VerifyOnHV())
	_, _ = lochness.SetHypervisorID(hypervisor.ID)
	s.NoError(hypervisor.VerifyOnHV())
}

func (s *HypervisorTestSuite) TestUpdateResources() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	_ = hypervisor.SetConfig("guestDiskDir", "/")

	flavor, _ := s.Context.Flavor(guest.FlavorID)
	_, _ = lochness.SetHypervisorID(hypervisor.ID)
	// Reset Resources
	hypervisor.TotalResources = lochness.Resources{}
	hypervisor.AvailableResources = lochness.Resources{}

	s.NoError(hypervisor.UpdateResources())
	tr := hypervisor.TotalResources
	ar := hypervisor.AvailableResources
	s.NotEqual(0, tr.Memory)
	s.NotEqual(0, tr.Disk)
	s.NotEqual(0, tr.CPU)
	s.Equal(tr.Memory-flavor.Memory, ar.Memory)
	s.Equal(tr.Disk-flavor.Disk, ar.Disk)
	// Total and available CPU are currently equal
	s.Equal(tr.CPU, ar.CPU)

	loadedHypervisor, _ := s.Context.Hypervisor(hypervisor.ID)
	tr = loadedHypervisor.TotalResources
	ar = loadedHypervisor.AvailableResources
	s.True(assert.ObjectsAreEqual(hypervisor.TotalResources, loadedHypervisor.TotalResources))
	s.True(assert.ObjectsAreEqual(hypervisor.AvailableResources, loadedHypervisor.AvailableResources))
}

func (s *HypervisorTestSuite) TestValidate() {
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
		h := &lochness.Hypervisor{ID: test.ID}
		err := h.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *HypervisorTestSuite) TestSave() {
	goodHypervisor := s.Context.NewHypervisor()

	clobberHypervisor := &lochness.Hypervisor{}
	*clobberHypervisor = *goodHypervisor

	tests := []struct {
		description string
		hypervisor  *lochness.Hypervisor
		expectedErr bool
	}{
		{"invalid hypervisor", &lochness.Hypervisor{}, true},
		{"valid hypervisor", goodHypervisor, false},
		{"existing hypervisor", goodHypervisor, false},
		{"existing hypervisor clobber", clobberHypervisor, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.hypervisor.Save()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

func (s *HypervisorTestSuite) TestAddSubnet() {
	tests := []struct {
		description string
		Hypervisor  *lochness.Hypervisor
		subnet      *lochness.Subnet
		expectedErr bool
	}{
		{"nonexisting Hypervisor, nonexisting subnet", s.Context.NewHypervisor(), s.Context.NewSubnet(), true},
		{"existing Hypervisor, nonexisting subnet", s.NewHypervisor(), s.Context.NewSubnet(), true},
		{"nonexisting Hypervisor, existing subnet", s.Context.NewHypervisor(), s.NewSubnet(), true},
		{"existing Hypervisor and subnet", s.NewHypervisor(), s.NewSubnet(), false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.Hypervisor.AddSubnet(test.subnet, "mistify0")
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Len(test.Hypervisor.Subnets(), 0, msg("fail should not add subnet to Hypervisor"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Len(test.Hypervisor.Subnets(), 1, msg("fail should add subnet to Hypervisor"))
		}
	}
}

func (s *HypervisorTestSuite) TestRemoveSubnet() {
	subnet := s.NewSubnet()
	hypervisor := s.NewHypervisor()
	_ = hypervisor.AddSubnet(subnet, "mistify0")

	tests := []struct {
		description string
		h           *lochness.Hypervisor
		s           *lochness.Subnet
		expectedErr bool
	}{
		{"nonexisting hypervisor, nonexisting subnet", s.Context.NewHypervisor(), s.Context.NewSubnet(), true},
		{"existing hypervisor, nonexisting subnet", hypervisor, s.Context.NewSubnet(), true},
		{"nonexisting hypervisor, existing subnet", s.Context.NewHypervisor(), subnet, true},
		{"existing hypervisor and subnet", hypervisor, subnet, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		hLen := len(test.h.Subnets())

		err := test.h.RemoveSubnet(test.s)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Len(test.h.Subnets(), hLen, msg("fail should not add subnet to hypervisor"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Len(test.h.Subnets(), hLen-1, msg("fail should add subnet to hypervisor"))
		}
	}
}

func (s *HypervisorTestSuite) TestSubnets() {
	hypervisor := s.NewHypervisor()
	_ = hypervisor.AddSubnet(s.NewSubnet(), "mistify0")

	s.Len(hypervisor.Subnets(), 1)
}

func (s *HypervisorTestSuite) TestHeartbeatAndIsAlive() {
	hypervisor := s.NewHypervisor()
	s.Error(hypervisor.Heartbeat(60 * time.Second))
	s.False(hypervisor.IsAlive())
	_, _ = lochness.SetHypervisorID(hypervisor.ID)
	s.NoError(hypervisor.Heartbeat(60 * time.Second))
	s.True(hypervisor.IsAlive())
}

func (s *HypervisorTestSuite) TestAddGuest() {
	guest := s.NewGuest()
	hypervisor := s.NewHypervisor()
	subnet := s.NewSubnet()
	network, _ := s.Context.Network(guest.NetworkID)
	_ = network.AddSubnet(subnet)
	_ = hypervisor.AddSubnet(subnet, "mistify0")

	tests := []struct {
		description string
		guest       *lochness.Guest
		hypervisor  *lochness.Hypervisor
		expectedErr bool
	}{
		{"neither exist", s.Context.NewGuest(), s.Context.NewHypervisor(), true},
		{"guest exist", s.NewGuest(), s.Context.NewHypervisor(), true},
		{"hypervisor exist", s.Context.NewGuest(), s.NewHypervisor(), true},
		{"both exist, wrong network", s.NewGuest(), s.NewHypervisor(), true},
		{"both exist, right network", guest, hypervisor, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.hypervisor.AddGuest(test.guest)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Empty(test.guest.HypervisorID, msg("should not set hypervisor id"))
			s.Len(test.hypervisor.Guests(), 0, msg("should not add to guest list"))
		} else {
			s.NoError(err, msg("should pass"))
			s.Equal(test.hypervisor.ID, test.guest.HypervisorID, msg("should set hypervisor id"))
			s.Len(test.hypervisor.Guests(), 1, msg("should add to guest list"))
		}
	}
}

func (s *HypervisorTestSuite) TestRemoveGuest() {
	hypervisor, guest := s.NewHypervisorWithGuest()

	s.Error(hypervisor.RemoveGuest(s.NewGuest()))
	s.Equal(hypervisor.ID, guest.HypervisorID)
	s.Len(hypervisor.Guests(), 1)

	s.NoError(hypervisor.RemoveGuest(guest))
	s.Empty(guest.HypervisorID)
	s.Len(hypervisor.Guests(), 0)
}

func (s *HypervisorTestSuite) TestGuests() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	guests := hypervisor.Guests()
	s.Len(guests, 1)
	s.Equal(guest.ID, guests[0])
}

func (s *HypervisorTestSuite) TestForEachGuest() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	guest2 := s.NewGuest()
	guest2.NetworkID = guest.NetworkID
	_ = hypervisor.AddGuest(guest2)

	expectedFound := map[string]bool{
		guest.ID:  true,
		guest2.ID: true,
	}

	resultFound := make(map[string]bool)

	err := hypervisor.ForEachGuest(func(g *lochness.Guest) error {
		resultFound[g.ID] = true
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(expectedFound, resultFound))

	returnErr := errors.New("an error")
	err = hypervisor.ForEachGuest(func(g *lochness.Guest) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

func (s *HypervisorTestSuite) TestFirstHypervisor() {
	_ = s.NewHypervisor()
	_ = s.NewHypervisor()
	h, err := s.Context.FirstHypervisor(func(h *lochness.Hypervisor) bool {
		return true
	})
	s.NoError(err)
	s.NotNil(h)
}

func (s *HypervisorTestSuite) TestForEachHypervisor() {
	hypervisor := s.NewHypervisor()
	hypervisor2 := s.NewHypervisor()
	expectedFound := map[string]bool{
		hypervisor.ID:  true,
		hypervisor2.ID: true,
	}

	resultFound := make(map[string]bool)

	err := s.Context.ForEachHypervisor(func(h *lochness.Hypervisor) error {
		resultFound[h.ID] = true
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(expectedFound, resultFound))

	returnErr := errors.New("an error")
	err = s.Context.ForEachHypervisor(func(h *lochness.Hypervisor) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

func (s *HypervisorTestSuite) TestSetConfig() {
	hypervisor := s.NewHypervisor()

	tests := []struct {
		description string
		key         string
		value       string
		expectedErr bool
	}{
		{"empty key", "", "bar", true},
		{"empty value", "bar", "", false},
		{"key and value", "foo", "bar", false},
		{"already set", "foo", "baz", false},
		{"nested key", "foobar/baz", "bang", false},
	}

	for _, test := range tests {
		err := hypervisor.SetConfig(test.key, test.value)
		if test.expectedErr {
			s.Error(err, test.description)
		} else {
			s.NoError(err, test.description)
		}
	}
}

func (s *HypervisorTestSuite) TestDestroy() {
	blank := s.Context.NewHypervisor()
	blank.ID = ""
	hypervisorWithGuest, _ := s.NewHypervisorWithGuest()

	tests := []struct {
		description string
		h           *lochness.Hypervisor
		expectedErr bool
	}{
		{"invalid hypervisor", blank, true},
		{"existing hypervisor", s.NewHypervisor(), false},
		{"nonexistant hypervisor", s.Context.NewHypervisor(), true},
		{"hypervisor with guest", hypervisorWithGuest, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.h.Destroy()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

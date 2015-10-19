package lochness_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/mistifyio/lochness"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type GuestTestSuite struct {
	CommonTestSuite
}

func TestGuestTestSuite(t *testing.T) {
	suite.Run(t, new(GuestTestSuite))
}

func (s *GuestTestSuite) TestJSON() {
	guest := s.newGuest()

	guestBytes, err := json.Marshal(guest)
	s.NoError(err)

	guestFromJSON := &lochness.Guest{}
	s.NoError(json.Unmarshal(guestBytes, guestFromJSON))
	s.Equal(guest.MAC, guestFromJSON.MAC)
	s.Equal(guest.IP, guestFromJSON.IP)
}

func (s *GuestTestSuite) TestNewGuest() {
	guest := s.Context.NewGuest()
	s.NotNil(uuid.Parse(guest.ID))
}

func (s *GuestTestSuite) TestGuest() {
	guest := s.newGuest()

	tests := []struct {
		description string
		ID          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid ID", "adf", true},
		{"nonexistant ID", uuid.New(), true},
		{"real ID", guest.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		g, err := s.Context.Guest(test.ID)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(g, msg("failure shouldn't return a guest"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(guest, g), msg("success should return correct data"))
		}
	}
}

func (s *GuestTestSuite) TestRefresh() {
	guest := s.newGuest()
	guestCopy := &lochness.Guest{}
	*guestCopy = *guest

	_ = guest.Save()
	s.NoError(guestCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(guest, guestCopy), "refresh should pull new data")

	newGuest := s.Context.NewGuest()
	s.Error(newGuest.Refresh(), "unsaved guest refresh should fail")
}

func (s *GuestTestSuite) TestValidate() {
	mac, _ := net.ParseMAC("4C:3F:B1:7E:54:64")
	tests := []struct {
		description string
		id          string
		flavor      string
		network     string
		mac         net.HardwareAddr
		expectedErr bool
	}{
		{"missing id", "", uuid.New(), uuid.New(), mac, true},
		{"non uuid id", "asdf", uuid.New(), uuid.New(), mac, true},
		{"missing flavor", uuid.New(), "", uuid.New(), mac, true},
		{"non uuid flavor", uuid.New(), "asdf", uuid.New(), mac, true},
		{"missing subnet", uuid.New(), uuid.New(), "", mac, true},
		{"non uuid subnet", uuid.New(), uuid.New(), "", mac, true},
		{"missing mac", uuid.New(), uuid.New(), uuid.New(), nil, true},
		{"all uuid", uuid.New(), uuid.New(), uuid.New(), mac, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		g := &lochness.Guest{
			ID:        test.id,
			FlavorID:  test.flavor,
			NetworkID: test.network,
			MAC:       test.mac,
		}
		err := g.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *GuestTestSuite) TestSave() {
	goodGuest := s.Context.NewGuest()
	flavor := s.newFlavor()
	network := s.newNetwork()
	mac, _ := net.ParseMAC("4C:3F:B1:7E:54:64")
	goodGuest.FlavorID = flavor.ID
	goodGuest.NetworkID = network.ID
	goodGuest.MAC = mac

	clobberGuest := &lochness.Guest{}
	*clobberGuest = *goodGuest

	tests := []struct {
		description string
		guest       *lochness.Guest
		expectedErr bool
	}{
		{"invalid guest", &lochness.Guest{}, true},
		{"valid guest", goodGuest, false},
		{"existing guest", goodGuest, false},
		{"existing guest clobber", clobberGuest, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.guest.Save()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

func (s *GuestTestSuite) TestDestroy() {
	blank := s.Context.NewGuest()
	blank.ID = ""

	hypervisor, guest := s.newHypervisorWithGuest()
	tests := []struct {
		description string
		g           *lochness.Guest
		expectedErr bool
	}{
		{"invalid guest", blank, true},
		{"existing guest not on hypervisor", s.newGuest(), false},
		{"existing guest on hypervisor", guest, false},
		{"nonexistant guest", s.Context.NewGuest(), true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.g.Destroy()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
			if test.g.HypervisorID != "" {
				_ = hypervisor.Refresh()
				s.Len(hypervisor.Guests(), 0, msg("should have been removed from hypervisor"))
			}
		}
	}
}

func (s *GuestTestSuite) TestCandidates() {
	guest := s.newGuest()
	subnet := s.newSubnet()
	network, _ := s.Context.Network(guest.NetworkID)
	_ = network.AddSubnet(subnet)

	hypervisors := lochness.Hypervisors{
		s.newHypervisor(), // Not alive
		s.newHypervisor(), // No correct subnet
		s.newHypervisor(), // Not enough resources
		s.newHypervisor(), // Everything
	}
	for i := 0; i < len(hypervisors); i++ {
		if i != 0 {
			_, _ = lochness.SetHypervisorID(hypervisors[i].ID)
			_ = hypervisors[i].Heartbeat(60 * time.Second)
		}
		if i != 1 {
			_ = hypervisors[i].AddSubnet(subnet, "mistify0")
		}
		if i == 2 {
			hypervisors[i].AvailableResources = lochness.Resources{}
			_ = hypervisors[i].Save()
		}
	}

	candidates, err := guest.Candidates(
		lochness.CandidateIsAlive,
		lochness.CandidateHasResources,
		lochness.CandidateHasSubnet,
	)
	s.NoError(err)
	s.Len(candidates, 1)
	s.Equal(hypervisors[3].ID, candidates[0].ID)
}

func (s *GuestTestSuite) TestCandidateIsAlive() {
	guest := s.newGuest()
	hypervisors := lochness.Hypervisors{
		s.newHypervisor(),
		s.newHypervisor(),
	}
	_, _ = lochness.SetHypervisorID(hypervisors[1].ID)
	_ = hypervisors[1].Heartbeat(60 * time.Second)

	candidates, err := lochness.CandidateIsAlive(guest, hypervisors)
	s.NoError(err)
	s.Len(candidates, 1)
	s.Equal(hypervisors[1].ID, candidates[0].ID)
}

func (s *GuestTestSuite) TestCandidateHasResources() {
	guest := s.newGuest()
	hypervisors := lochness.Hypervisors{
		s.newHypervisor(),
		s.newHypervisor(),
	}
	hypervisors[0].AvailableResources = lochness.Resources{}

	candidates, err := lochness.CandidateHasResources(guest, hypervisors)
	s.NoError(err)
	s.Len(candidates, 1)
	s.Equal(hypervisors[1].ID, candidates[0].ID)
}

func (s *GuestTestSuite) TestCandidateHasSubnet() {
	hypervisor, guest := s.newHypervisorWithGuest()
	hypervisors := lochness.Hypervisors{
		s.newHypervisor(),
		hypervisor,
	}

	candidates, err := lochness.CandidateHasSubnet(guest, hypervisors)
	s.NoError(err)
	s.Len(candidates, 1)
	s.Equal(hypervisors[1].ID, candidates[0].ID)

}

func (s *GuestTestSuite) TestCandidateRandomize() {
	guest := s.Context.NewGuest()
	candidates := make(lochness.Hypervisors, 10)
	for i := 0; i < cap(candidates); i++ {
		candidates[i] = s.Context.NewHypervisor()
	}

	randCandidates, err := lochness.CandidateRandomize(guest, candidates)
	if !s.NoError(err) {
		return
	}
	if !s.Len(randCandidates, len(candidates)) {
		return
	}
	different := 0
	for i, candidate := range candidates {
		var found bool
		for j, randCandidate := range randCandidates {
			if candidate.ID == randCandidate.ID {
				found = true
				break
			}
			if i != j {
				different++
			}
		}
		s.True(found, "original entry should be in randomized list")
	}
	if different < len(candidates)/2 {
		fmt.Printf("TestCandidateRandomize: Only %d of %d elements were in different location", different, len(candidates))
	}

}

func (s *GuestTestSuite) TestForEachGuest() {
	guest := s.newGuest()
	guest2 := s.newGuest()
	expectedFound := map[string]bool{
		guest.ID:  true,
		guest2.ID: true,
	}

	resultFound := make(map[string]bool)

	err := s.Context.ForEachGuest(func(g *lochness.Guest) error {
		resultFound[g.ID] = true
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(expectedFound, resultFound))

	returnErr := errors.New("an error")
	err = s.Context.ForEachGuest(func(g *lochness.Guest) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

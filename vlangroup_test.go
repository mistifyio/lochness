package lochness_test

import (
	"errors"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VLANGroupTestSuite struct {
	CommonTestSuite
}

func TestVLANGroupTestSuite(t *testing.T) {
	suite.Run(t, new(VLANGroupTestSuite))
}

func (s *VLANGroupTestSuite) TestNewVLANGroup() {
	vlangroup := s.Context.NewVLANGroup()
	s.NotNil(uuid.Parse(vlangroup.ID))
}

func (s *VLANGroupTestSuite) TestVLANGroup() {
	vlangroup := s.newVLANGroup()

	tests := []struct {
		description string
		ID          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid ID", "adf", true},
		{"nonexistant ID", uuid.New(), true},
		{"real ID", vlangroup.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		v, err := s.Context.VLANGroup(test.ID)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(v, msg("failure shouldn't return a vlangroup"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(vlangroup, v), msg("success should return correct data"))
		}
	}
}

func (s *VLANGroupTestSuite) TestRefresh() {
	vlangroup := s.newVLANGroup()
	vlangroupCopy := &lochness.VLANGroup{}
	*vlangroupCopy = *vlangroup
	_ = vlangroup.AddVLAN(s.newVLAN())

	_ = vlangroup.Save()
	s.NoError(vlangroupCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(vlangroup, vlangroupCopy), "refresh should pull new data")

	newVLANGroup := s.Context.NewVLANGroup()
	s.Error(newVLANGroup.Refresh(), "unsaved vlangroup refresh should fail")
}

func (s *VLANGroupTestSuite) TestValidate() {
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
		v := &lochness.VLANGroup{ID: test.ID}
		err := v.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *VLANGroupTestSuite) TestSave() {
	goodVLANGroup := s.Context.NewVLANGroup()

	clobberVLANGroup := &lochness.VLANGroup{}
	*clobberVLANGroup = *goodVLANGroup

	tests := []struct {
		description string
		vlangroup   *lochness.VLANGroup
		expectedErr bool
	}{
		{"invalid vlangroup", &lochness.VLANGroup{}, true},
		{"valid vlangroup", goodVLANGroup, false},
		{"existing vlangroup", goodVLANGroup, false},
		{"existing vlangroup clobber", clobberVLANGroup, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.vlangroup.Save()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

func (s *VLANGroupTestSuite) TestDestroy() {
	vlangroup := s.newVLANGroup()
	vlan := s.newVLAN()
	_ = vlangroup.AddVLAN(vlan)

	blankVG := s.Context.NewVLANGroup()
	blankVG.ID = ""

	tests := []struct {
		description string
		v           *lochness.VLANGroup
		expectedErr bool
	}{
		{"invalid vlangroup", blankVG, true},
		{"existing vlangroup", vlangroup, false},
		{"nonexistant vlangroup", s.Context.NewVLANGroup(), true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.v.Destroy()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
			_ = vlan.Refresh()
			s.Len(vlan.VLANGroups(), 0, msg("should remove vlan link"))
		}
	}

}

func (s *VLANGroupTestSuite) TestForEachVLANGroup() {
	vlangroup := s.newVLANGroup()
	vlangroup2 := s.newVLANGroup()
	expectedFound := map[string]bool{
		vlangroup.ID:  true,
		vlangroup2.ID: true,
	}

	resultFound := make(map[string]bool)

	err := s.Context.ForEachVLANGroup(func(v *lochness.VLANGroup) error {
		resultFound[v.ID] = true
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(expectedFound, resultFound))

	returnErr := errors.New("an error")
	err = s.Context.ForEachVLANGroup(func(v *lochness.VLANGroup) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

func (s *VLANGroupTestSuite) TestAddVLAN() {
	tests := []struct {
		description string
		vg          *lochness.VLANGroup
		v           *lochness.VLAN
		expectedErr bool
	}{
		{"nonexisting vlangroup, nonexisting vlan", s.Context.NewVLANGroup(), s.Context.NewVLAN(), true},
		{"existing vlangroup, nonexisting vlan", s.newVLANGroup(), s.Context.NewVLAN(), true},
		{"nonexisting vlangroup, existing vlan", s.Context.NewVLANGroup(), s.newVLAN(), true},
		{"existing vlangroup and vlan", s.newVLANGroup(), s.newVLAN(), false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.vg.AddVLAN(test.v)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Len(test.vg.VLANs(), 0, msg("fail should not add vlan to vlangroup"))
			s.Len(test.v.VLANGroups(), 0, msg("fail should not add vlangroup to vlan"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Len(test.vg.VLANs(), 1, msg("fail should add vlan to vlangroup"))
			s.Len(test.v.VLANGroups(), 1, msg("fail should add vlangroup to vlan"))
		}
	}
}

func (s *VLANGroupTestSuite) TestRemoveVLAN() {
	vlan := s.newVLAN()
	vlanGroup := s.newVLANGroup()
	_ = vlanGroup.AddVLAN(vlan)

	tests := []struct {
		description string
		vg          *lochness.VLANGroup
		v           *lochness.VLAN
		expectedErr bool
	}{
		{"nonexisting vlangroup, nonexisting vlan", s.Context.NewVLANGroup(), s.Context.NewVLAN(), true},
		{"existing vlangroup, nonexisting vlan", vlanGroup, s.Context.NewVLAN(), true},
		{"nonexisting vlangroup, existing vlan", s.Context.NewVLANGroup(), vlan, true},
		{"existing vlangroup and vlan", vlanGroup, vlan, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		vgLen := len(test.vg.VLANs())
		vLen := len(test.v.VLANGroups())

		err := test.vg.RemoveVLAN(test.v)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Len(test.vg.VLANs(), vgLen, msg("fail should not add vlan to vlangroup"))
			s.Len(test.v.VLANGroups(), vLen, msg("fail should not add vlangroup to vlan"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Len(test.vg.VLANs(), vgLen-1, msg("fail should add vlan to vlangroup"))
			s.Len(test.v.VLANGroups(), vLen-1, msg("fail should add vlangroup to vlan"))
		}
	}
}

func (s *VLANGroupTestSuite) TestVLANs() {
	vlanGroup := s.newVLANGroup()
	_ = vlanGroup.AddVLAN(s.newVLAN())

	s.Len(vlanGroup.VLANs(), 1)
}

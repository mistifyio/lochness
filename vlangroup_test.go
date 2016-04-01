package lochness_test

import (
	"errors"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestVLANGroup(t *testing.T) {
	suite.Run(t, new(VLANGroupSuite))
}

type VLANGroupSuite struct {
	common.Suite
}

func (s *VLANGroupSuite) TestNewVLANGroup() {
	vlangroup := s.Context.NewVLANGroup()
	s.NotNil(uuid.Parse(vlangroup.ID))
}

func (s *VLANGroupSuite) TestVLANGroup() {
	vlangroup := s.NewVLANGroup()

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
		msg := s.Messager(test.description)
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

func (s *VLANGroupSuite) TestRefresh() {
	vlangroup := s.NewVLANGroup()
	vlangroupCopy := &lochness.VLANGroup{}
	*vlangroupCopy = *vlangroup

	s.Require().NoError(vlangroup.AddVLAN(s.NewVLAN()))

	s.Require().NoError(vlangroup.Save())
	s.NoError(vlangroupCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(vlangroup, vlangroupCopy), "refresh should pull new data")

	NewVLANGroup := s.Context.NewVLANGroup()
	s.Error(NewVLANGroup.Refresh(), "unsaved vlangroup refresh should fail")
}

func (s *VLANGroupSuite) TestValidate() {
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
		msg := s.Messager(test.description)
		v := &lochness.VLANGroup{ID: test.ID}
		err := v.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *VLANGroupSuite) TestSave() {
	goodVLANGroup := s.Context.NewVLANGroup()

	clobberVLANGroup := *goodVLANGroup

	tests := []struct {
		description string
		vlangroup   *lochness.VLANGroup
		expectedErr bool
	}{
		{"invalid vlangroup", &lochness.VLANGroup{}, true},
		{"valid vlangroup", goodVLANGroup, false},
		{"existing vlangroup", goodVLANGroup, false},
		{"existing vlangroup clobber", &clobberVLANGroup, true},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		err := test.vlangroup.Save()
		if test.expectedErr {
			s.Error(err, msg("should fail"))
		} else {
			s.NoError(err, msg("should succeed"))
		}
	}
}

func (s *VLANGroupSuite) TestDestroy() {
	vlangroup := s.NewVLANGroup()
	vlan := s.NewVLAN()
	s.Require().NoError(vlangroup.AddVLAN(vlan))

	blankVG := s.Context.NewVLANGroup()
	blankVG.ID = ""

	tests := []struct {
		description  string
		v            *lochness.VLANGroup
		expectError  bool
		expectChange bool
	}{
		{"invalid vlangroup", blankVG, true, false},
		{"existing vlangroup", vlangroup, false, true},
		{"nonexistant vlangroup", s.Context.NewVLANGroup(), false, false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		err := test.v.Destroy()
		if test.expectError {
			s.Error(err, msg("should error"))
		} else {
			s.NoError(err, msg("should not error"))
		}
		if !test.expectChange {
			continue
		}
		_ = vlan.Refresh()
		s.Len(vlan.VLANGroups(), 0, msg("should remove vlan link"))
	}

}

func (s *VLANGroupSuite) TestForEachVLANGroup() {
	vlangroup := s.NewVLANGroup()
	vlangroup2 := s.NewVLANGroup()
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

func (s *VLANGroupSuite) TestAddVLAN() {
	tests := []struct {
		description string
		vg          *lochness.VLANGroup
		v           *lochness.VLAN
		expectedErr bool
	}{
		{"nonexisting vlangroup, nonexisting vlan", s.Context.NewVLANGroup(), s.Context.NewVLAN(), true},
		{"existing vlangroup, nonexisting vlan", s.NewVLANGroup(), s.Context.NewVLAN(), true},
		{"nonexisting vlangroup, existing vlan", s.Context.NewVLANGroup(), s.NewVLAN(), true},
		{"existing vlangroup and vlan", s.NewVLANGroup(), s.NewVLAN(), false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
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

func (s *VLANGroupSuite) TestRemoveVLAN() {
	vlan := s.NewVLAN()
	vlanGroup := s.NewVLANGroup()
	_ = vlanGroup.AddVLAN(vlan)

	tests := []struct {
		description  string
		vg           *lochness.VLANGroup
		v            *lochness.VLAN
		expectChange bool
	}{
		{"nonexisting vlangroup, nonexisting vlan", s.Context.NewVLANGroup(), s.Context.NewVLAN(), false},
		{"existing vlangroup, nonexisting vlan", vlanGroup, s.Context.NewVLAN(), false},
		{"nonexisting vlangroup, existing vlan", s.Context.NewVLANGroup(), vlan, false},
		{"existing vlangroup and vlan", vlanGroup, vlan, true},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		vgLen := len(test.vg.VLANs())
		vLen := len(test.v.VLANGroups())

		err := test.vg.RemoveVLAN(test.v)
		s.NoError(err)
		if test.expectChange {
			s.Len(test.vg.VLANs(), vgLen-1, msg("fail should add vlan to vlangroup"))
			s.Len(test.v.VLANGroups(), vLen-1, msg("fail should add vlangroup to vlan"))
		} else {
			s.Len(test.vg.VLANs(), vgLen, msg("fail should not add vlan to vlangroup"))
			s.Len(test.v.VLANGroups(), vLen, msg("fail should not add vlangroup to vlan"))
		}
	}
}

func (s *VLANGroupSuite) TestVLANs() {
	vlanGroup := s.NewVLANGroup()
	_ = vlanGroup.AddVLAN(s.NewVLAN())

	s.Len(vlanGroup.VLANs(), 1)
}

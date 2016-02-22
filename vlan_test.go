package lochness_test

import (
	"errors"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type VLANSuite struct {
	common.Suite
}

func TestVLAN(t *testing.T) {
	suite.Run(t, new(VLANSuite))
}

func (s *VLANSuite) TestNewVLAN() {
	vlan := s.Context.NewVLAN()
	s.Equal(1, vlan.Tag)
}

func (s *VLANSuite) TestVLAN() {
	vlan := s.NewVLAN()

	tests := []struct {
		description string
		tag         int
		expectedErr bool
	}{
		{"invalid tag", -1, true},
		{"nonexistant tag", 1000, true},
		{"real tag", vlan.Tag, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		v, err := s.Context.VLAN(test.tag)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(v, msg("failure shouldn't return a vlan"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(vlan, v), msg("success should return correct data"))
		}
	}
}

func (s *VLANSuite) TestRefresh() {
	vlan := s.NewVLAN()
	vlanCopy := &lochness.VLAN{}
	*vlanCopy = *vlan

	_ = vlan.Save()
	s.NoError(vlanCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(vlan, vlanCopy), "refresh should pull new data")

	NewVLAN := s.Context.NewVLAN()
	NewVLAN.Tag = 50
	s.Error(NewVLAN.Refresh(), "unsaved vlan refresh should fail")
}

func (s *VLANSuite) TestValidate() {
	tests := []struct {
		description string
		tag         int
		expectedErr bool
	}{
		{"negative tag", -1, true},
		{"zero tag", 0, true},
		{"min tag", 1, false},
		{"max tag", 4095, false},
		{"above max", 4096, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		v := &lochness.VLAN{Tag: test.tag}
		err := v.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *VLANSuite) TestSave() {
	goodVLAN := s.Context.NewVLAN()

	clobberVLAN := *goodVLAN

	tests := []struct {
		description string
		vlan        *lochness.VLAN
		expectedErr bool
	}{
		{"invalid vlan", &lochness.VLAN{}, true},
		{"valid vlan", goodVLAN, false},
		{"existing vlan", goodVLAN, false},
		{"existing vlan clobber", &clobberVLAN, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.vlan.Save()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *VLANSuite) TestDestroy() {
	vlan := s.NewVLAN()
	vlangroup := s.NewVLANGroup()
	_ = vlangroup.AddVLAN(vlan)

	invalidV := s.Context.NewVLAN()
	invalidV.Tag = 0

	tests := []struct {
		description string
		v           *lochness.VLAN
		expectedErr bool
	}{
		{"invalid vlan", invalidV, true},
		{"existing vlan", vlan, false},
		{"nonexistant vlan", s.Context.NewVLAN(), true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.v.Destroy()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
			_ = vlangroup.Refresh()
			s.Len(vlangroup.VLANs(), 0, msg("should remove vlan link"))
		}
	}
}

func (s *VLANSuite) TestForEachVLAN() {
	vlan := s.NewVLAN()
	vlan2 := s.NewVLAN()
	expectedFound := map[int]bool{
		vlan.Tag:  true,
		vlan2.Tag: true,
	}

	resultFound := make(map[int]bool)

	err := s.Context.ForEachVLAN(func(v *lochness.VLAN) error {
		resultFound[v.Tag] = true
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(expectedFound, resultFound))

	returnErr := errors.New("an error")
	err = s.Context.ForEachVLAN(func(v *lochness.VLAN) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

func (s *VLANSuite) TestVLANGroups() {
	vlanGroup := s.NewVLANGroup()
	vlan := s.NewVLAN()
	_ = vlanGroup.AddVLAN(vlan)
	s.Len(vlan.VLANGroups(), 1)
}

package lochness_test

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/common_test"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type FWGroupTestSuite struct {
	ct.CommonTestSuite
}

func TestFWGroupTestSuite(t *testing.T) {
	suite.Run(t, new(FWGroupTestSuite))
}

func (s *FWGroupTestSuite) TestNewFWGroup() {
	fw := s.Context.NewFWGroup()
	s.NotEmpty(uuid.Parse(fw.ID))
}

func (s *FWGroupTestSuite) TestFWGroup() {
	fwgroup := s.NewFWGroup()

	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid id", "asdf", true},
		{"nonexistant id", uuid.New(), true},
		{"real id", fwgroup.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		f, err := s.Context.FWGroup(test.id)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(f, msg("failure shouldn't return a fwgroup"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			// For some reason, asser.ObjectsAreEqual fails here
			s.Equal(fwgroup.ID, f.ID, msg("success should pull correct id"))
			s.Len(f.Rules, len(fwgroup.Rules), msg("success should pull the rules"))
		}
	}
}

func (s *FWGroupTestSuite) TestRefresh() {
	fwgroup := s.NewFWGroup()
	fwgroupCopy := &lochness.FWGroup{}
	*fwgroupCopy = *fwgroup
	fwgroup.Rules = lochness.FWRules{&lochness.FWRule{}}

	_ = fwgroup.Save()
	s.NoError(fwgroupCopy.Refresh(), "refresh existing should succeed")
	// For some reason, asser.ObjectsAreEqual fails here
	s.Equal(fwgroup.ID, fwgroupCopy.ID, "should pull correct id")
	s.Len(fwgroupCopy.Rules, len(fwgroup.Rules), "should pull the rules")

	newFWGroup := s.Context.NewFWGroup()
	s.Error(newFWGroup.Refresh(), "unsaved fwgroup refresh should fail")
}

func (s *FWGroupTestSuite) TestValidate() {
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
		fg := &lochness.FWGroup{ID: test.ID}
		err := fg.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *FWGroupTestSuite) TestSave() {
	goodFWGroup := s.Context.NewFWGroup()

	clobberFWGroup := &lochness.FWGroup{}
	*clobberFWGroup = *goodFWGroup

	tests := []struct {
		description string
		fwgroup     *lochness.FWGroup
		expectedErr bool
	}{
		{"valid fwgroup", goodFWGroup, false},
		{"existing fwgroup", goodFWGroup, false},
		{"existing fwgroup clobber changes", clobberFWGroup, true},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		err := test.fwgroup.Save()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *FWGroupTestSuite) TestJSON() {

	fwgroup := s.Context.NewFWGroup()
	_, n, _ := net.ParseCIDR("192.168.100.1/16")
	fwrule := &lochness.FWRule{
		Source:    n,
		PortStart: 1000,
		PortEnd:   2000,
	}
	fwgroup.Rules = lochness.FWRules{fwrule}

	fwgroupBytes, err := json.Marshal(fwgroup)
	s.NoError(err)

	fwgroupFromJSON := &lochness.FWGroup{}
	s.NoError(json.Unmarshal(fwgroupBytes, fwgroupFromJSON))
	// For some reason, asser.ObjectsAreEqual fails here
	s.Equal(fwgroup.ID, fwgroupFromJSON.ID, "should pull correct id")
	s.Len(fwgroupFromJSON.Rules, len(fwgroup.Rules), "should pull the rules")
	s.True(assert.ObjectsAreEqual(fwrule, fwgroupFromJSON.Rules[0]), "rules should be equal")

}

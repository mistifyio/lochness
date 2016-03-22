package lochness_test

import (
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestFlavor(t *testing.T) {
	suite.Run(t, new(FlavorSuite))
}

type FlavorSuite struct {
	common.Suite
}

func (s *FlavorSuite) TestNewFlavor() {
	f := s.Context.NewFlavor()
	s.NotEmpty(uuid.Parse(f.ID))
}

func (s *FlavorSuite) TestFlavor() {
	flavor := s.NewFlavor()

	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"invalid id", "asdf", true},
		{"nonexistant id", uuid.New(), true},
		{"real id", flavor.ID, false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		f, err := s.Context.Flavor(test.id)
		if test.expectedErr {
			s.Error(err, msg("lookup should fail"))
			s.Nil(f, msg("failure shouldn't return a flavor"))
		} else {
			s.NoError(err, msg("lookup should succeed"))
			s.True(assert.ObjectsAreEqual(flavor, f), msg("success should return correct data"))
		}
	}

}

func (s *FlavorSuite) TestRefresh() {
	flavor := s.NewFlavor()
	flavorCopy := &lochness.Flavor{}
	*flavorCopy = *flavor

	flavor.Image = uuid.New()
	s.Require().NoError(flavor.Save())

	s.NoError(flavorCopy.Refresh(), "refresh existing should succeed")
	s.True(assert.ObjectsAreEqual(flavor, flavorCopy), "refresh should pull new data")

	newFlavor := s.Context.NewFlavor()
	s.Error(newFlavor.Refresh(), "unsaved flavor refresh should fail")
}

func (s *FlavorSuite) TestValidate() {
	tests := []struct {
		description string
		flavor      *lochness.Flavor
		expectedErr bool
	}{
		{"missing id", &lochness.Flavor{}, true},
		{"invalid id", &lochness.Flavor{ID: "asdf"}, true},
		{"missing image", &lochness.Flavor{ID: uuid.New()}, true},
		{"invalid image", &lochness.Flavor{ID: uuid.New(), Image: "asdf"}, true},
		{"valid id and image", &lochness.Flavor{ID: uuid.New(), Image: uuid.New()}, false},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		err := test.flavor.Validate()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

func (s *FlavorSuite) TestSave() {
	goodFlavor := s.Context.NewFlavor()
	goodFlavor.Image = uuid.New()

	clobberFlavor := *goodFlavor
	clobberFlavor.Image = uuid.New()

	tests := []struct {
		description string
		flavor      *lochness.Flavor
		expectedErr bool
	}{
		{"invalid flavor", s.Context.NewFlavor(), true},
		{"valid flavor", goodFlavor, false},
		{"existing flavor", goodFlavor, false},
		{"existing flavor clobber changes", &clobberFlavor, true},
	}

	for _, test := range tests {
		msg := s.Messager(test.description)
		err := test.flavor.Save()
		if test.expectedErr {
			s.Error(err, msg("should be invalid"))
		} else {
			s.NoError(err, msg("should be valid"))
		}
	}
}

package lochness_test

import (
	"errors"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/stretchr/testify/suite"
)

type ContextTestSuite struct {
	CommonTestSuite
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(ContextTestSuite))
}

func (s *ContextTestSuite) TestNewContext() {
	s.NotNil(s.Context)
}

func (s *ContextTestSuite) TestIsKeyNotFound() {
	_, err := s.EtcdClient.Get(s.prefixKey("some-randon-non-existent-key"), false, false)

	s.Error(err)
	s.True(lochness.IsKeyNotFound(err))

	err = errors.New("some-random-non-key-not-found-error")
	s.False(lochness.IsKeyNotFound(err))
}

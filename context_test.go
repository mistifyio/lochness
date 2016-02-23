package lochness_test

import (
	"errors"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/suite"
)

func TestContext(t *testing.T) {
	suite.Run(t, new(ContextSuite))
}

type ContextSuite struct {
	common.Suite
}

func (s *ContextSuite) TestNewContext() {
	s.NotNil(s.Context)
}

func (s *ContextSuite) TestIsKeyNotFound() {
	_, err := s.EtcdClient.Get(s.PrefixKey("some-randon-non-existent-key"), false, false)

	s.Error(err)
	s.True(lochness.IsKeyNotFound(err))

	err = errors.New("some-random-non-key-not-found-error")
	s.False(lochness.IsKeyNotFound(err))
}

package lochness_test

import (
	"errors"
	"testing"

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
	_, err := s.KV.Get(s.PrefixKey("some-randon-non-existent-key"))

	s.Error(err)
	s.True(s.KV.IsKeyNotFound(err))

	err = errors.New("some-random-non-key-not-found-error")
	s.False(s.KV.IsKeyNotFound(err))
}

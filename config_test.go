package lochness_test

import (
	"errors"
	"testing"

	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	common.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (s *ConfigTestSuite) TestGetConfig() {
	_ = s.Context.SetConfig("TestGetConfig", "foo")
	_ = s.Context.SetConfig("TestGetConfigNested/foo", "bar")

	tests := []struct {
		description string
		key         string
		value       string
		expectedErr bool
	}{
		{"empty key", "", "", true},
		{"missing key", "bar", "", true},
		{"key present", "TestGetConfig", "foo", false},
		{"nested key present", "TestGetConfigNested/foo", "bar", false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		val, err := s.Context.GetConfig(test.key)
		s.Equal(test.value, val, msg("values should match"))
		if test.expectedErr {
			s.Error(err, msg("should error"))
		} else {
			s.NoError(err, msg("should not error"))
		}
	}

}

func (s *ConfigTestSuite) TestSetConfig() {
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
		err := s.Context.SetConfig(test.key, test.value)
		if test.expectedErr {
			s.Error(err, test.description)
		} else {
			s.NoError(err, test.description)
		}
	}
}

func (s *ConfigTestSuite) TestForEachConfig() {
	keyValues := map[string]string{
		"TestForEachConfig":       "foo",
		"TestGetConfigNested/foo": "bar",
	}
	for key, value := range keyValues {
		_ = s.Context.SetConfig(key, value)
	}

	resultKeyValues := make(map[string]string)

	err := s.Context.ForEachConfig(func(k, v string) error {
		resultKeyValues[k] = v
		return nil
	})
	s.NoError(err)
	s.True(assert.ObjectsAreEqual(keyValues, resultKeyValues))

	returnErr := errors.New("an error")
	err = s.Context.ForEachConfig(func(k, v string) error {
		return returnErr
	})
	s.Error(err)
	s.Equal(returnErr, err)
}

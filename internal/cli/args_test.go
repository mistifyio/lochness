package cli_test

import (
	"strings"
	"testing"

	"github.com/mistifyio/lochness/internal/cli"
	"github.com/stretchr/testify/suite"
)

type ReadTestSuite struct {
	suite.Suite
}

func TestReadTestSuite(t *testing.T) {
	suite.Run(t, new(ReadTestSuite))
}

func (s *ReadTestSuite) TestRead() {
	reader := strings.NewReader("")
	s.Len(cli.Read(reader), 0)
	reader = strings.NewReader("foo\nbar\nbaz\nbang")
	s.Len(cli.Read(reader), 4)
}

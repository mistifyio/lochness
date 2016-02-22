package cli_test

import (
	"strings"
	"testing"

	"github.com/mistifyio/lochness/internal/cli"
	"github.com/stretchr/testify/suite"
)

type CLISuite struct {
	suite.Suite
}

func TestCLI(t *testing.T) {
	suite.Run(t, new(CLISuite))
}

func (s *CLISuite) TestRead() {
	reader := strings.NewReader("")
	s.Len(cli.Read(reader), 0)
	reader = strings.NewReader("foo\nbar\nbaz\nbang")
	s.Len(cli.Read(reader), 4)
}

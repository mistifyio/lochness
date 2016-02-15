package main_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/suite"
)

type CmdSuite struct {
	common.Suite
	BinName        string
	WorkPath       string
	ConfigPath     string
	ConfigTemplate *template.Template
	Config         map[string][]string
	Hypervisor     *lochness.Hypervisor
}

func (s *CmdSuite) SetupSuite() {
	s.Suite.SetupSuite()

	log.SetLevel(log.FatalLevel)

	var err error
	// Fake Ansible
	s.WorkPath, err = ioutil.TempDir("", "nconfigdTest-")
	s.Require().NoError(err, "failed to create work dir")
	s.Require().NoError(os.Symlink("/bin/echo", filepath.Join(s.WorkPath, "run")),
		"failed to symlink echo to work dir")

	// Prepare Template
	s.ConfigTemplate, err = template.New("configTemplate").Parse(`{
	"{{ .Prefix }}/config": [],
	"{{ .Prefix }}/hypervisors/{{ .HypervisorID }}/config/foo": ["foo"],
	"{{ .Prefix }}/hypervisors/{{ .HypervisorID }}/config/bar": ["bar"],
	"{{ .Prefix }}/hypervisors/{{ .HypervisorID }}/config/foobar": ["foo","bar"]
}`)
	s.Require().NoError(err, "failed to parse config template")

	s.Require().NoError(common.Build(), "failed to build nconfigd")
	s.BinName = "nconfigd"
}

func (s *CmdSuite) SetupTest() {
	s.Suite.SetupTest()
	s.Hypervisor = s.NewHypervisor()

	configFile, err := ioutil.TempFile(s.WorkPath, "config-")
	s.Require().NoError(err, "failed to create config file in work dir")
	defer func() { _ = configFile.Close() }()
	s.ConfigPath = configFile.Name()

	configB := &bytes.Buffer{}

	s.Require().NoError(s.ConfigTemplate.Execute(configB, map[string]string{
		"Prefix":       s.EtcdPrefix,
		"HypervisorID": s.Hypervisor.ID,
	}), "failed to render config")

	_, err = configFile.Write(configB.Bytes())
	s.Require().NoError(err, "failed to write config file")

	var config map[string][]string
	s.Require().NoError(json.Unmarshal(configB.Bytes(), &config), "failed to unmarshal config json")
	s.Config = config
}

func (s *CmdSuite) TearDownSuite() {
	_ = os.RemoveAll(s.WorkPath)

	s.Suite.TearDownSuite()
}

func TestNConfigd(t *testing.T) {
	t.SkipNow()
	suite.Run(t, new(CmdSuite))
}

type testCase struct {
	description   string
	key           string
	value         string
	expectedRuns  int
	expectedRoles []string
}

func (s *CmdSuite) TestCmd() {
	args := []string{
		"-a", s.WorkPath,
		"-c", s.ConfigPath,
		"-e", s.EtcdURL,
	}

	tests := []testCase{
		{"unwatched", "/foobar", "true", 1, []string{}},
	}
	for key, roles := range s.Config {
		tests = append(tests, testCase{filepath.Base(key), key, "true", 2, roles})
	}

	for _, test := range tests {
		msg := common.TestMsgFunc(test.description)

		cmd, err := common.Exec("./"+s.BinName, args...)
		if !s.NoError(err, msg("command exec should not error")) {
			continue
		}

		_, err = s.EtcdClient.Set(test.key, test.value, 0)
		s.NoError(err)

		time.Sleep(1 * time.Second)
		s.NoError(cmd.Stop())
		status, _ := cmd.ExitStatus()
		s.Equal(-1, status, msg("expected status code to be that of a killed process"))

		output := strings.TrimSpace(cmd.Out.String())
		outputParts := strings.Split(output, "\n")
		s.Len(outputParts, test.expectedRuns, msg("wrong number of ansible runs"))

		if len(outputParts) == 2 {
			for _, role := range test.expectedRoles {
				s.Contains(outputParts[1], role, msg("should have run the role"))
			}
		}
	}
}

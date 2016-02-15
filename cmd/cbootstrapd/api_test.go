package main_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	logx "github.com/mistifyio/mistify-logrus-ext"
	"github.com/stretchr/testify/suite"
)

type API struct {
	ct.Suite
	Port           uint
	APIURL         string
	Opts           string
	DefaultVersion string
	Config         map[string]string
	ImageDir       string
	Versions       []string
	ImageNames     []string
	Hypervisor     *lochness.Hypervisor
	BinName        string
	Cmd            *ct.Cmd
}

func (s *API) SetupSuite() {
	s.Suite.SetupSuite()

	log.SetLevel(log.FatalLevel)
	s.Port = 51423
	s.APIURL = fmt.Sprintf("http://localhost:%d", s.Port)

	s.Opts = "--some bootopts"

	// Set up images to serve
	s.ImageDir, _ = ioutil.TempDir("", "cbootstrapdTest")
	s.Versions = []string{
		"0.1.0",
		"0.2.0",
		"0.3.0",
	}
	s.ImageNames = []string{
		"vmlinuz",
		"initrd",
	}
	for _, version := range s.Versions {
		s.Require().NoError(os.Mkdir(filepath.Join(s.ImageDir, version), 0777))
		for _, filename := range s.ImageNames {
			file, err := os.Create(filepath.Join(s.ImageDir, version, filename))
			s.Require().NoError(err)
			_, err = file.WriteString(fmt.Sprintf("%s-%s", version, filename))
			s.Require().NoError(err)
			s.Require().NoError(file.Close())
		}
	}

	// Set up Config
	s.DefaultVersion = "0.2.0"
	s.Config = map[string]string{
		"FOO": "bar",
		"BAZ": "bang",
	}
	for key, value := range s.Config {
		_ = s.Context.SetConfig(key, value)
	}

	// Run API
	s.Require().NoError(ct.Build())
	s.BinName = "cbootstrapd"
	args := []string{
		"-b", s.APIURL,
		"-e", s.EtcdURL,
		"-i", s.ImageDir,
		"-o", "'" + s.Opts + "'",
		"-p", strconv.Itoa(int(s.Port)),
		"-v", s.DefaultVersion,
	}

	var err error
	s.Cmd, err = ct.Exec("./"+s.BinName, args...)
	s.Require().NoError(err)
	time.Sleep(1 * time.Second)
}

func (s *API) SetupTest() {
	// Add Hypervisor
	s.Hypervisor = s.NewHypervisor()
	_ = s.Hypervisor.SetConfig("version", "0.3.0")
	_ = s.Hypervisor.SetConfig("ASDF", "qwerty")
	time.Sleep(100 * time.Millisecond)
}

func (s *API) TearDownSuite() {
	_ = s.Cmd.Stop()
	_ = os.RemoveAll(s.ImageDir)

	s.Suite.TearDownSuite()
}

func TestAPI(t *testing.T) {
	suite.Run(t, new(API))
}

func (s *API) TestIPXEGet() {
	defaultVersion := "0.1.0"
	_ = s.Context.SetConfig("defaultVersion", defaultVersion)

	hypervisor := s.NewHypervisor()
	hypervisor.IP = net.ParseIP("192.168.1.20")
	_ = hypervisor.Save()

	s.checkIPXE("hypervisor defined version", s.Hypervisor, s.Hypervisor.Config["version"])
	s.checkIPXE("etcd defined default version", hypervisor, defaultVersion)
	_ = s.Context.SetConfig("defaultVersion", "")
	s.checkIPXE("command flag defined default version", hypervisor, s.DefaultVersion)
}

func (s *API) checkIPXE(description string, h *lochness.Hypervisor, expectedVersion string) {
	msg := ct.TestMsgFunc(description)
	resp, err := http.Get(fmt.Sprintf("%s/ipxe/%s", s.APIURL, h.IP))
	s.NoError(err)
	defer logx.LogReturnedErr(resp.Body.Close, nil, "failed to close resp body")

	bodyB, _ := ioutil.ReadAll(resp.Body)
	expected := fmt.Sprintf(`#!ipxe
kernel %s/images/%s/vmlinuz uuid=%s '%s'
initrd %s/images/%s/initrd
boot
`,
		s.APIURL, expectedVersion, h.ID, s.Opts, s.APIURL, expectedVersion)
	s.Equal(expected, string(bodyB), msg("should return correct template"))
}

func (s *API) TestImageGet() {
	for _, version := range s.Versions {
		for _, filename := range s.ImageNames {
			resp, err := http.Get(fmt.Sprintf("%s/images/%s/%s", s.APIURL, version, filename))
			s.NoError(err)
			defer logx.LogReturnedErr(resp.Body.Close, nil, "failed to close resp body")

			bodyB, _ := ioutil.ReadAll(resp.Body)
			s.Equal(fmt.Sprintf("%s-%s", version, filename), string(bodyB))
		}
	}
}

func (s *API) TestConfigGet() {
	h := s.NewHypervisor()
	h.IP = net.ParseIP("192.168.1.90")
	_ = h.Save()

	for _, hypervisor := range []*lochness.Hypervisor{s.Hypervisor, h} {
		resp, err := http.Get(fmt.Sprintf("%s/config/%s", s.APIURL, hypervisor.IP))
		s.NoError(err)
		defer logx.LogReturnedErr(resp.Body.Close, nil, "failed to close resp body")

		bodyB, _ := ioutil.ReadAll(resp.Body)
		config, err := configRespToMap(string(bodyB))
		s.NoError(err)
		for key, value := range hypervisor.Config {
			if strings.ToUpper(key) == key {
				s.Equal(value, config[key])
			}
		}
		for key, value := range s.Config {
			if strings.ToUpper(key) == key {
				s.Equal(value, config[key])
			}
		}
	}
}

func configRespToMap(body string) (map[string]string, error) {
	result := make(map[string]string)
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("config line should be of form 'KEY=value': %s", line)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
}

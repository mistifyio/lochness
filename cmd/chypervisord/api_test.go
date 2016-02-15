package main

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/suite"
	"github.com/tylerb/graceful"
)

type API struct {
	common.Suite
	Port       uint
	APIServer  *graceful.Server
	Hypervisor *lochness.Hypervisor
	APIURL     string
}

func (s *API) SetupSuite() {
	s.Suite.SetupSuite()

	log.SetLevel(log.FatalLevel)
	s.Port = 51123
	s.APIURL = fmt.Sprintf("http://localhost:%d/hypervisors", s.Port)

	s.APIServer = Run(s.Port, s.Context)
	time.Sleep(100 * time.Millisecond)
}

func (s *API) SetupTest() {
	s.Suite.SetupTest()
	s.Hypervisor = s.NewHypervisor()
	_ = s.Hypervisor.SetConfig("foo", "bar")
}

func (s *API) TearDownSuite() {
	stopChan := s.APIServer.StopChan()
	s.APIServer.Stop(5 * time.Second)
	<-stopChan

	s.Suite.TearDownSuite()
}

func TestAPI(t *testing.T) {
	suite.Run(t, new(API))
}

func (s *API) TestHypervisorsList() {
	var hypervisors lochness.Hypervisors
	s.DoRequest("GET", s.APIURL, http.StatusOK, nil, &hypervisors)

	s.Len(hypervisors, 1)
	s.Equal(s.Hypervisor.ID, hypervisors[0].ID)
}

func (s *API) TestHypervisorAdd() {
	hypervisor := s.Context.NewHypervisor()
	hypervisor.IP = net.ParseIP("192.168.100.12")
	hypervisor.Netmask = net.ParseIP("225.225.225.225")
	hypervisor.Gateway = net.ParseIP("192.168.100.1")
	hypervisor.MAC, _ = net.ParseMAC("96:E0:51:F9:31:C2")
	hypervisor.TotalResources = lochness.Resources{
		Memory: 16 * 1024,
		Disk:   1024 * 1024,
		CPU:    32,
	}
	hypervisor.AvailableResources = hypervisor.TotalResources

	var hypervisorResp lochness.Hypervisor
	s.DoRequest("POST", s.APIURL, http.StatusCreated, hypervisor, &hypervisorResp)

	s.Equal(hypervisor.ID, hypervisorResp.ID)

	// Make sure it actually saved
	h, err := s.Context.Hypervisor(hypervisor.ID)
	s.NoError(err)
	s.Equal(hypervisor.ID, h.ID)
}

func (s *API) TestHypervisorGet() {
	var hypervisor lochness.Hypervisor
	s.DoRequest("GET", fmt.Sprintf("%s/%s", s.APIURL, s.Hypervisor.ID), http.StatusOK, nil, &hypervisor)

	s.Equal(s.Hypervisor.ID, hypervisor.ID)
}

func (s *API) TestHypervisorUpdate() {
	s.Hypervisor.IP = net.ParseIP("192.168.100.13")

	var hypervisorResp lochness.Hypervisor
	s.DoRequest("PATCH", fmt.Sprintf("%s/%s", s.APIURL, s.Hypervisor.ID), http.StatusOK, s.Hypervisor, &hypervisorResp)

	s.Equal(s.Hypervisor.ID, hypervisorResp.ID)

	// Make sure it actually saved
	h, err := s.Context.Hypervisor(s.Hypervisor.ID)
	s.NoError(err)
	s.Equal(s.Hypervisor.IP, h.IP)
}

func (s *API) TestHypervisorDestroy() {
	var hypervisorResp lochness.Hypervisor
	s.DoRequest("DELETE", fmt.Sprintf("%s/%s", s.APIURL, s.Hypervisor.ID), http.StatusOK, nil, &hypervisorResp)

	s.Equal(s.Hypervisor.ID, hypervisorResp.ID)

	// Make sure it actually saved
	_, err := s.Context.Hypervisor(s.Hypervisor.ID)
	s.Error(err)
}

func (s *API) TestHypervisorGetConfig() {
	var config map[string]string
	s.DoRequest("GET", fmt.Sprintf("%s/%s/config", s.APIURL, s.Hypervisor.ID), http.StatusOK, nil, &config)

	s.Equal(s.Hypervisor.Config, config)
}

func (s *API) TestHypervisorUpdateConfig() {
	configChanges := map[string]string{"asdf": "qwer"}
	var config map[string]string
	s.DoRequest("PATCH", fmt.Sprintf("%s/%s/config", s.APIURL, s.Hypervisor.ID), http.StatusOK, configChanges, &config)

	s.Equal(configChanges["asdf"], config["asdf"])

	// Make sure it actually saved
	hypervisor, err := s.Context.Hypervisor(s.Hypervisor.ID)
	s.NoError(err)
	s.Equal(configChanges["asdf"], hypervisor.Config["asdf"])
}

func (s *API) TestHypervisorSubnetList() {
	hypervisor, _ := s.NewHypervisorWithGuest()
	var subnets map[string]string
	s.DoRequest("GET", fmt.Sprintf("%s/%s/subnets", s.APIURL, hypervisor.ID), http.StatusOK, nil, &subnets)

	s.Len(subnets, 1)
	s.Equal(hypervisor.Subnets(), subnets)
}

func (s *API) TestHypervisorSubnetUpdate() {
	subnet := s.NewSubnet()
	_ = s.Hypervisor.AddSubnet(subnet, "foobar")

	var subnets map[string]string
	s.DoRequest("GET", fmt.Sprintf("%s/%s/subnets", s.APIURL, s.Hypervisor.ID), http.StatusOK, nil, &subnets)

	s.Equal(s.Hypervisor.Subnets(), subnets)
}

func (s *API) TestHypervisorSubnetRemove() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	var subnets map[string]string
	s.DoRequest("DELETE", fmt.Sprintf("%s/%s/subnets/%s", s.APIURL, hypervisor.ID, guest.SubnetID), http.StatusOK, nil, &subnets)

	s.Len(subnets, 0)

	// Make sure it actually saved
	h, _ := s.Context.Hypervisor(hypervisor.ID)
	s.Len(h.Subnets(), 0)
}

func (s *API) TestHypervisorGuestList() {
	hypervisor, guest := s.NewHypervisorWithGuest()
	var guests []string
	s.DoRequest("GET", fmt.Sprintf("%s/%s/guests", s.APIURL, hypervisor.ID), http.StatusOK, nil, &guests)

	s.Len(guests, 1)
	s.Equal(guest.ID, guests[0])
}

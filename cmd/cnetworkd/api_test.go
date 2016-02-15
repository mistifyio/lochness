package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/stretchr/testify/suite"
	"github.com/tylerb/graceful"
)

type APISuite struct {
	ct.Suite
	Port      uint
	APIServer *graceful.Server
	VLAN      *lochness.VLAN
	VLANGroup *lochness.VLANGroup
	APIURL    string
}

func (s *APISuite) SetupSuite() {
	s.Suite.SetupSuite()

	log.SetLevel(log.FatalLevel)
	s.Port = 51123
	s.APIURL = fmt.Sprintf("http://localhost:%d/vlans", s.Port)

	s.APIServer = Run(s.Port, s.Context)
	time.Sleep(100 * time.Millisecond)
}

func (s *APISuite) SetupTest() {
	s.Suite.SetupTest()
	s.VLAN = s.NewVLAN()
	s.VLANGroup = s.NewVLANGroup()
}

func (s *APISuite) TearDownSuite() {
	stopChan := s.APIServer.StopChan()
	s.APIServer.Stop(5 * time.Second)
	<-stopChan

	s.Suite.TearDownSuite()
}

func TestAPI(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func (s *APISuite) TestVLANList() {
	var vlans lochness.VLANs
	s.DoRequest("GET", fmt.Sprintf("%s/tags", s.APIURL), http.StatusOK, nil, &vlans)

	s.Len(vlans, 1)
	s.Equal(s.VLAN.Tag, vlans[0].Tag)
}

func (s *APISuite) TestVLANAdd() {
	vlan := s.Context.NewVLAN()
	vlan.Tag = 2

	var vlanResp lochness.VLAN
	s.DoRequest("POST", fmt.Sprintf("%s/tags", s.APIURL), http.StatusCreated, vlan, &vlanResp)

	s.Equal(vlan.Tag, vlanResp.Tag)

	// Make sure it actually saved
	v, err := s.Context.VLAN(vlan.Tag)
	s.NoError(err)
	s.Equal(vlan.Tag, v.Tag)
}

func (s *APISuite) TestVLANGet() {
	var vlan lochness.VLAN
	s.DoRequest("GET", fmt.Sprintf("%s/tags/%d", s.APIURL, s.VLAN.Tag), http.StatusOK, nil, &vlan)

	s.Equal(s.VLAN.Tag, vlan.Tag)
}

func (s *APISuite) TestVLANUpdate() {
	s.VLAN.Description = "foobar"

	var vlanResp lochness.VLAN
	s.DoRequest("PATCH", fmt.Sprintf("%s/tags/%d", s.APIURL, s.VLAN.Tag), http.StatusOK, s.VLAN, &vlanResp)

	s.Equal(s.VLAN.Tag, vlanResp.Tag)

	// Make sure it actually saved
	v, err := s.Context.VLAN(s.VLAN.Tag)
	s.NoError(err)
	s.Equal(s.VLAN.Description, v.Description)
}

func (s *APISuite) TestVLANDestroy() {
	var vlanResp lochness.VLAN
	s.DoRequest("DELETE", fmt.Sprintf("%s/tags/%d", s.APIURL, s.VLAN.Tag), http.StatusOK, nil, &vlanResp)

	s.Equal(s.VLAN.Tag, vlanResp.Tag)

	// Make sure it actually saved
	_, err := s.Context.VLAN(s.VLAN.Tag)
	s.Error(err)
}

func (s *APISuite) TestVLANGroupList() {
	var vlanGroups lochness.VLANGroups
	s.DoRequest("GET", fmt.Sprintf("%s/groups", s.APIURL), http.StatusOK, nil, &vlanGroups)

	s.Len(vlanGroups, 1)
	s.Equal(s.VLANGroup.ID, vlanGroups[0].ID)
}

func (s *APISuite) TestVLANGroupAdd() {
	vlanGroup := s.Context.NewVLANGroup()

	var vlanGroupResp lochness.VLANGroup
	s.DoRequest("POST", fmt.Sprintf("%s/groups", s.APIURL), http.StatusCreated, vlanGroup, &vlanGroupResp)

	s.Equal(vlanGroup.ID, vlanGroupResp.ID)

	// Make sure it actually saved
	v, err := s.Context.VLANGroup(vlanGroup.ID)
	s.NoError(err)
	s.Equal(vlanGroup.ID, v.ID)
}

func (s *APISuite) TestVLANGroupGet() {
	var vlanGroup lochness.VLANGroup
	s.DoRequest("GET", fmt.Sprintf("%s/groups/%s", s.APIURL, s.VLANGroup.ID), http.StatusOK, nil, &vlanGroup)

	s.Equal(s.VLANGroup.ID, vlanGroup.ID)
}

func (s *APISuite) TestVLANGroupUpdate() {
	s.VLANGroup.Description = "foobar"

	var vlanGroupResp lochness.VLANGroup
	s.DoRequest("PATCH", fmt.Sprintf("%s/groups/%s", s.APIURL, s.VLANGroup.ID), http.StatusOK, s.VLANGroup, &vlanGroupResp)

	s.Equal(s.VLANGroup.ID, vlanGroupResp.ID)

	// Make sure it actually saved
	v, err := s.Context.VLANGroup(s.VLANGroup.ID)
	s.NoError(err)
	s.Equal(s.VLANGroup.Description, v.Description)
}

func (s *APISuite) TestVLANGroupDestroy() {
	var vlanGroupResp lochness.VLANGroup
	s.DoRequest("DELETE", fmt.Sprintf("%s/groups/%s", s.APIURL, s.VLANGroup.ID), http.StatusOK, nil, &vlanGroupResp)

	s.Equal(s.VLANGroup.ID, vlanGroupResp.ID)

	// Make sure it actually saved
	_, err := s.Context.VLANGroup(s.VLANGroup.ID)
	s.Error(err)
}

func (s *APISuite) TestGetGroupsForVLAN() {
	_ = s.VLANGroup.AddVLAN(s.VLAN)

	var vlanGroups []string
	s.DoRequest("GET", fmt.Sprintf("%s/tags/%d/groups", s.APIURL, s.VLAN.Tag), http.StatusOK, nil, &vlanGroups)
	s.Len(vlanGroups, 1)
	s.Equal(s.VLANGroup.ID, vlanGroups[0])
}

func (s *APISuite) TestSetGroupsForVLAN() {
	newVLANGroups := []string{s.VLANGroup.ID}
	var vlanGroups []string
	s.DoRequest("POST", fmt.Sprintf("%s/tags/%d/groups", s.APIURL, s.VLAN.Tag), http.StatusOK, newVLANGroups, &vlanGroups)
	s.Len(vlanGroups, 1)
	s.Equal(s.VLANGroup.ID, vlanGroups[0])

	_ = s.VLAN.Refresh()
	s.Equal(s.VLANGroup.ID, s.VLAN.VLANGroups()[0])
}

func (s *APISuite) TestGetVLANsForGroup() {
	_ = s.VLANGroup.AddVLAN(s.VLAN)

	var vlans []int
	s.DoRequest("GET", fmt.Sprintf("%s/groups/%s/tags", s.APIURL, s.VLANGroup.ID), http.StatusOK, nil, &vlans)
	s.Len(vlans, 1)
	s.Equal(s.VLAN.Tag, vlans[0])
}

func (s *APISuite) TestSetVLANsForGroup() {
	newVLANs := []int{s.VLAN.Tag}
	var vlans []int
	s.DoRequest("POST", fmt.Sprintf("%s/groups/%s/tags", s.APIURL, s.VLANGroup.ID), http.StatusOK, newVLANs, &vlans)
	s.Len(vlans, 1)
	s.Equal(s.VLAN.Tag, vlans[0])

	_ = s.VLANGroup.Refresh()
	s.Equal(s.VLAN.Tag, s.VLANGroup.VLANs()[0])

}

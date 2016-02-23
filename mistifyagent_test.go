package lochness_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"testing"

	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	magent "github.com/mistifyio/mistify-agent"
	mnet "github.com/mistifyio/util/net"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

func TestMistifyAgent(t *testing.T) {
	suite.Run(t, new(MistifyAgentSuite))
}

type MistifyAgentSuite struct {
	common.Suite
	agent      *lochness.MistifyAgent
	api        *httptest.Server
	guest      *lochness.Guest
	hypervisor *lochness.Hypervisor
}

func (s *MistifyAgentSuite) SetupSuite() {
	s.Suite.SetupSuite()

	s.api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.String()
		actionRegexp := regexp.MustCompile(fmt.Sprintf("/guests/%s/\\w+", s.guest.ID))
		jobRegexp := regexp.MustCompile("/jobs/\\w+")
		switch {
		case path == fmt.Sprintf("/guests/%s", s.guest.ID):
			guestBytes, _ := json.Marshal(s.guest)
			_, _ = w.Write(guestBytes)
		case path == "/guests", actionRegexp.MatchString(path), path == "/images":
			w.Header().Set("X-Guest-Job-ID", uuid.New())
			w.WriteHeader(http.StatusAccepted)
		case jobRegexp.MatchString(path):
			job := &magent.Job{
				Status: magent.Complete,
			}
			jobBytes, _ := json.Marshal(job)
			_, _ = w.Write(jobBytes)
		}
	}))
}

func (s *MistifyAgentSuite) SetupTest() {
	s.Suite.SetupTest()
	u, _ := url.Parse(s.api.URL)
	host, sPort, _ := mnet.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(sPort)
	s.agent = s.Context.NewMistifyAgent(port)
	s.hypervisor, s.guest = s.NewHypervisorWithGuest()
	s.hypervisor.IP = net.ParseIP(host)
	_ = s.hypervisor.Save()
}

func (s *MistifyAgentSuite) TearDownSuite() {
	s.api.Close()
	s.Suite.TearDownSuite()
}

func (s *MistifyAgentSuite) TestNewMistifyAgent() {
	s.NotNil(s.agent)
}

func (s *MistifyAgentSuite) TestGetGuest() {
	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"nonuuid id", "asdf", true},
		{"nonexistent id", uuid.New(), true},
		{"existing id", s.guest.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		guest, err := s.agent.GetGuest(test.id)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(guest, msg("fail should not return guest"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.Equal(s.guest.ID, guest.ID, msg("should return guest"))
		}
	}
}

func (s *MistifyAgentSuite) TestCreateGuest() {
	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"nonuuid id", "asdf", true},
		{"nonexistent id", uuid.New(), true},
		{"existing id", s.guest.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		jobID, err := s.agent.CreateGuest(test.id)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(uuid.Parse(jobID), msg("fail should not return jobID"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(uuid.Parse(jobID), msg("should return jobID"))
		}
	}
}

func (s *MistifyAgentSuite) TestDeleteGuest() {
	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"nonuuid id", "asdf", true},
		{"nonexistent id", uuid.New(), true},
		{"existing id", s.guest.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		jobID, err := s.agent.DeleteGuest(test.id)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(uuid.Parse(jobID), msg("fail should not return jobID"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(uuid.Parse(jobID), msg("should return jobID"))
		}
	}
}

func (s *MistifyAgentSuite) TestGuestAction() {
	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"nonuuid id", "asdf", true},
		{"nonexistent id", uuid.New(), true},
		{"existing id", s.guest.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		jobID, err := s.agent.GuestAction(test.id, "restart")
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(uuid.Parse(jobID), msg("fail should not return jobID"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(uuid.Parse(jobID), msg("should return jobID"))
		}
	}
}

func (s *MistifyAgentSuite) TestCheckJobStatus() {
	tests := []struct {
		description string
		guestID     string
		jobID       string
		expectedErr bool
	}{
		{"missing guest id", "", uuid.New(), true},
		{"nonuuid guest id", "asdf", uuid.New(), true},
		{"nonexistent guest id", uuid.New(), uuid.New(), true},
		{"existing guest id, missing job id", s.guest.ID, "", true},
		{"existing guest id, uuid job id", s.guest.ID, uuid.New(), false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		done, err := s.agent.CheckJobStatus(test.guestID, test.jobID)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.False(done, msg("fail should not return done"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.True(done, msg("should return done"))
		}
	}
}

func (s *MistifyAgentSuite) TestFetchImage() {
	tests := []struct {
		description string
		id          string
		expectedErr bool
	}{
		{"missing id", "", true},
		{"nonuuid id", "asdf", true},
		{"nonexistent id", uuid.New(), true},
		{"existing id", s.guest.ID, false},
	}

	for _, test := range tests {
		msg := testMsgFunc(test.description)
		jobID, err := s.agent.FetchImage(test.id)
		if test.expectedErr {
			s.Error(err, msg("should fail"))
			s.Nil(uuid.Parse(jobID), msg("fail should not return jobID"))
		} else {
			s.NoError(err, msg("should succeed"))
			s.NotNil(uuid.Parse(jobID), msg("should return jobID"))
		}
	}
}

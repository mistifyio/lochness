package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-metrics"
	"github.com/bakins/go-metrics-map"
	"github.com/bakins/go-metrics-middleware"
	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/internal/tests/common"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/tylerb/graceful"
)

type APISuite struct {
	ct.Suite
	Port           uint
	BeanstalkdCmd  *exec.Cmd
	BeanstalkdPath string
	JobQueue       *jobqueue.Client
	MetricsContext *metricsContext
	APIServer      *graceful.Server
	Guest          *lochness.Guest
	APIURL         string
}

func (s *APISuite) SetupSuite() {
	s.Suite.SetupSuite()

	log.SetLevel(log.FatalLevel)
	s.Port = 51124
	s.APIURL = fmt.Sprintf("http://localhost:%d/guests", s.Port)

	// Metrics context
	sink := mapsink.New()
	fanout := metrics.FanoutSink{sink}
	conf := metrics.DefaultConfig("cguestdTEST")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, fanout)
	s.MetricsContext = &metricsContext{
		sink:    sink,
		metrics: m,
		mmw:     mmw.New(m),
	}

	// Beanstalkd
	port := "59872"
	s.BeanstalkdPath = fmt.Sprintf("127.0.0.1:%s", port)
	s.BeanstalkdCmd = exec.Command("beanstalkd", "-p", port)
	s.Require().NoError(s.BeanstalkdCmd.Start())

	beanstalkdReady := false
	for i := 0; i < 10; i++ {
		if _, err := beanstalk.Dial("tcp", s.BeanstalkdPath); err == nil {
			beanstalkdReady = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().True(beanstalkdReady)

	// Jobqueue
	s.JobQueue, _ = jobqueue.NewClient(s.BeanstalkdPath, s.EtcdClient)

	// Run the server
	s.APIServer = Run(s.Port, s.Context, s.JobQueue, s.MetricsContext)
	time.Sleep(100 * time.Millisecond)

}

func (s *APISuite) SetupTest() {
	s.Suite.SetupTest()
	s.Guest = s.NewGuest()
}

func (s *APISuite) TearDownSuite() {
	stopChan := s.APIServer.StopChan()
	s.APIServer.Stop(5 * time.Second)
	<-stopChan

	_ = s.BeanstalkdCmd.Process.Kill()
	_ = s.BeanstalkdCmd.Wait()

	s.Suite.TearDownSuite()
}

func TestCGuestdAPI(t *testing.T) {
	suite.Run(t, new(APISuite))
}

func (s *APISuite) TestGuestsList() {
	var guests lochness.Guests
	s.DoRequest("GET", s.APIURL, http.StatusOK, nil, &guests)

	s.Len(guests, 1)
	s.Equal(s.Guest.ID, guests[0].ID)
}

func (s *APISuite) TestGuestAdd() {
	s.Guest.ID = uuid.New()

	var guestResp lochness.Guest
	resp := s.DoRequest("POST", s.APIURL, http.StatusAccepted, s.Guest, &guestResp)
	s.NotEmpty(resp.Header.Get("X-Guest-Job-ID"))

	s.Equal(s.Guest.ID, guestResp.ID)
}

func (s *APISuite) TestGuestGet() {
	var guest lochness.Guest
	s.DoRequest("GET", fmt.Sprintf("%s/%s", s.APIURL, s.Guest.ID), http.StatusOK, nil, &guest)

	s.Equal(s.Guest.ID, guest.ID)
}

func (s *APISuite) TestGuestUpdate() {
	s.Guest.MAC, _ = net.ParseMAC("01:23:45:67:89:ab")

	var guestResp lochness.Guest
	s.DoRequest("PATCH", fmt.Sprintf("%s/%s", s.APIURL, s.Guest.ID), http.StatusOK, s.Guest, &guestResp)

	s.Equal(s.Guest.ID, guestResp.ID)

	// Make sure it actually saved
	g, err := s.Context.Guest(s.Guest.ID)
	s.NoError(err)
	s.Equal(s.Guest.MAC, g.MAC)
}

func (s *APISuite) TestGuestDestroy() {
	var guestResp lochness.Guest
	resp := s.DoRequest("DELETE", fmt.Sprintf("%s/%s", s.APIURL, s.Guest.ID), http.StatusAccepted, nil, &guestResp)
	s.NotEmpty(resp.Header.Get("X-Guest-Job-ID"))

	s.Equal(s.Guest.ID, guestResp.ID)
}

func (s *APISuite) TestGuestAction() {
	var guestResp lochness.Guest
	resp := s.DoRequest("POST", fmt.Sprintf("%s/%s/%s", s.APIURL, s.Guest.ID, "reboot"), http.StatusAccepted, nil, &guestResp)
	s.NotEmpty(resp.Header.Get("X-Guest-Job-ID"))

	s.Equal(s.Guest.ID, guestResp.ID)
}

func (s *APISuite) TestGuestJob() {
	var guestResp lochness.Guest
	resp := s.DoRequest("POST", fmt.Sprintf("%s/%s/%s", s.APIURL, s.Guest.ID, "reboot"), http.StatusAccepted, nil, &guestResp)
	jobID := resp.Header.Get("X-Guest-Job-ID")

	var job jobqueue.Job
	s.DoRequest("GET", fmt.Sprintf("http://localhost:%d/jobs/%s", s.Port, jobID), http.StatusOK, nil, &job)

	s.Equal(jobID, job.ID)
}

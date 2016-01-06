package main_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/cmd/common_test"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/mistifyio/mistify-agent"
	mnet "github.com/mistifyio/util/net"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/suite"
)

type CWorkerdTestSuite struct {
	ct.CommonTestSuite
	BinName        string
	BeanstalkdCmd  *exec.Cmd
	BeanstalkdPath string
	JobQueue       *jobqueue.Client
	Port           string
	Hypervisor     *lochness.Hypervisor
	Guest          *lochness.Guest
	Agent          *httptest.Server
	AgentPort      string
}

func (s *CWorkerdTestSuite) SetupSuite() {
	s.CommonTestSuite.SetupSuite()
	s.Require().NoError(ct.Build())
	s.BinName = "cworkerd"
	s.Port = "45363"

	bPort := "59873"
	s.BeanstalkdPath = fmt.Sprintf("127.0.0.1:%s", bPort)
	s.BeanstalkdCmd = exec.Command("beanstalkd", "-p", bPort)
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

	jobQueue, err := jobqueue.NewClient(s.BeanstalkdPath, s.EtcdClient)
	s.Require().NoError(err)
	s.JobQueue = jobQueue
}

func (s *CWorkerdTestSuite) SetupTest() {
	s.CommonTestSuite.SetupTest()

	s.Agent = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "guest") {
			w.Header().Set("X-Guest-Job-ID", uuid.New())
			w.WriteHeader(http.StatusAccepted)
		} else {
			w.Header().Set("X-Guest-Job-ID", uuid.New())
			w.WriteHeader(http.StatusOK)
			job := &agent.Job{
				Status: agent.Complete,
			}
			jobJSON, _ := json.Marshal(job)
			_, _ = w.Write(jobJSON)
		}
	}))
	agentURL, _ := url.Parse(s.Agent.URL)
	_, s.AgentPort, _ = mnet.SplitHostPort(agentURL.Host)

	s.Hypervisor, s.Guest = s.NewHypervisorWithGuest()
	s.Hypervisor.IP = net.IP{127, 0, 0, 1}
	_ = s.Hypervisor.Save()
}

func (s *CWorkerdTestSuite) TearDownTest() {
	_ = s.BeanstalkdCmd.Process.Kill()
	_ = s.BeanstalkdCmd.Wait()
	s.Agent.Close()
	s.CommonTestSuite.TearDownTest()
}

func TestCWorkerdTestSuite(t *testing.T) {
	suite.Run(t, new(CWorkerdTestSuite))
}

func (s *CWorkerdTestSuite) TestCmd() {

	tests := []struct {
		description string
		jobStatus   string
		jobAction   string
		guestID     string
		expectedErr bool
	}{
		{"bad job action",
			jobqueue.JobStatusNew, "foobar", s.Guest.ID, true},
		{"nonexistent guest id",
			jobqueue.JobStatusNew, "reboot", uuid.New(), true},
		{"valid",
			jobqueue.JobStatusNew, "reboot", s.Guest.ID, false},
	}

	for _, test := range tests {
		msg := ct.TestMsgFunc(test.description)

		job, _ := s.JobQueue.AddJob(test.guestID, test.jobAction)
		if test.jobStatus != jobqueue.JobStatusNew {
			job.Status = test.jobStatus
		}
		_ = job.Save(1 * time.Hour)

		// Start the daemon
		args := []string{
			"-p", s.Port,
			"-e", s.EtcdURL,
			"-b", s.BeanstalkdPath,
			"-a", s.AgentPort,
			"-l", "fatal",
		}
		cmd, err := ct.Exec("./"+s.BinName, args...)
		s.Require().NoError(err, msg("failed to execute daemon"))

		// Wait for processing
		for i := 0; i < 10; i++ {
			time.Sleep(1 * time.Second)
			_ = job.Refresh()
			if job.Status == jobqueue.JobStatusError || job.Status == jobqueue.JobStatusDone {
				break
			}
		}

		if test.expectedErr {
			s.Equal(jobqueue.JobStatusError, job.Status, msg("should have errored"))
			s.NotEmpty(job.Error, msg("should have error msg"))
		} else {
			s.Equal(jobqueue.JobStatusDone, job.Status, msg("should not have errored"))
			s.Empty(job.Error, msg("should not have error msg"))
		}

		workStats, _ := s.JobQueue.StatsWork()
		totalWorkJobs, _ := strconv.Atoi(workStats["current-jobs-total"])
		s.Equal(0, totalWorkJobs, msg("should not have task left in work queue"))

		// Stop the daemon
		_ = cmd.Stop()
		status, err := cmd.ExitStatus()
		s.Equal(-1, status, msg("expected status code to be that of a killed process"))
	}

}

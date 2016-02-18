package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-metrics"
	"github.com/bakins/go-metrics-map"
	"github.com/coreos/go-etcd/etcd"
	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/mistifyio/mistify-agent/config"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

func main() {
	var port, agentPort uint
	var etcdAddr, bstalk, logLevel string

	// Command line flags
	flag.StringVarP(&bstalk, "beanstalk", "b", "127.0.0.1:11300", "address of beanstalkd server")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.StringVarP(&etcdAddr, "etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	flag.UintVarP(&agentPort, "agent-port", "a", uint(lochness.AgentPort), "port on which agents listen")
	flag.UintVarP(&port, "http", "p", 7544, "http port to publish metrics. set to 0 to disable")
	flag.Parse()

	// Set up logger
	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"level": logLevel,
		}).Fatal("unable to to set up logrus")
	}

	etcdClient := etcd.NewClient([]string{etcdAddr})

	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"addr": etcdAddr,
		}).Fatal("unable to sync etcd cluster")
	}

	ctx := lochness.NewContext(etcdClient)

	log.WithField("address", bstalk).Info("connection to beanstalk")
	jobQueue, err := jobqueue.NewClient(bstalk, etcdClient)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": bstalk,
		}).Fatal("failed to create jobQueue client")
	}

	// Set up metrics
	m := setupMetrics(port)
	if m != nil {
	}

	agent := ctx.NewMistifyAgent(int(agentPort))

	// Start consuming
	consume(jobQueue, agent, m)
}

func consume(jobQueue *jobqueue.Client, agent *lochness.MistifyAgent, m *metrics.Metrics) {
	for {
		// Wait for and reserve a job
		task, err := jobQueue.NextWorkTask()
		if err != nil {
			if bCE, ok := err.(beanstalk.ConnError); ok {
				switch bCE {
				case beanstalk.ErrTimeout:
					// Empty queue, continue waiting
					continue
				case beanstalk.ErrDeadline:
					// See docs on beanstalkd deadline
					// We're just going to sleep to let the deadline'd job expire
					// and try to get another job
					m.IncrCounter([]string{"beanstalk", "error", "deadline"}, 1)
					log.Debug(beanstalk.ErrDeadline)
					time.Sleep(5 * time.Second)
					continue
				default:
					// You have failed me for the last time
					log.WithField("error", err).Fatal(err)
				}
			}

			log.WithFields(log.Fields{
				"task":  task,
				"error": err,
			}).Error("invalid task")

			if err := task.Delete(); err != nil {
				log.WithFields(log.Fields{
					"task":  task.ID,
					"error": err,
				}).Error("unable to delete")
			}
		}

		logFields := log.Fields{
			"task": task,
		}

		// Handle the task in its current state. Remove task when appropriate.
		removeTask, err := processTask(task, agent)

		if removeTask {
			if err != nil {
				log.WithFields(logFields).WithField("error", err).Error(err)
				if task.Job != nil {
					_ = updateJobStatus(task, jobqueue.JobStatusError, err)
				}
			} else {
				_ = updateJobStatus(task, jobqueue.JobStatusDone, nil)
			}
			if task.Job != nil {
				log.WithFields(logFields).WithField("status", task.Job.Status).Info("job status info")
			}

			updateMetrics(task, m)

			log.WithFields(logFields).Info("removing task")
			if err := task.Delete(); err != nil {
				log.WithFields(log.Fields{
					"task":  task,
					"error": err,
				}).Error("unable to delete")
			}
		} else {
			log.WithFields(logFields).Info("releasing task")
			if err := task.Release(); err != nil {
				log.WithFields(logFields).WithField("error", err).Fatal(err)
			}
		}
	}
}

// setupMetrics creates the metric sink and starts an optional http server
func setupMetrics(port uint) *metrics.Metrics {
	ms := mapsink.New()
	conf := metrics.DefaultConfig("cworkerd")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, ms)

	// Unless told not to, expose metrics via http
	if port != 0 {
		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ms)
		}))

		go func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		}()
	}

	return m
}

func processTask(task *jobqueue.Task, agent *lochness.MistifyAgent) (bool, error) {
	logFields := log.Fields{
		"task": task,
	}
	log.WithFields(logFields).Info("reserved task")

	switch task.Job.Status {
	case jobqueue.JobStatusDone:
		var err error
		if task.Job.Action == "delete" {
			err = postDelete(task)
		}
		return true, err
	case jobqueue.JobStatusError:
		return true, nil
	case jobqueue.JobStatusNew:
		if err := startJob(task, agent); err != nil {
			return true, err
		}
	case jobqueue.JobStatusWorking:
		if done, err := checkWorkingJob(task, agent); done || err != nil {
			log.WithFields(log.Fields{
				"task": task.ID,
			}).Info("JOB DONE")

			if err == nil && task.Job.Action == "delete" {
				err = postDelete(task)
			}
			return true, err
		}
	}

	return false, nil
}

func startJob(task *jobqueue.Task, agent *lochness.MistifyAgent) error {
	job := task.Job

	if task.Guest == nil {
		return errors.New("guest does not exist")
	}

	var err error
	var jobID string
	switch job.Action {
	case "fetch":
		jobID, err = agent.FetchImage(task.Guest.ID)
	case "create":
		jobID, err = agent.CreateGuest(task.Guest.ID)
	case "delete":
		jobID, err = agent.DeleteGuest(task.Guest.ID)
	default:
		if _, ok := config.ValidActions[job.Action]; !ok {
			return errors.New("invalid action")
		}
		jobID, err = agent.GuestAction(task.Guest.ID, job.Action)
	}

	if err != nil {
		return err
	}
	task.Job.RemoteID = jobID
	_ = updateJobStatus(task, jobqueue.JobStatusWorking, nil)
	return nil
}

func checkWorkingJob(task *jobqueue.Task, agent *lochness.MistifyAgent) (bool, error) {
	done, err := agent.CheckJobStatus(task.Guest.ID, task.Job.RemoteID)
	if err == nil && done && task.Job.Action == "fetch" {
		task.Job.Action = "create"
		task.Job.RemoteID = ""
		task.Job.Status = jobqueue.JobStatusNew

		done = false
		// Save Job Status
		if err := task.Job.Save(24 * time.Hour); err != nil {
			log.WithFields(log.Fields{
				"task":  task,
				"error": err,
			}).Error("unable to save")
			return done, err
		}
	}

	return done, err
}

func updateJobStatus(task *jobqueue.Task, status string, e error) error {
	task.Job.Status = status
	if e != nil {
		task.Job.Error = e.Error()
	}
	if task.Job.StartedAt.Equal(time.Time{}) {
		task.Job.StartedAt = time.Now()
	}
	if status == jobqueue.JobStatusError || status == jobqueue.JobStatusDone {
		task.Job.FinishedAt = time.Now()
	}

	// Save Job Status
	if err := task.Job.Save(24 * time.Hour); err != nil {
		log.WithFields(log.Fields{
			"task":  task,
			"error": err,
		}).Error("unable to save")
		return err
	}
	return nil
}

func postDelete(task *jobqueue.Task) error {
	log.WithFields(log.Fields{
		"task": task,
	}).Info("post delete")
	return task.Guest.Destroy()
}

func updateMetrics(task *jobqueue.Task, m *metrics.Metrics) {
	job := task.Job
	m.MeasureSince([]string{"action", job.Action, "time"}, job.StartedAt)
	m.MeasureSince([]string{"action", "time"}, job.StartedAt)
	m.IncrCounter([]string{"action", job.Action, "count"}, 1)
	m.IncrCounter([]string{"action", "count"}, 1)
	if job.Error != "" {
		m.IncrCounter([]string{"action", job.Action, "error"}, 1)
		m.IncrCounter([]string{"action", "error"}, 1)
	}
}

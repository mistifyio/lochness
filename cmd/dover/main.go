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
	flag "github.com/ogier/pflag"
)

const (
	// WorkTube is the name of the beanstalk tube for work tasks
	WorkTube = "work"
)

// Task is a "helper" struct to pull together information from
// beanstalk and etcd
type Task struct {
	ID      uint64 //id from beanstalkd
	Body    []byte // body from beanstalkd
	Job     *lochness.Job
	Guest   *lochness.Guest
	metrics *metrics.Metrics
	conn    *beanstalk.Conn
	ctx     *lochness.Context
}

func main() {
	// Command line flags
	bstalk := flag.StringP("beanstalk", "b", "127.0.0.1:11300", "address of beanstalkd server")
	logLevel := flag.StringP("log-level", "l", "warn", "log level")
	addr := flag.StringP("etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	port := flag.UintP("http", "p", 7544, "http port to publish metrics. set to 0 to disable")
	flag.Parse()

	// Set up logger
	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)

	// Set up beanstalk
	log.WithField("bstalk", *bstalk).Info("connecting to beanstalk")
	conn, err := beanstalk.Dial("tcp", *bstalk)
	if err != nil {
		log.Fatal(err)
	}
	ts := beanstalk.NewTubeSet(conn, WorkTube)

	// Set up etcd
	log.WithField("cluster", *addr).Info("connecting to etcd")
	etcdClient := etcd.NewClient([]string{*addr})
	// make sure we can talk to etcd
	if !etcdClient.SyncCluster() {
		log.WithField("cluster", *addr).Fatal("unable to sync etcd")
	}

	c := lochness.NewContext(etcdClient)

	// Set up metrics
	m := setupMetrics(*port)
	if m != nil {
	}

	// Start consuming
	consume(c, ts, m)
}

func consume(c *lochness.Context, ts *beanstalk.TubeSet, m *metrics.Metrics) {
	for {
		// Wait for and reserve a job
		id, body, err := ts.Reserve(10 * time.Hour)
		if err != nil {
			switch err.(beanstalk.ConnError) {
			case beanstalk.ErrTimeout:
				// Empty queue, continue waiting
				continue
			case beanstalk.ErrDeadline:
				// See docs on beanstalkd deadline
				// We're just going to sleep and try to get another job
				m.IncrCounter([]string{"beanstalk", "error", "deadline"}, 1)
				log.Debug(beanstalk.ErrDeadline)
				time.Sleep(5 * time.Second)
				continue
			default:
				// You have failed me for the last time
				log.WithField("error", err).Fatal(err)
			}
		}

		task := &Task{
			ID:      id,
			Body:    body,
			metrics: m,
			conn:    ts.Conn,
			ctx:     c,
		}

		logFields := log.Fields{
			"task": task.ID,
			"body": string(task.Body),
		}

		// Handle the task in its current state. Remove task when appropriate.
		removeTask, err := processTask(task)

		if removeTask {
			if err != nil {
				lf := copyFields(logFields)
				lf["error"] = err.Error()
				log.WithFields(lf).Error(err)
				if task.Job != nil {
					_ = updateJobStatus(task, lochness.JobStatusError, err)
				}
			} else {
				_ = updateJobStatus(task, lochness.JobStatusDone, nil)
			}
			if task.Job != nil {
				log.WithFields(logFields).Infof("job status: %s", task.Job.Status)
			}
			log.WithFields(logFields).Info("removing task")
			deleteTask(task)
		} else {
			log.WithFields(logFields).Info("releasing task")
			if err := task.conn.Release(task.ID, 0, 5*time.Second); err != nil {
				lf := copyFields(logFields)
				lf["error"] = err
				log.WithFields(lf).Fatal(err)
			}
		}
	}
}

// setupMetrics creates the metric sink and starts an optional http server
func setupMetrics(port uint) *metrics.Metrics {
	ms := mapsink.New()
	conf := metrics.DefaultConfig("dover")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, ms)

	// Unless told not to, expose metrics via http
	if port != 0 {
		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ms)
		}))

		go func() {
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
		}()
	}

	return m
}

// getJob retrieves the job for a task
func getJob(t *Task) error {
	id := string(t.Body)
	j, err := t.ctx.Job(id)
	if err != nil {
		return err
	}

	if err := j.Validate(); err != nil {
		return err
	}

	t.Job = j
	return nil
}

// getGuest retrieves the guest for a task's job
func getGuest(t *Task) error {
	if t.Job == nil {
		return errors.New("job missing guest")
	}

	g, err := t.ctx.Guest(t.Job.Guest)
	if err != nil {
		return err
	}

	t.Guest = g
	return nil
}

func processTask(task *Task) (bool, error) {
	logFields := log.Fields{
		"task": task.ID,
		"body": string(task.Body),
	}
	log.WithFields(logFields).Info("reserved task")

	// Look up the job info
	if err := getJob(task); err != nil {
		return true, err
	}
	log.WithFields(logFields).Infof("job status: %s", task.Job.Status)

	if err := getGuest(task); err != nil {
		return true, err
	}

	switch task.Job.Status {
	case lochness.JobStatusDone:
		return true, nil
	case lochness.JobStatusError:
		return true, nil
	case lochness.JobStatusNew:
		if err := startJob(task); err != nil {
			return true, err
		}
	case lochness.JobStatusWorking:
		if done, err := checkWorkingJob(task); done || err != nil {
			return true, err
		}
	}

	return false, nil
}

func startJob(task *Task) error {
	agent := task.ctx.NewMistifyAgent()
	job := task.Job

	var err error
	var jobID string
	switch job.Action {
	case "hypervisor-create":
		jobID, err = agent.CreateGuest(task.Guest.ID)
	case "create":
		jobID, err = agent.CreateGuest(task.Guest.ID)
	case "delete":
		jobID, err = agent.DeleteGuest(task.Guest.ID)
	default:
		jobID, err = agent.GuestAction(task.Guest.ID, job.Action)
	}

	if err != nil {
		return err
	}
	task.Job.RemoteID = jobID
	_ = updateJobStatus(task, lochness.JobStatusWorking, nil)
	return nil
}

func checkWorkingJob(task *Task) (bool, error) {
	agent := task.ctx.NewMistifyAgent()
	done, err := agent.CheckJobStatus(task.Job.Action, task.Guest.ID, task.Job.RemoteID)
	return done, err
}

func updateJobStatus(task *Task, status string, e error) error {
	task.Job.Status = status
	if e != nil {
		task.Job.Error = e.Error()
	}
	if (task.Job.StartedAt == time.Time{}) {
		task.Job.StartedAt = time.Now()
	}
	if status == lochness.JobStatusError || status == lochness.JobStatusDone {
		task.Job.FinishedAt = time.Now()

		// We're done with the task, so update the metrics
		task.metrics.MeasureSince([]string{"action", task.Job.Action, "time"}, task.Job.StartedAt)
		task.metrics.MeasureSince([]string{"action", "time"}, task.Job.StartedAt)
		task.metrics.IncrCounter([]string{"action", task.Job.Action, "count"}, 1)
		task.metrics.IncrCounter([]string{"action", "count"}, 1)
		if e != nil {
			task.metrics.IncrCounter([]string{"action", task.Job.Action, "error"}, 1)
			task.metrics.IncrCounter([]string{"action", "error"}, 1)
		}
	}

	// Save Job Status
	if err := task.Job.Save(24 * time.Hour); err != nil {
		log.WithFields(log.Fields{
			"task":  task.ID,
			"job":   task.Job.ID,
			"error": err,
		}).Error("unable to save")
		return err
	}
	return nil
}

func deleteTask(task *Task) {
	if err := task.conn.Delete(task.ID); err != nil {
		log.WithFields(log.Fields{
			"task":  task.ID,
			"error": err,
		}).Errorf("unable to delete")
	}
}

// hacky helper
func copyFields(fields log.Fields) log.Fields {
	f := log.Fields{}
	for k, v := range fields {
		f[k] = v
	}

	return f
}

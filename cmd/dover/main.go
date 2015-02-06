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
	"github.com/mistifyio/lochness"
	flag "github.com/ogier/pflag"
)

const (
	WorkTube = "work"
)

// Task is a "helper" struct to pull together information from
// beanstalk and etcd
type Task struct {
	ID    uint64 //id from beanstalkd
	Body  []byte // body from beanstalkd
	Job   *lochness.Job
	Guest *lochness.Guest
	conn  *beanstalk.Conn
	ctx   *lochness.Context
}

func main() {
	// Command line flags
	bstalk := flag.StringP("beanstalk", "b", "127.0.0.1:11300", "address of beanstalkd server")
	logLevel := flag.StringP("log-level", "l", "warn", "log level")
	addr := flag.StringP("etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	port := flag.UintP("http", "p", 7544, "address to http interface. set to 0 to disable")
	flag.Parse()

	// Set up logger
	log.SetFormatter(&log.JSONFormatter{})
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)

	// Set up beanstalk
	log.Infof("using beanstalk %s", *bstalk)
	conn, err := beanstalk.Dial("tcp", *bstalk)
	if err != nil {
		log.Fatal(err)
	}
	ts := beanstalk.NewTubeSet(conn, WorkTube)

	// Set up etcd
	log.Infof("using etcd %s", *addr)
	etcdClient := etcd.NewClient([]string{*addr})
	// make sure we can talk to etcd
	if !etcdClient.SyncCluster() {
		log.Fatal("unable to sync etcd at %s", *addr)
	}

	c := lochness.NewContext(etcdClient)

	// Set up metrics
	m := setupMetrics(*port)

	// Start consuming
	consume(ts)
}

func consume(ts *beanstalk.TubeSet) {
	for {
		// Wait for and reserve a job
		id, body, err := ts.Reserve(10 * time.Hour)
		if err != nil {
			switch err.(Beanstalk.ConnError) {
			case beanstalk.ErrTimeout:
				// Empty queue, continue waiting
				continue
			case beanstalk.ErrDeadline:
				// See docs on beanstalkd deadline
				// We're just going to sleep and try to get another job
				log.Info(beanstalk.ErrDeadline)
				time.Sleep(5 * time.Second)
				continue
			default:
				// You have failed me for the last time
				log.WithField("error", err).Fatal(err)
			}
		}

		task := &Task{
			ID:   id,
			Body: body,
			conn: conn,
			ctx:  c,
		}

		logFields := log.Fields{
			"task": task.ID,
			"body": string(task.Body),
		}

		// Handle the task in its current state. Remove task when appropriate.
		removeTask := processTask(task)
		log.WithFields(fields).Infof("job status: %s", task.Job.Status)
		if removeTask {
			if err := deleteTask(task); err != nil {
				logFields["error"] = err
				log.WithField(logFields).Fatal(err)
			}
		} else {
			if err := task.conn.Release(task.ID, 0, 5*time.Second); err != nil {
				logFields["error"] = err
				log.WithFields(fields).Fatal(err)
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
	if *port != 0 {
		http.Handle("/metrics", http.HandlerFunc(func(w htt.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ms)
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

func processTask(task *Task) bool {
	logFields := log.Fields{
		"task": task.ID,
		"body": string(task.Body),
	}
	log.WithFields(fields).Info("reserved task")

	// Look up the job info
	if err := getJob(task); err != nil {
		return true
	}

	switch task.Job.Status {
	case JobStatusDone:
		return true
	case JobStatusError:
		return true
	case JobStatusNew:
		if err := startJob(task); err != nil {
			return true
		}
	case JobStatusWorking:
		if err := checkWorkingJob(task); err != nil {
			return true
		}
	}

	return false
}

func startJob(task *Task) error {
	agent := task.ctx.NewMistifyAgent()

	var err error
	switch job.Action {
	case "hypervisor-create":
		//TODO: handle this
	case "delete":
		// TODO handle this
	default:
		// TODO: Update agent.GuestAction to take boolean for whether to wait
		// for completion or not
		_, err = agent.GuestAction(task.Guest.ID, job.Action)
	}

	if err != nil {
		_ := updateJobStatus(task, lochness.JobStatusError, err)
		return err
	}
	_ := updateJobStatus(task, lochness.JobStatusWorking, nil)
	return nil
}

func checkWorkingJob(task *Task) error {
	agent := task.ctx.NewMistifyAgent()
	// TODO: Export the job status checking.
	// TODO: figure out hte best way to align the job ids here with the job ids
	// from the agent's local queue (maybe allow external ids to be passed in on
	// action initiation)

}

// TODO: Decide best way to handle errors
func updateJobStatus(task *Task, status string, e error) error {
	task.Job.Status = status
	if e {
		task.Job.Error = err.Error()
	}
	if task.Job.Save(24 * time.Hour); err != nil {
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
	if err := task.conn.Delete(id); err != nil {
		log.WithFields(log.Fields{
			"task":  task.ID,
			"error": err,
		}).Errorf("unable to delete")
	}
}

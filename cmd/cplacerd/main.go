package main

// TODO: multiple beanstalkd servers

import (
	"encoding/json"
	_ "expvar"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"reflect"
	"runtime"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-metrics"
	"github.com/bakins/go-metrics-map"
	kv "github.com/coreos/go-etcd/etcd"
	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

// TaskFunc is a convenience wrapper for function calls on tasks
type TaskFunc struct {
	name     string
	function func(*jobqueue.Client, *jobqueue.Task) (bool, error)
	label    string // label for metrics
}

// TODO: restructure this as all the deletes for tube stuff is clunky.
// as we almost always delete the tube id, wrap in function and delete it?

func main() {
	var port uint
	var kvAddr, bstalk, logLevel string

	flag.StringVarP(&bstalk, "beanstalk", "b", "127.0.0.1:11300", "address of beanstalkd server")
	flag.StringVarP(&logLevel, "log-level", "l", "warn", "log level")
	flag.StringVarP(&kvAddr, "kv", "k", "http://127.0.0.1:4001", "address of kv server")
	flag.UintVarP(&port, "http", "p", 7543, "address for http interface. set to 0 to disable")
	flag.Parse()

	// Set up logger
	if err := logx.DefaultSetup(logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"level": logLevel,
		}).Fatal("failed to set up logging")
	}

	kvClient := kv.NewClient([]string{kvAddr})

	if !kvClient.SyncCluster() {
		log.WithFields(log.Fields{
			"addr": kvAddr,
		}).Fatal("unable to sync etcd cluster")
	}

	log.WithField("address", bstalk).Info("connection to beanstalk")
	jobQueue, err := jobqueue.NewClient(bstalk, kvClient)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": bstalk,
		}).Fatal("failed to create jobQueue client")
	}

	// setup metrics
	ms := mapsink.New()
	conf := metrics.DefaultConfig("cplacerd")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, ms)

	if port != 0 {

		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(ms); err != nil {
				log.WithField("error", err).Error(err)
			}
		}))

		go func() {
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
				log.WithFields(log.Fields{
					"error": err,
				}).Fatal("error serving")
			}
		}()

	}

	// XXX: we want to try to keep track of where a job is
	// in this pipeline? would have to persist in the job
	funcs := []TaskFunc{
		{
			name:     "check job status",
			function: checkJobStatus,
		},
		{
			name:     "check guest status",
			function: checkGuestStatus,
		},
		{
			name:     "select hypervisor candidate",
			function: selectHypervisor,
		},
		{
			name:     "update job action",
			function: changeJobAction,
		},
		{
			name:     "add task to worker",
			function: addJobToWorker,
		},
		{
			name:     "make task for deletion",
			function: deleteTask,
		},
	}

	for _, f := range funcs {
		f.label = strings.Split(runtime.FuncForPC(reflect.ValueOf(f.function).Pointer()).Name(), ".")[1]
	}

	for {
		task, err := jobQueue.NextCreateTask()
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

		for _, f := range funcs {

			fields := log.Fields{
				"task": task,
				"step": f.name,
			}

			log.WithFields(fields).Debug("running")

			start := time.Now()
			rm, err := f.function(jobQueue, task)

			m.MeasureSince([]string{f.label, "time"}, start)
			m.IncrCounter([]string{f.label, "count"}, 1)

			duration := int(time.Since(start).Seconds() * 1000)
			log.WithFields(fields).WithField("duration", duration).Info("done")

			if err != nil {

				m.IncrCounter([]string{f.label, "error"}, 1)

				log.WithFields(fields).WithField("error", err).Error("task error")

				task.Job.Status = jobqueue.JobStatusError
				task.Job.Error = err.Error()
				if err := task.Job.Save(24 * time.Hour); err != nil {
					log.WithFields(log.Fields{
						"task":  task,
						"error": err,
					}).Error("unable to save")
				}
			}

			if rm {
				if err := task.Delete(); err != nil {
					log.WithFields(log.Fields{
						"task":  task.ID,
						"error": err,
					}).Error("unable to delete")
				}
				break
			}
		}
	}
}

// these funcs return bool if task in beanstalk should be deleted (and loop stopped). loop also stops on error

func checkJobStatus(jobQueue *jobqueue.Client, t *jobqueue.Task) (bool, error) {
	if t.Job.Status != jobqueue.JobStatusNew {
		return true, fmt.Errorf("bad job status: %s", t.Job.Status)
	}
	if t.Job.Action != "select-hypervisor" {
		return true, fmt.Errorf("bad action: %s", t.Job.Action)
	}
	return false, nil
}

func checkGuestStatus(jobQueue *jobqueue.Client, t *jobqueue.Task) (bool, error) {

	if t.Guest.HypervisorID != "" {
		return true, fmt.Errorf("guest already has a hypervisor %s - %s", t.Guest.ID, t.Guest.HypervisorID)
	}
	return false, nil
}

func selectHypervisor(jobQueue *jobqueue.Client, t *jobqueue.Task) (bool, error) {
	candidates, err := t.Guest.Candidates(lochness.DefaultCandidateFunctions...)
	if err != nil {
		return true, fmt.Errorf("unable to select candidate %s - %s", t.Guest.ID, err)
	}

	if len(candidates) == 0 {
		return true, fmt.Errorf("no candidates found for %s", t.Guest.ID)
	}

	h := candidates[0]

	// the API for selecting a candidate and then adding to a hypervisor is clunky
	if err := h.AddGuest(t.Guest); err != nil {
		return true, fmt.Errorf("unable to add guest %s to %s - %s", t.Guest.ID, h.ID, err)
	}

	return false, nil
}

func changeJobAction(jobQueue *jobqueue.Client, t *jobqueue.Task) (bool, error) {
	t.Job.Action = "fetch"
	if err := t.Job.Save(24 * time.Hour); err != nil {
		return true, fmt.Errorf("unable to change job action - %s", err)
	}
	return false, nil
}

func addJobToWorker(jobQueue *jobqueue.Client, t *jobqueue.Task) (bool, error) {
	id, err := jobQueue.AddTask(t.Job)
	if err != nil {
		return true, fmt.Errorf("unable to put to work queue %s", err)
	}

	log.WithFields(log.Fields{
		"task": id,
		"job":  t.Job.ID,
	}).Debug("added job to work queue")

	return false, nil
}

// HACK: returning true trigegrs a task deletion in main
func deleteTask(jobQueue *jobqueue.Client, t *jobqueue.Task) (bool, error) {
	return true, nil
}

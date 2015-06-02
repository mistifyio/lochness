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
	"github.com/coreos/go-etcd/etcd"
	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
	logx "github.com/mistifyio/mistify-logrus-ext"
	flag "github.com/ogier/pflag"
)

// XXX: allow different tube names?
const (
	CreateTube = "create"
	WorkTube   = "work"
)

// TaskFunc is a convenience wrapper for function calls on tasks
type TaskFunc struct {
	name     string
	function func(*Task) (bool, error)
	label    string // label for metrics
}

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

// TODO: restructure this as all the deletes for tube stuff is clunky.
// as we almost always delete the tube id, wrap in function and delete it?

func main() {
	bstalk := flag.StringP("beanstalk", "b", "127.0.0.1:11300", "address of beanstalkd server")
	logLevel := flag.StringP("log-level", "l", "warn", "log level")
	addr := flag.StringP("etcd", "e", "http://127.0.0.1:4001", "address of etcd server")
	port := flag.UintP("http", "p", 7543, "address for http interface. set to 0 to disable")
	flag.Parse()

	// set with flag?
	if err := logx.DefaultSetup(*logLevel); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "logx.DefaultSetup",
			"level": *logLevel,
		}).Fatal("failed to set up logging")
	}

	log.WithField("address", *bstalk).Info("connection to beanstalk")
	conn, err := beanstalk.Dial("tcp", *bstalk)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": *bstalk,
		}).Fatal("failed to connect to beanstalk server")
	}

	log.WithField("address", *addr).Info("connection to etcd")
	etcdClient := etcd.NewClient([]string{*addr})

	// make sure we can actually talk to etcd
	if !etcdClient.SyncCluster() {
		log.WithFields(log.Fields{
			"error":   err,
			"address": *addr,
		}).Fatal("failed to connect to etcd cluster")
	}

	//inm := metrics.NewInmemSink(10*time.Second, 5*time.Minute)
	ms := mapsink.New()
	conf := metrics.DefaultConfig("cplacerd")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, ms)

	if *port != 0 {

		http.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(ms); err != nil {
				log.WithField("error", err).Error(err)
			}
		}))

		go func() {
			if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"func":  "http.ListenAndServe",
				}).Fatal("error serving")
			}
		}()

	}

	c := lochness.NewContext(etcdClient)

	ts := beanstalk.NewTubeSet(conn, CreateTube)

	// XXX: we want to try to keep track of where a job is
	// in this pipeline? would have to persist in the job
	funcs := []TaskFunc{
		TaskFunc{
			name:     "get a job",
			function: getJob,
		},
		TaskFunc{
			name:     "check job status",
			function: checkJobStatus,
		},
		TaskFunc{
			name:     "get guest",
			function: getGuest,
		},
		TaskFunc{
			name:     "check guest status",
			function: checkGuestStatus,
		},
		TaskFunc{
			name:     "select hypervisor candidate",
			function: selectHypervisor,
		},
		TaskFunc{
			name:     "update job action",
			function: changeJobAction,
		},
		TaskFunc{
			name:     "add task to worker",
			function: addJobToWorker,
		},
		TaskFunc{
			name:     "make task for deletion",
			function: deleteTask,
		},
	}

	for _, f := range funcs {
		f.label = strings.Split(runtime.FuncForPC(reflect.ValueOf(f.function).Pointer()).Name(), ".")[1]
	}

	for {
		id, body, err := ts.Reserve(10 * time.Hour)
		if err != nil {
			switch err.(beanstalk.ConnError) {
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

		task := &Task{
			ID:   id,
			Body: body,
			conn: conn,
			ctx:  c,
		}

		for _, f := range funcs {

			fields := log.Fields{
				"task": task.ID,
				"body": string(task.Body),
				"func": f.name,
			}

			if task.Job != nil {
				fields["job"] = task.Job.ID
			}

			log.WithFields(fields).Debug("running")

			start := time.Now()
			rm, err := f.function(task)

			m.MeasureSince([]string{f.label, "time"}, start)
			m.IncrCounter([]string{f.label, "count"}, 1)

			duration := int(time.Since(start).Seconds() * 1000)
			log.WithFields(fields).WithField("duration", duration).Info("done")

			if err != nil {

				m.IncrCounter([]string{f.label, "error"}, 1)

				log.WithFields(fields).WithField("error", err).Error("task error")

				if task.Job != nil {
					task.Job.Status = lochness.JobStatusError
					task.Job.Error = err.Error()
					if err := task.Job.Save(24 * time.Hour); err != nil {
						log.WithFields(log.Fields{
							"job":   task.Job.ID,
							"task":  task.ID,
							"error": err,
						}).Error("unable to save")
					}
				}
				rm = true
			}

			if rm {
				if err := conn.Delete(id); err != nil {
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

func getJob(t *Task) (bool, error) {
	id := string(t.Body)
	j, err := t.ctx.Job(id)

	if err != nil {
		return true, err
	}

	t.Job = j
	return false, nil

}

func checkJobStatus(t *Task) (bool, error) {
	if t.Job.Status != lochness.JobStatusNew {
		return true, fmt.Errorf("bad job status: %s", t.Job.Status)
	}
	if t.Job.Action != "select-hypervisor" {
		return true, fmt.Errorf("bad action: %s", t.Job.Action)
	}
	return false, nil
}

func getGuest(t *Task) (bool, error) {
	g, err := t.ctx.Guest(t.Job.Guest)
	if err != nil {
		return true, fmt.Errorf("unable to get guest %s - %s", t.Job.Guest, err)
	}
	t.Guest = g
	return false, nil
}

func checkGuestStatus(t *Task) (bool, error) {

	if t.Guest.HypervisorID != "" {
		return true, fmt.Errorf("guest already has a hypervisor %s - %s", t.Guest.ID, t.Guest.HypervisorID)
	}
	return false, nil
}

func selectHypervisor(t *Task) (bool, error) {
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

func changeJobAction(t *Task) (bool, error) {
	t.Job.Action = "fetch"
	if err := t.Job.Save(24 * time.Hour); err != nil {
		return true, fmt.Errorf("unable to change job action - %s", err)
	}
	return false, nil
}

func addJobToWorker(t *Task) (bool, error) {
	tube := beanstalk.Tube{
		Conn: t.conn,
		Name: WorkTube,
	}

	// TODO: should ttr be configurable?
	id, err := tube.Put([]byte(t.Job.ID), 0, 5*time.Second, 5*time.Minute)
	if err != nil {
		return true, fmt.Errorf("unable to put to work queue %s", err)
	}

	log.WithFields(log.Fields{
		"id":  id,
		"job": t.Job.ID,
	}).Debug("ading job to work queue")

	return false, nil
}

// HACK: returning true trigegrs a task deletion in main
func deleteTask(t *Task) (bool, error) {
	return true, nil
}

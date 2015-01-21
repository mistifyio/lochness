package main

// TODO: multiple beanstalkd servers

import (
	"flag"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/kr/beanstalk"
	"github.com/mistifyio/lochness"
)

// XXX: allow different tube names?
const (
	CREATE_TUBE = "create"
	WORK_TUBE   = "work"
)

type TaskFunc func(*Task) (bool, error)

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
	bstalk := flag.String("beanstalk", "127.0.0.1:11300", "address of beanstalkd server")
	logLevel := flag.String("log-level", "warn", "log level")
	addr := flag.String("etcd", "http://127.0.0.1:4001", "address of etcd server")
	flag.Parse()

	// set with flag?
	log.SetFormatter(&log.JSONFormatter{})

	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(level)

	log.Infof("using beanstalk %s", *bstalk)

	conn, err := beanstalk.Dial("tcp", *bstalk)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("using etcd %s", *addr)
	etcdClient := etcd.NewClient([]string{*addr})

	if !etcdClient.SyncCluster() {
		log.Fatal("unable to sync etcd at %s", *addr)
	}

	c := lochness.NewContext(etcdClient)

	ts := beanstalk.NewTubeSet(conn, CREATE_TUBE)

	// XXX: we want to try to keep track of where a job is
	// in this pipeline? would have to persist in the job
	funcs := []TaskFunc{
		getJob,
		checkJobStatus,
		getGuest,
		checkGuestStatus,
		selectHypervisor,
		changeJobStatus,
		addJobToWorker,
		deleteTask,
	}

	for {
		id, body, err := ts.Reserve(10 * time.Hour)
		if err != nil {
			if err.(beanstalk.ConnError).Err == beanstalk.ErrTimeout {
				// nothing queued, so just retry
				continue
			} else {
				// take a dirt nap
				log.Fatal(err)
			}
		}

		task := &Task{
			ID:   id,
			Body: body,
			Conn: conn,
			ctx:  c,
		}

		for _, f := range funcs {
			rm, err := f(task)

			if err != nil {
				log.Errorf("task error for %d: %s", id, err)
				if t.Job {
					t.Job.Status = "Error"
					t.Job.Error = err.Error()
					if err := t.Job.Save(); err != nil {
						log.Errorf("unable to save job %s - %s", t.Job.ID, err)
					}
				}
			}

			if rm {
				if err := conn.Delete(id); err != nil {
					log.Errorf("delete error for %d: %s", id, err)
				}
			}
		}

	}
}

// these funcs return bool if task in beanstalk should be deleted (and loop stopped). loop also stops on error

func getJob(t *Task) (bool, error) {
	j, err := t.ctx.Job(t.ID)

	if err != nil {
		log.Errorf("unable to get job %s: %s", t.ID, err)
		if lochness.IsKeyNotFound(err) {
			return true, nil
		}
	}

	t.Job = j
	return false, nil

}

func checkJobStatus(t *Task) (bool, error) {
	if t.Job.Status != "new" {
		//?? should we care?
		return true, fmt.Errorf("bad job status: %s", t.Job.Status)
	}
	if t.Job.Action != "select-hypervisor" {
		return true, fmt.Errorf("bad action: %s", t.Job.Action)
	}
	return false, nil
}

func getGuest(t *Task) (bool, error) {
	g, err := c.Guest(j.Guest)
	if err != nil {
		return true, fmt.Errorf("unable to get guest %s - %s", j.Guest, err)
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
	candidates, err := t.Guest.Candidates(lochness.DefaultCadidateFuctions...)
	if err != nil {
		return true, fmt.Errorf("unable to select candidate %s - %s", t.Guest.ID, err)
	}

	if len(candidates) == 0 {
		return true, fmt.Errorf("no candidates found for  %s", t.Guest.ID)
	}

	h := candidates[0]
	if err := h.AddGuest(g); err != nil {
		return true, fmt.Errorf("unable to add guest %s to %s - %s", t.Guest.ID, h.ID, err)
	}

	return false, nil
}

func changeJobStatus(t *Task) (bool, error) {
	t.Job.Action = "hypervisor-create"
	if err := t.Job.Save(); err != nil {
		return true, fmt.Errorf("unable to change job action - %s", err)
	}
	return false, nil
}

func addJobToWorker(t *Task) (bool, error) {
	tube := beanstalk.Tube{
		Conn: t.conn,
		Name: WORK_TUBE,
	}

	// TODO: should ttr be configurable?
	id, err := tube.Put([]byte(t.Job.Id), 0, 5*time.Minute)
	if err != nil {
		return true, fmt.Errorf("unable to put to work queue %s", err)
	}

	log.Debugf("added %d to work queue for %s", id, t.Job.Id)

	return false, nil
}

func deleteTask(t *Task) (bool, error) {
	return true, nil
}

package jobqueue

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/kr/beanstalk"
)

// Beanstalk tube names
const (
	// WorkTube is the name of the beanstalk tube for work tasks
	workTube = "work"
	// CreateTube is the name of the beanstalk tube for new guest creation
	createTube = "create"
)

type (
	// tubeSet holds a tube for publishing and tubeset for consuming a queue
	tubeSet struct {
		publish *beanstalk.Tube
		consume *beanstalk.TubeSet
	}

	// tubes holds the create and work tubeSets
	tubes struct {
		create *tubeSet
		work   *tubeSet
	}
)

// newTubeSet creates a new tubeSet for a tube name
func newTubeSet(conn *beanstalk.Conn, name string) *tubeSet {
	return &tubeSet{
		consume: beanstalk.NewTubeSet(conn, name),
		publish: &beanstalk.Tube{
			Conn: conn,
			Name: name,
		},
	}
}

// Put puts a job into the publish tube.
// See http://godoc.org/github.com/kr/beanstalk#Tube.Put
func (ts *tubeSet) Put(jobID string) (uint64, error) {
	body := []byte(jobID)
	id, err := ts.publish.Put(body, priority, delay, ttr)
	return id, err
}

// Reserve reserves and returns an item from the consume tubeset.
// See http://godoc.org/github.com/kr/beanstalk#TubeSet.Reserve
func (ts *tubeSet) Reserve() (uint64, string, error) {
	for {
		id, body, err := ts.consume.Reserve(timeout)
		if err != nil {
			switch err.(beanstalk.ConnError) {
			case beanstalk.ErrTimeout:
				// Empty queue, continue waiting
				continue
			case beanstalk.ErrDeadline:
				log.Debug("beanstalk.ErrDeadline")
				time.Sleep(reserveDelay)
				continue
			}
		}
		return id, string(body), err
	}
}

// newTubes creates a new tubes
func newTubes(conn *beanstalk.Conn) *tubes {
	return &tubes{
		create: newTubeSet(conn, createTube),
		work:   newTubeSet(conn, workTube),
	}
}

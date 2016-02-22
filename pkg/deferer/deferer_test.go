package deferer_test

import (
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/pkg/deferer"
	"github.com/stretchr/testify/suite"
)

// log.Fatal(msg) will log the msg and end with a call to os.Exit(). There's no
// way to handle an os.Exit, which means the tests will exit. To get around
// this, use a log hook, which runs first, that panics. The panic will break
// the code flow that normally leads to an exit, allowing recovery in the test,
// assertions to be made, and testing to continue.
var logHook *FatalTestHook

type FatalTestHook struct {
	lastEntry *log.Entry
}

func (fth *FatalTestHook) Levels() []log.Level {
	return []log.Level{
		log.FatalLevel,
	}
}

func (fth *FatalTestHook) Fire(e *log.Entry) error {
	fth.lastEntry = e
	panic("avoid fatal")
}

type DefererSuite struct {
	suite.Suite
}

func TestDeferer(t *testing.T) {
	suite.Run(t, new(DefererSuite))
}

func (s *DefererSuite) TestNewDeferer() {
	s.NotNil(deferer.NewDeferer(nil))
}

func (s *DefererSuite) TestDeferAndRun() {

	results := []int{}
	expectedResults := []int{4, 3, 7, 6, 5, 2, 1}

	func() {
		d := deferer.NewDeferer(nil)
		defer d.Run()
		d.Defer(func() { results = append(results, 1) })
		d.Defer(func() { results = append(results, 2) })

		func() {
			d2 := deferer.NewDeferer(d)
			defer d2.Run()
			d2.Defer(func() { results = append(results, 3) })
			d2.Defer(func() { results = append(results, 4) })
		}()

		d.Defer(func() { results = append(results, 5) })
		d.Defer(func() { results = append(results, 6) })

		results = append(results, 7)
	}()

	s.Equal(expectedResults, results)
}

func (s *DefererSuite) TestFatal() {
	fatalMsg := "fatal message"
	results := make([]int, 0)
	expectedResults := []int{2, 1}

	defer func() {
		if r := recover(); r != nil {
			s.Contains(logHook.lastEntry.Message, fatalMsg)
			s.Equal(expectedResults, results)
		}
	}()

	func() {
		d := deferer.NewDeferer(nil)
		defer d.Run()
		d.Defer(func() { results = append(results, 1) })

		func() {
			d2 := deferer.NewDeferer(d)
			defer d2.Run()
			d2.Defer(func() { results = append(results, 2) })
			d2.Fatal(fatalMsg)
		}()
	}()
}

func (s *DefererSuite) TestFatalWithFields() {
	fatalMsg := "fatal message"
	fatalFields := log.Fields{"foo": "bar"}
	results := make([]int, 0)
	expectedResults := []int{2, 1}

	defer func() {
		if r := recover(); r != nil {
			s.Contains(logHook.lastEntry.Message, fatalMsg)
			s.Equal(fatalFields, logHook.lastEntry.Data)
			s.Equal(expectedResults, results)
		}
	}()

	func() {
		d := deferer.NewDeferer(nil)
		defer d.Run()
		d.Defer(func() { results = append(results, 1) })

		func() {
			d2 := deferer.NewDeferer(d)
			defer d2.Run()
			d2.Defer(func() { results = append(results, 2) })
			d2.FatalWithFields(fatalFields, fatalMsg)
		}()
	}()
}

func init() {
	logHook = new(FatalTestHook)
	log.AddHook(logHook)
}

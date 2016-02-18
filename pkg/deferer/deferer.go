// Package deferer provides a way to use defer calls with log.Fatal. Using
// log.Fatal() is effecively the same as calling fmt.Println() followed by
// os.Exit(1). The normal defer methods are not run when os.Exit() is called
// but sometimes it is necessary (e.g. release a lock).
package deferer

import (
	"fmt"
	"path/filepath"
	"runtime"

	log "github.com/Sirupsen/logrus"
)

// Deferer holds a slice of deferred functions and an optional pointer to the
// caller's Deferrer
type Deferer struct {
	caller *Deferer
	fns    []func()
	ran    bool
}

// Defer adds to the array of defered function calls
func (d *Deferer) Defer(f func()) {
	d.fns = append(d.fns, f)
}

// Run calls each function in the defered array in reverse order. Common usage
// is to call `defer d.Run()` after creating the Deferer instance
func (d *Deferer) Run() {
	if d.ran {
		return
	}

	for i := len(d.fns) - 1; i >= 0; i-- {
		d.fns[i]()
	}
	d.ran = true
}

// Fatal runs each set of deferred functions, walking up the call change if the
// parent property is set, finishing with a call to log.Fatal()
func (d *Deferer) Fatal(v ...interface{}) {
	d.fatal()
	// Need to grab the original caller file and line number
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Fatal(v...)
	} else {
		base := filepath.Base(file)
		args := []interface{}{interface{}(fmt.Sprintf("%s:%d: ", base, line))}
		log.Fatal(append(args, v...)...)
	}
}

// FatalWithFields accepts additional logging fields for the fatal log
func (d *Deferer) FatalWithFields(fields log.Fields, v ...interface{}) {
	d.fatal()
	// Need to grab the original caller file and line number
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		log.WithFields(fields).Fatal(v...)
	} else {
		base := filepath.Base(file)
		fields["file"] = base
		fields["line"] = line
		log.WithFields(fields).Fatal(v...)
	}
}

func (d *Deferer) fatal() {
	d.Run()
	if d.caller != nil {
		d.caller.fatal()
	}
}

// NewDeferer returns a pointer to a new Deferer instance with the function
// slice initialized and the optional caller set
func NewDeferer(d *Deferer) *Deferer {
	return &Deferer{
		caller: d,
		fns:    make([]func(), 0),
	}
}

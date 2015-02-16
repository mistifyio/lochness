package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-systemd/dbus"
	"github.com/mistifyio/lochness/pkg/deferer"
	"github.com/mistifyio/lochness/pkg/lock"
	"github.com/mistifyio/lochness/pkg/sd"
)

const service = `[Unit]
Description=Cluster unique %s

[Service]
Type=oneshot
ExecStart=
ExecStart=%s
`

type params struct {
	Interval uint64     `json:"interval"`
	TTL      uint64     `json:"ttl"`
	Key      string     `json:"key"`
	Addr     string     `json:"addr"`
	Blocking bool       `json:"blocking"`
	ID       int        `json:"id"`
	Args     []string   `json:"args"`
	Lock     *lock.Lock `json:"lock"`
}

func runService(dc *deferer.Deferer, serviceDone chan struct{}, id int, name string, args []string) {
	d := deferer.NewDeferer(dc)
	defer d.Run()

	conn, err := dbus.New()
	if err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "dbus.New",
		}, "error creating new dbus connection")
	}

	f, err := os.Create("/run/systemd/system/" + name)
	if err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
		}, "error creating service file")
	}
	d.Defer(func() { f.Close() })

	base := filepath.Base(args[0])
	// For args with spaces, quote 'em
	for i, v := range args {
		if strings.Contains(v, " ") {
			args[i] = "'" + v + "'"
		}
	}
	dotService := fmt.Sprintf(service, base, strings.Join(args, " "))
	_, err = f.WriteString(dotService)
	if err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "f.WriteString",
		}, "error writing service file")
	}
	f.Sync()

	done := make(chan string)
	_, err = conn.StartUnit(name, "fail", done)
	if err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "conn.StartUnit",
		}, "error starting service")
	}

	status := <-done
	if status != "done" {
		d.FatalWithFields(log.Fields{
			"status": status,
			"func":   "StartUnit",
		}, "StartUnit returned a bad status")
	}

	serviceDone <- struct{}{}
}

func killService(name string, signal int32) error {
	conn, err := dbus.New()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "dbus.New",
		}).Error("error creating new dbus connection")
		return err
	}

	conn.KillUnit(name, signal)
	return nil
}

func refresh(lock *lock.Lock, interval uint64) chan struct{} {
	ch := make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		for {
			select {
			case <-ticker.C:
				if err := lock.Refresh(); err != nil {
					// TODO: Should we just log.Fatal here?
					// So that systemd kills the services
					// ASAP?
					ch <- struct{}{}
				}
			case <-ch:
				lock.Release()
				return
			}
		}
	}()

	return ch
}

func tickle(interval uint64) chan struct{} {
	tickler := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		for range ticker.C {
			sd.Notify("WATCHDOG=1")
		}
		tickler <- struct{}{}
	}()
	return tickler
}

func main() {
	d := deferer.NewDeferer(nil)
	defer d.Run()

	params := params{}
	arg, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "base64.DecodeString",
			"arg":   os.Args[1],
		}, "error decoding arg string")
	}
	if err := json.Unmarshal(arg, &params); err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "json.Unmarshal",
			"json":  arg,
		}, "error unmarshaling json")
	}

	l := params.Lock
	if err := l.Refresh(); err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "lock.Refresh",
		}, "failed to refresh lock")
	}
	d.Defer(func() { l.Release() })
	locker := refresh(l, params.Interval)

	sdttl, err := sd.WatchdogEnabled()
	if err != nil {
		d.FatalWithFields(log.Fields{
			"error": err,
			"func":  "sd.WatchdogEnabled",
		}, "failed to check watchdog configuration")
	}
	if uint64(sdttl.Seconds()) != params.TTL {
		d.FatalWithFields(log.Fields{
			"serviceTTL": sdttl,
			"paramTTL":   params.TTL,
		}, "params and systemd ttls do not match")
	}
	tickler := tickle(params.Interval)

	serviceDone := make(chan struct{})
	base := filepath.Base(params.Args[0])
	target := fmt.Sprintf("locked-%s-%d.service", base, params.ID)
	go runService(d, serviceDone, params.ID, target, params.Args)

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-locker:
		// TODO: should we never expect this?
		killService(target, int32(syscall.SIGINT))
	case <-serviceDone:
		close(locker)
	case s := <-sigs:
		log.WithField("signal", s).Info("signal received")
		killService(target, int32(s.(syscall.Signal)))
		close(locker)
	case <-tickler:
		// watchdog tickler stopped, uh oh we are going down pretty soon
		killService(target, int32(syscall.SIGINT))
		close(locker)
	}
}

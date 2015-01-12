package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
		d.Fatal(err)
	}

	f, err := os.Create("/run/systemd/system/" + name)
	if err != nil {
		d.Fatal(err)
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
		d.Fatal(err)
	}
	f.Sync()

	done := make(chan string)
	_, err = conn.StartUnit(name, "fail", done)
	if err != nil {
		d.Fatal(err)
	}

	status := <-done
	if status != "done" {
		d.Fatal(errors.New(name + " " + status))
	}
	log.Println("status:", status)

	serviceDone <- struct{}{}
}

func killService(name string, signal int32) error {
	conn, err := dbus.New()
	if err != nil {
		log.Println("err:", err)
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
					log.Println("sending to ch")
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

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	params := params{}
	arg, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		d.Fatal(err)
	}
	if err := json.Unmarshal(arg, &params); err != nil {
		d.Fatal(err)
	}

	l := params.Lock
	if err := l.Refresh(); err != nil {
		d.Fatal(err)
	}
	d.Defer(func() { l.Release() })
	locker := refresh(l, params.Interval)

	sdttl, err := sd.WatchdogEnabled()
	if err != nil {
		d.Fatal(err)
	}
	if uint64(sdttl.Seconds()) != params.TTL {
		d.Fatal("params and systemd ttls do not match")
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
		log.Println("got a sig:", s)
		killService(target, int32(s.(syscall.Signal)))
		close(locker)
	case <-tickler:
		// watchdog tickler stopped, uh oh we are going down pretty soon
		killService(target, int32(syscall.SIGINT))
		close(locker)
	}
}

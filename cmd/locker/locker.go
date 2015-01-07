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
	"syscall"
	"time"

	"github.com/coreos/go-systemd/dbus"
	"github.com/mistifyio/lochness/pkg/lock"
	"github.com/mistifyio/lochness/pkg/sd"
)

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

func startService(id int, name string, args []string) (chan struct{}, error) {
	conn, err := dbus.New()
	if err != nil {
		return nil, err
	}

	base := filepath.Base(args[0])
	desc := fmt.Sprintf("Cluster unique %s", base)
	props := []dbus.Property{
		dbus.PropDescription(desc),
		dbus.PropExecStart(args, false),
		dbus.PropBindsTo(fmt.Sprintf("%s-locker-%d.service", base, id)),
	}

	done := make(chan string)
	_, err = conn.StartTransientUnit(name, "fail", props, done)
	if err != nil {
		return nil, err
	}

	status := <-done
	if status != "done" {
		return nil, errors.New("failed to start service")
	}

	subset := conn.NewSubscriptionSet()
	subset.Add(name)
	statuses, errs := subset.Subscribe()

	statdone := make(chan struct{})
	go monService(statuses, errs, statdone)
	return statdone, nil
}

func monService(statuses <-chan map[string]*dbus.UnitStatus, errs <-chan error, resp chan<- struct{}) {
	for {
		select {
		case err := <-errs:
			log.Printf("error: %#v\n", err)
		case status := <-statuses:
			for _, v := range status {
				if v == nil {
					log.Println("nil, exiting")
					resp <- struct{}{}
					return
				}
				if v.ActiveState == "failed" {
					log.Println("service failed:", v)
					resp <- struct{}{}
					return
				}
				if v.ActiveState == "inactive" {
					log.Println("service is inactive:", v)
					resp <- struct{}{}
					return
				}
			}
		}
	}
}

func stopService(name string) error {
	conn, err := dbus.New()
	if err != nil {
		log.Println("err:", err)
		return err
	}

	done := make(chan string)
	_, err = conn.StopUnit(name, "fail", done)
	if err != nil {
		log.Println("err:", err)
		return err
	}

	status := <-done
	if status != "done" {
		log.Println("err:", err)
		return errors.New("failed to stop service")
	}
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
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	params := params{}
	arg, err := base64.StdEncoding.DecodeString(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(arg, &params); err != nil {
		log.Fatal(err)
	}

	l := params.Lock
	if err := l.Refresh(); err != nil {
		log.Fatal(err)
	}
	defer l.Release()
	locker := refresh(l, params.Interval)

	sdttl, err := sd.WatchdogEnabled()
	if err != nil {
		log.Fatal(err)
	}
	if uint64(sdttl.Seconds()) != params.TTL {
		log.Fatal("params and systemd ttls do not match")
	}
	tickler := tickle(params.Interval)

	base := filepath.Base(params.Args[0])
	target := fmt.Sprintf("%s-locked-%d.service", base, params.ID)
	service, err := startService(params.ID, target, params.Args)
	if err != nil {
		log.Fatal(err)
	}

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-locker:
		// TODO: should we never expect this?
		stopService(target)
	case <-service:
		close(locker)
	case <-sigs:
		stopService(target)
		close(locker)
	case <-tickler:
		// watchdog tickler stopped, uh oh we are going down pretty soon
		stopService(target)
		close(locker)
	}
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/coreos/go-systemd/dbus"
	"github.com/mistifyio/lochness/pkg/deferer"
	"github.com/mistifyio/lochness/pkg/lock"
)

const defaultAddr = "http://localhost:4001"
const service = `[Unit]
Description=Cluster unique %s locker

[Service]
ExecStart=
ExecStart=%s "%s"
WatchdogSec=%d
BoundBy=%s-locked-%d.service
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

func cmdrun(dc *deferer.Deferer, done chan struct{}, id int, ttl uint64, cmd, base, arg string) {
	d := deferer.NewDeferer(dc)
	defer d.Run()

	target := fmt.Sprintf("%s-locker-%d.service", base, id)
	exited, err := startService(id, ttl, target, cmd, base, arg)
	if err != nil {
		d.Fatal(err)
	}

	select {
	case <-done:
		stopService(target)
		done <- struct{}{}
	case <-exited:
		done <- struct{}{}
	}
}

func startService(id int, ttl uint64, target, cmd, base, arg string) (chan struct{}, error) {
	conn, err := dbus.New()
	if err != nil {
		return nil, err
	}

	f, err := os.Create("/run/systemd/system/" + target)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	arg = base64.StdEncoding.EncodeToString([]byte(arg))
	dotService := fmt.Sprintf(service, base, cmd, arg, ttl, base, id)
	_, err = f.WriteString(dotService)
	if err != nil {
		return nil, err
	}
	f.Sync()

	done := make(chan string)
	_, err = conn.StartUnit(target, "fail", done)
	if err != nil {
		return nil, err
	}

	status := <-done
	if status != "done" {
		return nil, errors.New("failed to start service")
	}
	log.Println("status:", status)

	subset := conn.NewSubscriptionSet()
	subset.Add(target)
	statuses, errs := subset.Subscribe()

	serviceDone := make(chan struct{})
	go monService(statuses, errs, serviceDone)

	log.Println("services names are:")
	log.Printf("%s-locker-%d\n", base, id)
	log.Printf("%s-locked-%d\n", base, id)
	return serviceDone, nil
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

func resolveCommand(command string) (string, error) {
	command, err := exec.LookPath(command)
	if err != nil {
		return "", err
	}
	return filepath.Abs(command)
}

func main() {
	d := deferer.NewDeferer(nil)
	defer d.Run()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	rand.Seed(time.Now().UnixNano())
	id := rand.Int()
	if ID := os.Getenv("ID"); ID != "" {
		fmt.Sscanf(ID, "%d", &id)
	}

	params := params{ID: id}
	flag.Uint64Var(&params.Interval, "interval", 30, "Interval in seconds to refresh lock")
	flag.Uint64Var(&params.TTL, "ttl", 0, "TTL for key in seconds, leave 0 for (2 * interval)")
	flag.StringVar(&params.Key, "key", "/lock", "Key to use as lock")
	flag.BoolVar(&params.Blocking, "block", false, "Block if we failed to acquire the lock")
	flag.StringVar(&params.Addr, "etcd", defaultAddr, "address of etcd machine")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: [options] -- command args\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\ncommand will be run with args via fork/exec not a shell\n")
	}
	flag.Parse()

	if params.TTL == 0 {
		params.TTL = params.Interval * 2
	}

	params.Args = flag.Args()
	if len(params.Args) < 1 {
		d.Fatal("command is required")
	}
	cmd, err := resolveCommand(params.Args[0])
	if err != nil {
		d.Fatal(err)
	}
	params.Args[0] = cmd

	hostname, err := os.Hostname()
	if err != nil {
		d.Fatal(err)
	}

	c := etcd.NewClient([]string{params.Addr})
	l, err := lock.Acquire(c, params.Key, hostname, params.TTL, params.Blocking)
	if err != nil {
		d.Fatal("failed to get lock", params.Key, err)
	}
	d.Defer(func() { l.Release() })
	params.Lock = l

	args, err := json.Marshal(&params)
	if err != nil {
		d.Fatal(err)
	}

	cmddone := make(chan struct{})
	base := filepath.Base(params.Args[0])
	locker, err := resolveCommand("locker")
	if err != nil {
		d.Fatal(err)
	}
	go cmdrun(d, cmddone, params.ID, params.TTL, locker, base, string(args))

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-cmddone:
		log.Println("cmd is done")
	case <-sigs:
		log.Println("got a sig")
		cmddone <- struct{}{}
		<-cmddone
	}
}

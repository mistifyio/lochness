package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/coreos/go-systemd/dbus"
	"github.com/mistifyio/lochness/pkg/deferer"
	"github.com/mistifyio/lochness/pkg/lock"
	flag "github.com/ogier/pflag"
)

const defaultAddr = "http://localhost:4001"
const service = `[Unit]
Description=Cluster unique %s locker

[Service]
Type=oneshot
ExecStart=
ExecStart=%s "%s"
WatchdogSec=%d
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

func runService(dc *deferer.Deferer, serviceDone chan struct{}, id int, ttl uint64, target, cmd, base, arg string) {
	d := deferer.NewDeferer(dc)
	defer d.Run()

	conn, err := dbus.New()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "dbus.New",
		}).Error("error creating new dbus connection")
		d.Fatal(err)
	}

	f, err := os.Create("/run/systemd/system/" + target)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Create",
		}).Error("error creating service file")
		d.Fatal(err)
	}
	d.Defer(func() { f.Close() })

	arg = base64.StdEncoding.EncodeToString([]byte(arg))
	dotService := fmt.Sprintf(service, base, cmd, arg, ttl)
	_, err = f.WriteString(dotService)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "f.WriteString",
		}).Error("error writing service file")
		d.Fatal(err)
	}
	f.Sync()

	log.WithFields(log.Fields{
		"locker": fmt.Sprintf("locker-%s-%d", base, id),
		"locked": fmt.Sprintf("locked-%s-%d", base, id),
	}).Info("created service names")

	done := make(chan string)
	_, err = conn.StartUnit(target, "fail", done)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "conn.StartUnit",
		}).Error("error starting service")
		d.Fatal(err)
	}

	status := <-done
	if status != "done" {
		log.WithFields(log.Fields{
			"status": status,
			"func":   "StartUnit",
		}).Error("StartUnit returned a bad status")
		d.Fatal(errors.New(target + " " + status))
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

func resolveCommand(command string) (string, error) {
	command, err := exec.LookPath(command)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"exec":  command,
			"func":  "exec.LookPath",
		}).Error("could not find executable in path")
		return "", err
	}
	return filepath.Abs(command)
}

func main() {
	d := deferer.NewDeferer(nil)
	defer d.Run()

	rand.Seed(time.Now().UnixNano())
	id := rand.Int()
	if ID := os.Getenv("ID"); ID != "" {
		fmt.Sscanf(ID, "%d", &id)
	}

	params := params{ID: id}
	flag.Uint64VarP(&params.Interval, "interval", "i", 30, "Interval in seconds to refresh lock")
	flag.Uint64VarP(&params.TTL, "ttl", "t", 0, "TTL for key in seconds, leave 0 for (2 * interval)")
	flag.StringVarP(&params.Key, "key", "k", "/lock", "Key to use as lock")
	flag.BoolVarP(&params.Blocking, "block", "b", false, "Block if we failed to acquire the lock")
	flag.StringVarP(&params.Addr, "etcd", "e", defaultAddr, "address of etcd machine")
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
		log.Fatal("command is required")
	}
	cmd, err := resolveCommand(params.Args[0])
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"exec":  params.Args[0],
			"func":  "resolveCommand",
		}).Error("failed to resolveCommand")
		d.Fatal()
	}
	params.Args[0] = cmd

	hostname, err := os.Hostname()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Hostname",
		}).Error("failed to get hostname")
		d.Fatal()
	}

	c := etcd.NewClient([]string{params.Addr})
	l, err := lock.Acquire(c, params.Key, hostname, params.TTL, params.Blocking)
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"func":     "lock.Acquire",
			"lock":     params.Key,
			"ttl":      params.TTL,
			"blocking": params.Blocking,
		}).Error("failed to get lock")
		d.Fatal()
	}

	d.Defer(func() { l.Release() })
	params.Lock = l

	args, err := json.Marshal(&params)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "json.Marshal",
		}).Error("failed to serialize params")
		d.Fatal(err)
	}

	serviceDone := make(chan struct{})
	base := filepath.Base(params.Args[0])
	target := fmt.Sprintf("locker-%s-%d.service", base, id)
	locker, err := resolveCommand("locker")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"exec":  params.Args[0],
			"func":  "resolveCommand",
		}).Error("failed to resolveCommand")
		d.Fatal(err)
	}
	go runService(d, serviceDone, params.ID, params.TTL, target, locker, base, string(args))

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-serviceDone:
		log.WithField("service_state", "done").Info("service is done")
	case s := <-sigs:
		log.WithField("signal", s).Info("signal received")
		killService(target, int32(s.(syscall.Signal)))
	}
}

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
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
		d.Fatal(err)
	}

	f, err := os.Create("/run/systemd/system/" + target)
	if err != nil {
		d.Fatal(err)
	}
	d.Defer(func() { f.Close() })

	arg = base64.StdEncoding.EncodeToString([]byte(arg))
	dotService := fmt.Sprintf(service, base, cmd, arg, ttl)
	_, err = f.WriteString(dotService)
	if err != nil {
		d.Fatal(err)
	}
	f.Sync()

	log.Println("services names are:")
	log.Printf("locker-%s-%d\n", base, id)
	log.Printf("locked-%s-%d\n", base, id)

	done := make(chan string)
	_, err = conn.StartUnit(target, "fail", done)
	if err != nil {
		d.Fatal(err)
	}

	status := <-done
	if status != "done" {
		d.Fatal(errors.New(target + " " + status))
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

	serviceDone := make(chan struct{})
	base := filepath.Base(params.Args[0])
	target := fmt.Sprintf("locker-%s-%d.service", base, id)
	locker, err := resolveCommand("locker")
	if err != nil {
		d.Fatal(err)
	}
	go runService(d, serviceDone, params.ID, params.TTL, target, locker, base, string(args))

	sigs := make(chan os.Signal)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	select {
	case <-serviceDone:
		log.Println("cmd is done")
	case s := <-sigs:
		log.Println("got a sig:", s)
		killService(target, int32(s.(syscall.Signal)))
	}
}

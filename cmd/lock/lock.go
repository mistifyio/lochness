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
	"github.com/spf13/cobra"
)

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

var p = params{
	Interval: 30,
	TTL:      0,
	Key:      "/lock",
	Blocking: false,
	Addr:     "http://localhost:4001",
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

func run(ccmd *cobra.Command, rawArgs []string) {
	d := deferer.NewDeferer(nil)
	defer d.Run()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	rand.Seed(time.Now().UnixNano())
	id := rand.Int()
	if ID := os.Getenv("ID"); ID != "" {
		fmt.Sscanf(ID, "%d", &id)
	}
	p.ID = id

	if p.TTL == 0 {
		p.TTL = p.Interval * 2
	}

	p.Args = rawArgs
	if len(p.Args) < 1 {
		d.Fatal("command is required")
	}

	cmd, err := resolveCommand(p.Args[0])
	if err != nil {
		d.Fatal(err)
	}
	p.Args[0] = cmd

	hostname, err := os.Hostname()
	if err != nil {
		d.Fatal(err)
	}

	c := etcd.NewClient([]string{p.Addr})
	l, err := lock.Acquire(c, p.Key, hostname, p.TTL, p.Blocking)
	if err != nil {
		d.Fatal("failed to get lock", p.Key, err)
	}
	d.Defer(func() { l.Release() })
	p.Lock = l

	args, err := json.Marshal(&p)
	if err != nil {
		d.Fatal(err)
	}

	serviceDone := make(chan struct{})
	base := filepath.Base(p.Args[0])
	target := fmt.Sprintf("locker-%s-%d.service", base, id)
	locker, err := resolveCommand("locker")
	if err != nil {
		d.Fatal(err)
	}
	go runService(d, serviceDone, p.ID, p.TTL, target, locker, base, string(args))

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

func main() {
	root := &cobra.Command{
		Use:  "lock",
		Long: "lock provides a way to obtain a cluster wide lock for a service",
		Run:  run,
	}
	root.Flags().Uint64VarP(&p.Interval, "interval", "i", p.Interval, "Interval in seconds to refresh lock")
	root.Flags().Uint64VarP(&p.TTL, "ttl", "t", p.TTL, "TTL for key in seconds, leave 0 for (2 * interval)")
	root.Flags().StringVarP(&p.Key, "key", "k", p.Key, "Key to use as lock")
	root.Flags().BoolVarP(&p.Blocking, "block", "b", p.Blocking, "Block if we failed to acquire the lock")
	root.Flags().StringVarP(&p.Addr, "etcd", "e", p.Addr, "address of etcd machine")

	root.Execute()
}

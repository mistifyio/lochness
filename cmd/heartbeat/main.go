package main

import (
	"log"
	"os"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	"github.com/spf13/cobra"
)

var (
	interval = 60
	ttl      = 2 * interval
	eaddr    = "http://localhost:4001"
	id       = ""
)

func run(cmd *cobra.Command, args []string) {
	e := etcd.NewClient([]string{eaddr})
	c := lochness.NewContext(e)

	hn, err := lochness.SetHypervisorID(id)
	if err != nil {
		log.Fatal(err)
	}

	hv, err := c.Hypervisor(hn)
	if err != nil {
		log.Fatal(err)
	}

	for {
		if err = hv.UpdateResources(); err != nil {
			log.Println(err)
		}
		if err = hv.Heartbeat(time.Duration(ttl)); err != nil {
			log.Println(err)
		}
		os.Stdout.WriteString("♥ ")
		os.Stdout.Sync()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func main() {
	root := &cobra.Command{
		Use:  "heartbeat",
		Long: "heartbeat periodically updates etcd with current resources and alive status for a hypervisor",
		Run:  run,
	}
	root.Flags().IntVarP(&interval, "interval", "i", interval, "update interval in seconds")
	root.Flags().IntVarP(&ttl, "ttl", "t", 2*(interval), "heartbeat ttl in seconds")
	root.Flags().StringVarP(&eaddr, "etcd", "e", eaddr, "address of etcd machine")
	root.Flags().StringVar(&id, "hid", id, "hypervisor id")

	root.Execute()
}

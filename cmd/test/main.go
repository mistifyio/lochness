package main

import (
	"log"

	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

func print(i interface{}) {
	log.Printf("%+v\n", i)
}
func main() {
	e := etcd.NewClient([]string{"http://127.0.0.1:4001"})
	c := lochness.NewContext(e)

	f := c.NewFlavor()
	f.CPU = 4
	f.Memory = 4096
	f.Disk = 8192
	if err := f.Save(); err != nil {
		log.Fatal(err)
	}
	print(f)

	h := c.NewHypervisor()
	if err := h.Save(); err != nil {
		log.Fatal(err)
	}
	print(h)

}

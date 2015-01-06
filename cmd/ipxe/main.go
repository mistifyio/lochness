package main

import (
	"flag"

	"log"
	"net/http"

	"text/template"

	"github.com/coreos/go-etcd/etcd"
	"github.com/gorilla/mux"
	"github.com/mistifyio/lochness"
)

func main() {
	address := flag.String("port", ":8888", "address to listen")
	eaddr := flag.String("etcd", "http://localhost:4001", "address of etcd machine")
	baseUrl := flag.String("base", "http://127.0.0.1:8080", "base address of bits request") // this could/should be discovered in etcd?
	defaultVersion := flag.String("version", "0.1.0", "If all else fails, what version to serve")
	imageDir := flag.String("images", "/var/lib/images", "directory containing the images")
	flag.Parse()

	e := etcd.NewClient([]string{*eaddr})
	c := lochness.NewContext(e)

	r := mux.NewRouter()

	r.PathPrefix("/images").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir(*imageDir))))

	// this is a little horrible... clean it up with a "Config" struct or something
	r.HandleFunc("/ipxe/{ip}",
		func(w http.ResponseWriter, r *http.Request) {
			// we should make sure it actually looks like a valid ip
			var found *lochness.Hypervisor

			// this currently loops over all hypervisors, no matter what...
			c.ForEachHypervisor(func(h *lochness.Hypervisor) error {
				ip := h.IP.String()
				if ip == mux.Vars(r)["ip"] {
					found = h
				}
				return nil
			})

			if found == nil {
				http.NotFound(w, r)
				return
			}

			// do we need to parse this evertime??
			t, err := template.New("ipxe").Parse(ipxeTemplate)

			version := found.Config["version"]
			if version == "" {
				version, err = c.GetConfig("defaultVersion")
				if err != nil {
					// not found? should be fatal?
				}
				if version == "" {
					version = *defaultVersion
				}
			}
			data := map[string]string{
				"BaseUrl": *baseUrl,
				"Options": "", // options would be any other things we set. probably want to pass along UUID, etc
				"Version": version,
			}
			err = t.Execute(w, data)

		})

	log.Fatal(http.ListenAndServe(*address, r))
}

const ipxeTemplate = `#!ipxe
kernel {{.BaseUrl}}/images/{{.Version}}/vmlinuz {{.Options}}
initrd {{.BaseUrl}}/images/{{.Version}}/initrd
boot
`

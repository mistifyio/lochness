package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"text/template"

	"github.com/bakins/net-http-recover"
	"github.com/coreos/go-etcd/etcd"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/mistifyio/lochness"
)

type Server struct {
	ctx            *lochness.Context
	t              *template.Template
	defaultVersion string
	baseUrl        string
}

const ipxeTemplate = `#!ipxe
kernel {{.BaseUrl}}/images/{{.Version}}/vmlinuz {{.Options}}
initrd {{.BaseUrl}}/images/{{.Version}}/initrd
boot
`

func main() {
	address := flag.String("port", ":8888", "address to listen")
	eaddr := flag.String("etcd", "http://localhost:4001", "address of etcd machine")
	baseUrl := flag.String("base", "http://ipxe.mistify.local:8888", "base address of bits request")
	defaultVersion := flag.String("version", "0.1.0", "If all else fails, what version to serve")
	imageDir := flag.String("images", "/var/lib/images", "directory containing the images")
	flag.Parse()

	e := etcd.NewClient([]string{*eaddr})
	c := lochness.NewContext(e)

	router := mux.NewRouter()
	router.StrictSlash(true)

	// default mux will have the profiler handlers
	router.PathPrefix("/debug/").Handler(http.DefaultServeMux)
	router.PathPrefix("/images").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir(*imageDir))))

	s := &Server{
		ctx:            c,
		t:              template.Must(template.New("ipxe").Parse(ipxeTemplate)),
		defaultVersion: *defaultVersion,
		baseUrl:        *baseUrl,
	}

	chain := alice.New(
		func(h http.Handler) http.Handler {
			return recovery.Handler(os.Stderr, h, true)
		},
		func(h http.Handler) http.Handler {
			return handlers.CombinedLoggingHandler(os.Stdout, h)
		},
		handlers.CompressHandler,
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				context.Set(r, "_server_", s)
				h.ServeHTTP(w, r)
			})
		},
	)

	router.Handle("/ipxe/{ip}", chain.ThenFunc(ipxeHandler))

	log.Fatal(http.ListenAndServe(*address, router))
}

func ipxeHandler(w http.ResponseWriter, r *http.Request) {

	s := context.Get(r, "_server_").(*Server)

	ip := mux.Vars(r)["ip"]

	if net.ParseIP(ip) == nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	var found *lochness.Hypervisor

	// this currently loops over all hypervisors. do we need a way to exit early?
	s.ctx.ForEachHypervisor(func(h *lochness.Hypervisor) error {
		if ip == h.IP.String() {
			found = h
		}
		return nil
	})

	if found == nil {
		http.NotFound(w, r)
		return
	}

	version := found.Config["version"]
	var err error
	if version == "" {
		version, err = s.ctx.GetConfig("defaultVersion")
		if err != nil && !lochness.IsKeyNotFound(err) {
			// XXX: should be fatal?
			log.Println(err)
		}
		if version == "" {
			version = s.defaultVersion
		}
	}

	options := map[string]string{
		"uuid": found.ID,
	}

	data := map[string]string{
		"BaseUrl": s.baseUrl,
		"Options": mapToOptions(options),
		"Version": version,
	}
	err = s.t.Execute(w, data)
	if err != nil {
		// we shouldn not get here
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func mapToOptions(m map[string]string) string {
	var parts []string

	for k, v := range m {
		// need to sanitize ?
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

package main

import (
	"encoding/json"
	_ "expvar"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"
	"text/template"

	"github.com/armon/go-metrics"
	"github.com/bakins/go-metrics-map"
	"github.com/bakins/go-metrics-middleware"
	"github.com/bakins/net-http-recover"
	"github.com/coreos/go-etcd/etcd"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/mistifyio/lochness"
	"github.com/spf13/cobra"
)

var (
	port           uint = 8888
	eaddr               = "http://127.0.0.1:4001"
	baseUrl             = "http://ipxe.mistify.local:8888"
	defaultVersion      = "0.1.0"
	imageDir            = "/var/lib/images"
	addOpts             = ""
	statsd              = ""
)

type Server struct {
	ctx            *lochness.Context
	t              *template.Template
	defaultVersion string
	baseUrl        string
	addOpts        string
}

const ipxeTemplate = `#!ipxe
kernel {{.BaseUrl}}/images/{{.Version}}/vmlinuz {{.Options}}
initrd {{.BaseUrl}}/images/{{.Version}}/initrd
boot
`

func run(cmd *cobra.Command, args []string) {
	e := etcd.NewClient([]string{eaddr})
	c := lochness.NewContext(e)

	router := mux.NewRouter()
	router.StrictSlash(true)

	s := &Server{
		ctx:            c,
		t:              template.Must(template.New("ipxe").Parse(ipxeTemplate)),
		defaultVersion: defaultVersion,
		baseUrl:        baseUrl,
		addOpts:        addOpts,
	}

	chain := alice.New(
		func(h http.Handler) http.Handler {
			return recovery.Handler(os.Stderr, h, true)
		},
		func(h http.Handler) http.Handler {
			return handlers.CombinedLoggingHandler(os.Stdout, h)
		},
		handlers.CompressHandler,
	)

	sink := mapsink.New()
	fanout := metrics.FanoutSink{sink}

	if statsd != "" {
		ss, _ := metrics.NewStatsdSink(statsd)
		fanout = append(fanout, ss)
	}

	conf := metrics.DefaultConfig("enfield")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, fanout)
	mw := mmw.New(m)

	router.PathPrefix("/debug/").Handler(chain.Append(mw.HandlerWrapper("debug")).Then((http.DefaultServeMux)))
	router.PathPrefix("/images").Handler(chain.Append(mw.HandlerWrapper("images")).Then(http.StripPrefix("/images/", http.FileServer(http.Dir(imageDir)))))

	router.Handle("/ipxe/{ip}", chain.Append(
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				context.Set(r, "_server_", s)
				h.ServeHTTP(w, r)
			})
		},
	).Append(mw.HandlerWrapper("ipxe")).ThenFunc(ipxeHandler))

	router.Handle("/metrics", chain.Append(mw.HandlerWrapper("metrics")).ThenFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(sink)
		}))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}

func ipxeHandler(w http.ResponseWriter, r *http.Request) {

	s := context.Get(r, "_server_").(*Server)

	ip := mux.Vars(r)["ip"]

	if net.ParseIP(ip) == nil {
		http.Error(w, "invalid address", http.StatusBadRequest)
		return
	}

	found, err := s.ctx.FirstHypervisor(func(h *lochness.Hypervisor) bool {
		return ip == h.IP.String()
	})

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if found == nil {
		http.NotFound(w, r)
		return
	}

	version := found.Config["version"]

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
		"Options": mapToOptions(options) + " " + s.addOpts,
		"Version": version,
	}
	err = s.t.Execute(w, data)
	if err != nil {
		// we shouldn not get here
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	root := &cobra.Command{
		Use:  "enfield",
		Long: "enfield is a simple web service for service ipxe configurations for LochNess",
		Run:  run,
	}
	root.Flags().UintVarP(&port, "port", "p", port, "address to listen")
	root.Flags().StringVarP(&eaddr, "etcd", "e", eaddr, "address of etcd machine")
	root.Flags().StringVarP(&baseUrl, "base", "b", baseUrl, "base address of bits request")
	root.Flags().StringVarP(&defaultVersion, "version", "v", defaultVersion, "If all else fails, what version to serve")
	root.Flags().StringVarP(&imageDir, "images", "i", imageDir, "directory containing the images")
	root.Flags().StringVarP(&addOpts, "options", "o", addOpts, "additional options to add to boot kernel")
	root.Flags().StringVarP(&statsd, "statsd", "s", statsd, "statsd address")

	root.Execute()
}

func mapToOptions(m map[string]string) string {
	var parts []string

	for k, v := range m {
		// need to sanitize ?
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, " ")
}

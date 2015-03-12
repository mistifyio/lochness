package main

import (
	"encoding/json"
	_ "expvar"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"regexp"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
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
	flag "github.com/ogier/pflag"
)

type server struct {
	ctx            *lochness.Context
	t              *template.Template
	c              *template.Template
	defaultVersion string
	baseURL        string
	addOpts        string
}

var envRegex = regexp.MustCompile("^[_A-Z][_A-Z0-9]*$")

const ipxeTemplate = `#!ipxe
kernel {{.BaseURL}}/images/{{.Version}}/vmlinuz {{.Options}}
initrd {{.BaseURL}}/images/{{.Version}}/initrd
boot
`
const configTemplate = `{{range $key, $value := .}}{{ printf "%s=%s\n" $key $value}}{{end}}`

func main() {
	port := flag.UintP("port", "p", 8888, "address to listen")
	eaddr := flag.StringP("etcd", "e", "http://127.0.0.1:4001", "address of etcd machine")
	baseURL := flag.StringP("base", "b", "http://ipxe.mistify.local:8888", "base address of bits request")
	defaultVersion := flag.StringP("version", "v", "0.1.0", "If all else fails, what version to serve")
	imageDir := flag.StringP("images", "i", "/var/lib/images", "directory containing the images")
	addOpts := flag.StringP("options", "o", "", "additional options to add to boot kernel")
	statsd := flag.StringP("statsd", "s", "", "statsd address")

	flag.Parse()

	e := etcd.NewClient([]string{*eaddr})
	c := lochness.NewContext(e)

	router := mux.NewRouter()
	router.StrictSlash(true)

	s := &server{
		ctx:            c,
		t:              template.Must(template.New("ipxe").Parse(ipxeTemplate)),
		c:              template.Must(template.New("config").Parse(configTemplate)),
		defaultVersion: *defaultVersion,
		baseURL:        *baseURL,
		addOpts:        *addOpts,
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

	if *statsd != "" {
		ss, _ := metrics.NewStatsdSink(*statsd)
		fanout = append(fanout, ss)
	}

	conf := metrics.DefaultConfig("enfield")
	conf.EnableHostname = false
	m, _ := metrics.New(conf, fanout)
	mw := mmw.New(m)

	router.PathPrefix("/debug/").Handler(chain.Append(mw.HandlerWrapper("debug")).Then((http.DefaultServeMux)))
	router.PathPrefix("/images").Handler(chain.Append(mw.HandlerWrapper("images")).Then(http.StripPrefix("/images/", http.FileServer(http.Dir(*imageDir)))))

	router.Handle("/ipxe/{ip}", chain.Append(
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				context.Set(r, "_server_", s)
				h.ServeHTTP(w, r)
			})
		},
	).Append(mw.HandlerWrapper("ipxe")).ThenFunc(ipxeHandler))

	router.Handle("/configs/{ip}", chain.Append(
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				context.Set(r, "_server_", s)
				h.ServeHTTP(w, r)
			})
		},
	).Append(mw.HandlerWrapper("config")).ThenFunc(configHandler))

	router.Handle("/metrics", chain.Append(mw.HandlerWrapper("metrics")).ThenFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			if err := json.NewEncoder(w).Encode(sink); err != nil {
				log.WithField("error", err).Error(err)
			}
		}))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), router); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "http.ListenAndServe",
		}).Fatal("ListenAndServe returned an error")
	}
}

func ipxeHandler(w http.ResponseWriter, r *http.Request) {

	s := context.Get(r, "_server_").(*server)

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
			log.WithFields(log.Fields{
				"error": err,
				"func":  "lochness.GetConfig",
			}).Error("failed to get a version")
		}
		if version == "" {
			version = s.defaultVersion
		}
	}

	options := map[string]string{
		"uuid": found.ID,
	}

	data := map[string]string{
		"BaseURL": s.baseURL,
		"Options": mapToOptions(options) + " " + s.addOpts,
		"Version": version,
	}
	err = s.t.Execute(w, data)
	if err != nil {
		// we should not get here
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func configHandler(w http.ResponseWriter, r *http.Request) {

	s := context.Get(r, "_server_").(*server)

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

	configs := map[string]string{}
	s.ctx.ForEachConfig(func(key, val string) error {
		if envRegex.MatchString(key) {
			configs[key] = val
		}
		return nil
	})

	for key, val := range found.Config {
		if envRegex.MatchString(key) {
			configs[key] = val
		}
	}

	err = s.c.Execute(w, configs)
	if err != nil {
		// we should not get here
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/bakins/logrus-middleware"
	"github.com/bakins/net-http-recover"
	"github.com/gorilla/context"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/jobqueue"
	"github.com/tylerb/graceful"
)

const (
	ctxKey string = "lochnessContext"
	jQKey  string = "lochnessJobQueue"
)

type (
	// HTTPResponse is a wrapper for http.ResponseWriter which provides access
	// to several convenience methods
	HTTPResponse struct {
		http.ResponseWriter
	}

	// HTTPError contains information for http error responses
	HTTPError struct {
		Message string   `json:"message"`
		Code    int      `json:"code"`
		Stack   []string `json:"stack"`
	}
)

// Run starts the server
func Run(port uint, ctx *lochness.Context, jobQueue *jobqueue.Client, m *metricsContext) *graceful.Server {
	router := mux.NewRouter()
	router.StrictSlash(true)

	// Common middleware applied to every request
	logrusMiddleware := logrusmiddleware.Middleware{
		Name: "cguestd",
	}
	commonMiddleware := alice.New(
		func(h http.Handler) http.Handler {
			return logrusMiddleware.Handler(h, "")
		},
		handlers.CompressHandler,
		func(h http.Handler) http.Handler {
			return recovery.Handler(os.Stderr, h, true)
		},
		func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				context.Set(r, ctxKey, ctx)
				context.Set(r, jQKey, jobQueue)
				h.ServeHTTP(w, r)
			})
		},
	)

	// NOTE: Due to weirdness with PrefixPath and StrictSlash, can't just pass
	// a prefixed subrouter to the register functions and have the base path
	// work cleanly. The register functions need to add a base path handler to
	// the main router before setting subhandlers on either main or subrouter

	RegisterGuestRoutes("/guests", router, m)
	RegisterJobRoutes("/jobs", router, m)

	router.HandleFunc("/metrics",
		func(w http.ResponseWriter, r *http.Request) {
			hr := HTTPResponse{w}
			hr.JSON(http.StatusOK, m.sink)
		})

	server := &graceful.Server{
		Timeout: 5 * time.Second,
		Server: &http.Server{
			Addr:           fmt.Sprintf(":%d", port),
			Handler:        commonMiddleware.Then(router),
			MaxHeaderBytes: 1 << 20,
		},
	}
	go listenAndServe(server)
	return server
}

func listenAndServe(server *graceful.Server) {
	if err := server.ListenAndServe(); err != nil {
		// Ignore the error from closing the listener, which is involved in the
		// graceful shutdown
		if !strings.Contains(err.Error(), "use of closed network connection") {
			log.WithField("error", err).Fatal("server error")
		}
	}
}

// JSON writes appropriate headers and JSON body to the http response
func (hr *HTTPResponse) JSON(code int, obj interface{}) {
	hr.Header().Set("Content-Type", "application/json")
	hr.WriteHeader(code)
	encoder := json.NewEncoder(hr)
	if err := encoder.Encode(obj); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
	}
}

// JSONError prepares an HTTPError with a stack trace and writes it with
// HTTPResponse.JSON
func (hr *HTTPResponse) JSONError(code int, err error) {
	httpError := &HTTPError{
		Message: err.Error(),
		Code:    code,
		Stack:   make([]string, 0, 4),
	}
	for i := 1; ; i++ { //
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		httpError.Stack = append(httpError.Stack, fmt.Sprintf("%s:%d (0x%x)", file, line, pc))
	}
	hr.JSON(code, httpError)
}

// JSONMsg is a convenience method to write a JSON response with just a message
// string
func (hr *HTTPResponse) JSONMsg(code int, msg string) {
	msgObj := map[string]string{
		"message": msg,
	}
	hr.JSON(code, msgObj)
}

// SetContext sets a lochness.Context value for a request
func SetContext(r *http.Request, ctx *lochness.Context) {
	context.Set(r, ctxKey, ctx)
}

// GetContext retrieves a lochness.Context value for a request
func GetContext(r *http.Request) *lochness.Context {
	if value := context.Get(r, ctxKey); value != nil {
		return value.(*lochness.Context)
	}
	return nil
}

// GetJobQueue retrieves a lochness.Context value for a request
func GetJobQueue(r *http.Request) *jobqueue.Client {
	if value := context.Get(r, jQKey); value != nil {
		return value.(*jobqueue.Client)
	}
	return nil
}

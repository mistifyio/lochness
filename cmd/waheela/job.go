package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// RegisterJobRoutes registers the guest routes and handlers
func RegisterJobRoutes(prefix string, router *mux.Router, m *metricsContext) {
	// TODO: Figure out a cleaner way to do middleware on the subrouter
	sub := router.PathPrefix(prefix).Subrouter()

	sub.Handle("/{jobID}", m.mmw.HandlerFunc(GetJob, "job")).Methods("GET")
}

// GetJob gets a job status
func GetJob(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vars := mux.Vars(r)
	ctx := GetContext(r)
	job, err := ctx.Job(vars["jobID"])
	if err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.JSON(http.StatusOK, job)
}

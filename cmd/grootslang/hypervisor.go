package main

import (
	"encoding/json"
	"net/http"

	"code.google.com/p/go-uuid/uuid"

	"github.com/gorilla/mux"
	"github.com/mistifyio/lochness"
)

// RegisterHypervisorRoutes registers the hypervisor routes and handlers
func RegisterHypervisorRoutes(prefix string, router *mux.Router) {
	router.HandleFunc(prefix, ListHypervisors).Methods("GET")
	router.HandleFunc(prefix, CreateHypervisor).Methods("POST")
	sub := router.PathPrefix(prefix).Subrouter()
	sub.HandleFunc("/{hypervisorID}", GetHypervisor).Methods("GET")
	sub.HandleFunc("/{hypervisorID}", UpdateHypervisor).Methods("PATCH")
	sub.HandleFunc("/{hypervisorID}", DestroyHypervisor).Methods("DELETE")
}

// ListHypervisors gets a list of all hypervisors
func ListHypervisors(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	ctx := GetContext(r)
	hypervisors := make(lochness.Hypervisors, 0)
	err := ctx.ForEachHypervisor(func(h *lochness.Hypervisor) error {
		hypervisors = append(hypervisors, h)
		return nil
	})
	if err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.JSON(http.StatusOK, hypervisors)
}

// GetHypervisor gets a particular hypervisor
func GetHypervisor(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}
	hr.JSON(http.StatusOK, hypervisor)
}

// CreateHypervisor creates a new hypervisor
func CreateHypervisor(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}

	hypervisor, err := decodeHypervisor(r, nil)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	if !saveHypervisorHelper(hr, hypervisor) {
		return
	}
	hr.JSON(http.StatusCreated, hypervisor)
}

// UpdateHypervisor updates an existing hypervisor
func UpdateHypervisor(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return // Specific response handled by getHypervisorHelper
	}

	// Parse Request
	_, err := decodeHypervisor(r, hypervisor)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	if !saveHypervisorHelper(hr, hypervisor) {
		return
	}
	hr.JSON(http.StatusOK, hypervisor)
}

// DestroyHypervisor deletes an existing hypervisor
func DestroyHypervisor(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}

	if err := hypervisor.Destroy(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.JSON(http.StatusOK, hypervisor)
}

// getHypervisorHelper gets the hypervisor object and handles sending a response
// in case of error
func getHypervisorHelper(hr HTTPResponse, r *http.Request) (*lochness.Hypervisor, bool) {
	ctx := GetContext(r)
	vars := mux.Vars(r)
	hypervisorID, ok := vars["hypervisorID"]
	if !ok {
		hr.JSONMsg(http.StatusBadRequest, "missing hypervisor id")
		return nil, false
	}
	if uuid.Parse(hypervisorID) == nil {
		hr.JSONMsg(http.StatusBadRequest, "invalid hypervisor id")
		return nil, false
	}
	hypervisor, err := ctx.Hypervisor(hypervisorID)
	if err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return nil, false
	}
	return hypervisor, true
}

// saveHypervisorHelper saves the hypervisor object and handles sending a
// response in case of error
func saveHypervisorHelper(hr HTTPResponse, hypervisor *lochness.Hypervisor) bool {
	if err := hypervisor.Validate(); err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return false
	}
	// Save
	if err := hypervisor.Save(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return false
	}
	return true
}

func decodeHypervisor(r *http.Request, hypervisor *lochness.Hypervisor) (*lochness.Hypervisor, error) {
	if hypervisor == nil {
		ctx := GetContext(r)
		hypervisor = ctx.NewHypervisor()
	}

	if err := json.NewDecoder(r.Body).Decode(hypervisor); err != nil {
		return nil, err
	}
	return hypervisor, nil
}

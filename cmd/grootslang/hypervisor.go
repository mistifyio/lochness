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
	sub.HandleFunc("/{hypervisorID}/config", GetHypervisorConfig).Methods("GET")
	sub.HandleFunc("/{hypervisorID}/config", UpdateHypervisorConfig).Methods("PATCH")
	sub.HandleFunc("/{hypervisorID}/subnets", ListHypervisorSubnets).Methods("GET")
	sub.HandleFunc("/{hypervisorID}/subnets", AddHypervisorSubnets).Methods("PATCH")
	sub.HandleFunc("/{hypervisorID}/subnets/{subnetID}", RemoveHypervisorSubnet).Methods("DELETE")
	sub.HandleFunc("/{hypervisorID}/guests", ListHypervisorGuests).Methods("GET")
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

// GetHypervisorConfig gets the set of key/value config options
func GetHypervisorConfig(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}

	hr.JSON(http.StatusOK, hypervisor.Config)
}

// UpdateHypervisorConfig sets key/value config options
func UpdateHypervisorConfig(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}
	var newConf map[string]string
	if err := json.NewDecoder(r.Body).Decode(&newConf); err != nil {
		hr.JSONError(http.StatusBadRequest, err)
		return
	}
	for k, v := range newConf {
		if err := hypervisor.SetConfig(k, v); err != nil {
			hr.JSONError(http.StatusInternalServerError, err)
			return
		}
	}
	hr.JSON(http.StatusOK, hypervisor.Config)
}

// ListHypervisorSubnets lists the subnets associated with a hypervisor
func ListHypervisorSubnets(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}

	hr.JSON(http.StatusOK, hypervisor.Subnets())
}

// AddHypervisorSubnets associates subnets with a hypervisor
func AddHypervisorSubnets(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	ctx := GetContext(r)
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}

	var subnets map[string]string
	if err := json.NewDecoder(r.Body).Decode(&subnets); err != nil {
		hr.JSONError(http.StatusBadRequest, err)
		return
	}

	for subnetID, bridge := range subnets {
		subnet, err := ctx.Subnet(subnetID)
		if err != nil {
			hr.JSONError(http.StatusNotFound, err)
			return
		}
		if err := hypervisor.AddSubnet(subnet, bridge); err != nil {
			hr.JSONError(http.StatusInternalServerError, err)
			return
		}
	}

	hr.JSON(http.StatusOK, hypervisor.Subnets())
}

// RemoveHypervisorSubnet removes a subnet from a Hypervisor
func RemoveHypervisorSubnet(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	ctx := GetContext(r)
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}

	vars := mux.Vars(r)
	subnet, err := ctx.Subnet(vars["subnetID"])
	if err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}

	if err := hypervisor.RemoveSubnet(subnet); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}

	hr.JSON(http.StatusOK, hypervisor.Subnets())
}

// ListHypervisorGuests returns a list of guests of the Hypervisor
func ListHypervisorGuests(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hypervisor, ok := getHypervisorHelper(hr, r)
	if !ok {
		return
	}

	hr.JSON(http.StatusOK, hypervisor.Guests())
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

// decodeHypervisor decodes request body JSON into a hypervisor object
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

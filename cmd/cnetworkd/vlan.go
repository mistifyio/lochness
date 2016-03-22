package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mistifyio/lochness"
)

// RegisterVLANRoutes registers the VLAN routes and handlers
func RegisterVLANRoutes(prefix string, router *mux.Router) {
	router.HandleFunc(prefix, ListVLANs).Methods("GET")
	router.HandleFunc(prefix, CreateVLAN).Methods("POST")

	sub := router.PathPrefix(prefix).Subrouter()
	sub.HandleFunc("/{vlanTag}", GetVLAN).Methods("GET")
	sub.HandleFunc("/{vlanTag}", UpdateVLAN).Methods("PATCH")
	sub.HandleFunc("/{vlanTag}", DestroyVLAN).Methods("DELETE")
	sub.HandleFunc("/{vlanTag}/groups", GetVLANGroupMembership).Methods("GET")
	sub.HandleFunc("/{vlanTag}/groups", UpdateVLANGroupMembership).Methods("POST")
}

// ListVLANs gets a list of all VLANs
func ListVLANs(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	ctx := GetContext(r)
	vlans := make(lochness.VLANs, 0)
	err := ctx.ForEachVLAN(func(vlan *lochness.VLAN) error {
		vlans = append(vlans, vlan)
		return nil
	})
	if err != nil && !ctx.IsKeyNotFound(err) {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.JSON(http.StatusOK, vlans)
}

// GetVLAN gets a particular VLAN
func GetVLAN(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlan, ok := getVLANHelper(hr, r)
	if !ok {
		return
	}
	hr.JSON(http.StatusOK, vlan)
}

// CreateVLAN creates a new VLAN
func CreateVLAN(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlan, err := decodeVLAN(r, nil)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	if !saveVLANHelper(hr, vlan) {
		return
	}
	hr.JSON(http.StatusCreated, vlan)
}

// UpdateVLAN updates a VLAN
func UpdateVLAN(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlan, ok := getVLANHelper(hr, r)
	if !ok {
		return
	}

	vlanTag := vlan.Tag

	_, err := decodeVLAN(r, vlan)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	// Don't allow tag redefinition
	vlan.Tag = vlanTag

	if !saveVLANHelper(hr, vlan) {
		return
	}

	hr.JSON(http.StatusOK, vlan)
}

// DestroyVLAN destroys a VLAN
func DestroyVLAN(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlan, ok := getVLANHelper(hr, r)
	if !ok {
		return
	}

	if err := vlan.Destroy(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
	}

	hr.JSON(http.StatusOK, vlan)
}

// GetVLANGroupMembership gets a VLAN's group membership
func GetVLANGroupMembership(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlan, ok := getVLANHelper(hr, r)
	if !ok {
		return
	}
	hr.JSON(http.StatusOK, vlan.VLANGroups())
}

// UpdateVLANGroupMembership updates a VLAN's group membership
func UpdateVLANGroupMembership(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlan, ok := getVLANHelper(hr, r)
	if !ok {
		return
	}
	var newGroups []string
	if err := json.NewDecoder(r.Body).Decode(&newGroups); err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	// Determine group modifications necessary
	// -1 : Removed
	//  0 : No change
	//  1 : Added
	ctx := GetContext(r)
	groupModifications := make(map[string]int)
	for _, oldGroup := range vlan.VLANGroups() {
		groupModifications[oldGroup] = -1
	}
	for _, newGroup := range newGroups {
		groupModifications[newGroup]++
	}
	// Make group modifications
	for groupID, action := range groupModifications {
		vlanGroup, err := ctx.VLANGroup(groupID)
		if err != nil {
			if ctx.IsKeyNotFound(err) {
				hr.JSONMsg(http.StatusBadRequest, "group not found")
			} else {
				hr.JSONMsg(http.StatusInternalServerError, err.Error())
			}
			return
		}
		switch action {
		case -1:
			if err := vlanGroup.RemoveVLAN(vlan); err != nil {
				hr.JSONError(http.StatusInternalServerError, err)
				return
			}
		case 0:
			break
		case 1:
			if err := vlanGroup.AddVLAN(vlan); err != nil {
				hr.JSONError(http.StatusInternalServerError, err)
				return
			}
		}
	}

	hr.JSON(http.StatusOK, vlan.VLANGroups())
}

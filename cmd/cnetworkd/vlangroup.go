package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mistifyio/lochness"
)

// RegisterVLANGroupRoutes registers the VLAN routes and handlers
func RegisterVLANGroupRoutes(prefix string, router *mux.Router) {
	router.HandleFunc(prefix, ListVLANGroups).Methods("GET")
	router.HandleFunc(prefix, CreateVLANGroup).Methods("POST")

	sub := router.PathPrefix(prefix).Subrouter()
	sub.HandleFunc("/{vlanGroupID}", GetVLANGroup).Methods("GET")
	sub.HandleFunc("/{vlanGroupID}", UpdateVLANGroup).Methods("PATCH")
	sub.HandleFunc("/{vlanGroupID}", DestroyVLANGroup).Methods("DELETE")
	sub.HandleFunc("/{vlanGroupID}/tags", GetVLANGroupVLANs).Methods("GET")
	sub.HandleFunc("/{vlanGroupID}/tags", UpdateVLANGroupVLANs).Methods("POST")
}

// ListVLANGroups gets a list of all VLANGroups
func ListVLANGroups(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	ctx := GetContext(r)
	vlanGroups := make(lochness.VLANGroups, 0)
	err := ctx.ForEachVLANGroup(func(vlanGroup *lochness.VLANGroup) error {
		vlanGroups = append(vlanGroups, vlanGroup)
		return nil
	})
	if err != nil && !ctx.IsKeyNotFound(err) {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.JSON(http.StatusOK, vlanGroups)
}

// GetVLANGroup gets a particular VLAN
func GetVLANGroup(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlanGroup, ok := getVLANGroupHelper(hr, r)
	if !ok {
		return
	}
	hr.JSON(http.StatusOK, vlanGroup)
}

// CreateVLANGroup creates a new VLAN
func CreateVLANGroup(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlanGroup, err := decodeVLANGroup(r, nil)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	if !saveVLANGroupHelper(hr, vlanGroup) {
		return
	}
	hr.JSON(http.StatusCreated, vlanGroup)
}

// UpdateVLANGroup updates a VLANGroup
func UpdateVLANGroup(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlanGroup, ok := getVLANGroupHelper(hr, r)
	if !ok {
		return
	}

	groupID := vlanGroup.ID

	_, err := decodeVLANGroup(r, vlanGroup)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	// Don't allow ID redefinition
	vlanGroup.ID = groupID

	if !saveVLANGroupHelper(hr, vlanGroup) {
		return
	}

	hr.JSON(http.StatusOK, vlanGroup)
}

// DestroyVLANGroup destroys a VLANGroup
func DestroyVLANGroup(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlanGroup, ok := getVLANGroupHelper(hr, r)
	if !ok {
		return
	}

	if err := vlanGroup.Destroy(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
	}

	hr.JSON(http.StatusOK, vlanGroup)
}

// GetVLANGroupVLANs gets a VLANGroup's VLANs
func GetVLANGroupVLANs(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlanGroup, ok := getVLANGroupHelper(hr, r)
	if !ok {
		return
	}
	hr.JSON(http.StatusOK, vlanGroup.VLANs())
}

// UpdateVLANGroupVLANs updates a VLANGroups's VLANs
func UpdateVLANGroupVLANs(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	vlanGroup, ok := getVLANGroupHelper(hr, r)
	if !ok {
		return
	}
	var newTags []int
	if err := json.NewDecoder(r.Body).Decode(&newTags); err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	// Determine group modifications necessary
	// -1 : Removed
	//  0 : No change
	//  1 : Added
	ctx := GetContext(r)
	groupModifications := make(map[int]int)
	for _, oldTag := range vlanGroup.VLANs() {
		groupModifications[oldTag] = -1
	}
	for _, newTag := range newTags {
		groupModifications[newTag]++
	}

	// Make group modifications
	for tag, action := range groupModifications {
		vlan, err := ctx.VLAN(tag)
		if err != nil {
			if ctx.IsKeyNotFound(err) {
				hr.JSONMsg(http.StatusBadRequest, "vlan not found")
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

	hr.JSON(http.StatusOK, vlanGroup.VLANs())
}

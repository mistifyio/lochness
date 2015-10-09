package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/pborman/uuid"

	"github.com/gorilla/mux"
	"github.com/mistifyio/lochness"
)

func getVLANHelper(hr HTTPResponse, r *http.Request) (*lochness.VLAN, bool) {
	ctx := GetContext(r)
	vars := mux.Vars(r)
	vt, ok := vars["vlanTag"]
	if !ok {
		hr.JSONMsg(http.StatusBadRequest, "missing vlan tag")
		return nil, false
	}
	vlanTag, err := strconv.Atoi(vt)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, "invalid vlan tag")
		return nil, false
	}

	vlan, err := ctx.VLAN(vlanTag)
	if err != nil {
		if lochness.IsKeyNotFound(err) {
			hr.JSONMsg(http.StatusNotFound, "tag not found")
		} else {
			hr.JSONError(http.StatusInternalServerError, err)
		}
		return nil, false
	}
	return vlan, true
}

func saveVLANHelper(hr HTTPResponse, vlan *lochness.VLAN) bool {
	if err := vlan.Validate(); err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return false
	}

	if err := vlan.Save(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return false
	}
	return true
}

func decodeVLAN(r *http.Request, vlan *lochness.VLAN) (*lochness.VLAN, error) {
	if vlan == nil {
		ctx := GetContext(r)
		vlan = ctx.NewVLAN()
	}

	if err := json.NewDecoder(r.Body).Decode(vlan); err != nil {
		return nil, err
	}
	return vlan, nil
}

func getVLANGroupHelper(hr HTTPResponse, r *http.Request) (*lochness.VLANGroup, bool) {
	ctx := GetContext(r)
	vars := mux.Vars(r)
	groupID, ok := vars["vlanGroupID"]
	if !ok {
		hr.JSONMsg(http.StatusBadRequest, "missing group id")
		return nil, false
	}
	if uuid.Parse(groupID) == nil {
		hr.JSONMsg(http.StatusBadRequest, "invalid group id")
		return nil, false
	}

	vlanGroup, err := ctx.VLANGroup(groupID)
	if err != nil {
		if lochness.IsKeyNotFound(err) {
			hr.JSONMsg(http.StatusNotFound, "group not found")
		} else {
			hr.JSONError(http.StatusInternalServerError, err)
		}
		return nil, false
	}
	return vlanGroup, true
}

func saveVLANGroupHelper(hr HTTPResponse, vlanGroup *lochness.VLANGroup) bool {
	if err := vlanGroup.Validate(); err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return false
	}

	if err := vlanGroup.Save(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return false
	}
	return true
}

func decodeVLANGroup(r *http.Request, vlanGroup *lochness.VLANGroup) (*lochness.VLANGroup, error) {
	if vlanGroup == nil {
		ctx := GetContext(r)
		vlanGroup = ctx.NewVLANGroup()
	}

	if err := json.NewDecoder(r.Body).Decode(vlanGroup); err != nil {
		return nil, err
	}
	return vlanGroup, nil
}

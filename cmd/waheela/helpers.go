package main

import (
	"encoding/json"
	"net/http"

	"code.google.com/p/go-uuid/uuid"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/mistifyio/lochness"
)

const guestKey = "guest"

// loadGuest is a middleware to load a guest into the request context and
// handles sending a response in case of error
func loadGuest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hr := HTTPResponse{w}
		ctx := GetContext(r)
		vars := mux.Vars(r)
		guestID, ok := vars["guestID"]
		if !ok {
			hr.JSONMsg(http.StatusBadRequest, "missing hypervisor id")
			return
		}
		if uuid.Parse(guestID) == nil {
			hr.JSONMsg(http.StatusBadRequest, "invalid guest id")
			return
		}
		guest, err := ctx.Guest(guestID)
		if err != nil {
			hr.JSONError(http.StatusInternalServerError, err)
			return
		}
		SetRequestGuest(r, guest)
		h.ServeHTTP(w, r)
	})
}

// saveGuestHelper saves the guest object and handles sending a response in case
// of error
func saveGuestHelper(hr HTTPResponse, guest *lochness.Guest) bool {
	if err := guest.Validate(); err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return false
	}
	// Save
	if err := guest.Save(); err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return false
	}
	return true
}

// decodeGuest decodes request body JSON into a guest object
func decodeGuest(r *http.Request, guest *lochness.Guest) (*lochness.Guest, error) {
	if guest == nil {
		ctx := GetContext(r)
		guest = ctx.NewGuest()
	}

	if err := json.NewDecoder(r.Body).Decode(guest); err != nil {
		return nil, err
	}
	return guest, nil

}

// guestNewJobHelper creates a new job for a guest action and handles sending a
// response
func guestNewJobHelper(hr HTTPResponse, r *http.Request, guest *lochness.Guest, action string) {
	jobQueue := GetJobQueue(r)
	job, err := jobQueue.AddJob(guest.ID, action)
	if err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.Header().Set("X-Guest-Job-ID", job.ID)
	hr.JSON(http.StatusAccepted, guest)
}

// SetRequestGuest saves the guest to the request context
func SetRequestGuest(r *http.Request, g *lochness.Guest) {
	context.Set(r, guestKey, g)
}

// GetRequestGuest retrieves the guest from the request context
func GetRequestGuest(r *http.Request) *lochness.Guest {
	return context.Get(r, guestKey).(*lochness.Guest)
}

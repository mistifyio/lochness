package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/mistifyio/lochness"
)

// RegisterGuestRoutes registers the guest routes and handlers
func RegisterGuestRoutes(prefix string, router *mux.Router) {
	guestMiddleware := alice.New(
		loadGuest,
	)

	router.HandleFunc(prefix, ListGuests).Methods("GET")
	router.HandleFunc(prefix, CreateGuest).Methods("POST")
	// TODO: Figure out a cleaner way to do middleware on the subrouter
	sub := router.PathPrefix(prefix).Subrouter()
	sub.Handle("/{guestID}", guestMiddleware.Then(http.HandlerFunc(GetGuest))).Methods("GET")
	sub.Handle("/{guestID}", guestMiddleware.Then(http.HandlerFunc(UpdateGuest))).Methods("PATCH")
	// TODO: Add guest delete method
	//sub.Handle("/{guestID}", guestMiddleware.Then(http.HandlerFunc(DeleteGuest))).Methods("DELETE")
}

// ListGuests gets a list of all guests
func ListGuests(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	ctx := GetContext(r)
	guests := make(lochness.Guests, 0)
	err := ctx.ForEachGuest(func(g *lochness.Guest) error {
		guests = append(guests, g)
		return nil
	})
	if err != nil {
		hr.JSONError(http.StatusInternalServerError, err)
		return
	}
	hr.JSON(http.StatusOK, guests)
}

// CreateGuest creates a new guest
func CreateGuest(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}

	guest, err := decodeGuest(r, nil)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	if !saveGuestHelper(hr, guest) {
		return
	}
	hr.JSON(http.StatusCreated, guest)
}

// GetGuest gets a particular guest
func GetGuest(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	hr.JSON(http.StatusOK, GetRequestGuest(r))
}

// UpdateGuest updates an existing guest
func UpdateGuest(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	guest := GetRequestGuest(r)

	_, err := decodeGuest(r, guest)
	if err != nil {
		hr.JSONMsg(http.StatusBadRequest, err.Error())
		return
	}

	if !saveGuestHelper(hr, guest) {
		return
	}
	hr.JSON(http.StatusOK, guest)
}

/*
func DeleteGuest(w http.ResponseWriter, r *http.Request) {
	hr := HTTPResponse{w}
	guest := GetRequestGuest(r)
}
*/

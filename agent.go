package lochness

import "github.com/mistifyio/mistify-agent/client"

type (
	// Agent is an interface that allows for communication with a hypervisor
	// agent
	Agent interface {
		GetGuest(string) (*client.Guest, error)
		CreateGuest(string) (*client.Guest, error)
		DeleteGuest(string) (*client.Guest, error)
		GuestAction(string, string) (*client.Guest, error)
	}
)

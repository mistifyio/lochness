package lochness

import "github.com/mistifyio/mistify-agent/client"

type (
	// Agent is an interface that allows for communication with a hypervisor
	// agent
	Agent interface {
		GetGuest(string) (*client.Guest, error)
		CreateGuest(string) (string, error)
		DeleteGuest(string) (string, error)
		GuestAction(string, string) (string, error)
		CheckJobStatus(string, string) (bool, error)
	}
)

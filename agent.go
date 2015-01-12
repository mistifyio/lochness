package lochness

import (
	"errors"
	"math/rand"
	"time"

	"github.com/mistifyio/mistify-agent/client"
)

type (
	// Agenter is an interface that allows for communication with a hypervisor
	// agent
	Agenter interface {
		GetGuest(string) (*client.Guest, error)
		CreateGuest(*client.Guest) (*client.Guest, error)
		DeleteGuest(string) (*client.Guest, error)
		GuestAction(string, string) (*client.Guest, error)
	}

	// AgentStubs is an Agenter with stubbed methods for testing
	AgentStubs struct {
		context     *Context
		rand        *rand.Rand
		failPercent int
	}
)

// NewAgentStubs creates a new AgentStubs instance within the context and
// initialies the random number generator for failures
func (context *Context) NewAgentStubs(failPercent int) *AgentStubs {
	return &AgentStubs{
		context:     context,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		failPercent: failPercent,
	}
}

// randomError simulates failure for a given percent of the time
func (agent *AgentStubs) randomError() error {
	if agent.rand.Intn(100) < agent.failPercent {
		return errors.New("Random Error")
	}
	return nil
}

// guestFromID creates a *client.Guest from a guestID using the datastore
func (agent *AgentStubs) guestFromID(guestID string) (*client.Guest, error) {
	g, err := agent.context.Guest(guestID)
	if err != nil {
		return nil, err
	}
	flavor, err := agent.context.Flavor(g.FlavorID)
	if err != nil {
		return nil, err
	}
	subnet, err := agent.context.Subnet(g.SubnetID)
	if err != nil {
		return nil, err
	}

	nic := client.Nic{
		Name:    "foobar",
		Network: g.NetworkID,
		Model:   "foobar",
		Mac:     g.MAC.String(),
		Address: g.IP.String(),
		Netmask: subnet.CIDR.Mask.String(),
		Gateway: subnet.Gateway.String(),
		Device:  "eth0",
	}

	disk := client.Disk{
		Bus:    "sata",
		Device: "sda1",
		Size:   flavor.Disk,
		Volume: "foo",
		Image:  "",
		Source: "/dev/zvol/foo",
	}

	return &client.Guest{
		Id:       g.ID,
		Type:     g.Type,
		Nics:     []client.Nic{nic},
		Disks:    []client.Disk{disk},
		State:    "running",
		Memory:   uint(flavor.Memory),
		Cpu:      uint(flavor.CPU),
		VNC:      1337,
		Metadata: g.Metadata,
	}, nil
}

// GetGuest is a stub for retrieving a guest via request to the agent.
func (agent *AgentStubs) GetGuest(guestID string) (*client.Guest, error) {
	if err := agent.randomError(); err != nil {
		return nil, err
	}
	guest, err := agent.guestFromID(guestID)
	return guest, err
}

// CreateGuest is a stub for creating a guest via request to the agent.
func (agent *AgentStubs) CreateGuest(guest *client.Guest) (*client.Guest, error) {
	if err := agent.randomError(); err != nil {
		return nil, err
	}
	return guest, nil
}

// DeleteGuest is a stub for deleting a guest via request to the agent.
func (agent *AgentStubs) DeleteGuest(guestID string) (*client.Guest, error) {
	if err := agent.randomError(); err != nil {
		return nil, err
	}
	guest, err := agent.guestFromID(guestID)
	return guest, err
}

// GuestAction is a stub for issuing other basic guest actions via request to
// the agent
func (agent *AgentStubs) GuestAction(guestID, actionName string) (*client.Guest, error) {
	if err := agent.randomError(); err != nil {
		return nil, err
	}
	guest, err := agent.guestFromID(guestID)
	if err != nil {
		return nil, err
	}
	for _, action := range []string{"shutdown", "poweroff", "suspend"} {
		if actionName == action {
			guest.State = "shutdown"
			continue
		}
	}
	return guest, nil
}

package lochness

import (
	"errors"
	"math/rand"
	"time"

	"github.com/mistifyio/mistify-agent/client"
)

type (
	// StubAgent is an Agenter with stubbed methods for testing
	StubAgent struct {
		context     *Context
		rand        *rand.Rand
		failPercent int
	}
)

// NewStubAgent creates a new StubAgent instance within the context and
// initialies the random number generator for failures
func (context *Context) NewStubAgent(failPercent int) *StubAgent {
	return &StubAgent{
		context:     context,
		rand:        rand.New(rand.NewSource(time.Now().UnixNano())),
		failPercent: failPercent,
	}
}

// randomError simulates failure for a given percent of the time
func (agent *StubAgent) randomError() error {
	if agent.rand.Intn(100) < agent.failPercent {
		return errors.New("Random Error")
	}
	return nil
}

// guestFromID creates a *client.Guest from a guestID using the datastore
func (agent *StubAgent) guestFromID(guestID string) (*client.Guest, error) {
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

// CheckJobStatus looks up whether a guest job has been completed or not.
func (agent *StubAgent) CheckJobStatus(action, guestID, jobID string) (bool, error) {
	if err := agent.randomError(); err != nil {
		return true, err
	}
	return true, nil
}

// GetGuest is a stub for retrieving a guest via request to the agent.
func (agent *StubAgent) GetGuest(guestID string) (*client.Guest, error) {
	if err := agent.randomError(); err != nil {
		return nil, err
	}
	guest, err := agent.guestFromID(guestID)
	return guest, err
}

// CreateGuest is a stub for creating a guest via request to the agent.
func (agent *StubAgent) CreateGuest(guestID string) (string, error) {
	if err := agent.randomError(); err != nil {
		return "", err
	}
	return "1234abcd-1234-abcd-1234-abcd1324abcd", nil
}

// DeleteGuest is a stub for deleting a guest via request to the agent.
func (agent *StubAgent) DeleteGuest(guestID string) (string, error) {
	if err := agent.randomError(); err != nil {
		return "", err
	}
	return "1234abcd-1234-abcd-1234-abcd1324abcd", nil
}

// GuestAction is a stub for issuing other basic guest actions via request to
// the agent
func (agent *StubAgent) GuestAction(guestID, actionName string) (string, error) {
	if err := agent.randomError(); err != nil {
		return "", err
	}
	return "1234abcd-1234-abcd-1234-abcd1324abcd", nil
}

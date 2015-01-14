package lochness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"path"
	"time"

	"github.com/mistifyio/mistify-agent/client"
)

type (
	// Agenter is an interface that allows for communication with a hypervisor
	// agent
	Agenter interface {
		GetGuest(string) (*client.Guest, error)
		CreateGuest(string) (*client.Guest, error)
		DeleteGuest(string) (*client.Guest, error)
		GuestAction(string, string) (*client.Guest, error)
	}

	// AgentStubs is an Agenter with stubbed methods for testing
	AgentStubs struct {
		context     *Context
		rand        *rand.Rand
		failPercent int
	}

	// Agent is an Agenter that communicates with a hypervisor agent to perform
	// actions relating to guests
	Agent struct {
		context *Context
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
func (agent *AgentStubs) CreateGuest(guestID string) (*client.Guest, error) {
	if err := agent.randomError(); err != nil {
		return nil, err
	}
	guest, err := agent.guestFromID(guestID)
	return guest, err
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

// generateURL crafts the agent url out of known and supplied parts
func (agent *Agent) generateURL(host string, guestID string, action string) string {
	// TODO: Get port from somewhere
	port := 1337
	urlPath := path.Join("guests", guestID, action)
	return fmt.Sprintf("http://%s:%d/%s", host, port, urlPath)
}

// request makes a request to the agent for a guest and checks response
func (agent *Agent) request(url, httpMethod string, expectedCode int, dataObj interface{}) (*client.Guest, error) {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	var resp *http.Response
	var err error
	if httpMethod == "POST" {
		dataJSON, err := json.Marshal(dataObj)
		if err != nil {
			return nil, err
		}
		resp, err = httpClient.Post(url, "application/json", bytes.NewReader(dataJSON))
	} else {
		resp, err = httpClient.Get(url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedCode {
		return nil, fmt.Errorf("Unexpected HTTP Response Code: Expected %d, Received %d", expectedCode, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var guest client.Guest
	if err := json.Unmarshal(body, &guest); err != nil {
		return nil, err
	}
	return &guest, nil
}

// requestGuestAction is a convenience wrapper for basic guest actions other
// than "get" and "create".
func (agent *Agent) requestGuestAction(guestID, actionName string) (*client.Guest, error) {
	g, err := agent.context.Guest(guestID)
	if err != nil {
		return nil, err
	}
	hypervisor, err := agent.context.Hypervisor(g.HypervisorID)
	if err != nil {
		return nil, err
	}

	url := agent.generateURL(string(hypervisor.IP), guestID, actionName)
	guest, err := agent.request(url, "POST", 202, nil)
	return guest, err
}

// GetGuest retrieves information on a guest from an agent
func (agent *Agent) GetGuest(guestID string) (*client.Guest, error) {
	g, err := agent.context.Guest(guestID)
	if err != nil {
		return nil, err
	}
	hypervisor, err := agent.context.Hypervisor(g.HypervisorID)
	if err != nil {
		return nil, err
	}
	url := agent.generateURL(string(hypervisor.IP), guestID, "")
	guest, err := agent.request(url, "GET", 200, nil)
	return guest, err
}

// CreateGuest tries to create a new guest on a hypervisor selected from a list
// of viable candidates
func (agent *Agent) CreateGuest(guestID string) (*client.Guest, error) {
	g, err := agent.context.Guest(guestID)
	if err != nil {
		return nil, err
	}

	candidates, err := g.Candidates(DefaultCadidateFuctions...)
	if err != nil {
		return nil, err
	}

	for _, hypervisor := range candidates {
		url := agent.generateURL(string(hypervisor.IP), "", "")
		guest, err := agent.request(url, "POST", 202, g)
		if err == nil {
			return guest, nil
		}
	}

	return nil, err
}

// DeleteGuest deletes a guest from a hypervisor
func (agent *Agent) DeleteGuest(guestID string) (*client.Guest, error) {
	guest, err := agent.requestGuestAction(guestID, "delete")
	return guest, err
}

// GuestAction is used to run various actions on a guest under a hypervisor
// Actions: "shutdown", "reboot", "restart", "poweroff", "start", "suspend"
func (agent *Agent) GuestAction(guestID, actionName string) (*client.Guest, error) {
	guest, err := agent.requestGuestAction(guestID, actionName)
	return guest, err
}

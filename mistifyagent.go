package lochness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"time"

	magent "github.com/mistifyio/mistify-agent"
	"github.com/mistifyio/mistify-agent/client"
)

type (
	// MistifyAgent is an Agent that communicates with a hypervisor agent to perform
	// actions relating to guests
	MistifyAgent struct {
		context *Context
	}
)

// NewMistifyAgent creates a new MistifyAgent instance within the context
func (context *Context) NewMistifyAgent() *MistifyAgent {
	return &MistifyAgent{
		context: context,
	}
}

// generateClientGuest creates a client.Guest object based on the stored guest
// properties. Used during guest creation
func (agent *MistifyAgent) generateClientGuest(g *Guest) (*client.Guest, error) {
	flavor, err := agent.context.Flavor(g.FlavorID)

	if err != nil {
		return nil, err
	}

	subnet, err := agent.context.Subnet(g.SubnetID)
	if err != nil {
		return nil, err
	}

	nic := client.Nic{
		Network: g.Bridge,
		Model:   "virtio", // TODO: Check whether this is alwalys the case
		Address: g.IP.String(),
		Netmask: subnet.CIDR.Mask.String(),
		Gateway: subnet.Gateway.String(),
	}

	disk := client.Disk{
		Size: flavor.Disk,
	}

	return &client.Guest{
		Id:       g.ID,
		Type:     g.Type,
		Nics:     []client.Nic{nic},
		Disks:    []client.Disk{disk},
		Memory:   uint(flavor.Memory),
		Cpu:      uint(flavor.CPU),
		Metadata: g.Metadata,
	}, nil
}

// guestActionURL crafts the guest action url
func (agent *MistifyAgent) guestActionURL(host, guestID, action string) string {
	port := 1337 // TODO: Get port from somewhere. Config?

	// Create and Get don't have the action name in the URL, so blank it out
	// Create doesn't specify a guest id in the URL
	if action == "create" || action == "get" {
		if action == "create" {
			guestID = ""
		}
		action = ""
	}
	// Join with appropriate seperators whether action is blank or not
	urlPath := path.Join("guests", guestID, action)

	return fmt.Sprintf("http://%s:%d/%s", host, port, urlPath)
}

// guestJobURL crafts the job status url
func (agent *MistifyAgent) guestJobURL(host, guestID, jobID string) string {
	port := 1337 // TODO: Get port from somewhere
	return fmt.Sprintf("http://%s:%d/guests/%s/jobs/%s", host, port, guestID, jobID)
}

// request is the generic way to hit an agent endpoint with minimal response
// checking. It returns the body string for later parsing and an optional jobID.
// Generally don't use directly; other, more convenient methods will wrap this
func (agent *MistifyAgent) request(url, httpMethod string, expectedCode int, dataObj interface{}) ([]byte, string, error) {
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	// Make the request. POST sends JSON data, GET doesn't
	var resp *http.Response
	var reqErr error
	if httpMethod == "POST" {
		dataJSON, err := json.Marshal(dataObj)
		if err != nil {
			return nil, "", err
		}
		resp, reqErr = httpClient.Post(url, "application/json", bytes.NewReader(dataJSON))
	} else {
		resp, reqErr = httpClient.Get(url)
	}
	if reqErr != nil {
		return nil, "", reqErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != expectedCode {
		return nil, "", fmt.Errorf("Unexpected HTTP Response Code: Expected %d, Received %d", expectedCode, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	return body, resp.Header.Get("X-Guest-Job-ID"), err
}

// getHypervisor loads the hypervisor based on guest id
func (agent *MistifyAgent) getHypervisor(guestID string) (*Hypervisor, error) {
	guest, err := agent.context.Guest(guestID)
	if err != nil {
		return nil, err
	}
	hypervisor, err := agent.context.Hypervisor(guest.HypervisorID)
	if err != nil {
		return nil, err
	}
	return hypervisor, nil
}

// requestGuestAction makes requests for a guest to a hypervisor agent.
func (agent *MistifyAgent) requestGuestAction(guestID, actionName string) (string, error) {
	hypervisor, err := agent.getHypervisor(guestID)
	if err != nil {
		return "", err
	}
	url := agent.guestActionURL(hypervisor.IP.String(), guestID, actionName)

	// Make the request
	_, jobID, err := agent.request(url, "POST", http.StatusAccepted, nil)
	if err != nil {
		return "", err
	}

	return jobID, nil
}

// checkJobStatus looks up whether a guest job has been completed or not.
func (agent *MistifyAgent) CheckJobStatus(guestID, jobID string) (bool, error) {
	hypervisor, err := agent.getHypervisor(guestID)
	if err != nil {
		return false, err
	}
	url := agent.guestJobURL(hypervisor.IP.String(), guestID, jobID)
	body, _, err := agent.request(url, "GET", http.StatusOK, nil)
	if err != nil {
		return false, err
	}

	var job magent.GuestJob
	if err := json.Unmarshal(body, &job); err != nil {
		return false, err
	}

	switch job.Status {
	case magent.Complete:
		return true, nil
	case magent.Errored:
		return true, errors.New(job.Message)
	default:
		return false, nil
	}
}

// GetGuest retrieves information on a guest from an agent
func (agent *MistifyAgent) GetGuest(guestID string) (*client.Guest, error) {
	hypervisor, err := agent.getHypervisor(guestID)
	if err != nil {
		return nil, err
	}
	url := agent.guestActionURL(hypervisor.IP.String(), guestID, "get")
	body, _, err := agent.request(url, "GET", http.StatusOK, nil)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var g client.Guest
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, err
	}
	return &g, err
}

// CreateGuest tries to create a new guest on a hypervisor selected from a list
// of viable candidates
func (agent *MistifyAgent) CreateGuest(guestID string) (string, error) {
	guest, err := agent.context.Guest(guestID)
	if err != nil {
		return "", err
	}
	hypervisor, err := agent.context.Hypervisor(guest.HypervisorID)
	if err != nil {
		return "", err
	}

	g, err := agent.generateClientGuest(guest)
	if err != nil {
		return "", err
	}

	url := agent.guestActionURL(hypervisor.IP.String(), guestID, "create")
	_, jobID, err := agent.request(url, "POST", http.StatusAccepted, g)
	return jobID, err
}

// DeleteGuest deletes a guest from a hypervisor
func (agent *MistifyAgent) DeleteGuest(guestID string) (string, error) {
	jobID, err := agent.requestGuestAction(guestID, "delete")
	return jobID, err
}

// GuestAction is used to run various actions on a guest under a hypervisor
// Actions: "shutdown", "reboot", "restart", "poweroff", "start", "suspend"
func (agent *MistifyAgent) GuestAction(guestID, actionName string) (string, error) {
	jobID, err := agent.requestGuestAction(guestID, actionName)
	return jobID, err
}

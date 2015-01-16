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

	mistifyagent "github.com/mistifyio/mistify-agent"
	"github.com/mistifyio/mistify-agent/client"
)

type (
	// Agent is an Agenter that communicates with a hypervisor agent to perform
	// actions relating to guests
	Agent struct {
		context *Context
	}
)

// NewAgent creates a new Agent instance within the context
func (context *Context) NewAgent() *Agent {
	return &Agent{
		context: context,
	}
}

// generateClientGuest creates a client.Guest object based on the stored guest
// properties. Used during guest creation
func (agent *Agent) generateClientGuest(g *Guest) (*client.Guest, error) {
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
func (agent *Agent) guestActionURL(host, guestID, action string) string {
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
func (agent *Agent) guestJobURL(host, guestID, jobID string) string {
	port := 1337 // TODO: Get port from somewhere
	return fmt.Sprintf("http://%s:%d/guests/%s/jobs/%s", host, port, guestID, jobID)
}

// request is the generic way to hit an agent endpoint with minimal response
// checking. It returns the body string for later parsing and an optional jobID.
// Generally don't use directly; other, more convenient methods will wrap this
func (agent *Agent) request(url, httpMethod string, expectedCode int, dataObj interface{}) ([]byte, string, error) {
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

// guestRequest makes requests for a guest to a hypervisor agent. It wraps
// Agent.request, generating the url, parsing the result, and handling job
// status polling
func (agent *Agent) guestRequest(hypervisor *Hypervisor, guestID string, actionName string, dataObj interface{}) (*client.Guest, error) {
	url := agent.guestActionURL(hypervisor.IP.String(), guestID, actionName)

	// Determine appropriate http method and response code
	httpCode := http.StatusAccepted
	httpMethod := "POST"
	if actionName == "get" {
		httpCode = http.StatusOK
		httpMethod = "GET"
	}

	// Make the request
	body, jobID, err := agent.request(url, httpMethod, httpCode, dataObj)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var guest client.Guest
	if err := json.Unmarshal(body, &guest); err != nil {
		return nil, err
	}

	// If there's no jobID to poll for, return now
	if jobID == "" {
		return &guest, nil
	}

	return &guest, agent.waitForGuestJob(hypervisor, guestID, jobID)
}

// requestGuestAction makes requests for a guest to a hypervisor agent. It wraps
// Agent.guestRequest, looking up the hypervisor
func (agent *Agent) requestGuestAction(guestID, actionName string) (*client.Guest, error) {
	g, err := agent.context.Guest(guestID)
	if err != nil {
		return nil, err
	}
	hypervisor, err := agent.context.Hypervisor(g.HypervisorID)
	if err != nil {
		return nil, err
	}

	guest, err := agent.guestRequest(hypervisor, guestID, actionName, nil)
	return guest, err
}

// waitForGuestJob polls the hypervisor for the job status until it errors or
// finishes
func (agent *Agent) waitForGuestJob(hypervisor *Hypervisor, guestID, jobID string) error {
	url := agent.guestJobURL(hypervisor.IP.String(), guestID, jobID)
	done := false
	var err error
	for !done {
		done, err = agent.checkJobStatus(url)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// checkJobStatus looks up whether a guest job has been completed or not.
// Since it is likely to be called multiple times, it takes a url rather than
// regenerating it every time from components
func (agent *Agent) checkJobStatus(url string) (bool, error) {
	body, _, err := agent.request(url, "GET", http.StatusOK, nil)
	if err != nil {
		return false, err
	}

	var job mistifyagent.GuestJob
	if err := json.Unmarshal(body, &job); err != nil {
		return false, err
	}

	switch job.Status {
	case mistifyagent.Complete:
		return true, nil
	case mistifyagent.Errored:
		return false, errors.New(job.Message)
	default:
		return false, nil
	}
}

// GetGuest retrieves information on a guest from an agent
func (agent *Agent) GetGuest(guestID string) (*client.Guest, error) {
	guest, err := agent.requestGuestAction(guestID, "get")
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

	guest, err := agent.generateClientGuest(g)
	if err != nil {
		return nil, err
	}

	// Cycle through the candidates until one successfully creates. Track which
	// one via the guest's HypervisorID
	for _, hypervisor := range candidates {
		guest, err = agent.guestRequest(hypervisor, guestID, "create", guest)
		if err == nil {
			g.HypervisorID = hypervisor.ID
			if err := g.Save(); err != nil {
				fmt.Printf("Guest created on Hypervisor %s, but guest info persist failed", hypervisor.ID)
			}
			return guest, nil
		}
		fmt.Println("Guest Create Error:", err)
	}

	return nil, errors.New("failed to create guest")
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

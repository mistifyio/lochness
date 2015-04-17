// Package cli provides a client and utilities for lochness cli applications to
// interact with agents.
package cli

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Client interacts with an http api
type Client struct {
	c      http.Client
	t      string //type
	scheme string
	addr   string
}

// NewClient creates a new Client
func NewClient(address string) *Client {
	strings := strings.SplitN(address, "://", 2)
	return &Client{scheme: strings[0], addr: strings[1], t: "application/json"}
}

// URLString generates the full url given an endpoint path
func (c *Client) URLString(endpoint string) string {
	return c.scheme + "://" + path.Join(c.addr, endpoint)
}

// GetMany GETs a set of resources
func (c *Client) GetMany(title, endpoint string) ([]map[string]interface{}, *http.Response) {
	resp, err := c.c.Get(c.URLString(endpoint))
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	ret := []map[string]interface{}{}
	processResponse(resp, title, "get", http.StatusOK, &ret)
	return ret, resp
}

// GetList GETs an array of string (e.g. IDs)
func (c *Client) GetList(title, endpoint string) ([]string, *http.Response) {
	resp, err := c.c.Get(c.URLString(endpoint))
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	ret := []string{}
	processResponse(resp, title, "get", http.StatusOK, &ret)
	return ret, resp
}

// Get GETs a single resource
func (c *Client) Get(title, endpoint string) (map[string]interface{}, *http.Response) {
	resp, err := c.c.Get(c.URLString(endpoint))
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	ret := map[string]interface{}{}
	processResponse(resp, title, "get", http.StatusOK, &ret)
	return ret, resp
}

// Post POSTs a body
func (c *Client) Post(title, endpoint, body string) (map[string]interface{}, *http.Response) {
	resp, err := c.c.Post(c.URLString(endpoint), c.t, strings.NewReader(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  body,
		}).Fatal("unable to create new " + title)
	}
	ret := map[string]interface{}{}
	processResponse(resp, title, "create", http.StatusCreated, &ret)
	return ret, resp
}

// Delete DELETEs a resource
func (c *Client) Delete(title, endpoint string) (map[string]interface{}, *http.Response) {
	addr := c.URLString(endpoint)
	req, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
		}).Fatal("unable to form request")
	}
	req.Header.Add("ContentType", c.t)
	resp, err := c.c.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
		}).Fatal("unable to complete request")
	}

	ret := map[string]interface{}{}
	processResponse(resp, title, "delete", http.StatusOK, &ret)
	return ret, resp
}

// Patch PATCHes a resource
func (c *Client) Patch(title, endpoint, body string) (map[string]interface{}, *http.Response) {
	addr := c.URLString(endpoint)
	req, err := http.NewRequest("PATCH", addr, strings.NewReader(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
			"body":    body,
		}).Fatal("unable to form request")
	}
	req.Header.Add("ContentType", c.t)
	resp, err := c.c.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
			"body":    body,
		}).Fatal("unable to complete request")
	}
	ret := map[string]interface{}{}
	processResponse(resp, title, "update", http.StatusOK, &ret)
	return ret, resp
}

func parseError(dec *json.Decoder) (string, []interface{}) {
	jmap := JMap{}
	err := dec.Decode(&jmap)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "parseError",
		}).Error("failed to parse json")
		return "", []interface{}{}
	}

	msg := ""
	iface, ok := jmap["message"]
	if ok {
		if str, ok := iface.(string); ok {
			msg = str
		}
	}

	stack := []interface{}{}
	iface, ok = jmap["stack"]
	if ok {
		if slice, ok := iface.([]interface{}); ok {
			stack = slice
		}
	}
	return msg, stack
}

func processResponse(response *http.Response, title, action string, status int, dest interface{}) {
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)
	if response.StatusCode == status {
		if err := dec.Decode(dest); err != nil {
			log.WithField("error", err).Fatal("failed to parse json")
		}
		return
	}

	fields := log.Fields{
		"status": response.Status,
		"code":   response.StatusCode,
	}

	msg, stack := parseError(dec)
	if msg != "" {
		fields["message"] = msg
	}
	if len(stack) > 0 {
		if log.GetLevel() >= log.DebugLevel {
			fields["stack"] = stack
		}
	}

	log.WithFields(fields).Fatal("failed to " + action + " " + title)
}

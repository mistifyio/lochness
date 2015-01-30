package main

import (
	"encoding/json"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type client struct {
	http.Client
	t    string //type
	addr string
}

func newClient(address string) *client {
	if strings.HasSuffix(address, "/") {
		return &client{addr: address, t: "application/json"}
	}
	return &client{addr: address + "/", t: "application/json"}
}

func (c *client) getMany(title, endpoint string) []map[string]interface{} {
	resp, err := c.Get(c.addr + endpoint)
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to get " + title)
	}

	ret := []map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return ret
}

func (c *client) getList(title, endpoint string) []string {
	resp, err := c.Get(c.addr + endpoint)
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to get " + title)
	}

	ret := []string{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return ret
}

func (c *client) get(title, endpoint string) map[string]interface{} {
	resp, err := c.Get(c.addr + endpoint)
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to get " + title)
	}

	ret := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return ret
}

func (c *client) post(title, endpoint, body string) map[string]interface{} {
	resp, err := c.Post(c.addr+endpoint, c.t, strings.NewReader(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  body,
		}).Fatal("unable to create new " + title)
	}
	if resp.StatusCode != http.StatusCreated {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to create " + title)
	}
	defer resp.Body.Close()

	ret := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return ret
}

func (c *client) del(title, endpoint string) map[string]interface{} {
	addr := c.addr + endpoint
	req, err := http.NewRequest("DELETE", addr, nil)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
		}).Fatal("unable to form request")
	}
	req.Header.Add("ContentType", c.t)
	resp, err := c.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
		}).Fatal("unable to complete request")
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to delete " + title)
	}
	defer resp.Body.Close()

	ret := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return ret
}

func (c *client) put(title, endpoint, body string) map[string]interface{} {
	addr := c.addr + endpoint
	req, err := http.NewRequest("PATCH", addr, strings.NewReader(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
			"body":    body,
		}).Fatal("unable to form request")
	}
	req.Header.Add("ContentType", c.t)
	resp, err := c.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
			"body":    body,
		}).Fatal("unable to complete request")
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to modify " + title)
	}
	defer resp.Body.Close()

	ret := map[string]interface{}{}
	if err := json.NewDecoder(resp.Body).Decode(&ret); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return ret
}

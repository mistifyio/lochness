package cli

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type Client struct {
	c      http.Client
	t      string //type
	scheme string
	addr   string
}

func NewClient(address string) *Client {
	strings := strings.SplitN(address, "://", 2)
	return &Client{scheme: strings[0], addr: strings[1], t: "application/json"}
}

func (c *Client) URLString(endpoint string) string {
	return c.scheme + "://" + path.Join(c.addr, endpoint)
}

func (c *Client) GetMany(title, endpoint string) []map[string]interface{} {
	resp, err := c.c.Get(c.URLString(endpoint))
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	ret := []map[string]interface{}{}
	processResponse(resp, title, http.StatusOK, &ret)
	return ret
}

func (c *Client) GetList(title, endpoint string) []string {
	resp, err := c.c.Get(c.URLString(endpoint))
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	ret := []string{}
	processResponse(resp, title, http.StatusOK, &ret)
	return ret
}

func (c *Client) Get(title, endpoint string) map[string]interface{} {
	resp, err := c.c.Get(c.URLString(endpoint))
	if err != nil {
		log.WithField("error", err).Fatal("failed to get " + title)
	}
	ret := map[string]interface{}{}
	processResponse(resp, title, http.StatusOK, &ret)
	return ret
}

func (c *Client) Post(title, endpoint, body string) map[string]interface{} {
	resp, err := c.c.Post(c.addr+endpoint, c.t, strings.NewReader(body))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"body":  body,
		}).Fatal("unable to create new " + title)
	}
	ret := map[string]interface{}{}
	processResponse(resp, title, http.StatusCreated, &ret)
	return ret
}

func (c *Client) Del(title, endpoint string) map[string]interface{} {
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
	processResponse(resp, title, http.StatusOK, &ret)
	return ret
}

func (c *Client) Patch(title, endpoint, body string) map[string]interface{} {
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
	processResponse(resp, title, http.StatusOK, &ret)
	return ret
}

func processResponse(response *http.Response, title string, status int, dest interface{}) {
	defer response.Body.Close()

	if response.StatusCode != status {
		log.WithFields(log.Fields{
			"status": response.Status,
			"code":   response.StatusCode,
		}).Fatal("failed to get " + title)
	}

	if err := json.NewDecoder(response.Body).Decode(dest); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
}

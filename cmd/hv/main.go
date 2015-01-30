package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	c       = &client{addr: "http://localhost:17000/", t: "application/json"}
	verbose = false
)

type client struct {
	http.Client
	t    string //type
	addr string
}

type hypervisor map[string]interface{}

func (h hypervisor) ID() string {
	return h["id"].(string)
}

func (h hypervisor) String() string {
	buf, err := json.Marshal(&h)
	if err != nil {
		return ""
	}
	return string(buf)
}

func createHV(c *client, spec string) (hypervisor, error) {
	resp, err := c.Post(c.addr+"hypervisors", c.t, strings.NewReader(spec))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"spec":  spec,
		}).Error()
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		log.WithField("code", resp.StatusCode).Error("failed to create hypervisor")
	}
	defer resp.Body.Close()

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		log.WithField("error", err).Error("failed to parse json")
		return nil, err
	}
	return hv, nil
}

func getHVs(c *client) []hypervisor {
	resp, err := c.Get(c.addr + "hypervisors")
	if err != nil {
		log.WithField("error", err).Fatal("failed to get list of hypervisors")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithField("code", resp.StatusCode).Fatal("failed to get list of hypervisors")
	}

	hvs := []hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hvs); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return hvs
}

func getHV(c *client, id string) hypervisor {
	resp, err := c.Get(c.addr + "hypervisors/" + id)
	if err != nil {
		log.WithField("error", err).Fatal("failed to get hypervisor")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithField("code", resp.StatusCode).Fatal("failed to get hypervisor")
	}

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return hv
}

func list(cmd *cobra.Command, args []string) {
	hvs := []hypervisor{}
	if len(args) == 0 {
		hvs = getHVs(c)
	} else {
		for _, id := range args {
			hvs = append(hvs, getHV(c, id))
		}
	}
	for _, hv := range hvs {
		if verbose {
			fmt.Println(hv)
		} else {
			fmt.Println(hv.ID())
		}
	}
}

func create(cmd *cobra.Command, specs []string) {
	for _, spec := range specs {
		hv, err := createHV(c, spec)
		if err != nil {
			return
		}
		if verbose {
			fmt.Println(hv)
		} else {
			fmt.Println(hv["id"])
		}
	}
}

func main() {

	root := &cobra.Command{
		Use:   "hv",
		Short: "hv is the cli interface to grootslang",
	}
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", verbose, "print full hv description")

	cmdHVs := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "list the hypervisor(s)",
		Run:   list,
	}

	cmdCreate := &cobra.Command{
		Use:   "create <spec>...",
		Short: "create new hypervisor(s)",
		Long: `Create a new hypervisor using "spec" as the initial values. "spec" must be
valid json and contain the required fields, "mac" and "ip".`,
		Run: create,
	}
	root.AddCommand(cmdHVs, cmdCreate)
	root.Execute()
}

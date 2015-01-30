package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	verbose = false
	c       = &client{addr: "http://localhost:17000/", t: "application/json"}
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

func getHVs(c *client) ([]hypervisor, error) {
	resp, err := c.Get(c.addr + "hypervisors")
	if err != nil {
		log.WithField("error", err).Error("failed to get list of hypervisors")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithField("code", resp.StatusCode).Error("failed to get list of hypervisors")
	}

	hvs := []hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hvs); err != nil {
		log.WithField("error", err).Error("failed to parse json")
		return nil, err
	}
	return hvs, nil
}

func getHV(c *client, id string) (hypervisor, error) {
	resp, err := c.Get(c.addr + "hypervisors/" + id)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		panic(err)
	}
	return hv, nil
}

func list(cmd *cobra.Command, args []string) {
	hvs := []hypervisor{}
	if len(args) > 0 {
		for _, id := range args {
			hv, err := getHV(c, id)
			if err != nil {
				return
			}
			hvs = append(hvs, hv)
		}
	} else {
		var err error
		hvs, err = getHVs(c)
		if err != nil {
			return
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
	ret := 0

	root := &cobra.Command{
		Use:   "hv",
		Short: "hv is the cli interface to grootslang",
	}
	root.PersistentFlags().BoolVarP(&verbose, "verbose", "v", verbose, "print full hv description")

	cmdHVs := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "list the hypervisor(s)",
		Run: list,
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
	os.Exit(ret)
}

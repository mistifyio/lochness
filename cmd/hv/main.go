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
	server  = "http://localhost:17000/"
	verbose = false
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

func getHVs(c *client) []hypervisor {
	resp, err := c.Get(c.addr + "hypervisors")
	if err != nil {
		log.WithField("error", err).Fatal("failed to get list of hypervisors")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to get hypervisors")
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
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to get hypervisor")
	}

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return hv
}

func createHV(c *client, spec string) hypervisor {
	resp, err := c.Post(c.addr+"hypervisors", c.t, strings.NewReader(spec))
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"spec":  spec,
		}).Fatal("unable to create new hypervisor")
	}
	if resp.StatusCode != http.StatusCreated {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to create hypervisor")
	}
	defer resp.Body.Close()

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return hv
}

func modifyHV(c *client, id string, spec string) hypervisor {
	addr := c.addr + "hypervisors/" + id
	req, err := http.NewRequest("PATCH", addr, strings.NewReader(spec))
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
			"spec":    spec,
		}).Fatal("unable to form request")
	}
	req.Header.Add("ContentType", c.t)
	resp, err := c.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": addr,
			"spec":    spec,
		}).Fatal("unable to complete request")
	}
	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status": resp.Status,
			"code":   resp.StatusCode,
		}).Fatal("failed to modify hypervisor")
	}
	defer resp.Body.Close()

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return hv
}

func deleteHV(c *client, id string) hypervisor {
	addr := c.addr + "hypervisors/" + id
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
		}).Fatal("failed to delete hypervisor")
	}
	defer resp.Body.Close()

	hv := hypervisor{}
	if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
		log.WithField("error", err).Fatal("failed to parse json")
	}
	return hv
}

func list(cmd *cobra.Command, args []string) {
	c := newClient(server)
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
	c := newClient(server)
	for _, spec := range specs {
		hv := createHV(c, spec)
		if verbose {
			fmt.Println(hv)
		} else {
			fmt.Println(hv["id"])
		}
	}
}

func modify(cmd *cobra.Command, args []string) {
	c := newClient(server)
	for _, arg := range args {
		idSpec := strings.SplitN(arg, "=", 2)
		if len(idSpec) != 2 {
			log.WithFields(log.Fields{
				"arg": arg,
			}).Fatal("invalid argument")
		}
		id := idSpec[0]
		spec := idSpec[1]
		hv := modifyHV(c, id, spec)
		if verbose {
			fmt.Println(hv)
		} else {
			fmt.Println(hv["id"])
		}
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	for _, id := range ids {
		hv := deleteHV(c, id)
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
	root.PersistentFlags().StringVarP(&server, "server", "s", server, "server address to connect to")

	cmdList := &cobra.Command{
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
	cmdMod := &cobra.Command{
		Use:   "modify id=<spec>...",
		Short: "modify hypervisor(s)",
		Long:  `Modify given hypervisor(s). Where "spec" is a valid json string.`,
		Run:   modify,
	}
	cmdDel := &cobra.Command{
		Use:   "delete <id>...",
		Short: "delete the hypervisor(s)",
		Run:   del,
	}
	root.AddCommand(cmdList, cmdCreate, cmdMod, cmdDel)
	root.Execute()
}

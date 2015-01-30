package main

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	server  = "http://localhost:17000/"
	verbose = false
)

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
	ret := c.getMany("hypervisors", "hypervisors")
	// wasteful you say?
	hvs := make([]hypervisor, len(ret))
	for i := range ret {
		hvs[i] = ret[i]
	}
	return hvs
}

func getHV(c *client, id string) hypervisor {
	return c.get("hypervisor", "hypervisors/"+id)
}

func createHV(c *client, spec string) hypervisor {
	return c.post("hypervisor", "hypervisors", spec)
}

func modifyHV(c *client, id string, spec string) hypervisor {
	return c.put("hypervisor", "hypervisors/"+id, spec)
}

func deleteHV(c *client, id string) hypervisor {
	return c.del("hypervisor", "hypervisors/"+id)
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

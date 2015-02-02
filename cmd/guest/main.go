package main

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	server  = "http://localhost:18000/"
	jsonout = false
	t       = "application/json"
)

type (
	jmap map[string]interface{}
)

func (j jmap) ID() string {
	return j["id"].(string)
}

func (j jmap) String() string {
	buf, err := json.Marshal(&j)
	if err != nil {
		return ""
	}
	return string(buf)
}

func (g jmap) Print() {
	if jsonout {
		fmt.Println(g)
	} else {
		fmt.Println(g.ID())
	}
}

func getGuests(c *client) []jmap {
	ret := c.getMany("guests", "guests")
	guests := make([]jmap, len(ret))
	for i := range ret {
		guests[i] = ret[i]
	}
	return guests
}

func getGuest(c *client, id string) jmap {
	return c.get("guest", "guests/"+id)
}

func createGuest(c *client, spec string) jmap {
	return c.post("guest", "guests", spec)
}

func modifyGuest(c *client, id string, spec string) jmap {
	return c.patch("guest", "guests/"+id, spec)
}

func deleteGuest(c *client, id string) jmap {
	return c.del("hypervisor", "guests/"+id)
}

func list(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	guests := []jmap{}

	if len(ids) == 0 {
		guests = getGuests(c)
	} else {
		for _, id := range ids {
			guests = append(guests, getGuest(c, id))
		}
	}

	for _, guest := range guests {
		guest.Print()
	}
}

func create(cmd *cobra.Command, specs []string) {
	c := newClient(server)
	for _, spec := range specs {
		guest := createGuest(c, spec)
		guest.Print()
	}
}

func modify(cmd *cobra.Command, args []string) {
	c := newClient(server)
	for _, arg := range args {
		idSpec := strings.SplitN(arg, "=", 2)
		if len(idSpec) != 2 {
			log.WithField("arg", arg).Fatal("invalid argument")
		}
		id := idSpec[0]
		spec := idSpec[1]
		guest := modifyGuest(c, id, spec)
		guest.Print()
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	for _, id := range ids {
		guest := deleteGuest(c, id)
		guest.Print()
	}
}

func main() {
	root := &cobra.Command{
		Use:   "guest",
		Short: "guest is the cli interface to waheela",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Help()
		},
	}
	root.PersistentFlags().BoolVarP(&jsonout, "jsonout", "j", jsonout, "output in json")
	root.PersistentFlags().StringVarP(&server, "server", "s", server, "server address to connect to")

	cmdList := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "list the guest(s)",
		Run:   list,
	}

	cmdCreate := &cobra.Command{
		Use:   "create <spec>...",
		Short: "create guest(s)",
		Long:  `Create new guest(s) using "spec"(s) as the initial values. Where "spec" is a valid json string.`,
		Run:   modify,
	}

	cmdModify := &cobra.Command{
		Use:   "modify id=<spec>...",
		Short: "modify guest(s)",
		Long:  `Modify given guest(s). Where "spec" is a valid json string.`,
		Run:   modify,
	}

	cmdDelete := &cobra.Command{
		Use:   "delete <id>...",
		Short: "delete guest(s)",
		Run:   del,
	}

	root.AddCommand(cmdList, cmdCreate, cmdModify, cmdDelete)
	root.Execute()
}

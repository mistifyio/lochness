package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/Sirupsen/logrus"
	"github.com/mistifyio/lochness/pkg/internal/cli"
	"github.com/spf13/cobra"
)

var (
	server  = "http://localhost:18000/"
	jsonout = false
	t       = "application/json"
)

type (
	jmap      map[string]interface{}
	jmapSlice []jmap
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

func (j jmap) Print() {
	if jsonout {
		fmt.Println(j)
	} else {
		fmt.Println(j.ID())
	}
}

func (js jmapSlice) Len() int {
	return len(js)
}

func (js jmapSlice) Less(i, j int) bool {
	return js[i].ID() < js[j].ID()
}

func (js jmapSlice) Swap(i, j int) {
	js[j], js[i] = js[i], js[j]
}

func assertID(id string) {
	if uuid := uuid.Parse(id); uuid == nil {
		log.WithField("id", id).Fatal("invalid id")
	}
}

func assertSpec(spec string) {
	j := jmap{}
	if err := json.Unmarshal([]byte(spec), &j); err != nil {
		log.WithFields(log.Fields{
			"spec":  spec,
			"error": err,
		}).Fatal("invalid spec")
	}
}

func help(cmd *cobra.Command, _ []string) {
	cmd.Help()
}

func getGuests(c *cli.Client) []jmap {
	ret := c.GetMany("guests", "guests")
	guests := make([]jmap, len(ret))
	for i := range ret {
		guests[i] = ret[i]
	}
	return guests
}

func getGuest(c *cli.Client, id string) jmap {
	return c.Get("guest", "guests/"+id)
}

func createGuest(c *cli.Client, spec string) jmap {
	return c.Post("guest", "guests", spec)
}

func modifyGuest(c *cli.Client, id string, spec string) jmap {
	return c.Patch("guest", "guests/"+id, spec)
}

func deleteGuest(c *cli.Client, id string) jmap {
	return c.Del("hypervisor", "guests/"+id)
}

func list(cmd *cobra.Command, ids []string) {
	c := cli.New(server)
	guests := []jmap{}

	if len(ids) == 0 {
		guests = getGuests(c)
		sort.Sort(jmapSlice(guests))
	} else {
		for _, id := range ids {
			assertID(id)
			guests = append(guests, getGuest(c, id))
		}
	}

	for _, guest := range guests {
		guest.Print()
	}
}

func create(cmd *cobra.Command, specs []string) {
	c := cli.New(server)
	for _, spec := range specs {
		assertSpec(spec)
		guest := createGuest(c, spec)
		guest.Print()
	}
}

func modify(cmd *cobra.Command, args []string) {
	c := cli.New(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even number of args")
	}
	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		guest := modifyGuest(c, id, spec)
		guest.Print()
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := cli.New(server)
	for _, id := range ids {
		assertID(id)
		guest := deleteGuest(c, id)
		guest.Print()
	}
}

func main() {
	root := &cobra.Command{
		Use:   "guest",
		Short: "guest is the cli interface to waheela",
		Run:   help,
	}
	root.PersistentFlags().BoolVarP(&jsonout, "jsonout", "j", jsonout, "output in json")
	root.PersistentFlags().StringVarP(&server, "server", "s", server, "server address to connect to")

	cmdList := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "List the guests",
		Run:   list,
	}

	cmdCreate := &cobra.Command{
		Use:   "create <spec>...",
		Short: "Create guests",
		Long:  `Create new guest(s) using "spec"(s) as the initial values. Where "spec" is a valid json string.`,
		Run:   create,
	}

	cmdModify := &cobra.Command{
		Use:   "modify (<id> <spec>)...",
		Short: "Modify guests",
		Long:  `Modify given guest(s). Where "spec" is a valid json string.`,
		Run:   modify,
	}

	cmdDelete := &cobra.Command{
		Use:   "delete <id>...",
		Short: "Delete guests",
		Run:   del,
	}

	root.AddCommand(cmdList, cmdCreate, cmdModify, cmdDelete)
	root.Execute()
}

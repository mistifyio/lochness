package main

import (
	"os"
	"sort"

	log "github.com/Sirupsen/logrus"
	"github.com/andrew-d/go-termutil"
	"github.com/mistifyio/lochness/pkg/internal/cli"
	"github.com/spf13/cobra"
)

var (
	server  = "http://localhost:18000/"
	jsonout = false
	t       = "application/json"
)

func help(cmd *cobra.Command, _ []string) {
	cmd.Help()
}

func getGuests(c *cli.Client) []cli.JMap {
	ret := c.GetMany("guests", "guests")
	guests := make([]cli.JMap, len(ret))
	for i := range ret {
		guests[i] = ret[i]
	}
	return guests
}

func getGuest(c *cli.Client, id string) cli.JMap {
	return c.Get("guest", "guests/"+id)
}

func createGuest(c *cli.Client, spec string) cli.JMap {
	return c.Post("guest", "guests", spec)
}

func modifyGuest(c *cli.Client, id string, spec string) cli.JMap {
	return c.Patch("guest", "guests/"+id, spec)
}

func deleteGuest(c *cli.Client, id string) cli.JMap {
	return c.Del("hypervisor", "guests/"+id)
}

func list(cmd *cobra.Command, args []string) {
	c := cli.NewClient(server)
	guests := []cli.JMap{}
	if len(args) == 0 {
		if termutil.Isatty(os.Stdin.Fd()) {
			guests = getGuests(c)
			sort.Sort(cli.JMapSlice(guests))
		} else {
			args = cli.Read(os.Stdin)
		}
	}
	if len(guests) == 0 {
		for _, id := range args {
			cli.AssertID(id)
			guests = append(guests, getGuest(c, id))
		}
	}

	for _, guest := range guests {
		guest.Print(jsonout)
	}
}

func create(cmd *cobra.Command, specs []string) {
	c := cli.NewClient(server)
	if len(specs) == 0 {
		specs = cli.Read(os.Stdin)
	}

	for _, spec := range specs {
		cli.AssertSpec(spec)
		guest := createGuest(c, spec)
		guest.Print(jsonout)
	}
}

func modify(cmd *cobra.Command, args []string) {
	c := cli.NewClient(server)
	if len(args) == 0 {
		args = cli.Read(os.Stdin)
	}
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even number of args")
	}

	for i := 0; i < len(args); i += 2 {
		id := args[i]
		cli.AssertID(id)
		spec := args[i+1]
		cli.AssertSpec(spec)

		guest := modifyGuest(c, id, spec)
		guest.Print(jsonout)
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := cli.NewClient(server)
	if len(ids) == 0 {
		ids = cli.Read(os.Stdin)
	}

	for _, id := range ids {
		cli.AssertID(id)
		guest := deleteGuest(c, id)
		guest.Print(jsonout)
	}
}

func main() {
	root := &cobra.Command{
		Use:  "guest",
		Long: "guest is the cli interface to waheela. All commands support arguments via command line or stdin.",
		Run:  help,
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

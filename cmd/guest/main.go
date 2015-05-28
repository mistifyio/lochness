package main

import (
	"fmt"
	"os"
	"sort"
	"unicode"
	"unicode/utf8"

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
	if err := cmd.Help(); err != nil {
		log.WithField("error", err).Fatal("help")
	}
}

func getGuests(c *cli.Client) []cli.JMap {
	ret, _ := c.GetMany("guests", "guests")
	guests := make([]cli.JMap, len(ret))
	for i := range ret {
		guests[i] = ret[i]
	}
	return guests
}

func getGuest(c *cli.Client, id string) cli.JMap {
	guest, _ := c.Get("guest", "guests/"+id)
	return guest
}

func createGuest(c *cli.Client, spec string) cli.JMap {
	guest, resp := c.Post("guest", "guests", spec)
	j := cli.JMap{
		"id":    resp.Header.Get("x-guest-job-id"),
		"guest": guest,
	}
	return j
}

func modifyGuest(c *cli.Client, id string, spec string) cli.JMap {
	guest, _ := c.Patch("guest", "guests/"+id, spec)
	return guest
}

func deleteGuest(c *cli.Client, id string) cli.JMap {
	guest, resp := c.Delete("guest", "guests/"+id)
	j := cli.JMap{
		"id":    resp.Header.Get("x-guest-job-id"),
		"guest": guest,
	}
	return j
}

func guestAction(c *cli.Client, id, action string) cli.JMap {
	guest, resp := c.Post("guest", fmt.Sprintf("guests/%s/%s", id, action), "")
	j := cli.JMap{
		"id":    resp.Header.Get("x-guest-job-id"),
		"guest": guest,
	}

	return j
}

func getJob(c *cli.Client, id string) cli.JMap {
	job, _ := c.Get("job", "jobs/"+id)
	return job
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
		j := createGuest(c, spec)
		j.Print(jsonout)
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
		j := deleteGuest(c, id)
		j.Print(jsonout)
	}
}

func generateActionHandler(action string) func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, ids []string) {
		c := cli.NewClient(server)
		if len(ids) == 0 {
			ids = cli.Read(os.Stdin)
		}

		for _, id := range ids {
			cli.AssertID(id)
			j := guestAction(c, id, action)
			j.Print(jsonout)
		}
	}
}

func job(cmd *cobra.Command, ids []string) {
	c := cli.NewClient(server)
	if len(ids) == 0 {
		ids = cli.Read(os.Stdin)
	}

	for _, id := range ids {
		cli.AssertID(id)
		job := getJob(c, id)
		job.Print(jsonout)
	}
}

func main() {
	root := &cobra.Command{
		Use:  "guest",
		Long: "guest is the cli interface to cguestd. All commands support arguments via command line or stdin.",
		Run:  help,
	}
	root.PersistentFlags().BoolVarP(&jsonout, "json", "j", jsonout, "output in json")
	root.PersistentFlags().StringVarP(&server, "server", "s", server, "server address to connect to")

	cmdList := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "List the guests",
		Run:   list,
	}
	root.AddCommand(cmdList)

	cmdCreate := &cobra.Command{
		Use:   "create <spec>...",
		Short: "Create guests asynchronously",
		Long:  `Create new guest(s) using "spec"(s) as the initial values. Where "spec" is a valid json string.`,
		Run:   create,
	}
	root.AddCommand(cmdCreate)

	cmdModify := &cobra.Command{
		Use:   "modify (<id> <spec>)...",
		Short: "Modify guests",
		Long:  `Modify given guest(s). Where "spec" is a valid json string.`,
		Run:   modify,
	}
	root.AddCommand(cmdModify)

	cmdDelete := &cobra.Command{
		Use:   "delete <id>...",
		Short: "Delete guests asynchronously",
		Run:   del,
	}
	root.AddCommand(cmdDelete)

	for _, action := range []string{"shutdown", "reboot", "restart", "poweroff", "start", "suspend"} {
		a, n := utf8.DecodeRuneInString(action)
		cmdAction := &cobra.Command{
			Use:   fmt.Sprintf("%s <id>...", action),
			Short: fmt.Sprintf("%s guests asynchronously", string(unicode.ToUpper(a))+action[n:]),
			Run:   generateActionHandler(action),
		}
		root.AddCommand(cmdAction)
	}

	cmdJob := &cobra.Command{
		Use:   "job <id>...",
		Short: "Check status of guest jobs",
		Run:   job,
	}
	root.AddCommand(cmdJob)

	if err := root.Execute(); err != nil {
		log.WithField("error", err).Fatal("failed to execute root command")
	}
}

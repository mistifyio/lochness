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
	server  = "http://localhost:17000"
	jsonout = false
)

func assertID(id string) {
	if uuid := uuid.Parse(id); uuid == nil {
		log.WithFields(log.Fields{
			"id": id,
		}).Fatal("invalid id")
	}
}

func assertSpec(spec string) {
	j := cli.JMap{}
	if err := json.Unmarshal([]byte(spec), &j); err != nil {
		log.WithFields(log.Fields{
			"spec":  spec,
			"error": err,
		}).Fatal("invalid spec")
	}
}

func printTreeMap(id, key string, m map[string]interface{}) {
	if jsonout {
		c := cli.JMap{"id": id}
		if len(m) != 0 {
			c[key] = m
		}
		fmt.Println(c)
	} else {
		fmt.Println(id)
		if len(m) == 0 {
			return
		}
		keys := make([]string, len(m))
		keys = keys[0:0]
		for key := range m {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		if len(m) > 1 {
			for _, key := range keys[:len(keys)-1] {
				fmt.Print("├── ", key, ":", m[key], "\n")
			}
		}
		key := keys[len(keys)-1]
		fmt.Print("└── ", key, ":", m[key], "\n")
	}
}

func printTreeSlice(id, key string, s []string) {
	if jsonout {
		c := cli.JMap{
			"id": id,
		}
		if len(s) != 0 {
			c[key] = s
		}
		fmt.Println(c)
	} else {
		fmt.Println(id)
		if len(s) == 0 {
			return
		}
		sort.Strings(s)
		if len(s) > 1 {
			for _, item := range s[:len(s)-1] {
				fmt.Println("├──", item)
			}
		}
		fmt.Println("└──", s[len(s)-1])
	}
}

func help(cmd *cobra.Command, _ []string) {
	cmd.Help()
}

func getHVs(c *cli.Client) []cli.JMap {
	ret := c.GetMany("hypervisors", "hypervisors")
	// wasteful you say?
	hvs := make([]cli.JMap, len(ret))
	for i := range ret {
		hvs[i] = ret[i]
	}
	return hvs
}

func getGuests(c *cli.Client, id string) []string {
	return c.GetList("guests", "hypervisors/"+id+"/guests")
}

func getHV(c *cli.Client, id string) cli.JMap {
	return c.Get("hypervisor", "hypervisors/"+id)
}

func createHV(c *cli.Client, spec string) cli.JMap {
	return c.Post("hypervisor", "hypervisors", spec)
}

func modifyHV(c *cli.Client, id string, spec string) cli.JMap {
	return c.Patch("hypervisor", "hypervisors/"+id, spec)
}

func modifyConfig(c *cli.Client, id string, spec string) cli.JMap {
	return c.Patch("config", "hypervisors/"+id+"/config", spec)
}

func modifySubnets(c *cli.Client, id string, spec string) cli.JMap {
	return c.Patch("subnets", "hypervisors/"+id+"/subnets", spec)
}

func deleteHV(c *cli.Client, id string) cli.JMap {
	return c.Del("hypervisor", "hypervisors/"+id)
}

func deleteSubnet(c *cli.Client, hv, subnet string) cli.JMap {
	return c.Del("subnet", "hypervisors/"+hv+"/subnets/"+subnet)
}

func list(cmd *cobra.Command, args []string) {
	c := cli.New(server)
	hvs := []cli.JMap{}
	if len(args) == 0 {
		hvs = getHVs(c)
		sort.Sort(cli.JMapSlice(hvs))
	} else {
		for _, id := range args {
			assertID(id)
			hvs = append(hvs, getHV(c, id))
		}
	}

	for _, hv := range hvs {
		hv.Print(jsonout)
	}
}

func create(cmd *cobra.Command, specs []string) {
	c := cli.New(server)
	for _, spec := range specs {
		assertSpec(spec)
		hv := createHV(c, spec)
		hv.Print(jsonout)
	}
}

func modify(cmd *cobra.Command, args []string) {
	c := cli.New(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}

	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		hv := modifyHV(c, id, spec)
		hv.Print(jsonout)
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := cli.New(server)
	for _, id := range ids {
		assertID(id)
		hv := deleteHV(c, id)
		hv.Print(jsonout)
	}
}

func guests(cmd *cobra.Command, ids []string) {
	c := cli.New(server)
	if len(ids) == 0 {
		for _, hv := range getHVs(c) {
			ids = append(ids, hv["id"].(string))
		}
		sort.Strings(ids)
	} else {
		for _, id := range ids {
			assertID(id)
		}
	}

	for _, id := range ids {
		guests := getGuests(c, id)
		printTreeSlice(id, "guests", guests)
	}
}

func config(cmd *cobra.Command, ids []string) {
	c := cli.New(server)
	if len(ids) == 0 {
		for _, hv := range getHVs(c) {
			ids = append(ids, hv["id"].(string))
		}
		sort.Strings(ids)
	} else {
		for _, id := range ids {
			assertID(id)
		}
	}

	for _, id := range ids {
		config := c.Get("config", "hypervisors/"+id+"/config")
		printTreeMap(id, "config", config)
	}
}

func config_modify(cmd *cobra.Command, args []string) {
	c := cli.New(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}

	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		config := modifyConfig(c, id, spec)
		printTreeMap(id, "config", config)
	}
}

func subnets(cmd *cobra.Command, ids []string) {
	c := cli.New(server)
	if len(ids) == 0 {
		for _, hv := range getHVs(c) {
			ids = append(ids, hv["id"].(string))
		}
		sort.Strings(ids)
	} else {
		for _, id := range ids {
			assertID(id)
		}
	}

	for _, id := range ids {
		subnet := c.Get("subnet", "hypervisors/"+id+"/subnets")
		printTreeMap(id, "subnet", subnet)
	}
}

func subnets_modify(cmd *cobra.Command, args []string) {
	c := cli.New(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}

	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		subnet := modifySubnets(c, id, spec)
		printTreeMap(id, "subnet", subnet)
	}
}

func subnets_del(cmd *cobra.Command, args []string) {
	c := cli.New(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}

	for i := 0; i < len(args); i += 2 {
		hv := args[i]
		assertID(hv)
		subnet := args[i+1]
		assertSpec(subnet)

		deleted := deleteSubnet(c, hv, subnet)
		deleted.Print(jsonout)
	}
}

func main() {
	root := &cobra.Command{
		Use:   "hv",
		Short: "hv is the cli interface to grootslang",
		Run:   help,
	}
	root.PersistentFlags().BoolVarP(&jsonout, "json", "j", jsonout, "output in json")
	root.PersistentFlags().StringVarP(&server, "server", "s", server, "server address to connect to")

	cmdList := &cobra.Command{
		Use:   "list [<hv>...]",
		Short: "List the hypervisors",
		Run:   list,
	}
	cmdCreate := &cobra.Command{
		Use:   "create <spec>...",
		Short: "Create new hypervisors",
		Long: `Create a new hypervisor using "spec" as the initial values. "spec" must be
valid json and contain the required fields, "mac" and "ip".`,
		Run: create,
	}
	cmdMod := &cobra.Command{
		Use:   "modify (<hv> <spec>)...",
		Short: "Modify hypervisors",
		Long:  `Modify given hypervisor. Where "spec" is a valid json string.`,
		Run:   modify,
	}
	cmdDel := &cobra.Command{
		Use:   "delete <hv>...",
		Short: "Delete hypervisors",
		Run:   del,
	}
	cmdGuestsRoot := &cobra.Command{
		Use:   "guests",
		Short: "Operate on hypervisor guests",
		Run:   help,
	}
	cmdGuestsList := &cobra.Command{
		Use:   "list [<hv>...]",
		Short: "List the guests belonging to hypervisor",
		Run:   guests,
	}
	cmdConfigRoot := &cobra.Command{
		Use:   "config",
		Short: "Operate on hypervisor config",
		Run:   help,
	}
	cmdConfigList := &cobra.Command{
		Use:   "list [<hv>...]",
		Short: "Get hypervisor config",
		Run:   config,
	}
	cmdConfigMod := &cobra.Command{
		Use:   "modify (<hv> <spec>)...",
		Short: "Modify hypervisor config",
		Long:  `Modify the config of given hypervisor. Where "spec" is a valid json string.`,
		Run:   config_modify,
	}
	cmdSubnetsRoot := &cobra.Command{
		Use:   "subnets",
		Short: "Operate on hypervisor subnets",
		Run:   help,
	}
	cmdSubnetsList := &cobra.Command{
		Use:   "list [<hv>...]",
		Short: "Get hypervisor subnets",
		Run:   subnets,
	}
	cmdSubnetsMod := &cobra.Command{
		Use:   "modify (<hv> <spec>)...",
		Short: "Modify hypervisor subnets",
		Long:  `Modify the subnets of given hypervisor. Where "spec" is a valid json string.`,
		Run:   subnets_modify,
	}
	cmdSubnetsDel := &cobra.Command{
		Use:   "delete (<hv> <subnet>)...",
		Short: "Delete hypervisor subnets",
		Run:   subnets_del,
	}

	root.AddCommand(cmdList,
		cmdCreate,
		cmdDel,
		cmdMod,
		cmdGuestsRoot,
		cmdConfigRoot,
		cmdSubnetsRoot)
	cmdConfigRoot.AddCommand(cmdConfigList, cmdConfigMod)
	cmdGuestsRoot.AddCommand(cmdGuestsList)
	cmdSubnetsRoot.AddCommand(cmdSubnetsList, cmdSubnetsMod, cmdSubnetsDel)
	root.Execute()
}

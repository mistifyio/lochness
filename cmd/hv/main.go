package main

import (
	"encoding/json"
	"fmt"

	"code.google.com/p/go-uuid/uuid"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	server  = "http://localhost:17000"
	jsonout = false
)

type jmap map[string]interface{}

func (j jmap) ID() string {
	return j["id"].(string)
}

func (h jmap) String() string {
	buf, err := json.Marshal(&h)
	if err != nil {
		return ""
	}
	return string(buf)
}

func assertID(id string) {
	if uuid := uuid.Parse(id); uuid == nil {
		log.WithFields(log.Fields{
			"id": id,
		}).Fatal("invalid id")
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

func getHVs(c *client) []jmap {
	ret := c.getMany("hypervisors", "hypervisors")
	// wasteful you say?
	hvs := make([]jmap, len(ret))
	for i := range ret {
		hvs[i] = ret[i]
	}
	return hvs
}

func getGuests(c *client, id string) []string {
	return c.getList("guests", "hypervisors/"+id+"/guests")
}

func getHV(c *client, id string) jmap {
	return c.get("hypervisor", "hypervisors/"+id)
}

func createHV(c *client, spec string) jmap {
	return c.post("hypervisor", "hypervisors", spec)
}

func modifyHV(c *client, id string, spec string) jmap {
	return c.patch("hypervisor", "hypervisors/"+id, spec)
}

func modifyConfig(c *client, id string, spec string) jmap {
	return c.patch("config", "hypervisors/"+id+"/config", spec)
}

func modifySubnets(c *client, id string, spec string) jmap {
	return c.patch("subnets", "hypervisors/"+id+"/subnets", spec)
}

func deleteHV(c *client, id string) jmap {
	return c.del("hypervisor", "hypervisors/"+id)
}

func list(cmd *cobra.Command, args []string) {
	c := newClient(server)
	hvs := []jmap{}
	if len(args) == 0 {
		hvs = getHVs(c)
	} else {
		for _, id := range args {
			assertID(id)
			hvs = append(hvs, getHV(c, id))
		}
	}
	for _, hv := range hvs {
		if jsonout {
			fmt.Println(hv)
		} else {
			fmt.Println(hv.ID())
		}
	}
}

func create(cmd *cobra.Command, specs []string) {
	c := newClient(server)
	for _, spec := range specs {
		assertSpec(spec)
		hv := createHV(c, spec)
		if jsonout {
			fmt.Println(hv)
		} else {
			fmt.Println(hv["id"])
		}
	}
}

func modify(cmd *cobra.Command, args []string) {
	c := newClient(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}
	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		hv := modifyHV(c, id, spec)
		if jsonout {
			fmt.Println(hv)
		} else {
			fmt.Println(hv["id"])
		}
	}
}

func del(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	for _, id := range ids {
		assertID(id)
		hv := deleteHV(c, id)
		if jsonout {
			fmt.Println(hv)
		} else {
			fmt.Println(hv["id"])
		}
	}
}

func guests(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	if len(ids) == 0 {
		for _, hv := range getHVs(c) {
			ids = append(ids, hv["id"].(string))
		}
	} else {
		for _, id := range ids {
			assertID(id)
		}
	}
	for _, id := range ids {
		fmt.Println(id)
		guests := getGuests(c, id)

		if jsonout {
			j := jmap{
				"id": id,
			}
			if len(guests) != 0 {
				j["guests"] = guests
			}
			fmt.Println(j)
		} else {
			switch len(guests) {
			case 0:
			default:
				for _, guest := range guests[:len(guests)-1] {
					fmt.Println("├──", guest)
				}
				fallthrough
			case 1:
				fmt.Println("└──", guests[len(guests)-1])
			}
		}
	}
}

func config(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	if len(ids) == 0 {
		for _, hv := range getHVs(c) {
			ids = append(ids, hv["id"].(string))
		}
	} else {
		for _, id := range ids {
			assertID(id)
		}
	}
	for _, id := range ids {
		config := c.get("config", "hypervisors/"+id+"/config")
		if jsonout {
			c := jmap{
				"id": id,
			}
			if len(config) != 0 {
				c["config"] = config
			}
			fmt.Println(c)
		} else {
			fmt.Println(id)
			if len(config) == 0 {
				continue
			}
			fmt.Print("└── ")
			for k, v := range config {
				fmt.Print(k, ":", v, " ")
			}
			fmt.Println()
		}
	}
}

func config_modify(cmd *cobra.Command, args []string) {
	c := newClient(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}
	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		config := modifyConfig(c, id, spec)
		if jsonout {
			c := jmap{
				"id": id,
			}
			if len(config) != 0 {
				c["config"] = config
			}
			fmt.Println(c)
		} else {
			fmt.Println(id)
			if len(config) == 0 {
				continue
			}
			fmt.Print("└── ")
			for k, v := range config {
				fmt.Print(k, ":", v, " ")
			}
			fmt.Println()
		}
	}
}

func subnets(cmd *cobra.Command, ids []string) {
	c := newClient(server)
	if len(ids) == 0 {
		for _, hv := range getHVs(c) {
			ids = append(ids, hv["id"].(string))
		}
	} else {
		for _, id := range ids {
			assertID(id)
		}
	}
	for _, id := range ids {
		subnet := c.get("subnet", "hypervisors/"+id+"/subnets")
		if jsonout {
			c := jmap{
				"id": id,
			}
			if len(subnet) != 0 {
				c["subnet"] = subnet
			}
			fmt.Println(c)
		} else {
			fmt.Println(id)
			if len(subnet) == 0 {
				continue
			}
			fmt.Print("└── ")
			for k, v := range subnet {
				fmt.Print(k, ":", v, " ")
			}
			fmt.Println()
		}
	}
}

func subnets_modify(cmd *cobra.Command, args []string) {
	c := newClient(server)
	if len(args)%2 != 0 {
		log.WithField("num", len(args)).Fatal("expected an even amount of args")
	}
	for i := 0; i < len(args); i += 2 {
		id := args[i]
		assertID(id)
		spec := args[i+1]
		assertSpec(spec)

		subnet := modifySubnets(c, id, spec)
		if jsonout {
			c := jmap{
				"id": id,
			}
			if len(subnet) != 0 {
				c["subnet"] = subnet
			}
			fmt.Println(c)
		} else {
			fmt.Println(id)
			if len(subnet) == 0 {
				continue
			}
			fmt.Print("└── ")
			for k, v := range subnet {
				fmt.Print(k, ":", v, " ")
			}
			fmt.Println()
		}
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

	root.AddCommand(cmdList,
		cmdCreate,
		cmdDel,
		cmdMod,
		cmdGuestsRoot,
		cmdConfigRoot,
		cmdSubnetsRoot)
	cmdConfigRoot.AddCommand(cmdConfigList, cmdConfigMod)
	cmdGuestsRoot.AddCommand(cmdGuestsList)
	cmdSubnetsRoot.AddCommand(cmdSubnetsList, cmdSubnetsMod)
	root.Execute()
}

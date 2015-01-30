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

func main() {
	client := &client{addr: "http://localhost:17000/", t: "application/json"}
	ret := 0

	root := &cobra.Command{
		Use:   "hv",
		Short: "hv is the cli interface to grootslang",
	}

	full := false
	cmdHVs := &cobra.Command{
		Use:   "list [<id>...]",
		Short: "list the hypervisor(s)",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				for _, id := range args {
					resp, err := client.Get(client.addr + "hypervisors/" + id)
					if err != nil {
						log.Fatal(err)
					}
					defer resp.Body.Close()

					hv := hypervisor{}
					if err := json.NewDecoder(resp.Body).Decode(&hv); err != nil {
						panic(err)
					}
					if full {
						fmt.Println(hv)
					} else {
						fmt.Println(hv.ID)
					}
				}
				return
			}
			resp, err := client.Get(client.addr + "hypervisors")
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			hvs := []hypervisor{}
			if err := json.NewDecoder(resp.Body).Decode(&hvs); err != nil {
				panic(err)
			}
			for _, hv := range hvs {
				if full {
					fmt.Println(hv)
				} else {
					fmt.Println(hv.ID())
				}
			}
		},
	}
	cmdHVs.PersistentFlags().BoolVarP(&full, "full", "f", false, "give full hv output")

	cmdCreate := &cobra.Command{
		Use:   "create <spec>...",
		Short: "create new hypervisor(s)",
		Long: `Create a new hypervisor using "spec" as the initial values. "spec" must be
valid json and contain the required fields, "mac" and "ip".`,
		Run: func(cmd *cobra.Command, specs []string) {
			for _, spec := range specs {
				hv, err := createHV(client, spec)
				if err != nil {
					ret = 1
					return
				}
				if full {
					fmt.Println(hv)
				} else {
					fmt.Println(hv["id"])
				}
			}
		},
	}
	root.AddCommand(cmdHVs, cmdCreate)
	root.Execute()
	os.Exit(ret)
}

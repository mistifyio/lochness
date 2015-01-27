package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

const (
	nftSinglePort = "ip daddr %s %s dport %d %s"
	nftPortRange  = "ip daddr %s %s dport %d - %d %s"
)

var (
	tmpl *template.Template
	hv   *lochness.Hypervisor
)

type group struct {
	Name int
	ID   string
	IPs  []string
}

type templateData struct {
	IP      string
	Rules   []string
	Sources []group
}

func genRules(e *etcd.Client, c *lochness.Context) (templateData, error) {
	fwgroups := map[string]*lochness.FWGroup{}
	rules := []string{}
	// map of FWGroupID -> set name a.k.a g0,g1,g2
	groups := map[string]int{}
	max := len(groups)
	err := hv.ForEachGuest(func(guest *lochness.Guest) error {
		group, ok := fwgroups[guest.FWGroupID]
		if !ok {
			var err error
			group, err = c.FWGroup(guest.FWGroupID)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err,
					"func":  "context.FWGroup",
					"group": guest.FWGroupID,
				}).Error("failed to get firewall group")
				return err
			}
			fwgroups[guest.FWGroupID] = group
		}

		for _, rule := range group.Rules {
			source := ""
			if rule.Group != "" {
				i, ok := groups[rule.Group]
				if !ok {
					i = max
					max++
					groups[rule.Group] = i
				}
				source = " ip saddr @g" + strconv.Itoa(i)
			}
			if rule.Source != nil && rule.Source.String() != "" {
				source += " ip saddr " + rule.Source.String()
			}

			nftRule := ""
			if rule.PortStart == rule.PortEnd {
				nftRule = fmt.Sprintf(nftSinglePort,
					guest.IP,
					rule.Protocol,
					rule.PortEnd,
					source)
			} else if rule.PortStart < rule.PortEnd {
				nftRule = fmt.Sprintf(nftPortRange,
					guest.IP,
					rule.Protocol,
					rule.PortStart,
					rule.PortEnd,
					source)
			} else {
				log.WithFields(log.Fields{
					"start": rule.PortStart,
					"stop":  rule.PortEnd,
					"error": "invalid port range",
				}).Error("invalid port range specified")
				return errors.New("invalid port range specified")
			}
			rules = append(rules, nftRule)
		}
		return nil
	})
	if err != nil {
		return templateData{}, err
	}

	sort.Strings(rules)
	tData := templateData{
		IP:      hv.IP.String(),
		Rules:   rules,
		Sources: make([]group, len(groups)),
	}
	for id, i := range groups {
		tData.Sources[i] = group{Name: i, ID: id}
	}

	err = c.ForEachGuest(func(guest *lochness.Guest) error {
		i, ok := groups[guest.FWGroupID]
		if !ok {
			return nil
		}

		ips := tData.Sources[i].IPs
		ips = append(ips, guest.IP.String())
		tData.Sources[i].IPs = ips
		return nil
	})
	if err != nil {
		return templateData{}, err
	}

	return tData, nil
}

func applyRules(td templateData) error {
	temp, err := ioutil.TempFile("", "nft-")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "ioutil.TempFile",
		}).Error("failed to create temporary file")
		return err
	}
	defer os.Remove(temp.Name())

	err = tmpl.Execute(temp, td)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "template.Execute",
		}).Error("template returned an error")
		temp.Close()
		return err
	}
	temp.Close()

	// TODO: store rules file and do atomic-update/rollbacks?
	cmd := exec.Command("nft", "-f", temp.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "cmd.Run",
		}).Error("nft command returned an error")
	}
	return err
}

func watch(c *etcd.Client, prefix string, stop chan bool, ch chan struct{}) {
	single := strings.TrimRight(filepath.Base(prefix), "s")

	responses := make(chan *etcd.Response, 1)
	go func() {
		for r := range responses {
			log.WithFields(log.Fields{
				"type":   single,
				"node":   r.Node.Key,
				"action": r.Action,
			}).Info(prefix, " was updated")
			ch <- struct{}{}
		}
	}()

	_, err := c.Watch(prefix, 0, true, responses, stop)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Watch",
		}).Fatal("etcd watch returned an error")
	}
}

func main() {
	eaddr := flag.String("etcd", "http://localhost:4001", "etcd cluster address")
	flag.Parse()

	e := etcd.NewClient([]string{*eaddr})
	c := lochness.NewContext(e)

	var err error
	hn := os.Getenv("TEST_HOSTNAME")
	if hn == "" {
		hn, err = os.Hostname()
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Hostname",
		}).Fatal("failed to get hostname")
	} else {
		log.WithField("hostname", hn).Warn("environment is overriding hostname")
	}
	hv, err = c.Hypervisor(hn)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "context.Hypervisor",
		}).Fatal("failed to fetch hypervisor info")
	}

	tmpl = template.Must(template.New("nft").Parse(ruleset))

	stop := make(chan bool)
	ch := make(chan struct{})
	go watch(e, "/lochness/guests/", stop, ch)
	go watch(e, "/lochness/fwgroups/", stop, ch)

	go func() {
		// load rules at startup
		ch <- struct{}{}
	}()

	// TODO: batching?
	for range ch {
		td, err := genRules(e, c)
		if err != nil {
			continue
		}
		if err := applyRules(td); err != nil {
			continue
		}
	}
}

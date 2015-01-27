package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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

type templateData struct {
	IP      string
	Rules   []string
	Sources map[string][]string
}

func genRules(e *etcd.Client, c *lochness.Context) (templateData, error) {
	tData := templateData{
		IP:      hv.IP.String(),
		Rules:   nil,
		Sources: map[string][]string{},
	}

	fwgroups := map[string]*lochness.FWGroup{}
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
				source = " ip saddr @group_" + rule.Group
				if _, ok := tData.Sources[rule.Group]; !ok {
					tData.Sources[rule.Group] = nil
				}
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
			tData.Rules = append(tData.Rules, nftRule)
		}
		return nil
	})
	if err != nil {
		return templateData{}, err
	}

	err = c.ForEachGuest(func(guest *lochness.Guest) error {
		if s, ok := tData.Sources[guest.FWGroupID]; ok {
			s = append(s, guest.IP.String())
			tData.Sources[guest.FWGroupID] = s
		}
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

	hn, err := os.Hostname()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Hostname",
		}).Fatal("failed to get hostname")
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

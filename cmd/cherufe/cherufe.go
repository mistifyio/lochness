package main

//go:generate ego -package=main -o=nftables.ego.go

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
	flag "github.com/ogier/pflag"
)

const (
	nftSinglePort = "ip daddr %s %s dport %d %s"
	nftPortRange  = "ip daddr %s %s dport %d - %d %s"
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

func genRules(hv *lochness.Hypervisor, c *lochness.Context) (templateData, error) {
	if err := hv.Refresh(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "Hypervisor.Refresh",
			"id":    hv.ID,
		}).Error("could not refresh hypervisor")
		return templateData{}, err
	}

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

func applyRules(filename string, td templateData) error {
	dir := filepath.Dir(filename)
	prefix := filepath.Base(filename) + ".tmp"
	temp, err := ioutil.TempFile(dir, prefix)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "ioutil.TempFile",
		}).Error("failed to create temporary file")
		return err
	}

	err = nftWrite(temp, td.IP, td.Sources, td.Rules)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "nftWrite",
		}).Error("template returned an error")
		temp.Close()
		return err
	}
	temp.Close()

	return checkAndReload(filename, temp.Name())
}

func checkAndReload(permanet, temporary string) error {
	cmd := exec.Command("nft", "-f", temporary)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "cmd.Run",
		}).Error("nft does not like the generated rules file")
		return err
	}

	if err = os.Rename(temporary, permanet); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "os.Rename",
			"file":  temporary,
		}).Error("failed to overwrite nftables.conf")
		return err
	}

	cmd = exec.Command("systemctl", "reload", "nftables.service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "systemctl restart nftables.service",
		}).Error("systemctl returned and error")
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

func cleanStaleFiles(rulesfile string) {
	dir := filepath.Dir(rulesfile)
	prefix := filepath.Base(rulesfile) + ".tmp"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "ioutil.ReadDir",
		}).Fatal("failed to find stale temporary files")
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !strings.HasPrefix(file.Name(), prefix) {
			continue
		}

		name := filepath.Join(dir, file.Name())
		log.WithField("file", name).Info("removing stale file")
		if err := os.Remove(name); err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"file":  name,
			}).Error("failed to remove file")
		}
	}
}

func getHV(hn string, e *etcd.Client, c *lochness.Context) *lochness.Hypervisor {
	var err error
	hn, err = lochness.SetHypervisorID(hn)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "lochness.SetHypervisorID",
		}).Fatal("failed")
	}

	log.WithField("hypervisor_id", hn).Info("using id")

	hv, err := c.Hypervisor(hn)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "context.Hypervisor",
		}).Fatal("failed to fetch hypervisor info")
	}
	return hv
}

func canonicalizeRules(rules string) string {
	path, err := filepath.Abs(rules)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "filepath.Abs",
			"file":  rules,
		}).Fatal("failed to get absolute filename")
	}
	return path
}

func main() {
	eaddr := "http://localhost:4001"
	hn := ""
	rules := "/etc/nftables.conf"
	flag.StringVarP(&eaddr, "etcd", "e", eaddr, "etcd cluster address")
	flag.StringVarP(&hn, "id", "i", hn, "hypervisor id")
	flag.StringVarP(&rules, "file", "f", rules, "nft configuration file")
	flag.Parse()

	rules = canonicalizeRules(rules)
	cleanStaleFiles(rules)

	e := etcd.NewClient([]string{eaddr})
	c := lochness.NewContext(e)
	hv := getHV(hn, e, c)

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
		td, err := genRules(hv, c)
		if err != nil {
			continue
		}
		applyRules(rules, td)
	}
}

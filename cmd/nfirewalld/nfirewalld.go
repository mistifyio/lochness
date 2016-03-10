package main

//go:generate ego -package=main -o=nftables.go

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	ln "github.com/mistifyio/lochness"
	"github.com/mistifyio/lochness/pkg/kv"
	_ "github.com/mistifyio/lochness/pkg/kv/etcd"
	"github.com/mistifyio/lochness/pkg/watcher"
	flag "github.com/ogier/pflag"
)

const (
	nftSinglePort = "%s dport %d %s"
	nftPortRange  = "%s dport %d - %d %s"
)

type groupVal struct {
	num   int
	id    string
	ips   []string
	rules []string
}

type templateData struct {
	ip     string
	groups groupMap
	guests guestMap
}

type groupMap map[string]groupVal

// definitely not concurrent-safe
func (g groupMap) Index(id string) int {
	group, ok := g[id]
	if !ok {
		group.num = len(g)
		group.id = id
		g[id] = group
	}
	return group.num
}

type guestMap map[string]int

// genNFRules iterates through each FWRule and creates the nft rule line
func genNFRules(groups groupMap, fwrules ln.FWRules) []string {
	var nftrules []string
	for _, rule := range fwrules {
		source := ""
		if rule.Group != "" {
			source += "ip saddr @s" + strconv.Itoa(groups.Index(rule.Group))
		}
		if rule.Source != nil {
			if source != "" {
				source += " "
			}
			source += "ip saddr " + rule.Source.String()
		}

		nftRule := ""
		if rule.PortStart == rule.PortEnd {
			nftRule = fmt.Sprintf(nftSinglePort,
				rule.Protocol,
				rule.PortEnd,
				source)
		} else if rule.PortStart < rule.PortEnd {
			nftRule = fmt.Sprintf(nftPortRange,
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
			continue
		}
		nftrules = append(nftrules, nftRule)
	}
	return nftrules
}

func getGuestsFWGroups(c *ln.Context, hv *ln.Hypervisor) (groupMap, guestMap) {
	guests := guestMap{}
	groups := groupMap{}
	n := len(groups)

	_ = hv.ForEachGuest(func(guest *ln.Guest) error {
		// check if in cache
		g, ok := groups[guest.FWGroupID]
		if ok {
			// link the guest to the FWGroup, via the FWGroup's index
			guests[guest.IP.String()] = g.num
			return nil
		}

		// nope not cached
		fw, err := c.FWGroup(guest.FWGroupID)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"func":  "context.FWGroup",
				"group": guest.FWGroupID,
			}).Error("failed to get firewall group")
			return err
		}

		g = groupVal{
			num:   n,
			id:    fw.ID,
			rules: genNFRules(groups, fw.Rules),
		}
		n++
		groups[guest.FWGroupID] = g

		// link the guest to the FWGroup, via the FWGroup's index
		guests[guest.IP.String()] = g.num
		return nil
	})
	return groups, guests
}

func populateGroupMembers(c *ln.Context, groups groupMap) {
	_ = c.ForEachGuest(func(guest *ln.Guest) error {
		group, ok := groups[guest.FWGroupID]
		if !ok {
			// not a FWGroup referenced by any guest's FWGroup
			return nil
		}

		ips := group.ips
		ips = append(ips, guest.IP.String())
		group.ips = ips
		groups[guest.FWGroupID] = group
		return nil
	})
}

func genRules(hv *ln.Hypervisor, c *ln.Context) (templateData, error) {
	if err := hv.Refresh(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "Hypervisor.Refresh",
			"id":    hv.ID,
		}).Error("could not refresh hypervisor")
		return templateData{}, err
	}

	groups, guests := getGuestsFWGroups(c, hv)

	populateGroupMembers(c, groups)
	td := templateData{
		ip:     hv.IP.String(),
		groups: groups,
		guests: guests,
	}
	return td, nil
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

	err = nftWrite(temp, td.ip, td.groups, td.guests)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "nftWrite",
		}).Error("template returned an error")
		_ = temp.Close()
		return err
	}
	_ = temp.Close()

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

func getHV(hn string, c *ln.Context) *ln.Hypervisor {
	var err error
	hn, err = ln.SetHypervisorID(hn)
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
	kvAddr := "http://localhost:4001"
	hn := ""
	rules := "/etc/nftables.conf"
	flag.StringVarP(&kvAddr, "kv", "k", kvAddr, "kv cluster address")
	flag.StringVarP(&hn, "id", "i", hn, "hypervisor id")
	flag.StringVarP(&rules, "file", "f", rules, "nft configuration file")
	flag.Parse()

	rules = canonicalizeRules(rules)
	cleanStaleFiles(rules)

	KV, err := kv.New(kvAddr)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "kv.New",
		}).Fatal("failed to connect to kv")
	}

	c := ln.NewContext(KV)
	hv := getHV(hn, c)

	watcher, err := watcher.New(KV)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "watcher.New",
		}).Fatal("failed to start watcher")
	}

	if err = watcher.Add("/lochness/guests"); err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"func":   "watcher.Add",
			"prefix": "/lochness/guests",
		}).Fatal("failed to add prefix to watch list")
	}

	if err := watcher.Add("/lochness/fwgroups"); err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"func":   "watcher.Add",
			"prefix": "/lochness/fwgroups",
		}).Fatal("failed to add prefix to watch list")
	}

	// load rules at startup
	td, err := genRules(hv, c)
	if err != nil {
		log.WithField("error", err).Fatal("could not load intial rules")
	}
	if err := applyRules(rules, td); err != nil {
		log.WithField("error", err).Fatal("could not apply intial rules")
	}

	for watcher.Next() {
		td, err := genRules(hv, c)
		if err != nil {
			continue
		}
		if err := applyRules(rules, td); err != nil {
			log.WithField("error", err).Fatal("could not apply rules")
		}
	}
	if err := watcher.Err(); err != nil {
		log.Fatal(err)
	}
}

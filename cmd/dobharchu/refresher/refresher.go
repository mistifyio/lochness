package refresher

import (
	"io"
	"sort"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/mistifyio/lochness"
)

type (
	Refresher struct {
		Domain      string
		Context     *lochness.Context
		EtcdClient  *etcd.Client
		hypervisors *map[string]*lochness.Hypervisor
		subnets     *map[string]*lochness.Subnet
		guests      *map[string]*lochness.Guest
	}

	TemplateHelper struct {
		Domain      string
		Hypervisors []HypervisorHelper
		Guests      []GuestHelper
	}

	HypervisorHelper struct {
		ID      string
		MAC     string
		IP      string
		Gateway string
		Netmask string
	}

	GuestHelper struct {
		ID      string
		MAC     string
		IP      string
		Gateway string
		CIDR    string
	}
)

var HypervisorsTemplate = `
# Generated by Dobharchu

group hypervisors {
    option domain-name "nodes.{{.Domain}}";
    if exists user-class and option user-class = "iPXE" {
        filename "http://ipxe.services.{{.Domain}}:8888/ipxe/${net0/ip}";
    } else {
        next-server tftp.services.{{.Domain}};
        filename "undionly.kpxe";
    }
{{range $h := .Hypervisors}}
    host {{$h.ID}} {
        hardware ethernet "{{$h.MAC}}";
        fixed-address "{{$h.IP}}";
        option routers "{{$h.Gateway}}";
        option subnet-mask "{{$h.Netmask}}";
    }
{{end}}
}
`

var GuestsTemplate = `
# Generated by Dobharchu

group guests {
    option domain-name "guests.{{.Domain}}";
{{range $g := .Guests}}
    host {{$g.ID}} {
        hardware ethernet "{{$g.MAC}}";
        fixed-address "{{$g.IP}}";
        option routers "{{$g.Gateway}}";
        option subnet-mask "{{$g.CIDR}}";
    }
{{end}}
}
`

func NewRefresher(domain string, etcdAddress string) *Refresher {
	e := etcd.NewClient([]string{etcdAddress})
	c := lochness.NewContext(e)
	return &Refresher{
		Domain:      domain,
		Context:     c,
		EtcdClient:  e,
		hypervisors: nil,
		subnets:     nil,
		guests:      nil,
	}
}

func (r *Refresher) fetchHypervisors() (*map[string]*lochness.Hypervisor, error) {
	if r.hypervisors != nil {
		return r.hypervisors, nil
	}
	res, err := r.EtcdClient.Get("lochness/hypervisors/", true, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Error("Could not retrieve hypervisors from etcd")
		return nil, err
	}
	hypervisors := make(map[string]*lochness.Hypervisor)
	for _, node := range res.Node.Nodes {
		for _, hnode := range node.Nodes {
			if strings.Contains(hnode.Key, "metadata") {
				hv := r.Context.NewHypervisor()
				hv.UnmarshalJSON([]byte(hnode.Value))
				hypervisors[hv.ID] = hv
			}
		}
	}
	log.WithFields(log.Fields{
		"hypervisorCount": len(hypervisors),
	}).Debug("Fetched hypervisors metadata")
	r.hypervisors = &hypervisors
	return r.hypervisors, nil
}

func (r *Refresher) fetchGuests() (*map[string]*lochness.Guest, error) {
	if r.guests != nil {
		return r.guests, nil
	}
	res, err := r.EtcdClient.Get("lochness/guests/", true, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Error("Could not retrieve guests from etcd")
		return nil, err
	}
	guests := make(map[string]*lochness.Guest)
	for _, node := range res.Node.Nodes {
		for _, gnode := range node.Nodes {
			if strings.Contains(gnode.Key, "metadata") {
				g := r.Context.NewGuest()
				g.UnmarshalJSON([]byte(gnode.Value))
				guests[g.ID] = g
			}
		}
	}
	log.WithFields(log.Fields{
		"guestCount": len(guests),
	}).Debug("Fetched guests metadata")
	r.guests = &guests
	return r.guests, nil
}

func (r *Refresher) fetchSubnets() (*map[string]*lochness.Subnet, error) {
	res, err := r.EtcdClient.Get("lochness/subnets/", true, true)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Error("Could not retrieve subnets from etcd")
		return nil, err
	}
	subnets := make(map[string]*lochness.Subnet)
	for _, node := range res.Node.Nodes {
		for _, snode := range node.Nodes {
			if strings.Contains(snode.Key, "metadata") {
				s := r.Context.NewSubnet()
				s.UnmarshalJSON([]byte(snode.Value))
				subnets[s.ID] = s
			}
		}
	}
	log.WithFields(log.Fields{
		"subnetCount": len(subnets),
	}).Debug("Fetched subnets metadata")
	r.subnets = &subnets
	return r.subnets, nil
}

func (r *Refresher) WriteHypervisorsConfigFile(w io.Writer) error {
	vals := new(TemplateHelper)
	vals.Domain = r.Domain

	// Fetch and sort keys
	href, err := r.fetchHypervisors()
	if err != nil {
		return err
	}
	hypervisors := *href
	hkeys := make([]string, len(hypervisors))
	i := 0
	for id, _ := range hypervisors {
		hkeys[i] = id
		i++
	}
	sort.Strings(hkeys)

	// Loop through and build up the TemplateHelper
	for _, id := range hkeys {
		hv := hypervisors[id]
		vals.Hypervisors = append(vals.Hypervisors, HypervisorHelper{
			ID:      hv.ID,
			MAC:     strings.ToUpper(hv.MAC.String()),
			IP:      hv.IP.String(),
			Gateway: hv.Gateway.String(),
			Netmask: hv.Netmask.String(),
		})
	}

	// Execute template
	t := template.New("hypervisors.conf")
	t, err = t.Parse(HypervisorsTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "template.Parse",
		}).Error("Could not parse hypervisors.conf template")
		return err
	}
	t.Execute(w, vals)
	return nil
}

func (r *Refresher) WriteGuestsConfigFile(w io.Writer) error {
	vals := new(TemplateHelper)
	vals.Domain = r.Domain

	// Fetch and sort keys
	sref, err := r.fetchSubnets()
	subnets := *sref
	if err != nil {
		return err
	}
	gref, err := r.fetchGuests()
	if err != nil {
		return err
	}
	guests := *gref
	gkeys := make([]string, len(guests))
	i := 0
	for id, _ := range guests {
		gkeys[i] = id
		i++
	}
	sort.Strings(gkeys)

	// Loop through and build up the TemplateHelper
	for _, id := range gkeys {
		g := guests[id]
		if g.HypervisorID == "" || g.SubnetID == "" {
			continue
		}
		s, ok := subnets[g.SubnetID]
		if !ok {
			continue
		}
		vals.Guests = append(vals.Guests, GuestHelper{
			ID:      g.ID,
			MAC:     strings.ToUpper(g.MAC.String()),
			IP:      g.IP.String(),
			Gateway: s.Gateway.String(),
			CIDR:    s.CIDR.IP.String(),
		})
	}

	// Execute template
	t := template.New("guests.conf")
	t, err = t.Parse(GuestsTemplate)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"func":  "template.Parse",
		}).Error("Could not parse guests.conf template")
		return err
	}
	t.Execute(w, vals)
	return nil
}

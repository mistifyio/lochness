package main

import (
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/hashicorp/go-multierror"
	"github.com/mistifyio/lochness"
)

type (
	// Refresher grabs keys from etcd and writes out the dhcp configuration files
	// hypervisors.conf and guests.conf
	Refresher struct {
		Domain             string
		Context            *lochness.Context
		EtcdClient         *etcd.Client
		hypervisors        *map[string]*lochness.Hypervisor
		guests             *map[string]*lochness.Guest
		subnets            *map[string]*lochness.Subnet
		hypervisorsFetched time.Time
		guestsFetched      time.Time
		subnetsFetched     time.Time
	}

	// templateHelper is used for inserting values into the templates
	templateHelper struct {
		Domain      string
		Hypervisors []hypervisorHelper
		Guests      []guestHelper
	}

	// hypervisorHelper is used for inserting a hypervisor's values into the template
	hypervisorHelper struct {
		ID      string
		MAC     string
		IP      string
		Gateway string
		Netmask string
	}

	// guestHelper is used for inserting a guest's values into the template
	guestHelper struct {
		ID      string
		MAC     string
		IP      string
		Gateway string
		CIDR    string
	}
)

const fetchTimeout = 10 * time.Second

var hypervisorsTemplate = `
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

var guestsTemplate = `
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

var matchKeys = regexp.MustCompile("^/lochness/(hypervisors|subnets|guests)/([0-9a-f\\-]+)(/([^/]+))?(/.*)?")

// NewRefresher creates a new refresher
func NewRefresher(domain string, etcdAddress string) *Refresher {
	e := etcd.NewClient([]string{etcdAddress})
	c := lochness.NewContext(e)
	return &Refresher{
		Domain:     domain,
		Context:    c,
		EtcdClient: e,
	}
}

// Fetch pulls the hypervisors, guests, and subnets from etcd
func (r *Refresher) Fetch() error {
	var errs *multierror.Error
	err := r.fetchHypervisors()
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	err = r.fetchSubnets()
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	err = r.fetchGuests()
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	return errs.ErrorOrNil()
}

// fetchHypervisors pulls the hypervisors from etcd
func (r *Refresher) fetchHypervisors() error {
	hypervisors := make(map[string]*lochness.Hypervisor)
	res, err := r.EtcdClient.Get("lochness/hypervisors/", true, true)
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode == 100 {
			// key missing; log and set empty slice
			log.WithFields(log.Fields{
				"error": err,
				"func":  "etcd.Get",
			}).Warning("No hypervisors are stored in etcd")
			r.hypervisors = &hypervisors
			return nil
		}
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Error("Could not retrieve hypervisors from etcd")
		return err
	}
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
	}).Info("Fetched hypervisors metadata")
	r.hypervisors = &hypervisors
	r.hypervisorsFetched = time.Now()
	return nil
}

// fetchGuests pulls the guests from etcd
func (r *Refresher) fetchGuests() error {
	guests := make(map[string]*lochness.Guest)
	res, err := r.EtcdClient.Get("lochness/guests/", true, true)
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode == 100 {
			// key missing; log and set empty slice
			log.WithFields(log.Fields{
				"error": err,
				"func":  "etcd.Get",
			}).Warning("No guests are stored in etcd")
			r.guests = &guests
			return nil
		}
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Error("Could not retrieve guests from etcd")
		return err
	}
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
	}).Info("Fetched guests metadata")
	r.guests = &guests
	r.guestsFetched = time.Now()
	return nil
}

// fetchSubnets pulls the subnets from etcd
func (r *Refresher) fetchSubnets() error {
	subnets := make(map[string]*lochness.Subnet)
	res, err := r.EtcdClient.Get("lochness/subnets/", true, true)
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode == 100 {
			// key missing; log and set empty slice
			log.WithFields(log.Fields{
				"error": err,
				"func":  "etcd.Get",
			}).Warning("No subnets are stored in etcd")
			r.subnets = &subnets
			return nil
		}
		log.WithFields(log.Fields{
			"error": err,
			"func":  "etcd.Get",
		}).Error("Could not retrieve subnets from etcd")
		return err
	}
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
	}).Info("Fetched subnets metadata")
	r.subnets = &subnets
	r.subnetsFetched = time.Now()
	return nil
}

// GetHypervisors retrieves the stored hypervisors, or re-fetches them if the
// timeout window has passed
func (r *Refresher) GetHypervisors() (map[string]*lochness.Hypervisor, error) {
	if time.Now().Sub(r.hypervisorsFetched) > fetchTimeout {
		err := r.fetchHypervisors()
		if err != nil {
			return nil, err
		}
	}
	return *r.hypervisors, nil
}

// GetGuests retrieves the stored guests, or re-fetches them if the timeout
// window has passed
func (r *Refresher) GetGuests() (map[string]*lochness.Guest, error) {
	if time.Now().Sub(r.guestsFetched) > fetchTimeout {
		err := r.fetchGuests()
		if err != nil {
			return nil, err
		}
	}
	return *r.guests, nil
}

// GetSubnets retrieves the stored subnets, or re-fetches them if the
// timeout window has passed
func (r *Refresher) GetSubnets() (map[string]*lochness.Subnet, error) {
	if time.Now().Sub(r.subnetsFetched) > fetchTimeout {
		err := r.fetchSubnets()
		if err != nil {
			return nil, err
		}
	}
	return *r.subnets, nil
}

// IntegrateResponse takes an etcd reponse and updates our list of hypervisors, subnets, or guests
func (r *Refresher) IntegrateResponse(e *etcd.Response) error {
	matches := matchKeys.FindStringSubmatch(e.Node.Key)
	if len(matches) < 2 {
		err := errors.New("Caught response from etcd that did not match; re-fetching")
		log.WithFields(log.Fields{
			"key":    e.Node.Key,
			"action": e.Action,
			"regexp": matchKeys.String(),
		}).Warning(err.Error())
		err2 := r.Fetch()
		if err2 != nil {
			log.WithFields(log.Fields{
				"error": err2,
				"func":  "refresher.Fetch",
			}).Error("Could not re-fetch lists from etcd")
		}
		return err
	}
	element := matches[1]
	id := matches[2]
	vtype := matches[4]
	_ = r.integrationError("debug", "Response received", e, element, id, vtype)
	switch {
	case element == "hypervisors":
		return r.integrateHypervisorChange(e, element, id, vtype)
	case element == "guests":
		return r.integrateGuestChange(e, element, id, vtype)
	case element == "subnets":
		return r.integrateSubnetChange(e, element, id, vtype)
	}
	return nil
}

// integrationError logs a uniform message when a response is not integrated
// and returns the error
func (r *Refresher) integrationError(level string, message string, e *etcd.Response, element string, id string, vtype string) error {
	err := errors.New(message)
	fields := log.Fields{
		"action":  e.Action,
		"element": element,
		"id":      id,
		"vtype":   vtype,
		"key":     e.Node.Key,
	}
	if level == "debug" {
		log.WithFields(fields).Debug(err.Error())
	} else if level == "info" {
		log.WithFields(fields).Info(err.Error())
	} else if level == "warning" {
		log.WithFields(fields).Warning(err.Error())
	} else if level == "error" {
		log.WithFields(fields).Error(err.Error())
	}
	return err
}

// integrateHypervisorChange updates our hypervisors using an etcd response
func (r *Refresher) integrateHypervisorChange(e *etcd.Response, element string, id string, vtype string) error {
	hypervisors := *r.hypervisors
	switch {
	case e.Action == "create":
		if vtype != "metadata" {
			return r.integrationError("debug", "Create on something other than the main hypervisor; ignoring", e, element, id, vtype)
		}
		if _, ok := hypervisors[id]; ok {
			return r.integrationError("warning", "Caught response creating a hypervisor that already exists", e, element, id, vtype)
		}
		hv := r.Context.NewHypervisor()
		hv.UnmarshalJSON([]byte(e.Node.Value))
		hypervisors[id] = hv
		_ = r.integrationError("info", "Added hypervisor", e, element, id, vtype)

	case e.Action == "compareAndSwap":
		if vtype != "metadata" {
			return r.integrationError("debug", "Edit on something other than the main hypervisor; ignoring", e, element, id, vtype)
		}
		if _, ok := hypervisors[id]; !ok {
			return r.integrationError("warning", "Caught response editing a hypervisor that doesn't exist", e, element, id, vtype)
		}
		hv := r.Context.NewHypervisor()
		hv.UnmarshalJSON([]byte(e.Node.Value))
		hypervisors[id] = hv
		_ = r.integrationError("info", "Updated hypervisor", e, element, id, vtype)

	case e.Action == "delete":
		if vtype != "metadata" && vtype != "" {
			return r.integrationError("debug", "Delete on something other than the main hypervisor; ignoring", e, element, id, vtype)
		}
		if _, ok := hypervisors[id]; !ok {
			return r.integrationError("warning", "Caught response deleting a hypervisor that doesn't exist", e, element, id, vtype)
		}
		delete(hypervisors, id)
		_ = r.integrationError("info", "Deleted hypervisor", e, element, id, vtype)

	default:
		return r.integrationError("debug", "Action doesn't affect the config; ignoring", e, element, id, vtype)
	}
	r.hypervisors = &hypervisors
	return nil
}

// integrateGuestChange updates our guests using an etcd response
func (r *Refresher) integrateGuestChange(e *etcd.Response, element string, id string, vtype string) error {
	guests := *r.guests
	switch {
	case e.Action == "create":
		if vtype != "metadata" {
			return r.integrationError("debug", "Create on something other than the main guest; ignoring", e, element, id, vtype)
		}
		if _, ok := guests[id]; ok {
			return r.integrationError("warning", "Caught response creating a guest that already exists", e, element, id, vtype)
		}
		g := r.Context.NewGuest()
		g.UnmarshalJSON([]byte(e.Node.Value))
		guests[id] = g
		_ = r.integrationError("info", "Created guest", e, element, id, vtype)

	case e.Action == "compareAndSwap":
		if vtype != "metadata" {
			return r.integrationError("debug", "Edit on something other than the main guest; ignoring", e, element, id, vtype)
		}
		if _, ok := guests[id]; !ok {
			return r.integrationError("warning", "Caught response editing a guest that doesn't exist", e, element, id, vtype)
		}
		g := r.Context.NewGuest()
		g.UnmarshalJSON([]byte(e.Node.Value))
		guests[id] = g
		_ = r.integrationError("info", "Updated guest", e, element, id, vtype)

	case e.Action == "delete":
		if vtype != "metadata" && vtype != "" {
			return r.integrationError("debug", "Delete on something other than the main guest; ignoring", e, element, id, vtype)
		}
		if _, ok := guests[id]; !ok {
			return r.integrationError("warning", "Caught response deleting a guest that doesn't exist", e, element, id, vtype)
		}
		delete(guests, id)
		_ = r.integrationError("info", "Deleted guest", e, element, id, vtype)

	default:
		return r.integrationError("debug", "Action doesn't affect the config; ignoring", e, element, id, vtype)
	}
	r.guests = &guests
	return nil
}

// integrateSubnetChange updates our subnets using an etcd response
func (r *Refresher) integrateSubnetChange(e *etcd.Response, element string, id string, vtype string) error {
	subnets := *r.subnets
	switch {
	case e.Action == "create":
		if vtype != "metadata" {
			return r.integrationError("debug", "Create on something other than the main subnet; ignoring", e, element, id, vtype)
		}
		if _, ok := subnets[id]; ok {
			return r.integrationError("warning", "Caught response creating a subnet that already exists", e, element, id, vtype)
		}
		s := r.Context.NewSubnet()
		s.UnmarshalJSON([]byte(e.Node.Value))
		subnets[id] = s
		_ = r.integrationError("info", "Created subnet", e, element, id, vtype)

	case e.Action == "compareAndSwap":
		if vtype != "metadata" {
			return r.integrationError("debug", "Edit on something other than the main subnet; ignoring", e, element, id, vtype)
		}
		if _, ok := subnets[id]; !ok {
			return r.integrationError("warning", "Caught response editing a subnet that doesn't exist", e, element, id, vtype)
		}
		s := r.Context.NewSubnet()
		s.UnmarshalJSON([]byte(e.Node.Value))
		subnets[id] = s
		_ = r.integrationError("info", "Updated subnet", e, element, id, vtype)

	case e.Action == "delete":
		if vtype != "metadata" && vtype != "" {
			return r.integrationError("debug", "Delete on something other than the main subnet; ignoring", e, element, id, vtype)
		}
		if _, ok := subnets[id]; !ok {
			return r.integrationError("warning", "Caught response deleting a subnet that doesn't exist", e, element, id, vtype)
		}
		delete(subnets, id)
		_ = r.integrationError("info", "Deleted subnet", e, element, id, vtype)

	default:
		return r.integrationError("debug", "Action doesn't affect the config; ignoring", e, element, id, vtype)
	}
	r.subnets = &subnets
	return nil
}

// WriteHypervisorsConfigFile writes out the hypervisors config file using the given writer
func (r *Refresher) WriteHypervisorsConfigFile(w io.Writer) error {
	vals := new(templateHelper)
	vals.Domain = r.Domain
	hypervisors, err := r.GetHypervisors()
	if err != nil {
		return err
	}

	// Sort keys
	hkeys := make([]string, len(hypervisors))
	i := 0
	for id := range hypervisors {
		hkeys[i] = id
		i++
	}
	sort.Strings(hkeys)

	// Loop through and build up the templateHelper
	for _, id := range hkeys {
		hv := hypervisors[id]
		vals.Hypervisors = append(vals.Hypervisors, hypervisorHelper{
			ID:      hv.ID,
			MAC:     strings.ToUpper(hv.MAC.String()),
			IP:      hv.IP.String(),
			Gateway: hv.Gateway.String(),
			Netmask: hv.Netmask.String(),
		})
	}

	// Execute template
	t := template.New("hypervisors.conf")
	t, err = t.Parse(hypervisorsTemplate)
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

// WriteGuestsConfigFile writes out the guests config file using the given writer
func (r *Refresher) WriteGuestsConfigFile(w io.Writer) error {
	vals := new(templateHelper)
	vals.Domain = r.Domain
	guests, err := r.GetGuests()
	if err != nil {
		return err
	}
	subnets, err := r.GetSubnets()
	if err != nil {
		return err
	}

	// Sort guest keys
	gkeys := make([]string, len(guests))
	i := 0
	for id := range guests {
		gkeys[i] = id
		i++
	}
	sort.Strings(gkeys)

	// Loop through and build up the templateHelper
	for _, id := range gkeys {
		g := guests[id]
		if g.HypervisorID == "" || g.SubnetID == "" {
			continue
		}
		s, ok := subnets[g.SubnetID]
		if !ok {
			continue
		}
		vals.Guests = append(vals.Guests, guestHelper{
			ID:      g.ID,
			MAC:     strings.ToUpper(g.MAC.String()),
			IP:      g.IP.String(),
			Gateway: s.Gateway.String(),
			CIDR:    s.CIDR.IP.String(),
		})
	}

	// Execute template
	t := template.New("guests.conf")
	t, err = t.Parse(guestsTemplate)
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

package main

import (
	"errors"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/hashicorp/go-multierror"
	"github.com/mistifyio/lochness"
)

type (
	// Fetcher grabs keys from etcd and maintains lists of hypervisors, guests, and
	// subnets
	Fetcher struct {
		context            *lochness.Context
		etcdClient         *etcd.Client
		hypervisors        map[string]*lochness.Hypervisor
		guests             map[string]*lochness.Guest
		subnets            map[string]*lochness.Subnet
		hypervisorsFetched bool
		guestsFetched      bool
		subnetsFetched     bool
	}

	// ilogFields defines what needs to be passed to logIntegrationMessage()
	ilogFields struct {
		r *etcd.Response
		m string
		i string
		v string
		e error
		f string
	}
)

var matchKeys = regexp.MustCompile(`^/lochness/(hypervisors|subnets|guests)/([0-9a-f\-]+)(/([^/]+))?(/.*)?`)

// NewFetcher creates a new fetcher
func NewFetcher(etcdAddress string) *Fetcher {
	e := etcd.NewClient([]string{etcdAddress})
	c := lochness.NewContext(e)
	return &Fetcher{
		context:     c,
		etcdClient:  e,
		hypervisors: make(map[string]*lochness.Hypervisor),
		guests:      make(map[string]*lochness.Guest),
		subnets:     make(map[string]*lochness.Subnet),
	}
}

// FetchAll pulls the hypervisors, guests, and subnets from etcd
func (f *Fetcher) FetchAll() error {
	var errs *multierror.Error
	if err := f.fetchHypervisors(); err != nil {
		errs = multierror.Append(errs, err)
	}
	if err := f.fetchSubnets(); err != nil {
		errs = multierror.Append(errs, err)
	}
	if err := f.fetchGuests(); err != nil {
		errs = multierror.Append(errs, err)
	}
	return errs.ErrorOrNil()
}

// fetchHypervisors pulls the hypervisors from etcd
func (f *Fetcher) fetchHypervisors() error {
	f.hypervisors = make(map[string]*lochness.Hypervisor)
	err := f.context.ForEachHypervisor(func(hv *lochness.Hypervisor) error {
		f.hypervisors[hv.ID] = hv
		return nil
	})
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode == 100 {
			// key missing; log warning but return no error
			log.WithFields(log.Fields{
				"error": err,
				"func":  "context.ForEachHypervisor",
			}).Warning("No hypervisors are stored in etcd")
			return nil
		}
		log.WithFields(log.Fields{
			"error": err,
			"func":  "context.ForEachHypervisor",
		}).Error("Could not retrieve hypervisors from etcd")
		return err
	}
	log.WithFields(log.Fields{
		"hypervisorCount": len(f.hypervisors),
	}).Info("Fetched hypervisors metadata")
	f.hypervisorsFetched = true
	return nil
}

// fetchGuests pulls the guests from etcd
func (f *Fetcher) fetchGuests() error {
	f.guests = make(map[string]*lochness.Guest)
	err := f.context.ForEachGuest(func(g *lochness.Guest) error {
		f.guests[g.ID] = g
		return nil
	})
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode == 100 {
			// key missing; log warning but return no error
			log.WithFields(log.Fields{
				"error": err,
				"func":  "context.ForEachGuest",
			}).Warning("No guests are stored in etcd")
			return nil
		}
		log.WithFields(log.Fields{
			"error": err,
			"func":  "context.ForEachGuest",
		}).Error("Could not retrieve guests from etcd")
		return err
	}
	log.WithFields(log.Fields{
		"guestCount": len(f.guests),
	}).Info("Fetched guests metadata")
	f.guestsFetched = true
	return nil
}

// fetchSubnets pulls the subnets from etcd
func (f *Fetcher) fetchSubnets() error {
	f.subnets = make(map[string]*lochness.Subnet)
	res, err := f.etcdClient.Get("lochness/subnets/", true, true)
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode == 100 {
			// key missing; log warning but return no error
			log.WithFields(log.Fields{
				"error": err,
				"func":  "etcd.Get",
			}).Warning("No subnets are stored in etcd")
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
				s := f.context.NewSubnet()
				if err := s.UnmarshalJSON([]byte(snode.Value)); err != nil {
					log.WithFields(log.Fields{
						"error": err,
						"func":  "subnet.UnmarshalJSON",
					}).Error("Could not unmarshal subnet json")
				}
				f.subnets[s.ID] = s
			}
		}
	}
	log.WithFields(log.Fields{
		"subnetCount": len(f.subnets),
	}).Info("Fetched subnets metadata")
	f.subnetsFetched = true
	return nil
}

// GetHypervisors retrieves the stored hypervisors, or re-fetches them if the
// timeout window has passed
func (f *Fetcher) GetHypervisors() (map[string]*lochness.Hypervisor, error) {
	if !f.hypervisorsFetched {
		if err := f.fetchHypervisors(); err != nil {
			return nil, err
		}
	}
	return f.hypervisors, nil
}

// GetGuests retrieves the stored guests, or re-fetches them if the timeout
// window has passed
func (f *Fetcher) GetGuests() (map[string]*lochness.Guest, error) {
	if !f.guestsFetched {
		if err := f.fetchGuests(); err != nil {
			return nil, err
		}
	}
	return f.guests, nil
}

// GetSubnets retrieves the stored subnets, or re-fetches them if the
// timeout window has passed
func (f *Fetcher) GetSubnets() (map[string]*lochness.Subnet, error) {
	if !f.subnetsFetched {
		if err := f.fetchSubnets(); err != nil {
			return nil, err
		}
	}
	return f.subnets, nil
}

// IntegrateResponse takes an etcd reponse and updates our list of hypervisors, subnets, or guests
func (f *Fetcher) IntegrateResponse(r *etcd.Response) error {

	// Parse the key
	matches := matchKeys.FindStringSubmatch(r.Node.Key)
	if len(matches) < 2 {
		err := errors.New("Caught response from etcd that did not match; re-fetching")
		log.WithFields(log.Fields{
			"key":    r.Node.Key,
			"action": r.Action,
			"regexp": matchKeys.String(),
		}).Warning(err.Error())
		err2 := f.FetchAll()
		if err2 != nil {
			log.WithFields(log.Fields{
				"error": err2,
				"func":  "fetcher.FetchAll",
			}).Error("Could not re-fetch lists from etcd")
		}
		return err
	}
	element := matches[1]
	id := matches[2]
	vtype := matches[4]
	f.logIntegrationMessage("debug", "Response received", ilogFields{r: r, m: element, i: id, v: vtype})

	// Filter out actions we don't care about
	switch {
	case r.Action == "create":
		if vtype != "metadata" {
			msg := "Create on something other than the main element; ignoring"
			f.logIntegrationMessage("debug", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
	case r.Action == "compareAndSwap":
		if vtype != "metadata" {
			msg := "Edit on something other than the main element; ignoring"
			f.logIntegrationMessage("debug", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
	case r.Action == "delete":
		if vtype != "metadata" && vtype != "" {
			msg := "Delete on something other than the main element; ignoring"
			f.logIntegrationMessage("debug", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
	default:
		msg := "Action doesn't affect the config; ignoring"
		f.logIntegrationMessage("debug", msg, ilogFields{r: r, m: element, i: id, v: vtype})
		return errors.New(msg)
	}

	// Process each element
	switch {
	case element == "hypervisors":
		return f.integrateHypervisorChange(r, element, id, vtype)
	case element == "guests":
		return f.integrateGuestChange(r, element, id, vtype)
	case element == "subnets":
		return f.integrateSubnetChange(r, element, id, vtype)
	}
	return nil
}

// logIntegrationMessage logs a uniform message when a response is not
// integrated
func (f *Fetcher) logIntegrationMessage(level string, message string, fields ilogFields) {
	logfields := log.Fields{
		"action":  fields.r.Action,
		"element": fields.m,
		"id":      fields.i,
		"vtype":   fields.v,
		"key":     fields.r.Node.Key,
	}
	if fields.e != nil {
		logfields["error"] = fields.e
		logfields["func"] = fields.f
	}
	if level == "debug" {
		log.WithFields(logfields).Debug(message)
	} else if level == "info" {
		log.WithFields(logfields).Info(message)
	} else if level == "warning" {
		log.WithFields(logfields).Warning(message)
	} else if level == "error" {
		log.WithFields(logfields).Error(message)
	}
}

// integrateHypervisorChange updates our hypervisors using an etcd response
func (f *Fetcher) integrateHypervisorChange(r *etcd.Response, element string, id string, vtype string) error {
	switch {
	case r.Action == "create":
		if _, ok := f.hypervisors[id]; ok {
			msg := "Caught response creating an element that already exists"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		hv := f.context.NewHypervisor()
		if err := hv.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
			msg := "Could not unmarshal etcd response"
			f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype, e: err, f: "hypervisor.UnmarshalJSON"})
			return errors.New(msg)
		}
		f.hypervisors[id] = hv
		f.logIntegrationMessage("info", "Added hypervisor", ilogFields{r: r, m: element, i: id, v: vtype})
	case r.Action == "compareAndSwap":
		if _, ok := f.hypervisors[id]; !ok {
			msg := "Caught response editing an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		hv := f.context.NewHypervisor()
		if err := hv.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
			msg := "Could not unmarshal etcd response"
			f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype, e: err, f: "hypervisor.UnmarshalJSON"})
			return errors.New(msg)
		}
		f.hypervisors[id] = hv
		f.logIntegrationMessage("info", "Updated hypervisor", ilogFields{r: r, m: element, i: id, v: vtype})
	case r.Action == "delete":
		if _, ok := f.hypervisors[id]; !ok {
			msg := "Caught response deleting an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		delete(f.hypervisors, id)
		f.logIntegrationMessage("info", "Deleted hypervisor", ilogFields{r: r, m: element, i: id, v: vtype})
	}
	return nil
}

// integrateGuestChange updates our guests using an etcd response
func (f *Fetcher) integrateGuestChange(r *etcd.Response, element string, id string, vtype string) error {
	switch {
	case r.Action == "create":
		if _, ok := f.guests[id]; ok {
			msg := "Caught response creating an element that already exists"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		g := f.context.NewGuest()
		if err := g.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
			msg := "Could not unmarshal etcd response"
			f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype, e: err, f: "guest.UnmarshalJSON"})
			return errors.New(msg)
		}
		f.guests[id] = g
		f.logIntegrationMessage("info", "Created guest", ilogFields{r: r, m: element, i: id, v: vtype})
	case r.Action == "compareAndSwap":
		if _, ok := f.guests[id]; !ok {
			msg := "Caught response editing an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		g := f.context.NewGuest()
		if err := g.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
			msg := "Could not unmarshal etcd response"
			f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype, e: err, f: "guest.UnmarshalJSON"})
			return errors.New(msg)
		}
		f.guests[id] = g
		f.logIntegrationMessage("info", "Updated guest", ilogFields{r: r, m: element, i: id, v: vtype})
	case r.Action == "delete":
		if _, ok := f.guests[id]; !ok {
			msg := "Caught response deleting an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		delete(f.guests, id)
		f.logIntegrationMessage("info", "Deleted guest", ilogFields{r: r, m: element, i: id, v: vtype})
	}
	return nil
}

// integrateSubnetChange updates our subnets using an etcd response
func (f *Fetcher) integrateSubnetChange(r *etcd.Response, element string, id string, vtype string) error {
	switch {
	case r.Action == "create":
		if _, ok := f.subnets[id]; ok {
			msg := "Caught response creating an element that already exists"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		s := f.context.NewSubnet()
		if err := s.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
			msg := "Could not unmarshal etcd response"
			f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype, e: err, f: "subnet.UnmarshalJSON"})
			return errors.New(msg)
		}
		f.subnets[id] = s
		f.logIntegrationMessage("info", "Created subnet", ilogFields{r: r, m: element, i: id, v: vtype})
	case r.Action == "compareAndSwap":
		if _, ok := f.subnets[id]; !ok {
			msg := "Caught response editing an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		s := f.context.NewSubnet()
		if err := s.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
			msg := "Could not unmarshal etcd response"
			f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype, e: err, f: "subnet.UnmarshalJSON"})
			return errors.New(msg)
		}
		f.subnets[id] = s
		f.logIntegrationMessage("info", "Updated subnet", ilogFields{r: r, m: element, i: id, v: vtype})
	case r.Action == "delete":
		if _, ok := f.subnets[id]; !ok {
			msg := "Caught response deleting an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilogFields{r: r, m: element, i: id, v: vtype})
			return errors.New(msg)
		}
		delete(f.subnets, id)
		f.logIntegrationMessage("info", "Deleted subnet", ilogFields{r: r, m: element, i: id, v: vtype})
	}
	return nil
}

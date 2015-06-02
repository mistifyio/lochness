package main

import (
	"errors"
	"regexp"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/hashicorp/go-multierror"
	"github.com/mistifyio/lochness"
)

type (
	// Fetcher grabs keys from etcd and maintains lists of hypervisors, guests, and
	// subnets
	Fetcher struct {
		context     *lochness.Context
		etcdClient  *etcd.Client
		hypervisors map[string]*lochness.Hypervisor
		guests      map[string]*lochness.Guest
		subnets     map[string]*lochness.Subnet
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
		context:    c,
		etcdClient: e,
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
		if _, ok := err.(*etcd.EtcdError); ok && err.(*etcd.EtcdError).ErrorCode == 100 {
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
	return nil
}

// fetchSubnets pulls the subnets from etcd
func (f *Fetcher) fetchSubnets() error {
	f.subnets = make(map[string]*lochness.Subnet)
	err := f.context.ForEachSubnet(func(s *lochness.Subnet) error {
		f.subnets[s.ID] = s
		return nil
	})
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
	log.WithFields(log.Fields{
		"subnetCount": len(f.subnets),
	}).Info("Fetched subnets metadata")
	return nil
}

// Hypervisors retrieves the stored hypervisors, or fetches them if they
// aren't stored yet
func (f *Fetcher) Hypervisors() (map[string]*lochness.Hypervisor, error) {
	if f.hypervisors == nil {
		if err := f.fetchHypervisors(); err != nil {
			return nil, err
		}
	}
	return f.hypervisors, nil
}

// Guests retrieves the stored guests, or fetches them if they aren't stored
// yet
func (f *Fetcher) Guests() (map[string]*lochness.Guest, error) {
	if f.guests == nil {
		if err := f.fetchGuests(); err != nil {
			return nil, err
		}
	}
	return f.guests, nil
}

// Subnets retrieves the stored subnets, or fetches them if they aren't
// stored yet
func (f *Fetcher) Subnets() (map[string]*lochness.Subnet, error) {
	if f.subnets == nil {
		if err := f.fetchSubnets(); err != nil {
			return nil, err
		}
	}
	return f.subnets, nil
}

// IntegrateResponse takes an etcd reponse and updates our list of hypervisors,
// subnets, or guests, then returns whether a refresh should happen
func (f *Fetcher) IntegrateResponse(r *etcd.Response) (bool, error) {

	// Parse the key
	matches := matchKeys.FindStringSubmatch(r.Node.Key)
	if len(matches) < 2 {
		msg := "Caught response from etcd that did not match"
		log.WithFields(log.Fields{
			"key":    r.Node.Key,
			"action": r.Action,
			"regexp": matchKeys.String(),
		}).Warning(msg)
		return false, errors.New(msg)
	}
	element := matches[1]
	id := matches[2]
	vtype := matches[4]
	f.logIntegrationMessage("debug", "Response received", ilogFields{r: r, m: element, i: id, v: vtype})

	// Error out if we haven't fetched the element in question yet
	if (element == "hypervisors" && f.hypervisors == nil) || (element == "guests" && f.guests == nil) || (element == "subnets" && f.subnets == nil) {
		msg := "Cannot integrate elements when no initial fetch has occurred"
		f.logIntegrationMessage("error", msg, ilogFields{r: r, m: element, i: id, v: vtype})
		return false, errors.New(msg)
	}

	// Filter out actions we don't care about
	switch r.Action {
	case "create", "compareAndSwap", "set", "delete":
		if vtype != "metadata" {
			f.logIntegrationMessage("debug", "Action on something other than the main element; ignoring", ilogFields{r: r, m: element, i: id, v: vtype})
			return false, nil
		}
	default:
		f.logIntegrationMessage("debug", "Action doesn't affect the config; ignoring", ilogFields{r: r, m: element, i: id, v: vtype})
		return false, nil
	}

	// Process each element
	var err error
	switch element {
	case "hypervisors":
		err = f.integrateHypervisorChange(r, element, id, vtype)
	case "guests":
		err = f.integrateGuestChange(r, element, id, vtype)
	case "subnets":
		err = f.integrateSubnetChange(r, element, id, vtype)
	default:
		f.logIntegrationMessage("debug", "Unknown element; ignoring", ilogFields{r: r, m: element, i: id, v: vtype})
		return false, nil
	}
	rewrite := true
	if err != nil {
		rewrite = false
	}
	return rewrite, err
}

// logIntegrationMessage logs a uniform message during integration
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

// integrateHypervisorChange updates our hypervisors using an etcd response,
// then returns whether a refresh should happen
func (f *Fetcher) integrateHypervisorChange(r *etcd.Response, element string, id string, vtype string) error {
	ilf := ilogFields{r: r, m: element, i: id, v: vtype}

	// Sanity check
	if _, ok := f.hypervisors[id]; ok {
		if r.Action == "create" {
			msg := "Caught response creating an element that already exists"
			f.logIntegrationMessage("warning", msg, ilf)
			return errors.New(msg)
		}
	} else {
		if r.Action != "create" {
			msg := "Caught response operating on an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilf)
			return errors.New(msg)
		}
	}

	// Delete
	if r.Action == "delete" {
		delete(f.hypervisors, id)
		f.logIntegrationMessage("info", "Deleted hypervisor", ilf)
		return nil
	}

	// Add/update
	hv := f.context.NewHypervisor()
	if err := hv.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
		ilf.e = err
		ilf.f = "hypervisor.UnmarshalJSON"
		f.logIntegrationMessage("error", "Could not unmarshal etcd response", ilf)
		return err
	}
	f.hypervisors[id] = hv
	f.logIntegrationMessage("info", "Integrated hypervisor", ilf)

	return nil
}

// integrateGuestChange updates our guests using an etcd response
func (f *Fetcher) integrateGuestChange(r *etcd.Response, element string, id string, vtype string) error {
	ilf := ilogFields{r: r, m: element, i: id, v: vtype}

	// Sanity check
	if _, ok := f.guests[id]; ok {
		if r.Action == "create" {
			msg := "Caught response creating an element that already exists"
			f.logIntegrationMessage("warning", msg, ilf)
			return errors.New(msg)
		}
	} else {
		if r.Action != "create" {
			msg := "Caught response operating on an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilf)
			return errors.New(msg)
		}
	}

	// Delete
	if r.Action == "delete" {
		delete(f.guests, id)
		f.logIntegrationMessage("info", "Deleted guest", ilf)
		return nil
	}

	// Add/update
	g := f.context.NewGuest()
	if err := g.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
		ilf.e = err
		ilf.f = "guest.UnmarshalJSON"
		f.logIntegrationMessage("error", "Could not unmarshal etcd response", ilf)
		return err
	}
	f.guests[id] = g
	f.logIntegrationMessage("info", "Integrated guest", ilogFields{r: r, m: element, i: id, v: vtype})

	return nil
}

// integrateSubnetChange updates our subnets using an etcd response
func (f *Fetcher) integrateSubnetChange(r *etcd.Response, element string, id string, vtype string) error {
	ilf := ilogFields{r: r, m: element, i: id, v: vtype}

	// Sanity check
	if _, ok := f.subnets[id]; ok {
		if r.Action == "create" {
			msg := "Caught response creating an element that already exists"
			f.logIntegrationMessage("warning", msg, ilf)
			return errors.New(msg)
		}
	} else {
		if r.Action != "create" {
			msg := "Caught response operating on an element that doesn't exist"
			f.logIntegrationMessage("warning", msg, ilf)
			return errors.New(msg)
		}
	}

	// Delete
	if r.Action == "delete" {
		delete(f.subnets, id)
		f.logIntegrationMessage("info", "Deleted subnet", ilf)
		return nil
	}

	// Add/update
	s := f.context.NewSubnet()
	if err := s.UnmarshalJSON([]byte(r.Node.Value)); err != nil {
		ilf.e = err
		ilf.f = "subnet.UnmarshalJSON"
		f.logIntegrationMessage("error", "Could not unmarshal etcd response", ilf)
		return err
	}
	f.subnets[id] = s
	f.logIntegrationMessage("info", "Integrated subnet", ilogFields{r: r, m: element, i: id, v: vtype})

	return nil
}

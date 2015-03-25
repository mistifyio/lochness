package main

import (
	"errors"
	"regexp"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-etcd/etcd"
	"github.com/hashicorp/go-multierror"
	"github.com/mistifyio/lochness"
)

type (
	// Fetcher grabs keys from etcd and maintains lists of hypervisors, guests, and
	// subnets
	Fetcher struct {
		Context            *lochness.Context
		EtcdClient         *etcd.Client
		hypervisors        map[string]*lochness.Hypervisor
		guests             map[string]*lochness.Guest
		subnets            map[string]*lochness.Subnet
		hypervisorsFetched time.Time
		guestsFetched      time.Time
		subnetsFetched     time.Time
	}
)

const fetchTimeout = 10 * time.Second

var matchKeys = regexp.MustCompile(`^/lochness/(hypervisors|subnets|guests)/([0-9a-f\-]+)(/([^/]+))?(/.*)?`)

// NewFetcher creates a new fetcher
func NewFetcher(etcdAddress string) *Fetcher {
	e := etcd.NewClient([]string{etcdAddress})
	c := lochness.NewContext(e)
	return &Fetcher{
		Context:     c,
		EtcdClient:  e,
		hypervisors: make(map[string]*lochness.Hypervisor),
		guests:      make(map[string]*lochness.Guest),
		subnets:     make(map[string]*lochness.Subnet),
	}
}

// FetchAll pulls the hypervisors, guests, and subnets from etcd
func (f *Fetcher) FetchAll() error {
	var errs *multierror.Error
	err := f.fetchHypervisors()
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	err = f.fetchSubnets()
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	err = f.fetchGuests()
	if err != nil {
		errs = multierror.Append(errs, err)
	}
	return errs.ErrorOrNil()
}

// fetchHypervisors pulls the hypervisors from etcd
func (f *Fetcher) fetchHypervisors() error {
	f.hypervisors = make(map[string]*lochness.Hypervisor)
	err := f.Context.ForEachHypervisor(func(hv *lochness.Hypervisor) error {
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
	f.hypervisorsFetched = time.Now()
	return nil
}

// fetchGuests pulls the guests from etcd
func (f *Fetcher) fetchGuests() error {
	f.guests = make(map[string]*lochness.Guest)
	err := f.Context.ForEachGuest(func(g *lochness.Guest) error {
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
	f.guestsFetched = time.Now()
	return nil
}

// fetchSubnets pulls the subnets from etcd
func (f *Fetcher) fetchSubnets() error {
	f.subnets = make(map[string]*lochness.Subnet)
	res, err := f.EtcdClient.Get("lochness/subnets/", true, true)
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
				s := f.Context.NewSubnet()
				s.UnmarshalJSON([]byte(snode.Value))
				f.subnets[s.ID] = s
			}
		}
	}
	log.WithFields(log.Fields{
		"subnetCount": len(f.subnets),
	}).Info("Fetched subnets metadata")
	f.subnetsFetched = time.Now()
	return nil
}

// GetHypervisors retrieves the stored hypervisors, or re-fetches them if the
// timeout window has passed
func (f *Fetcher) GetHypervisors() (map[string]*lochness.Hypervisor, error) {
	if time.Now().Sub(f.hypervisorsFetched) > fetchTimeout {
		err := f.fetchHypervisors()
		if err != nil {
			return nil, err
		}
	}
	return f.hypervisors, nil
}

// GetGuests retrieves the stored guests, or re-fetches them if the timeout
// window has passed
func (f *Fetcher) GetGuests() (map[string]*lochness.Guest, error) {
	if time.Now().Sub(f.guestsFetched) > fetchTimeout {
		err := f.fetchGuests()
		if err != nil {
			return nil, err
		}
	}
	return f.guests, nil
}

// GetSubnets retrieves the stored subnets, or re-fetches them if the
// timeout window has passed
func (f *Fetcher) GetSubnets() (map[string]*lochness.Subnet, error) {
	if time.Now().Sub(f.subnetsFetched) > fetchTimeout {
		err := f.fetchSubnets()
		if err != nil {
			return nil, err
		}
	}
	return f.subnets, nil
}

// IntegrateResponse takes an etcd reponse and updates our list of hypervisors, subnets, or guests
func (f *Fetcher) IntegrateResponse(e *etcd.Response) error {

	// Parse the key
	matches := matchKeys.FindStringSubmatch(e.Node.Key)
	if len(matches) < 2 {
		err := errors.New("Caught response from etcd that did not match; re-fetching")
		log.WithFields(log.Fields{
			"key":    e.Node.Key,
			"action": e.Action,
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
	_ = f.logIntegrationMessage("debug", "Response received", e, element, id, vtype)

	// Filter out actions we don't care about
	switch {
	case e.Action == "create":
		if vtype != "metadata" {
			return f.logIntegrationMessage("debug", "Create on something other than the main "+element+"; ignoring", e, element, id, vtype)
		}
	case e.Action == "compareAndSwap":
		if vtype != "metadata" {
			return f.logIntegrationMessage("debug", "Edit on something other than the main "+element+"; ignoring", e, element, id, vtype)
		}
	case e.Action == "delete":
		if vtype != "metadata" && vtype != "" {
			return f.logIntegrationMessage("debug", "Delete on something other than the main "+element+"; ignoring", e, element, id, vtype)
		}
	default:
		return f.logIntegrationMessage("debug", "Action doesn't affect the config; ignoring", e, element, id, vtype)
	}

	// Process each element
	switch {
	case element == "hypervisors":
		return f.integrateHypervisorChange(e, element, id, vtype)
	case element == "guests":
		return f.integrateGuestChange(e, element, id, vtype)
	case element == "subnets":
		return f.integrateSubnetChange(e, element, id, vtype)
	}
	return nil
}

// logIntegrationMessage logs a uniform message when a response is not
// integrated and returns the error
func (f *Fetcher) logIntegrationMessage(level string, message string, e *etcd.Response, element string, id string, vtype string) error {
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
func (f *Fetcher) integrateHypervisorChange(e *etcd.Response, element string, id string, vtype string) error {
	switch {
	case e.Action == "create":
		if _, ok := f.hypervisors[id]; ok {
			return f.logIntegrationMessage("warning", "Caught response creating a hypervisor that already exists", e, element, id, vtype)
		}
		hv := f.Context.NewHypervisor()
		hv.UnmarshalJSON([]byte(e.Node.Value))
		f.hypervisors[id] = hv
		_ = f.logIntegrationMessage("info", "Added hypervisor", e, element, id, vtype)
	case e.Action == "compareAndSwap":
		if _, ok := f.hypervisors[id]; !ok {
			return f.logIntegrationMessage("warning", "Caught response editing a hypervisor that doesn't exist", e, element, id, vtype)
		}
		hv := f.Context.NewHypervisor()
		hv.UnmarshalJSON([]byte(e.Node.Value))
		f.hypervisors[id] = hv
		_ = f.logIntegrationMessage("info", "Updated hypervisor", e, element, id, vtype)
	case e.Action == "delete":
		if _, ok := f.hypervisors[id]; !ok {
			return f.logIntegrationMessage("warning", "Caught response deleting a hypervisor that doesn't exist", e, element, id, vtype)
		}
		delete(f.hypervisors, id)
		_ = f.logIntegrationMessage("info", "Deleted hypervisor", e, element, id, vtype)
	}
	return nil
}

// integrateGuestChange updates our guests using an etcd response
func (f *Fetcher) integrateGuestChange(e *etcd.Response, element string, id string, vtype string) error {
	switch {
	case e.Action == "create":
		if _, ok := f.guests[id]; ok {
			return f.logIntegrationMessage("warning", "Caught response creating a guest that already exists", e, element, id, vtype)
		}
		g := f.Context.NewGuest()
		g.UnmarshalJSON([]byte(e.Node.Value))
		f.guests[id] = g
		_ = f.logIntegrationMessage("info", "Created guest", e, element, id, vtype)
	case e.Action == "compareAndSwap":
		if _, ok := f.guests[id]; !ok {
			return f.logIntegrationMessage("warning", "Caught response editing a guest that doesn't exist", e, element, id, vtype)
		}
		g := f.Context.NewGuest()
		g.UnmarshalJSON([]byte(e.Node.Value))
		f.guests[id] = g
		_ = f.logIntegrationMessage("info", "Updated guest", e, element, id, vtype)
	case e.Action == "delete":
		if _, ok := f.guests[id]; !ok {
			return f.logIntegrationMessage("warning", "Caught response deleting a guest that doesn't exist", e, element, id, vtype)
		}
		delete(f.guests, id)
		_ = f.logIntegrationMessage("info", "Deleted guest", e, element, id, vtype)
	}
	return nil
}

// integrateSubnetChange updates our subnets using an etcd response
func (f *Fetcher) integrateSubnetChange(e *etcd.Response, element string, id string, vtype string) error {
	switch {
	case e.Action == "create":
		if _, ok := f.subnets[id]; ok {
			return f.logIntegrationMessage("warning", "Caught response creating a subnet that already exists", e, element, id, vtype)
		}
		s := f.Context.NewSubnet()
		s.UnmarshalJSON([]byte(e.Node.Value))
		f.subnets[id] = s
		_ = f.logIntegrationMessage("info", "Created subnet", e, element, id, vtype)
	case e.Action == "compareAndSwap":
		if _, ok := f.subnets[id]; !ok {
			return f.logIntegrationMessage("warning", "Caught response editing a subnet that doesn't exist", e, element, id, vtype)
		}
		s := f.Context.NewSubnet()
		s.UnmarshalJSON([]byte(e.Node.Value))
		f.subnets[id] = s
		_ = f.logIntegrationMessage("info", "Updated subnet", e, element, id, vtype)
	case e.Action == "delete":
		if _, ok := f.subnets[id]; !ok {
			return f.logIntegrationMessage("warning", "Caught response deleting a subnet that doesn't exist", e, element, id, vtype)
		}
		delete(f.subnets, id)
		_ = f.logIntegrationMessage("info", "Deleted subnet", e, element, id, vtype)
	}
	return nil
}

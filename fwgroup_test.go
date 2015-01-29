package lochness_test

import (
	"encoding/json"
	"strings"
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
)

func TestFWGroupsAlias(t *testing.T) {
	_ = lochness.FWGroups([]*lochness.FWGroup{})
}

func TestFWRulesAlias(t *testing.T) {
	_ = lochness.FWRules([]*lochness.FWRule{})
}

func TestFWGroupJson(t *testing.T) {
	data := `{"id": "EF8D7367-F14F-49C9-B960-2625947CA929", "rules": [ {"source": "192.168.1.0/24", "portStart": 80, "portEnd": 80, "protocol": "tcp", "action": "allow"} ] }`

	f := lochness.FWGroup{}
	err := json.Unmarshal([]byte(data), &f)
	h.Ok(t, err)
	h.Equals(t, "EF8D7367-F14F-49C9-B960-2625947CA929", f.ID)
	h.Equals(t, 1, len(f.Rules))
	h.Equals(t, "192.168.1.0", f.Rules[0].Source.IP.String())

	b, err := json.Marshal(&f)
	h.Ok(t, err)

	h.Assert(t, strings.Contains(string(b), "192.168.1.0/24"), "incorrect source information")
}

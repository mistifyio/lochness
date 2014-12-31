package lochness_test

import (
	"testing"

	"github.com/mistifyio/lochness"
)

func TestFWGroupsAlias(t *testing.T) {
	_ = lochness.FWGroups([]*lochness.FWGroup{})
}

func TestFWRulesAlias(t *testing.T) {
	_ = lochness.FWRules([]*lochness.FWRule{})
}

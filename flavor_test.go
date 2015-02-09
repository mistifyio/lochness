package lochness_test

import (
	"strings"
	"testing"

	h "github.com/bakins/test-helpers"
	"github.com/mistifyio/lochness"
)

func TestFlavorsAlias(t *testing.T) {
	_ = lochness.Flavors([]*lochness.Flavor{})
}

func TestFlavorWithBadID(t *testing.T) {
	c := newContext(t)
	_, err := c.Flavor("")
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "invalid UUID"), "unexpected error")

	_, err = c.Flavor("foo")
	h.Assert(t, err != nil, "should have got an error")
	h.Assert(t, strings.Contains(err.Error(), "invalid UUID"), "unexpected error")

}

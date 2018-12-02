package bupserver

import (
	"github.com/function61/gokit/assert"
	"strings"
	"testing"
)

func dump(ss []string) string {
	return strings.Join(ss, ",")
}

// shorthand for making string slice
func ss(items ...string) []string {
	return items
}

func TestFilter(t *testing.T) {
	notJoonas := func(item string) bool {
		return item != "Joonas"
	}

	allPeople := ss("Foo", "Joonas", "Bar")
	stupidPeople := filter(allPeople, notJoonas)

	assert.EqualString(t, dump(stupidPeople), "Foo,Bar")
}

func TestMissingFromLeftHandSide(t *testing.T) {

	one := func(lhs []string, rhs []string) string {
		return dump(missingFromLeftHandSide(lhs, rhs))
	}

	assert.EqualString(t, one(ss("vol1"), ss("vol1", "vol2")), "vol2")
	assert.EqualString(t, one(ss(), ss("vol1", "vol2")), "vol1,vol2")
	assert.EqualString(t, one(ss("vol1", "vol2", "vol3"), ss("vol1", "vol2")), "")
}

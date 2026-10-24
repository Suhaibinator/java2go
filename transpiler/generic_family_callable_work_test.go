package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// Generic method signatures examine every candidate leaf owner. Structure may
// be collected once while each target's eligibility remains a fresh query.
func TestGenericFamilyCallableTraversalWork(t *testing.T) {
	var source strings.Builder
	for index := 0; index < 16; index++ {
		fmt.Fprintf(&source, "class Cell%d<T>{}\n", index)
	}
	source.WriteString("class Factory{")
	for index := 0; index < 16; index++ {
		fmt.Fprintf(&source, "<T> Cell%d<T> make%d(){return null;}\n", index, index)
	}
	source.WriteString("} class Padding{void noise(){int value=0;")
	for index := 0; index < 1024; index++ {
		source.WriteString("value++;")
	}
	source.WriteString("}}")
	helper := setupParseHelper(t, source.String())
	for sample := 0; sample < 3; sample++ {
		if plans := discoverCanonicalGenericFamilies(helper.Ctx); len(plans) != 0 {
			t.Fatalf("leaf layout admission changed: families=%d", len(plans))
		}
	}
}

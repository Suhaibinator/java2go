package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// Runtime-owned reference names are legal Java and cannot create source family
// seeds. Their inherited lookup work must not dominate a source-demand scan.
func TestGenericFamilyImpossibleSourceDemandWork(t *testing.T) {
	var source strings.Builder
	source.WriteString("class Base{} class Child extends Base{")
	for index := 0; index < 256; index++ {
		fmt.Fprintf(&source, "String text%d; Object object%d;", index, index)
	}
	source.WriteString("} class Cell<T>{} class Use{Cell<?> demand;}")
	helper := setupParseHelper(t, source.String())
	for sample := 0; sample < 3; sample++ {
		plans := discoverCanonicalGenericFamilies(helper.Ctx)
		if len(plans) != 1 || len(plans[0].members) != 1 {
			t.Fatalf("source demand changed: families=%d", len(plans))
		}
		if _, ok := plans[0].members[helper.File.Symbols.FindClassScope("Cell")]; !ok {
			t.Fatal("source family lost")
		}
	}
}

package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// Family-specific decisions consume source events. Traversing unrelated method
// bodies once per admitted seed must not multiply the complete AST inventory.
func TestGenericFamilySourceEvents_WholeASTTraversalIsPerDiscovery(t *testing.T) {
	var source strings.Builder
	for index := 0; index < 64; index++ {
		fmt.Fprintf(&source, "class Cell%d<T>{}\n", index)
	}
	source.WriteString("class Demands{\n")
	for index := 0; index < 64; index++ {
		fmt.Fprintf(&source, "Cell%d<?> value%d;\n", index, index)
	}
	source.WriteString("}class Padding{void noise(){int value=0;\n")
	for index := 0; index < 4096; index++ {
		source.WriteString("value++;\n")
	}
	source.WriteString("}}")
	helper := setupParseHelper(t, source.String())
	t.Run("admitted-families", func(t *testing.T) {
		seeds := sourceGenericViewDemandSeeds(helper.Ctx)
		if len(seeds) != 64 {
			t.Fatalf("seeds=%d,want64", len(seeds))
		}
		plans := discoverCanonicalGenericFamilies(helper.Ctx)
		if len(plans) != 64 {
			t.Fatalf("admitted families=%d,want64", len(plans))
		}
		for _, plan := range plans {
			if len(plan.members) != 1 || len(plan.binders) != 1 || len(plan.locals) != 0 || len(plan.anonymous) != 0 {
				t.Fatalf("isolatedfamily inventory changed: %#v", plan)
			}
		}
	})
	t.Run("inventory-budget", func(t *testing.T) {
		inventory := testing.AllocsPerRun(1, func() {
			sourceGenericViewDemandSeeds(helper.Ctx)
			genericFamilyLocalInterfaceEdges(allSourceClassScopes(), helper.Ctx)
		})
		discovery := testing.AllocsPerRun(1, func() {
			if plans := discoverCanonicalGenericFamilies(helper.Ctx); len(plans) != 64 {
				t.Fatal("family admission changed")
			}
		})
		t.Logf("same admitted64family graph: single inventory allocations=%.0f discovery allocations=%.0f ratio=%.2f", inventory, discovery, discovery/inventory)
		if discovery > 6*inventory {
			t.Fatalf("GENERIC_FAMILY_REPEATED_SOURCE_EVENT_SCAN: %.0f allocations >6*single inventory %.0f", discovery, inventory)
		}
	})
	t.Run("local-method-binder-and-anonymous", func(t *testing.T) {
		local := setupParseHelper(t, `interface Left<T>{T read();}interface Right<T>{T read();}class Use{<X> void make(){class Local implements Left<X>,Right<X>{public X read(){return null;}}}Left<String> anonymous(){return new Left<String>(){public String read(){return "ok";}};}}`)
		seed := local.File.Symbols.FindClassScope("Left")
		plan, err := planGenericFamily(seed, local.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.members) != 2 || len(plan.locals) != 1 || len(plan.anonymous) != 1 {
			t.Fatalf("source event inventory changed: members=%d locals=%d anonymous=%d", len(plan.members), len(plan.locals), len(plan.anonymous))
		}
		use := local.File.Symbols.FindClassScope("Use")
		captured := false
		for _, method := range use.Methods {
			if method.OriginalName == "make" {
				if len(method.TypeParameters) != 1 {
					t.Fatal("missing source method binder")
				}
				_, captured = plan.binders[method.TypeParameters[0].Declaration]
			}
		}
		if !captured {
			t.Fatal("local interface implementation lost method-owned binder identity")
		}
		for key, event := range plan.locals {
			if key.file != local.File.Symbols || event.owner != use || event.node == nil {
				t.Fatal("local event lost source ownership")
			}
		}
		for key, event := range plan.anonymous {
			if key.file != local.File.Symbols || event.owner != use || event.parent != seed || event.javaType != "Left<String>" {
				t.Fatal("anonymous event lost declaringcontext")
			}
		}
	})
}

package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// Rejected families still need the complete source-local interface inventory,
// but that immutable inventory belongs to the discovery, not each seed. The
// source is legal Java: enums may implement instantiated generic interfaces.
func TestGenericFamilyInventory_WholeGraphWorkIsPerDiscovery(t *testing.T) {
	var source strings.Builder
	for index := 0; index < 64; index++ {
		fmt.Fprintf(&source, "interface Slot%d<T>{T read();} enum Item%d implements Slot%d<String>{ONE;public String read(){return \"ok\";}}\n", index, index, index)
	}
	source.WriteString("class Demands{\n")
	for index := 0; index < 64; index++ {
		fmt.Fprintf(&source, "Slot%d<?> value%d;\n", index, index)
	}
	source.WriteString("} class Padding{void noise(){int value=0;\n")
	for index := 0; index < 4096; index++ {
		source.WriteString("value++;\n")
	}
	source.WriteString("}}")
	helper := setupParseHelper(t, source.String())
	t.Run("rejected-families", func(t *testing.T) {
		if seeds := sourceGenericViewDemandSeeds(helper.Ctx); len(seeds) != 64 {
			t.Fatalf("demand seeds=%d,want64", len(seeds))
		}
		if plans := discoverCanonicalGenericFamilies(helper.Ctx); len(plans) != 0 {
			t.Fatalf("enum family admission changed: plans=%d", len(plans))
		}
	})
	t.Run("inventory-budget", func(t *testing.T) {
		inventory := testing.AllocsPerRun(1, func() {
			sourceGenericViewDemandSeeds(helper.Ctx)
			genericFamilyLocalInterfaceEdges(allSourceClassScopes(), helper.Ctx)
		})
		discovery := testing.AllocsPerRun(1, func() {
			if plans := discoverCanonicalGenericFamilies(helper.Ctx); len(plans) != 0 {
				t.Fatal("rejected enum family admitted")
			}
		})
		t.Logf("same complete source graph: single inventory allocations=%.0f discovery allocations=%.0f ratio=%.2f", inventory, discovery, discovery/inventory)
		if discovery > 6*inventory {
			t.Fatalf("GENERIC_DISCOVERY_REPEATED_WHOLE_AST: %.0f allocations >6*single inventory %.0f", discovery, inventory)
		}
	})
	t.Run("local-interface-edges", func(t *testing.T) {
		local := setupParseHelper(t, `interface Left<T>{T read();}interface Right<T>{T read();}class Use{void make(){class Local implements Left<String>,Right<String>{public String read(){return "ok";}}}}`)
		edges := genericFamilyLocalInterfaceEdges(allSourceClassScopes(), local.Ctx)
		if len(edges) != 1 || len(edges[0]) != 2 {
			t.Fatalf("local interface edges lost: %#v", edges)
		}
		names := map[string]bool{}
		for _, scope := range edges[0] {
			names[scope.Class.OriginalName] = true
		}
		if !names["Left"] || !names["Right"] {
			t.Fatal("local interfaces resolved against wrong declaration")
		}
	})
	t.Run("accepted-connected-family", func(t *testing.T) {
		accepted := setupParseHelper(t, `interface Slot<T>{T read();}abstract class Adapter<T> implements Slot<T>{public abstract T read();}class Memory<T> extends Adapter<T>{T value;public T read(){return value;}}class Use{Adapter<?> demand;Slot<?> mirror;Adapter<String> make(){return new Adapter<String>(){public String read(){return "ok";}};}}`)
		seed := accepted.File.Symbols.FindClassScope("Adapter")
		plan, err := planGenericFamily(seed, accepted.Ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(plan.members) != 3 || len(plan.anonymous) != 1 {
			t.Fatalf("complete admitted inventory changed: members=%d anonymous=%d", len(plan.members), len(plan.anonymous))
		}
		for _, name := range []string{"Slot", "Adapter", "Memory"} {
			if _, ok := plan.members[accepted.File.Symbols.FindClassScope(name)]; !ok {
				t.Errorf("missing connected %s", name)
			}
		}
	})
}

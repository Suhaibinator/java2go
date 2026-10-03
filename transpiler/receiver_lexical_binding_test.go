package transpiler

import (
	"fmt"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func TestSourceClassReceiverLexicalValueBindings(t *testing.T) {
	cases := []struct {
		name, declaration string
		inner             bool
		wantClasses       []string
	}{
		{"parameter", `class Probe { static int run(Binder Cache) { return Cache.pick(); } }`, false, []string{""}},
		{"field", `class Probe { Binder Depot; int run() { return Depot.pick(); } }`, false, []string{""}},
		{"inherited field", `class Parent { Binder Cache; } class Probe extends Parent { int run() { return Cache.pick(); } }`, false, []string{""}},
		{"enclosing field", `class Probe { Binder Depot; class Inner { int run() { return Depot.pick(); } } }`, true, []string{""}},
		{"prior local", `class Probe { static int run() { Binder Cache = new Binder(); return Cache.pick(); } }`, false, []string{""}},
		{"later local", `class Probe { static int run() { int first = Depot.pick(); Binder Depot = new Binder(); return first + Depot.pick(); } }`, false, []string{"Depot", ""}},
		{"nested block ends", `class Probe { static int run() { { Binder Cache = new Binder(); Cache.pick(); } return Cache.pick(); } }`, false, []string{"", "Cache"}},
		{"declarator order", `class Probe { static int run() { Binder first = Depot.make(), Depot = new Binder(first.pick()); return Depot.pick(); } }`, false, []string{"Depot", ""}},
		{"basic for", `class Probe { static int run() { for (Binder Cache = new Binder(); Cache.pick() == 42;) { Cache.pick(); break; } return Cache.pick(); } }`, false, []string{"", "", "Cache"}},
		{"for declarator order", `class Probe { static int run() { for (Binder first = Depot.make(), Depot = new Binder(); Depot.pick() == 42;) { break; } return Depot.pick(); } }`, false, []string{"Depot", "", "Depot"}},
		{"enhanced for", `class Probe { static int run() { for (Binder Cache : Cache.items()) { Cache.pick(); } return Cache.pick(); } }`, false, []string{"Cache", "", "Cache"}},
		{"catch", `class Probe { static int run() { try { throw new Exception(); } catch (Exception Depot) { Depot.getMessage(); } return Depot.pick(); } }`, false, []string{"", "Depot"}},
		{"inferred lambda", `class Probe { static int run() { java.util.function.Function<Binder, Integer> f = Cache -> Cache.pick(); return Cache.pick(); } }`, false, []string{"", "Cache"}},
		{"typed lambda", `class Probe { static int run() { java.util.function.Function<Binder, Integer> f = (Binder Depot) -> Depot.pick(); return Depot.pick(); } }`, false, []string{"", "Depot"}},
		{"resources", `class Probe { static int run() { try (Binder first = Cache.make(); Binder Cache = first) { Cache.pick(); } return Cache.pick(); } }`, false, []string{"Cache", "", "Cache"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(`class Binder implements AutoCloseable { Binder() {} Binder(int ignored) {} public int pick() { return 42; } public void close() {} }
class Cache { static int pick() { return 7; } static Binder make() { return new Binder(); } static Binder[] items() { return new Binder[0]; } }
class Depot { static int pick() { return 9; } static Binder make() { return new Binder(); } }
` + tc.declaration)
			helper := setupParseHelper(t, string(source))
			ctx := helper.Ctx
			ctx.currentClass = helper.File.Symbols.FindClassScope("Probe")
			if ctx.currentClass == nil {
				t.Fatal("missing probe class")
			}
			if tc.inner {
				ctx.currentClass = ctx.currentClass.Subclasses[0]
			}
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil {
				t.Fatal("missing probe method")
			}
			var receivers []*sitter.Node
			var visit func(*sitter.Node)
			visit = func(node *sitter.Node) {
				if node.Type() == "method_invocation" {
					object := node.ChildByFieldName("object")
					if object != nil && object.Type() == "identifier" && (object.Content(source) == "Cache" || object.Content(source) == "Depot") {
						receivers = append(receivers, object)
					}
				}
				for index := 0; index < int(node.NamedChildCount()); index++ {
					visit(node.NamedChild(index))
				}
			}
			visit(ctx.localScope.DeclarationNode)
			if len(receivers) != len(tc.wantClasses) {
				t.Fatalf("receiver count = %d, want %d", len(receivers), len(tc.wantClasses))
			}
			for index, receiver := range receivers {
				t.Run(fmt.Sprint(index), func(t *testing.T) {
					scope := resolveClassScopeByIdentifier(ctx, source, receiver)
					actual := ""
					if scope != nil {
						actual = scope.Class.OriginalName
					}
					if actual != tc.wantClasses[index] {
						t.Fatalf("receiver %s resolved class %q, want %q (empty means lexical value)", receiver.Content(source), actual, tc.wantClasses[index])
					}
				})
			}
		})
	}
}

package transpiler

import (
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestDeclaredMethodInferredOriginExactAndRefusalMatrix(t *testing.T) {
	const declarations = `class Base {} class Child extends Base {} class Other extends Base {} class Box<X> {} class Owner {
 <T extends Base> int pick(Box<T> value){return 1;} long pick(Object value){return 2;}
 <T extends Base> int pair(Box<T> a,Box<T> b){return 3;} long pair(Object a,Object b){return 4;}
 <T extends Base> int array(Box<T>[] value){return 5;} long array(Object value){return 6;}
 }
 `
	for _, tc := range []struct {
		name, caller, method string
		exact                bool
	}{
		{"unique invariant", `static int run(Owner owner,Box<Child> value){return owner.pick(value);}`, "pick", true},
		{"actual caller binder", `static <T extends Base> int run(Owner owner,Box<T> value){return owner.pick(value);}`, "pick", true},
		{"explicit actual origin", `static int run(Owner owner,Box<Child> value){return owner.<Child>pick(value);}`, "pick", true},
		// This is an invalid-Java parser shield, not a JVM parity fixture.
		{"explicit incompatible origin", `static int run(Owner owner,Box<Child> value){return owner.<Other>pick(value);}`, "pick", false},
		{"multiple identical constraints", `static int run(Owner owner,Box<Child> a,Box<Child> b){return owner.pair(a,b);}`, "pair", true},
		{"conflicting invariant constraints", `static long run(Owner owner,Box<Child> a,Box<Other> b){return owner.pair(a,b);}`, "pair", false},
		{"null only", `static int run(Owner owner){return owner.pick(null);}`, "pick", false},
		{"wildcard capture remains deferred", `static int run(Owner owner,Box<? extends Base> value){return owner.pick(value);}`, "pick", false},
		{"generic array", `static int run(Owner owner,Box<Child>[] value){return owner.array(value);}`, "array", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, declarations+"class Use {"+tc.caller+"}")
			ctx := helper.Ctx.Clone()
			ctx.currentClass = helper.File.Symbols.FindClassScope("Use")
			ctx.className = ctx.currentClass.Class.Name
			for _, m := range ctx.currentClass.Methods {
				if m.OriginalName == "run" {
					ctx.localScope = m
				}
			}
			owner := helper.File.Symbols.FindClassScope("Owner")
			var method *symbol.Definition
			for _, m := range owner.Methods {
				if m.OriginalName == tc.method && len(m.TypeParameters) == 1 {
					method = m
				}
			}
			if method == nil || ctx.localScope == nil {
				t.Fatal("actual declarations missing")
			}
			formals := methodFormalOrigins(method, owner, nil, ctx)
			before := append([]inferredJavaTypeOrigin(nil), formals...)
			ctx.expectedType = "java.lang.Object"
			var invocation *sitter.Node
			var walk func(*sitter.Node)
			walk = func(n *sitter.Node) {
				if n.Type() == "method_invocation" && n.ChildByFieldName("name").Content(helper.File.Source) == tc.method {
					invocation = n
				}
				for i := 0; i < int(n.NamedChildCount()); i++ {
					walk(n.NamedChild(i))
				}
			}
			walk(ctx.localScope.DeclarationNode)
			if invocation == nil {
				t.Fatal("actual invocation missing")
			}
			args := nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments"))
			projected, proved := declaredMethodArgumentInferredOrigin(formals[0], method, formals, args, false, ctx, helper.File.Source)
			var actual inferredJavaTypeOrigin
			typ, _ := inferExprJavaType(args[0], ctx, helper.File.Source, &actual)
			actual.javaType = typ
			_, exact, applies, handled := declaredOriginConversion(actual, projected, ctx)
			got := proved && exact && applies && handled
			if got != tc.exact {
				t.Errorf("exact conversion=%t proved=%t conversion=%t/%t/%t, want %t", got, proved, exact, applies, handled, tc.exact)
			}
			if ctx.expectedType != "java.lang.Object" || !reflect.DeepEqual(before, formals) {
				t.Fatal("origin refinement mutated caller context or formal origins")
			}
			if tc.name == "actual caller binder" && got {
				if len(projected.arguments) != 1 || projected.arguments[0].parameter != ctx.localScope.TypeParameters[0].Declaration || projected.arguments[0].parameter == method.TypeParameters[0].Declaration {
					t.Fatal("caller and method declaration identities collapsed")
				}
			}
		})
	}
}

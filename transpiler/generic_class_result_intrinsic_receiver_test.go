package transpiler

import (
	"go/ast"
	"os"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

const genericResultReceiverProbe = `
class SourceTimestamp { long getTime(){return 7L;} }
class Unproved<A> { Object read(){return null;} }
class MethodBinder<A> { <M> M read(){return null;} }
class ReceiverProbe {
 static long time(Base<java.sql.Timestamp> view){return view.read().getTime();}
 static long nanos(Base<java.sql.Timestamp> view){return view.read().getNanos();}
 static long shadow(Base<SourceTimestamp> view){return view.read().getTime();}
 static Object foreign(Base<java.net.URI> view){return view.read();}
 static Object unproved(Unproved<java.sql.Timestamp> view){return view.read();}
 static Object mismatched(MethodBinder<java.sql.Timestamp> view){return view.<java.sql.Timestamp>read();}
}`

func genericResultReceiverTestContext(t *testing.T, name string) (Ctx, []byte, *sitter.Node) {
	t.Helper()
	fixture, err := os.ReadFile("testdata/generic_date_bound/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	source := append(append([]byte(nil), fixture...), []byte(genericResultReceiverProbe)...)
	helper := setupParseHelper(t, string(source))
	owner := helper.File.Symbols.FindClassScope("ReceiverProbe")
	ctx := classScopeCtx(owner, helper.Ctx)
	for _, method := range owner.Methods {
		if method.OriginalName == name {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil {
		t.Fatal("missing exact probe method")
	}
	var inner *sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" {
			if n := node.ChildByFieldName("name"); n != nil && n.Content(source) == "read" {
				inner = node
				return
			}
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if inner == nil {
		t.Fatal("missing actual result invocation")
	}
	return ctx, source, inner
}

func TestGenericClassResultIntrinsicReceiverProjection(t *testing.T) {
	for _, test := range []struct{ name, method string }{{"time", "getTime"}, {"nanos", "getNanos"}} {
		t.Run(test.name, func(t *testing.T) {
			ctx, source, inner := genericResultReceiverTestContext(t, test.name)
			view, erasure, known := directOwnerMethodResultView(inner, ctx, source)
			if !known || view != "java.sql.Timestamp" || erasure != "Object" {
				t.Fatalf("declared result/physical proof missing: %q %q %v", view, erasure, known)
			}
			target, resolution := resolvedInstanceInvocation(inner, ctx, source)
			if resolution == nil || resolution.def == nil || resolution.owner == nil || len(resolution.def.TypeParameters) != 0 {
				t.Fatal("not the exact ordinary class-generic result declaration")
			}
			if len(resolution.owner.TypeParameters[0].Bounds) != 0 {
				t.Fatal("probe binder is not the implicit Object bound")
			}
			declared := definitionDeclaredJavaType(resolution.def)
			if declared.TypeParameterBindings["A"] != resolution.owner.TypeParameters[0].Declaration {
				t.Fatal("result lost its declaring class binder identity")
			}
			var origin inferredJavaTypeOrigin
			recordMethodTypeOrigin([]*inferredJavaTypeOrigin{&origin}, resolution, target, inner, ctx, source, view)
			if !completeJavaTypeOrigin(origin) || origin.externalName != "java.sql.Timestamp" || origin.parameter != nil || origin.nominalScope != nil {
				t.Fatalf("substituted result origin not canonical Timestamp: %#v", origin)
			}
			expression, ok := tryInstanceIntrinsic(inner, test.method, source, ctx)
			if !ok || expression == nil {
				t.Fatal("supported native result method was not selected")
			}
			projected := false
			ast.Inspect(expression, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				indexed, ok := call.Fun.(*ast.IndexExpr)
				if !ok {
					return true
				}
				selector, ok := indexed.X.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "ObjectView" {
					return true
				}
				pointer, ok := indexed.Index.(*ast.StarExpr)
				if !ok {
					return true
				}
				typ, ok := pointer.X.(*ast.SelectorExpr)
				if ok && typ.Sel.Name == "SQLTimestamp" {
					projected = true
				}
				return true
			})
			if !projected {
				t.Fatal("proved class-generic reference result reached native method receiver without nominal projection")
			}
		})
	}
}

func TestGenericClassResultIntrinsicReceiverProofControls(t *testing.T) {
	for _, name := range []string{"source_shadow", "foreign_owner", "unproved_result", "method_binder_mismatch", "mismatched_receiver", "binary_Date_no_cast"} {
		t.Run(name, func(t *testing.T) {
			probe := map[string]string{"source_shadow": "shadow", "foreign_owner": "foreign", "unproved_result": "unproved", "method_binder_mismatch": "mismatched", "mismatched_receiver": "time", "binary_Date_no_cast": "time"}[name]
			ctx, source, inner := genericResultReceiverTestContext(t, probe)
			if name == "mismatched_receiver" {
				other, _, _ := genericResultReceiverTestContext(t, "unproved")
				ctx.localScope = other.localScope
			}
			view, erasure, known := directOwnerMethodResultView(inner, ctx, source)
			switch name {
			case "source_shadow":
				if !known || view != "SourceTimestamp" || erasure != "Object" {
					t.Fatal("source shadow borrowed native Timestamp proof")
				}
				if _, selected := tryInstanceIntrinsic(inner, "getTime", source, ctx); selected {
					t.Fatal("source method borrowed an external native intrinsic")
				}
			case "foreign_owner":
				if !known || view != "java.net.URI" || erasure != "Object" {
					t.Fatal("foreign result borrowed Timestamp identity")
				}
			case "unproved_result", "method_binder_mismatch", "mismatched_receiver":
				if known {
					t.Fatal("unproved or mismatched result donated a class-binder view")
				}
			case "binary_Date_no_cast":
				if !known || view != "java.sql.Timestamp" {
					t.Fatal("missing Date result control")
				}
				original := &ast.Ident{Name: "erasedResult"}
				if projectDirectOwnerErasedIntrinsicReceiver(original, inner, ctx, source) != original {
					t.Fatal("binary/scalar Date operand acquired a new reference cast")
				}
			}
		})
	}
}

package transpiler

import (
	"go/ast"
	"os"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// Use the original application unchanged: nestedInference has a caller T extends
// B and combines B with widen(T). Both applicability and lowering must retain B.
func TestCI107NestedGenericStaticDependentWitness(t *testing.T) {
	source, err := os.ReadFile("../testfiles/applications/dependent_generic_conversion_tdd/src/parity/dependentgeneric/DependentGenericConversionApplication.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(source))
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("DependentGenericConversionApplication")
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "nestedInference" {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil {
		t.Fatal("missing original nestedInference")
	}
	ctx.executionContextName = executionParameterName(ctx.localScope.DeclarationNode, source, ctx)
	ctx.dependentTypeWitnesses = planConcreteDependentTypeWitnesses(ctx.localScope, source, ctx)
	var invocation *sitter.Node
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" && node.Content(source) == "combine(anchor, widen(concrete))" {
			invocation = node
			return
		}
		for i := 0; i < int(node.NamedChildCount()); i++ {
			walk(node.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if invocation == nil {
		t.Fatal("missing original combine invocation")
	}
	if got := javaInferenceLeastUpperBound([]string{"B", "T"}, ctx); got != "B" {
		t.Errorf("caller dependent LUB = %q, want B", got)
	}
	resolved := findBestMethodInHierarchy(ctx.currentClass, "combine", invocation.ChildByFieldName("arguments"), false, true, ctx, source)
	if resolved == nil {
		t.Fatal("nested generic static combine must remain applicable")
	}
	call, ok := ParseExpr(invocation, source, ctx).(*ast.CallExpr)
	if !ok || len(call.Args) != 4 {
		t.Fatalf("combine needs execution, dependent witness and both source arguments: %#v", call)
	}
	nested, ok := call.Args[3].(*ast.CallExpr)
	if !ok {
		t.Fatal("nested widen must remain a direct generic call")
	}
	if _, ok := nested.Fun.(*ast.IndexListExpr); !ok {
		t.Fatalf("target-typed B result must not be rewrapped as T: %#v", nested.Fun)
	}
	if len(dependentTypeWitnessInvocationArguments(resolved.def, invocation, ctx, source)) != 1 {
		t.Fatal("combine must carry its Base projection witness")
	}

}

func TestCI107DependentLUBKeepsShadowedDeclarations(t *testing.T) {
	helper := setupParseHelper(t, `class Base {} class Other {}
class Outer<B extends Base,T extends B> { <B extends Other> void compare(T first,B second) {} }`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Outer")
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "compare" {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil {
		t.Fatal("missing shadowing method")
	}
	outerB, outerT := ctx.currentClass.TypeParameters[0].EmittedName(), ctx.currentClass.TypeParameters[1].EmittedName()
	methodB := ctx.localScope.TypeParameters[0].EmittedName()
	if methodB == outerB {
		t.Fatal("fixture requires separate emitted binders")
	}
	for _, pair := range [][2]string{{outerT, outerB}, {outerB, outerT}} {
		if got := javaInferencePairLeastUpperBound(pair[0], pair[1], ctx); got != outerB {
			t.Errorf("outer chain LUB(%s,%s)=%s, want %s", pair[0], pair[1], got, outerB)
		}
	}
	for _, pair := range [][2]string{{outerT, methodB}, {methodB, outerT}} {
		if got := javaInferencePairLeastUpperBound(pair[0], pair[1], ctx); got == methodB || got == outerT {
			t.Errorf("unrelated shadowed chain LUB(%s,%s)=%s", pair[0], pair[1], got)
		}
	}
}

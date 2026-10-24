package transpiler

import (
	"go/ast"
	"go/token"
	"os"
	"reflect"
	"strconv"
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
	// The source-owned static helper now uses each declared first-bound slot.
	// The consuming B view is restored once after that physical call; requiring
	// a direct widen[B,T] call would assert the older generated ABI instead.
	base := resolveClassScopeByQualifiedName(ctx, "Base")
	if base == nil || base.Class == nil {
		t.Fatal("missing original source Base declaration")
	}
	basePointer := &ast.StarExpr{X: qualifiedNameExpr(base.Class.Name, findJavaPackageForClassScope(base), ctx)}
	view, ok := nested.Fun.(*ast.IndexExpr)
	if !ok || !reflect.DeepEqual(view.X, stdjavaQualifiedExpr("ObjectView", ctx)) || !reflect.DeepEqual(view.Index, basePointer) {
		t.Fatalf("nested B consumer must have the owned runtime Base view: %#v", nested.Fun)
	}
	if len(nested.Args) != 2 {
		t.Fatalf("nested consumer needs exactly the physical call and nominal descriptor: %#v", nested.Args)
	}
	baseDescriptor := &ast.CallExpr{Fun: stdjavaQualifiedExpr("TypeID", ctx), Args: []ast.Expr{
		&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("parity.dependentgeneric.DependentGenericConversionApplication$Base")},
	}}
	if !reflect.DeepEqual(nested.Args[1], baseDescriptor) {
		t.Fatalf("B consumer descriptor must name the original Base, not T or Object: %#v", nested.Args[1])
	}
	physical, ok := nested.Args[0].(*ast.CallExpr)
	if !ok {
		t.Fatalf("consumer must contain the original source widen call: %#v", nested.Args[0])
	}
	widen := findBestMethodInHierarchy(ctx.currentClass, "widen", invocation.ChildByFieldName("arguments").NamedChild(1).ChildByFieldName("arguments"), false, true, ctx, source)
	if widen == nil || len(widen.def.TypeParameters) != 2 ||
		widen.def.TypeParameters[0].Declaration == nil || widen.def.TypeParameters[1].Declaration == nil ||
		widen.def.TypeParameters[0].Declaration == widen.def.TypeParameters[1].Declaration ||
		!methodTypeParameterDependsOn("T", "B", widen.def.TypeParameters) {
		t.Fatal("original widen must retain distinct dependent B/T declarations")
	}
	physicalFun, ok := physical.Fun.(*ast.IndexListExpr)
	ownedWiden := qualifiedNameExpr(executionMethodCallName(widen.def, widen.owner, ctx), findJavaPackageForClassScope(widen.owner), ctx)
	if !ok || !reflect.DeepEqual(physicalFun.X, ownedWiden) || len(physicalFun.Indices) != 2 ||
		!reflect.DeepEqual(physicalFun.Indices[0], basePointer) || !reflect.DeepEqual(physicalFun.Indices[1], basePointer) {
		t.Fatalf("physical helper must be the source-owned widen with both Base slots: %#v", physical.Fun)
	}
	if len(physical.Args) != 4 || !reflect.DeepEqual(physical.Args[0], ast.NewIdent(ctx.executionContextName)) {
		t.Fatalf("physical widen needs execution, both dependent witnesses and concrete argument: %#v", physical.Args)
	}
	for _, argument := range physical.Args[1:3] {
		witness, ok := argument.(*ast.FuncLit)
		if !ok || witness.Type.Params == nil || len(witness.Type.Params.List) != 1 ||
			len(witness.Type.Params.List[0].Names) != 1 || !reflect.DeepEqual(witness.Type.Params.List[0].Type, basePointer) ||
			witness.Type.Results == nil || len(witness.Type.Results.List) != 1 || !reflect.DeepEqual(witness.Type.Results.List[0].Type, basePointer) ||
			witness.Body == nil || len(witness.Body.List) != 1 {
			t.Fatalf("physical dependent witness must preserve the Base-to-Base view: %#v", argument)
		}
		returned, ok := witness.Body.List[0].(*ast.ReturnStmt)
		if !ok || len(returned.Results) != 1 || !reflect.DeepEqual(returned.Results[0], ast.NewIdent(witness.Type.Params.List[0].Names[0].Name)) {
			t.Fatalf("physical dependent witness must retain the same value: %#v", witness.Body)
		}
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

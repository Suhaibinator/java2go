package transpiler

import (
	"go/ast"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestDeferredToCollectionActualConsumerRendering(t *testing.T) {
	for _, row := range []struct{ declarations, result, factory, element string }{
		{"", "java.util.ArrayList<String>", "java.util.ArrayList<String>::new", "String"},
		{sourceArrayListNativeGuardRoot, "Values<String>", "Values<String>::new", "String"},
		{sourceArrayListNativeGuardRoot, "java.util.List<String>", "Values<String>::new", "String"},
		{`final class Words extends java.util.ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}`, "Words", "Words::new", "String"},
		{`class Plain extends java.util.ArrayList<Object>{}`, "Plain", "()->new Plain()", "Object"},
	} {
		t.Run(row.result, func(t *testing.T) {
			file := sourceArrayListP2GuardFile(t, `import java.util.stream.Stream;import java.util.stream.Collectors;`+row.declarations+`class Use{`+row.result+` run(){return Stream.<`+row.element+`>of(null).collect(Collectors.toCollection(`+row.factory+`));}}`)
			body := sourceArrayListP2GuardText(sourceArrayListP2GuardMethod(t, file, "RunJava2goExecution").Body)
			for _, required := range []string{"CollectorToCollectionPlan[", "SupplierReferenceView(", "StreamCollectPlanExecution(__java2goExecution", "ObjectView["} {
				if !strings.Contains(body, required) {
					t.Fatalf("real consumer lacks %s: %s", required, body)
				}
			}
			if strings.Contains(body, "CollectorToCollectionPlan[any,") || strings.Contains(body, "CollectorToListPlan") || strings.Contains(body, "NewExecution(") {
				t.Fatalf("consumer selected wrong recipe, typed result cast or fresh Execution: %s", body)
			}
		})
	}
}

func TestDeferredToCollectionDeclaredSupplierAndSourceSAM(t *testing.T) {
	for _, declaration := range []string{
		`class Factory{static java.util.function.Supplier<Values<String>> factory=Values<String>::new;}`,
		`class Owned implements java.util.function.Supplier<Values<String>>{public Values<String> get(){return new Values<String>();}} class Factory{static Owned factory=new Owned();}`,
	} {
		h, ctx, collect := sourceArrayListA4CollectorInvocation(t, sourceArrayListNativeGuardRoot+declaration, "Values<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(Factory.factory))`)
		proof, known := deferredToCollectionCertificate(invocationArgumentNode(collect, 0), ctx, h.File.Source)
		if !known || proof.result.nominalScope != h.Ctx.currentFile.FindClassScope("Values") {
			t.Fatal("declared/source Supplier lost its selected C origin")
		}
		plan, _ := lowerDeferredToCollectionPlan(invocationArgumentNode(collect, 0), ctx, h.File.Source)
		if plan == nil || strings.Count(sourceArrayListP2GuardText(plan), "Factory") > 1 {
			t.Fatal("declared Supplier unavailable or reevaluated")
		}
	}
}

func TestDeferredToCollectionInheritedElementAndReorderedBinders(t *testing.T) {
	for _, source := range []string{
		sourceArrayListNativeGuardRoot + `class Parent<A,B>{java.util.function.Supplier<Values<B>> factory;}class Use<X,Y> extends Parent<Y,X>{Values<X> run(java.util.stream.Stream<X> input){return input.collect(java.util.stream.Collectors.toCollection(factory));}}`,
		sourceArrayListNativeGuardRoot + `class Parent{static class Item{} java.util.function.Supplier<Values<Item>> factory;java.util.stream.Stream<Item> input;}class Use extends Parent{static class Item{} Values<Parent.Item> run(){return input.collect(java.util.stream.Collectors.toCollection(factory));}}`,
	} {
		h := setupParseHelper(t, source)
		owner := h.Ctx.currentFile.FindClassScope("Use")
		method := owner.FindMethodByName("run", nil)
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = method
		collect := findNode(method.DeclarationNode.ChildByFieldName("body"), "method_invocation")
		proof, known := deferredToCollectionCertificate(invocationArgumentNode(collect, 0), ctx, h.File.Source)
		if !known {
			t.Fatal("declaration-owned inherited/reordered element lost its terminal result")
		}
		target, known := sourceArrayListNativeBoundaryOrigin(collect, ctx, h.File.Source)
		if !known || !sameJavaTypeOrigin(proof.result, target) {
			t.Fatal("supplier C was reparsed under caller member or binder homonym")
		}
	}
}

func TestDeferredToCollectionCanonicalOwnerAndInvariantSlotGuards(t *testing.T) {
	for _, row := range []struct{ declaration, factory string }{
		{`class Collectors{}`, `ArrayList<String>::new`},
		{`class ArrayList<T>{}`, `ArrayList<String>::new`},
		{`class Factory{static Supplier<ArrayList<Object>> factory;}`, `Factory.factory`},
		{`class Supplier<T>{}class Factory{static Supplier<ArrayList<String>> factory;}`, `Factory.factory`},
		{`class Factory{static java.util.function.Supplier<java.util.ArrayList<String>> factory[];}`, `Factory.factory`},
	} {
		h, ctx, collect := sourceArrayListA4CollectorInvocation(t, row.declaration, "Object", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(`+row.factory+`))`)
		if _, known := deferredToCollectionCertificate(invocationArgumentNode(collect, 0), ctx, h.File.Source); known {
			t.Fatalf("foreign, mismatched or suffix-array factory acquired collector authority: %s", row.declaration)
		}
	}
}

func TestDeferredToCollectionRejectsForeignExecutableAndChangedSource(t *testing.T) {
	h, ctx, collect := sourceArrayListA4CollectorInvocation(t, "", "ArrayList<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<String>::new))`)
	collector := invocationArgumentNode(collect, 0)
	registered := symbol.GlobalScope
	foreign, other, foreignCollect := sourceArrayListA4CollectorInvocation(t, "", "ArrayList<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(ArrayList<String>::new))`)
	if _, known := deferredToCollectionCertificate(invocationArgumentNode(foreignCollect, 0), other, foreign.File.Source); !known {
		t.Fatal("foreign graph did not establish its own positive premise")
	}
	symbol.GlobalScope = registered
	if _, known := deferredToCollectionCertificate(collector, ctx, h.File.Source); !known {
		t.Fatal("original graph not restored before source/owner negatives")
	}
	for _, row := range []struct {
		ctx    Ctx
		source []byte
	}{{other, h.File.Source}, {ctx, append([]byte(nil), h.File.Source...)}} {
		if _, known := deferredToCollectionCertificate(collector, row.ctx, row.source); known {
			t.Fatal("equal-span foreign owner or copied source gained collector authority")
		}
	}
	if _, known := deferredToCollectionCertificate(invocationArgumentNode(foreignCollect, 0), ctx, h.File.Source); known {
		t.Fatal("foreign occurrence borrowed original executable")
	}
	_ = foreign
}

func TestDeferredToCollectionPhysicalABIAndActualConsumerBoundary(t *testing.T) {
	h, ctx, collect := sourceArrayListA4CollectorInvocation(t, sourceArrayListNativeGuardRoot, "Values<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(Values<String>::new))`)
	receiver := ParseExpr(collect.ChildByFieldName("object"), h.File.Source, ctx)
	makeRaw := func() ast.Expr { return lowerCollectCall(receiver, collect, ctx, h.File.Source) }
	raw := makeRaw()
	proof, known := deferredToCollectionCertificate(invocationArgumentNode(collect, 0), ctx, h.File.Source)
	if !known || !deferredToCollectionRawResult(raw, proof, ctx) {
		t.Fatal("actual public raw runtime producer lacks its certified erased result envelope")
	}
	projected := coerceArgumentToExpectedType(raw, collect, "Values<String>", ctx, h.File.Source)
	valuesName := h.Ctx.currentFile.FindClassScope("Values").Class.Name
	if got := sourceArrayListP2GuardText(projected); !strings.Contains(got, "ObjectView[*"+valuesName+"[*stdjava.JavaString]]") || strings.Count(got, "StreamCollectPlanExecution(") != 1 {
		t.Fatalf("actual consuming return lacks one original-object C projection: %s", got)
	}
	badE := makeRaw().(*ast.CallExpr)
	badE.Args[0] = ast.NewIdent("foreignExecution")
	badName := makeRaw().(*ast.CallExpr)
	badName.Fun = &ast.SelectorExpr{X: ast.NewIdent("unowned"), Sel: ast.NewIdent("StreamCollectPlanExecution")}
	badT := makeRaw().(*ast.CallExpr)
	badPlan := badT.Args[2].(*ast.CallExpr)
	badPlan.Fun.(*ast.IndexExpr).Index = ast.NewIdent("any")
	for _, value := range []ast.Expr{badE, badName, badT, ast.NewIdent("arbitraryAny"), &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}}}} {
		if deferredToCollectionRawResult(value, proof, ctx) {
			t.Fatal("wrong currentE/T/runtime binding or arbitrary any closure donated physical authority")
		}
	}
}

func TestDeferredToCollectionObjectConsumerHasNoEarlyCheckcast(t *testing.T) {
	h, ctx, collect := sourceArrayListA4CollectorInvocation(t, sourceArrayListNativeGuardRoot, "Object", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(Values<String>::new))`)
	raw := lowerCollectCall(ParseExpr(collect.ChildByFieldName("object"), h.File.Source, ctx), collect, ctx, h.File.Source)
	converted := coerceArgumentToExpectedType(raw, collect, "Object", ctx, h.File.Source)
	if got := sourceArrayListP2GuardText(converted); strings.Contains(got, "ObjectView[") || strings.Count(got, "StreamCollectPlanExecution(") != 1 {
		t.Fatalf("Object consumer narrowed or reevaluated erased C: %s", got)
	}
}

func TestDeferredToCollectionActualFieldLocalAssignmentAndReturn(t *testing.T) {
	expression := `java.util.stream.Stream.<String>of("x").collect(java.util.stream.Collectors.toCollection(Values<String>::new))`
	source := sourceArrayListNativeGuardRoot + `class Use{static Values<String> shared=` + expression + `;Values<String> field=` + expression + `;Values<String> run(){Values<String> local=` + expression + `;local=` + expression + `;return ` + expression + `;}}`
	declarations := setupParseHelper(t, source)
	valuesName := declarations.Ctx.currentFile.FindClassScope("Values").Class.Name
	file := sourceArrayListP2GuardFile(t, source)
	out := sourceArrayListP2GuardText(file)
	projected := 0
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && len(call.Args) == 2 && sourceArrayListP2GuardText(call.Fun) == "stdjava.ObjectView[*"+valuesName+"[*stdjava.JavaString]]" && strings.Count(sourceArrayListP2GuardText(call.Args[0]), "CollectorToCollectionPlan[") == 1 {
			projected++
		}
		return true
	})
	if projected != 5 || strings.Count(out, "CollectorToCollectionPlan[") != 5 {
		t.Fatalf("actual static/instance field, local, assignment or return producer lost C projection: %s", out)
	}
}

func TestDeferredToCollectionDownstreamTypedResultRemainsHeld(t *testing.T) {
	h, ctx, collect := sourceArrayListA4CollectorInvocation(t, "", "Object", `Stream.<String>of(first,null,last).collect(Collectors.groupingBy(value->value,Collectors.toCollection(ArrayList<String>::new)))`)
	grouping := invocationArgumentNode(collect, 0)
	downstream := invocationArgumentNode(grouping, 1)
	if _, known := deferredToCollectionCertificate(downstream, ctx, h.File.Source); known {
		t.Fatal("unproved typed downstream R composition was silently admitted")
	}
}

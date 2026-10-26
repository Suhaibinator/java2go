package transpiler

import (
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// Ported controls preserve all six authored supplier forms and public legacy
// toList routing. They exercise the new owned direct toCollection branch,
// without changing the independently frozen application or JVM observations.
func sourceArrayListA4CollectorInvocation(t *testing.T, declarations, result, expression string) (*ParseHelper, Ctx, *sitter.Node) {
	t.Helper()
	source := `import java.util.ArrayList;import java.util.List;import java.util.function.Supplier;import java.util.stream.Stream;import java.util.stream.Collectors;` + declarations + `class Use{static ` + result + ` run(String first,String last){return ` + expression + `;}}`
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName("run", nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	statement := findNode(method.DeclarationNode.ChildByFieldName("body"), "return_statement")
	if statement == nil || statement.NamedChildCount() != 1 {
		t.Fatal("actual return expression missing")
	}
	return h, ctx, statement.NamedChild(0)
}

func TestSourceArrayListA4ExplicitStreamOfReceiverControl(t *testing.T) {
	h, ctx, invocation := sourceArrayListA4CollectorInvocation(t, "", "Stream<String>", `Stream.<String>of(first,null,last)`)
	elements := receiverElementJavaTypes(invocation, ctx, h.File.Source)
	if len(elements) != 1 || sourceArrayListP2GuardText(javaTypeStringToGoTypeExpr(elements[0], inScopeTypeParameters(ctx), ctx)) != "*stdjava.JavaString" {
		t.Fatalf("explicit Stream.<String>of lost its actual String element: %v", elements)
	}
	got := sourceArrayListP2GuardText(ParseExpr(invocation, h.File.Source, ctx))
	if !strings.Contains(got, "NewStream[*stdjava.JavaString]") || strings.Contains(got, "Unsupported") || strings.Contains(got, "unsupported") {
		t.Fatalf("explicit Stream.of receiver remains unsupported: %s", got)
	}
}

func TestSourceArrayListA4ExplicitStreamOfToListControl(t *testing.T) {
	strictRoutingState(t)
	out := renderGoFileFromJava(t, `import java.util.List;import java.util.stream.Stream;import java.util.stream.Collectors;class Use{static List<String> run(String first,String last){return Stream.<String>of(first,null,last).collect(Collectors.toList());}}`)
	for _, want := range []string{"NewStream[*stdjava.JavaString]", ".ToList()"} {
		if !strings.Contains(out, want) {
			t.Fatalf("explicit receiver plus existing toList path lacks %s: %s", want, out)
		}
	}
	if strings.Contains(out, "CollectorToCollectionPlan[") || strings.Contains(out, "StreamFromValues[") || strings.Contains(out, "StreamCollectPlanExecution(") {
		t.Fatalf("public legacy toList fallback changed: %s", out)
	}
	if strings.Contains(out, "Unsupported") || strings.Contains(out, "unsupported") {
		t.Fatalf("toList control retained a strict unsupported path: %s", out)
	}
}

func sourceArrayListA4RequireCustomCollector(t *testing.T, declarations, result, supplier, stream string) {
	t.Helper()
	h, ctx, collect := sourceArrayListA4CollectorInvocation(t, declarations, result, stream+`.collect(Collectors.toCollection(`+supplier+`))`)
	receiver := collect.ChildByFieldName("object")
	elements := receiverElementJavaTypes(receiver, ctx, h.File.Source)
	if len(elements) != 1 {
		t.Fatalf("receiver failed before custom collector selection: %v", elements)
	}
	collector := invocationArgumentNode(collect, 0)
	name, known := deferredToCollectionFactoryName(collector, h.File.Source, ctx)
	if !known || name != "toCollection" || invocationArgumentCount(collector) != 1 {
		t.Fatalf("actual canonical factory was not selected: name=%q known=%v", name, known)
	}
	plan, loweredResult := lowerDeferredToCollectionPlan(collector, ctx, h.File.Source)
	inferredResult, inferred := inferCollectorJavaType(collector, elements[0], ctx, h.File.Source)
	if plan == nil || loweredResult == "" || !inferred || inferredResult == "" {
		t.Fatalf("canonical toCollection missing after receiver/factory proof: supplier=%s element=%s planNil=%v lowerResult=%q inferred=%v inferredResult=%q", supplier, elements[0], plan == nil, loweredResult, inferred, inferredResult)
	}
}

func TestSourceArrayListA4NativeToCollectionSupplierSelection(t *testing.T) {
	for _, row := range []struct{ name, declaration, supplier string }{
		{"constructor_reference", "", `ArrayList<String>::new`},
		{"lambda", "", `()->new ArrayList<String>()`},
		{"declared_supplier", `class Factories{static Supplier<ArrayList<String>> factory=ArrayList<String>::new;}`, `Factories.factory`},
	} {
		t.Run(row.name, func(t *testing.T) {
			sourceArrayListA4RequireCustomCollector(t, row.declaration, "ArrayList<String>", row.supplier, `Stream.<String>of(first,null,last)`)
		})
	}
}

func TestSourceArrayListA4SourceToCollectionSupplierSelection(t *testing.T) {
	for _, row := range []struct{ name, declaration, result, supplier, stream string }{
		{"generic_constructor", `class Values<E> extends ArrayList<E>{int calls;public boolean add(E value){calls++;return super.add(value);}}`, "Values<String>", `Values<String>::new`, `Stream.<String>of(first,null,last)`},
		{"String_constructor", `final class Words extends ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}`, "Words", `Words::new`, `Stream.<String>of(first,null,last)`},
		{"Object_lambda", `class Plain extends ArrayList<Object>{}`, "Plain", `()->new Plain()`, `Stream.<Object>of(first,null,last)`},
	} {
		t.Run(row.name, func(t *testing.T) {
			sourceArrayListA4RequireCustomCollector(t, row.declaration, row.result, row.supplier, row.stream)
		})
	}
}

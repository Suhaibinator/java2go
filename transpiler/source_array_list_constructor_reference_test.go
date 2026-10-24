package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
	"testing"
)

func sourceArrayListA4DirectConstructorRender(t *testing.T, source string) (out string) {
	t.Helper()
	strictRoutingState(t)
	defer func() {
		if failure := recover(); failure != nil {
			t.Fatalf("SOURCE_CTOR_RENDER_REFUSED: %T %v", failure, failure)
		}
	}()
	return renderGoFileFromJava(t, source)
}

func TestSourceArrayListConstructorReferenceNativeControl(t *testing.T) {
	source := `import java.util.ArrayList;import java.util.function.Supplier;class Use{static Supplier<ArrayList<String>> make(){return ArrayList<String>::new;}}`
	out := sourceArrayListA4DirectConstructorRender(t, source)
	if !strings.Contains(out, "NewArrayList[*stdjava.JavaString]") || !strings.Contains(out, "NewSupplierFuncAdapter[*stdjava.List[*stdjava.JavaString]]") {
		t.Fatalf("native constructor reference control lost typed lazy Supplier: %s", out)
	}
	t.Log("NATIVE_CTOR_REFERENCE_CONTROL_PASS")
}

func TestSourceArrayListConstructorReferenceNonGenericControl(t *testing.T) {
	source := `import java.util.function.Supplier;final class Words extends java.util.ArrayList<String>{int calls;public boolean add(String value){calls++;return super.add(value);}}class Use{static Supplier<Words> make(){return Words::new;}}`
	h := setupParseHelper(t, source)
	name := h.Ctx.currentFile.FindClassScope("Words").Class.Name
	out := sourceArrayListA4DirectConstructorRender(t, source)
	if !strings.Contains(out, "NewSupplierFuncAdapter[*"+name+"]") {
		t.Fatalf("non-generic source constructor reference control lost its declared product: %s", out)
	}
	t.Log("NONGENERIC_SOURCE_CTOR_REFERENCE_CONTROL_PASS")
}

func TestSourceArrayListConstructorReferenceGenericActualRendering(t *testing.T) {
	source := `import java.util.function.Supplier;` + sourceArrayListNativeGuardRoot + `class Use{static Supplier<Values<String>> make(){return Values<String>::new;}}`
	h := setupParseHelper(t, source)
	name := h.Ctx.currentFile.FindClassScope("Values").Class.Name
	out := sourceArrayListA4DirectConstructorRender(t, source)
	if !strings.Contains(out, "NewSupplierFuncAdapter[*"+name+"[*stdjava.JavaString]]") {
		t.Fatalf("generic source constructor reference lacks its actual declared C/String instantiation: %s", out)
	}
}

func sourceArrayListConstructorReferenceOccurrence(t *testing.T, source, methodName string) (*ParseHelper, Ctx, *sitter.Node) {
	t.Helper()
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Use")
	method := owner.FindMethodByName(methodName, nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	body := method.DeclarationNode.ChildByFieldName("body")
	statement := body.NamedChild(int(body.NamedChildCount()) - 1)
	node := findNode(statement, "method_reference")
	if node == nil {
		t.Fatal("actual outer constructor occurrence missing")
	}
	return h, ctx, node
}

func TestSourceArrayListConstructorReferenceLazyExecutionFunction(t *testing.T) {
	for _, root := range []string{sourceArrayListNativeGuardRoot} {
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, `import java.util.function.Supplier;`+root+`class Use{static Supplier<Values<String>> make(){return Values<String>::new;}}`, "make")
		value, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx)
		call, isCall := value.(*ast.CallExpr)
		owner := h.Ctx.currentFile.FindClassScope("Values")
		constructor := constructorFuncName(owner)
		if constructor == "" {
			constructor = defaultConstructorName(owner.Class.Name)
		}
		expected := executionConstructorImplementationName(constructor, owner)
		if !known || !isCall || len(call.Args) != 1 {
			t.Fatal("no typed Supplier function value")
		}
		typed, isTyped := call.Args[0].(*ast.IndexExpr)
		if !isTyped || sourceArrayListP2GuardText(typed.X) != expected || sourceArrayListP2GuardText(typed.Index) != "*stdjava.JavaString" {
			t.Fatalf("lazy callback is not actual E-aware typed constructor function: %s", sourceArrayListP2GuardText(value))
		}
		// The adapter's only operand is a function value: there is no construction,
		// closure, fresh Execution or captured caller Execution at reference creation.
		text := sourceArrayListP2GuardText(value)
		if strings.Contains(text, "func(") || strings.Contains(text, "NewExecution(") || strings.Contains(text, expected+"(") || strings.Count(text, expected) != 1 {
			t.Fatalf("constructor was invoked or Execution was captured: %s", text)
		}
	}
}

func TestSourceArrayListConstructorReferenceActualBoundaries(t *testing.T) {
	source := `import java.util.function.Supplier;` + sourceArrayListNativeGuardRoot + `class Use{static Supplier<Values<String>> shared=Values<String>::new;Supplier<Values<String>> field=Values<String>::new;Supplier<Values<String>> make(){Supplier<Values<String>> local=Values<String>::new;local=Values<String>::new;field=Values<String>::new;this.field=Values<String>::new;shared=Values<String>::new;return Values<String>::new;}}`
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope("Values")
	name := owner.Class.Name
	out := sourceArrayListA4DirectConstructorRender(t, source)
	if strings.Count(out, "NewSupplierFuncAdapter[*"+name+"[*stdjava.JavaString]]") != 8 {
		t.Fatalf("actual static/instance/local/assignment/return boundaries lost invariant Supplier<C>: %s", out)
	}
}

func TestSourceArrayListConstructorReferenceAttachedCollector(t *testing.T) {
	h, ctx, collect := sourceArrayListA4CollectorInvocation(t, sourceArrayListNativeGuardRoot, "Values<String>", `Stream.<String>of(first,null,last).collect(Collectors.toCollection(Values<String>::new))`)
	collector := invocationArgumentNode(collect, 0)
	node := invocationArgumentNode(collector, 0)
	proof, known := deferredToCollectionCertificate(collector, ctx, h.File.Source)
	if !known || !proof.factory.Equal(node) {
		t.Fatal("actual collector factory premise missing")
	}
	ctx.expectedType = "java.util.function.Supplier<" + proof.resultJava + ">"
	value, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx)
	if !known || !strings.Contains(sourceArrayListP2GuardText(value), "NewSupplierFuncAdapter[") {
		t.Fatal("attached direct canonical toCollection did not supply its JDK-owned target")
	}
	other := ctx
	other.expectedType = "java.util.function.Supplier<java.util.List<String>>"
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, other); known {
		t.Fatal("covariant native consumer target replaced exact C")
	}
}

func TestSourceArrayListConstructorReferenceSourceAuthority(t *testing.T) {
	source := `import java.util.function.Supplier;` + sourceArrayListNativeGuardRoot + `class Use{static Supplier<Values<String>> make(){return Values<String>::new;}static Supplier<Values<String>> other(){return Values<String>::new;}}`
	h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, source, "make")
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
		t.Fatal("actual attached positive failed")
	}
	graph := symbol.GlobalScope
	_, foreignCtx, foreign := sourceArrayListConstructorReferenceOccurrence(t, source, "make")
	symbol.GlobalScope = graph
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
		t.Fatal("actual graph not restored before negatives")
	}
	other := ctx
	other.localScope = ctx.currentClass.FindMethodByName("other", nil)
	for _, row := range []struct {
		node   *sitter.Node
		ctx    Ctx
		source []byte
	}{{node, other, h.File.Source}, {foreign, ctx, h.File.Source}, {node, foreignCtx, h.File.Source}, {node, ctx, append([]byte(nil), h.File.Source...)}, {node, ctx, nil}, {node, ctx, h.File.Source[:1]}} {
		if _, known := lowerSourceArrayListConstructorReference(row.node, row.source, row.ctx); known {
			t.Fatal("foreign executable/node/source donated constructor authority")
		}
	}
	saved := ctx.localScope.DeclarationNode
	ctx.localScope.DeclarationNode = foreignCtx.localScope.DeclarationNode
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
		t.Fatal("equal-span foreign rebound method donated constructor authority")
	}
	ctx.localScope.DeclarationNode = saved
	at := int(node.StartByte())
	savedBytes := h.File.Source[at]
	h.File.Source[at] = 'X'
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
		t.Fatal("mutated source snapshot donated constructor authority")
	}
	h.File.Source[at] = savedBytes
}

func TestSourceArrayListConstructorReferenceTargetAndRankGuards(t *testing.T) {
	for _, body := range []string{
		`static java.util.function.Function<Object,Values<String>> make(){return Values<String>::new;}`,
		`static java.util.function.Supplier<java.util.List<String>> make(){return Values<String>::new;}`,
		`static java.util.function.Supplier<Values<Object>> make(){return Values<String>::new;}`,
		`static java.util.function.Supplier<Values<String>>[] make(){return Values<String>::new;}`,
		`static java.util.function.Supplier<Values<String>> make()[]{return Values<String>::new;}`,
		`static Object make(){return Values<String>::new;}`,
		`static java.util.function.Supplier<Values<String[]>> make(){return Values<String[]>::new;}`,
		`static java.util.function.Supplier<Values<java.util.ArrayList<String>>> make(){return Values<java.util.ArrayList<String>>::new;}`,
	} {
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+`class Use{`+body+`}`, "make")
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
			t.Fatalf("wrong SAM/invariant C/rank acquired source constructor ABI: %s", body)
		}
	}
	h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+`class Use{static Object make(){return Values<String>::new;}}`, "make")
	ctx.expectedType = "java.util.function.Supplier<Values<String>>"
	ctx.expectedTypeRoot = node
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
		t.Fatal("target-only sidecar replaced actual Object boundary")
	}
}

func TestSourceArrayListConstructorReferenceRetainedTypeNamespace(t *testing.T) {
	for _, body := range []string{
		`static java.util.function.Supplier<Values<String>> make(){class Values<E>{}return Values<String>::new;}`,
		`static java.util.function.Supplier<Values<java.lang.Object>> make(){class Object{}return Values<Object>::new;}`,
		`static java.util.function.Supplier<Values<java.lang.String>> make(){class java{static class lang{static class String{}}}return Values<java.lang.String>::new;}`,
		`static Object make(){class Supplier<T>{}Supplier<Values<String>> local=Values<String>::new;return null;}`,
		`static Object make(){class Supplier<T>{}Supplier<Values<String>> target=null;Object ignored=new Object(){java.util.function.Supplier<Values<String>> target;};target=Values<String>::new;return null;}`,
		`static Object make(){class java{static class util{static class function{static class Supplier<T>{}}}}java.util.function.Supplier<Values<String>> local=Values<String>::new;return null;}`,
	} {
		h := setupParseHelper(t, `import java.util.function.Supplier;`+sourceArrayListNativeGuardRoot+`class Use{`+body+`}`)
		owner := h.Ctx.currentFile.FindClassScope("Use")
		ctx := classScopeCtx(owner, h.Ctx)
		ctx.localScope = owner.FindMethodByName("make", nil)
		node := findNode(ctx.localScope.DeclarationNode.ChildByFieldName("body"), "method_reference")
		if node == nil || !sourceArrayListNativeSourceContext(node, ctx, h.File.Source) {
			t.Fatal("outer attached method namespace premise failed")
		}
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
			t.Fatalf("retained local TYPE donated source/Supplier/element authority: %s", body)
		}
	}
	for _, body := range []string{
		`static java.util.function.Supplier<Values<java.lang.String>> make(){class String{}return Values<java.lang.String>::new;}`,
		`static java.util.function.Supplier<Values<String>> make(){{class Values<E>{}}return Values<String>::new;}`,
	} {
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+`class Use{`+body+`}`, "make")
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
			t.Fatal("unrelated tail-name or closed block shadowed actual qualified TYPE")
		}
	}
}

func TestSourceArrayListConstructorReferenceConstructorProvenance(t *testing.T) {
	for _, ctor := range []string{`Values(){super();}`, `Values(int size){super();}`, `<T> Values(){super();}`, `Values(Object... values){super();}`} {
		root := `class Values<E> extends java.util.ArrayList<E>{` + ctor + `int calls;public boolean add(E value){calls++;return super.add(value);}}`
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, root+`class Use{static java.util.function.Supplier<Values<String>> make(){return Values<String>::new;}}`, "make")
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
			t.Fatalf("unproved arity/method binder/varargs constructor activated: %s", ctor)
		}
	}
	h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+`class Use{static java.util.function.Supplier<Values<String>> make(){return Values<String>::new;}}`, "make")
	owner := h.Ctx.currentFile.FindClassScope("Values")
	owner.Methods = append(owner.Methods, &symbol.Definition{Constructor: true, Name: "forged"})
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
		t.Fatal("detached synthetic constructor metadata donated implicit constructor ABI")
	}
}

func TestSourceArrayListConstructorReferenceOwnedBinderIdentity(t *testing.T) {
	source := sourceArrayListNativeGuardRoot + `class Parent<A,B>{java.util.function.Supplier<Values<B>> target;}class Use<X,Y> extends Parent<Y,X>{java.util.function.Supplier<Values<X>> make(){target=Values<X>::new;return Values<X>::new;}}`
	h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, source, "make")
	value, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx)
	expected := ctx.currentClass.TypeParameters[0].EmittedName()
	if !known || !strings.Contains(sourceArrayListP2GuardText(value), "["+expected+"]") {
		t.Fatal("source receiver parameter was rebound to reordered parent binder")
	}
	assignment := findNode(ctx.localScope.DeclarationNode.ChildByFieldName("body"), "assignment_expression").ChildByFieldName("right")
	if _, known := lowerSourceArrayListConstructorReference(assignment, h.File.Source, ctx); !known {
		t.Fatal("declaration-owned inherited Supplier substitution lost exact caller binder")
	}
	ctx.expectedType = "java.util.function.Supplier<Values<Y>>"
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
		t.Fatal("conflicting same-spelled target donated different parameter declaration")
	}
}

func TestSourceArrayListConstructorReferenceInheritedMemberTypeAuthority(t *testing.T) {
	for _, prefix := range []string{
		`class Parent{static class Values<T>{}}class Use extends Parent{java.util.function.Supplier<Values<String>> make(){return Values<String>::new;}}`,
		`class Parent{interface Supplier<T>{void run();}}class Use extends Parent{Supplier<Values<String>> make(){return Values<String>::new;}}`,
	} {
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+prefix, "make")
		header := classHeaderTypeCtx(ctx.currentClass, ctx)
		header.localScope = ctx.localScope
		if !sourceArrayListNativeSourceContext(node, header, h.File.Source) {
			t.Fatal("actual body ownership must precede header-role negative")
		}
		for _, candidate := range []Ctx{ctx, header} {
			if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, candidate); known {
				t.Fatal("header sidecar hid the body's actual inherited Values/Supplier TYPE")
			}
		}
	}
}

func TestSourceArrayListConstructorReferenceCollectorReceiverTypeAuthority(t *testing.T) {
	for _, source := range []string{
		sourceArrayListNativeGuardRoot + `class Use{Object make(){class Stream<T>{}Stream<String> input=null;return input.collect(java.util.stream.Collectors.toCollection(Values<String>::new));}}`,
		sourceArrayListNativeGuardRoot + `class Use{Object make(){class java{static class util{static class stream{static class Stream<T>{}}}}java.util.stream.Stream<String> input=null;return input.collect(java.util.stream.Collectors.toCollection(Values<String>::new));}}`,
	} {
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, source, "make")
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
			t.Fatal("retained local Stream/java TYPE donated canonical terminal receiver")
		}
	}
	for _, source := range []string{
		sourceArrayListNativeGuardRoot + `class Use{Object make(java.util.stream.Stream<String> input){return input.collect(java.util.stream.Collectors.toCollection(Values<String>::new));}}`,
		sourceArrayListNativeGuardRoot + `class Use{java.util.stream.Stream<String> input;Object make(){return input.collect(java.util.stream.Collectors.toCollection(Values<String>::new));}}`,
		sourceArrayListNativeGuardRoot + `class Use{java.util.stream.Stream<String> input(){return null;}Object make(){return input().collect(java.util.stream.Collectors.toCollection(Values<String>::new));}}`,
	} {
		h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, source, "make")
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
			t.Fatal("actual canonical parameter/field/source-return receiver declaration lost Supplier<C>")
		}
	}
}

func TestSourceArrayListConstructorReferenceEmittedCachesCannotDonate(t *testing.T) {
	h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+`class Wrong<E>{}class Use{java.util.function.Supplier<Wrong<String>> make(){return Wrong<String>::new;}}`, "make")
	values := h.Ctx.currentFile.FindClassScope("Values")
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
		t.Fatal("unredirected unsupported Wrong unexpectedly admitted")
	}
	for _, role := range []string{"local", "anonymous"} {
		forged := ctx
		if role == "local" {
			forged.localClasses = map[string]*localClassInfo{"Wrong": {structName: "Wrong", scope: values}}
		} else {
			forged.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{{start: node.StartByte(), end: node.EndByte()}: {structName: "Wrong", scope: values}}
		}
		if resolveClassScopeByQualifiedName(forged, "Wrong") != values {
			t.Fatal("same-T emitted-cache donor premise not active")
		}
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, forged); known {
			t.Fatal("emitted cache redirected actual Wrong declaration to supported Values constructor")
		}
	}
	h, ctx, node = sourceArrayListConstructorReferenceOccurrence(t, sourceArrayListNativeGuardRoot+`class Wrong<E>{}class Use{java.util.function.Supplier<Values<String>> make(){return Values<String>::new;}}`, "make")
	wrong := h.Ctx.currentFile.FindClassScope("Wrong")
	ctx.localClasses = map[string]*localClassInfo{"Values": {structName: "Values", scope: wrong}}
	ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{{start: node.StartByte(), end: node.EndByte()}: {structName: "Values", scope: wrong}}
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
		t.Fatal("stale emitted aliases hid genuine declaration-owned Values positive")
	}
}

func TestSourceArrayListConstructorReferenceOriginalImportNamespace(t *testing.T) {
	for _, imported := range []string{"unsupported", "good"} {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"good/Values.java":        `package good;` + strings.Replace(sourceArrayListNativeGuardRoot, "class Values", "public class Values", 1),
			"unsupported/Values.java": `package unsupported;public class Values<E>{}`,
			"app/Use.java":            `package app;import ` + imported + `.Values;import java.util.function.Supplier;public class Use{Supplier<Values<String>> make(){return Values<String>::new;}}`,
		})
		owner := findQualifiedSourceClass("app.Use")
		ctx = classScopeCtx(owner, ctx)
		ctx.localScope = owner.FindMethodByName("make", nil)
		node := findNode(ctx.localScope.DeclarationNode.ChildByFieldName("body"), "method_reference")
		_, known := lowerSourceArrayListConstructorReference(node, ctx.currentFile.Source, ctx)
		if known != (imported == "good") {
			t.Fatal("unmutated original imported source authority control failed")
		}
		if imported == "unsupported" {
			ctx.currentFile.Imports["Values"] = "good"
			if resolveClassScopeByQualifiedName(ctx, "Values") != findQualifiedSourceClass("good.Values") {
				t.Fatal("same-T import donor premise missing")
			}
			if _, known := lowerSourceArrayListConstructorReference(node, ctx.currentFile.Source, ctx); known {
				t.Fatal("mutable import table donated another source declaration's constructor")
			}
			ctx.currentFile.Imports["Values"] = "unsupported"
		} else {
			saved := ctx.currentFile.Package
			ctx.currentFile.Package = "foreign"
			if _, known := lowerSourceArrayListConstructorReference(node, ctx.currentFile.Source, ctx); known {
				t.Fatal("mutable package table replaced original declaration namespace")
			}
			ctx.currentFile.Package = saved
			targetFile := findFileScopeForClassScope(findQualifiedSourceClass("good.Values"), ctx)
			targetFile.Imports["ArrayList"] = "java.util"
			if _, known := lowerSourceArrayListConstructorReference(node, ctx.currentFile.Source, ctx); known {
				t.Fatal("mutated target declaration imports donated root ABI")
			}
			delete(targetFile.Imports, "ArrayList")
			if _, known := lowerSourceArrayListConstructorReference(node, ctx.currentFile.Source, ctx); !known {
				t.Fatal("restored source namespaces lost genuine positive")
			}
		}
	}
}

func TestSourceArrayListConstructorReferenceOriginalImportRoot(t *testing.T) {
	source := `import java.util.function.*;` + sourceArrayListNativeGuardRoot + `class Use{Supplier<Values<String>> make(){return Values<String>::new;}}`
	h, ctx, node := sourceArrayListConstructorReferenceOccurrence(t, source, "make")
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
		t.Fatal("actual original wildcard import root positive failed")
	}
	graph := symbol.GlobalScope
	foreign := setupParseHelper(t, source)
	symbol.GlobalScope = graph
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
		t.Fatal("actual source graph not restored before root negatives")
	}
	saved := ctx.currentFile.BaseClass
	for _, donor := range []*symbol.ClassScope{nil, foreign.Ctx.currentFile.BaseClass, ctx.currentClass} {
		ctx.currentFile.BaseClass = donor
		if sourceArrayListConstructorFileNamespace(ctx.currentFile) {
			t.Fatal("missing/foreign/non-first BaseClass donated import root")
		}
		if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); known {
			t.Fatal("legacy import entry point escaped original retained ParsedRoot")
		}
	}
	ctx.currentFile.BaseClass = saved
	if _, known := lowerSourceArrayListConstructorReference(node, h.File.Source, ctx); !known {
		t.Fatal("restored original BaseClass lost genuine constructor target")
	}
}

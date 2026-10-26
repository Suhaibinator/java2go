package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
	"testing"
)

func TestDeferredToCollectionExplicitStringFactoryPhysicalControl(t *testing.T) {
	h, ctx, receiver := sourceArrayListA4CollectorInvocation(t, "", "Stream<String>", `Stream.<String>of(first,null,last)`)
	got := sourceArrayListP2GuardText(ParseExpr(receiver, h.File.Source, ctx))
	element := receiverElementJavaTypes(receiver, ctx, h.File.Source)
	t.Logf("EXPLICIT_STRING_ACTUAL_GO=%s ELEMENTS=%v", got, element)
	if len(element) != 1 || !strings.Contains(got, "NewStream[*stdjava.JavaString]") || sourceArrayListP2GuardText(javaTypeStringToGoTypeExpr(element[0], inScopeTypeParameters(ctx), ctx)) != "*stdjava.JavaString" {
		t.Fatal("existing String factory physical control failed before widening proof")
	}
}

func TestDeferredToCollectionExplicitObjectFactoryPhysicalElement(t *testing.T) {
	h, ctx, receiver := sourceArrayListA4CollectorInvocation(t, "", "Stream<Object>", `Stream.<Object>of(first,null,last)`)
	arguments := receiver.ChildByFieldName("type_arguments")
	if arguments == nil || arguments.NamedChildCount() != 1 || arguments.NamedChild(0).Content(h.File.Source) != "Object" {
		t.Fatal("actual written Object type argument missing")
	}
	selected, known := intrinsicStaticClassName(receiver.ChildByFieldName("object"), ctx, h.File.Source)
	if !known || selected != "Stream" {
		t.Fatal("actual static Stream factory selection failed")
	}
	got := sourceArrayListP2GuardText(ParseExpr(receiver, h.File.Source, ctx))
	element := receiverElementJavaTypes(receiver, ctx, h.File.Source)
	t.Logf("EXPLICIT_OBJECT_ACTUAL_GO=%s ELEMENTS=%v", got, element)
	if len(element) != 1 || !strings.Contains(got, "NewStream[any]") || sourceArrayListP2GuardText(javaTypeStringToGoTypeExpr(element[0], inScopeTypeParameters(ctx), ctx)) != "any" {
		t.Fatal("explicit Object Stream factory lost its declared physical element after actual written argument and static factory proof")
	}
}

func explicitStreamFactoryGuardInvocation(t *testing.T, source, ownerName, methodName string) (*ParseHelper, Ctx, *sitter.Node) {
	t.Helper()
	h := setupParseHelper(t, source)
	owner := h.Ctx.currentFile.FindClassScope(ownerName)
	method := owner.FindMethodByName(methodName, nil)
	ctx := classScopeCtx(owner, h.Ctx)
	ctx.localScope = method
	body := method.DeclarationNode.ChildByFieldName("body")
	statement := body.NamedChild(int(body.NamedChildCount()) - 1)
	if statement == nil || statement.Type() != "return_statement" || statement.NamedChildCount() != 1 {
		t.Fatal("actual factory occurrence missing")
	}
	return h, ctx, statement.NamedChild(0)
}

func TestExplicitStreamFactoryCanonicalOwnerGuards(t *testing.T) {
	for _, source := range []string{
		`class Stream{static <T> Object of(Object a,Object b,Object c){return null;}} class Use{static Object run(String first,String last){return Stream.<Object>of(first,null,last);}}`,
		`import other.Stream;class Use{static Object run(String first,String last){return Stream.<Object>of(first,null,last);}}`,
		`import java.util.stream.Stream;class Fake{static <T> Object of(Object a,Object b,Object c){return null;}} class Use{static Fake Stream;static Object run(String first,String last){return Stream.<Object>of(first,null,last);}}`,
		`import java.util.stream.Stream;class Use{static <Stream> Object run(String first,String last){return Stream.<Object>of(first,null,last);}}`,
		`class java{static class util{static class stream{static class Stream{static <T> Object of(Object a,Object b,Object c){return null;}}}}} class Use{static Object run(String first,String last){return java.util.stream.Stream.<Object>of(first,null,last);}}`,
	} {
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
		if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); known {
			t.Fatalf("source/value/import/binder prefix donated canonical Stream authority: %s", source)
		}
	}
	for _, expression := range []string{`java.util.stream.Stream.<Object>of(first,null,last)`, `Stream.<Object>of(first,null,last)`} {
		source := `import java.util.stream.*;class Use{static Object run(String first,String last){return ` + expression + `;}}`
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
		origin, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source)
		if !known || origin.externalName != "java.lang.Object" || !strings.Contains(sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx)), "NewStream[any]") {
			t.Fatal("qualified/on-demand canonical Stream lost written Object")
		}
	}
}

func TestExplicitStreamFactorySourceAuthority(t *testing.T) {
	source := `import java.util.stream.Stream;class Use{static Object run(String first,String last){return Stream.<Object>of(first,null,last);}static Object other(String first,String last){return Stream.<Object>of(first,null,last);}}`
	h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
	if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); !known {
		t.Fatal("attached positive lost source ownership")
	}
	other := ctx
	other.localScope = ctx.currentClass.FindMethodByName("other", nil)
	registered := symbol.GlobalScope
	_, foreignCtx, foreign := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
	symbol.GlobalScope = registered
	if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); !known {
		t.Fatal("original attached graph was not restored before authority negatives")
	}
	for _, row := range []struct {
		node   *sitter.Node
		ctx    Ctx
		source []byte
	}{{node, other, h.File.Source}, {node, foreignCtx, h.File.Source}, {foreign, ctx, h.File.Source}, {node, ctx, append([]byte(nil), h.File.Source...)}, {node, ctx, nil}, {node, ctx, h.File.Source[:1]}} {
		if _, _, known := explicitStreamFactoryElement(row.node, row.ctx, row.source); known {
			t.Fatal("foreign/copied/wrong executable donated written element")
		}
	}
	func() {
		start := int(node.StartByte()) + strings.Index(node.Content(h.File.Source), "Object")
		old := append([]byte(nil), h.File.Source[start:start+len("Object")]...)
		defer copy(h.File.Source[start:start+len(old)], old)
		copy(h.File.Source[start:start+len("String")], []byte("String"))
		if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); known {
			t.Fatal("changed original source slice donated written element")
		}
	}()
	saved := ctx.localScope.DeclarationNode
	ctx.localScope.DeclarationNode = foreignCtx.localScope.DeclarationNode
	if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); known {
		t.Fatal("equal-span rebound method donated written element")
	}
	ctx.localScope.DeclarationNode = saved
}

func TestExplicitStreamFactoryInheritedAndReorderedBinders(t *testing.T) {
	source := `import java.util.stream.Stream;class Parent<A,B>{B value;}class Use<B,A> extends Parent<A,B>{Stream<B> run(){return Stream.<B>of(value,null,value);}}`
	h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
	expected := ctx.currentClass.TypeParameters[0]
	element, written, known := explicitStreamFactoryElement(node, ctx, h.File.Source)
	if !known || element.parameter != expected.Declaration || element.parameter == ctx.currentClass.TypeParameters[1].Declaration {
		t.Fatal("written method element borrowed reordered parent/caller binder")
	}
	var receiver inferredJavaTypeOrigin
	if _, known := inferExprJavaType(node, ctx, h.File.Source, &receiver); !known || len(receiver.arguments) != 1 || receiver.arguments[0].parameter != expected.Declaration {
		t.Fatal("derived receiver lost attached written binder")
	}
	operand := invocationArgumentNode(node, 0)
	var value inferredJavaTypeOrigin
	if _, known := inferExprJavaType(operand, ctx, h.File.Source, &value); !known || !sameJavaTypeOrigin(value, element) {
		t.Fatal("inherited field element was recaptured in child namespace")
	}
	if written != expected.Declaration.SourceName || !strings.Contains(sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx)), "NewStream["+expected.EmittedName()+"]") {
		t.Fatal("physical factory lost actual declaration binder")
	}
}

func TestExplicitStreamFactorySupportedScalarBoundary(t *testing.T) {
	for _, expression := range []string{`Stream.of(first,null,last)`, `Stream.<String[]>of((String[])null,(String[])null)`, `Stream.<java.util.ArrayList<String>>of(null,null,null)`, `Stream.<?>of(null,null,null)`, `Stream.<Object,String>of(first,null,last)`, `Stream.<Missing>of(null,null,null)`} {
		h, ctx, node := sourceArrayListA4CollectorInvocation(t, "", "Object", expression)
		if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); known {
			t.Fatalf("unproved scalar/rank/typearg shape was activated: %s", expression)
		}
	}
}

func TestExplicitStreamFactoryWrittenElementPrecedesExpectedTarget(t *testing.T) {
	h, ctx, node := sourceArrayListA4CollectorInvocation(t, "", "Stream<Object>", `Stream.<Object>of(first,null,last)`)
	ctx.expectedType = "Stream<String>"
	ctx.expectedTypeRoot = node
	if !expectedTypeTargetsExpression(ctx, node) {
		t.Fatal("conflicting exact target control not active")
	}
	got := sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx))
	element := receiverElementJavaTypes(node, ctx, h.File.Source)
	if !strings.Contains(got, "NewStream[any]") || len(element) != 1 || sourceArrayListP2GuardText(javaTypeStringToGoTypeExpr(element[0], inScopeTypeParameters(ctx), ctx)) != "any" {
		t.Fatal("outer target or operand LUB replaced explicit Object")
	}
	// The actual declaration, rather than a spelling-only sidecar, still owns T.
	origin, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source)
	expected := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.lang.Object"}, ctx)
	if !known || !sameJavaTypeOrigin(origin, expected) {
		t.Fatal("written canonical Object origin changed under target context")
	}
}

func TestExplicitStreamFactoryRetainedLocalTypeNamespace(t *testing.T) {
	for _, source := range []string{
		`import java.util.stream.Stream;class Use{static Object run(String first,String last){class Stream{static <T> Object of(Object a,Object b,Object c){return null;}}return Stream.<java.lang.Object>of(first,null,last);}}`,
		`import java.util.stream.Stream;class Use{static java.lang.Object run(String first,String last){class Object{}return Stream.<Object>of(new Object(),null,new Object());}}`,
		`import java.util.stream.Stream;class Use{static Object run(String first,String last){class java{static class util{static class stream{static class Stream{static <T> Object of(Object a,Object b,Object c){return null;}}}}}return java.util.stream.Stream.<String>of(first,null,last);}}`,
		`import java.util.stream.Stream;class Use{static java.lang.Object run(String first,String last){class java{static class lang{static class Object{}}}return Stream.<java.lang.Object>of(new java.lang.Object(),null,new java.lang.Object());}}`,
	} {
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
		if ctx.localClasses != nil {
			t.Fatal("original API local-type cache control was not empty")
		}
		if !sourceArrayListNativeSourceContext(node, ctx, h.File.Source) {
			t.Fatal("actual outer executable ownership failed before local namespace proof")
		}
		if _, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source); known {
			t.Fatalf("retained local TYPE declaration donated native owner/element: %s", source)
		}
	}
	for _, source := range []string{
		`import java.util.stream.Stream;class Use{static Object run(String first,String last){{class Stream{}}return Stream.<Object>of(first,null,last);}}`,
		`import java.util.stream.Stream;class Use{static Object run(String first,String last){class Stream{}return java.util.stream.Stream.<Object>of(first,null,last);}}`,
	} {
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
		element, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source)
		if !known || element.externalName != "java.lang.Object" || !strings.Contains(sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx)), "NewStream[any]") {
			t.Fatal("closed-block or unrelated simple local name shadowed exact native occurrence")
		}
	}
}

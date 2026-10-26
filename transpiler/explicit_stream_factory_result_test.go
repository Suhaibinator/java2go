package transpiler

import (
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestExplicitStreamFactoryResultKeepsActualReorderedBinder(t *testing.T) {
	source := `import java.util.stream.Stream;class Parent<A,B>{B value;}class Use<B,A> extends Parent<A,B>{Stream<B> run(){return Stream.<B>of(value,null,value);}}`
	h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
	expected := ctx.currentClass.TypeParameters[0]
	element, _, known := explicitStreamFactoryElement(node, ctx, h.File.Source)
	physical := sourceArrayListP2GuardText(ParseExpr(node, h.File.Source, ctx))
	if !known || element.parameter != expected.Declaration || !strings.Contains(physical, "NewStream["+expected.EmittedName()+"]") {
		t.Fatal("actual written/physical binder premise failed")
	}
	var actual inferredJavaTypeOrigin
	java, known := inferExprJavaType(node, ctx, h.File.Source, &actual)
	t.Logf("ACTUAL_STREAM_RESULT=%s KNOWN=%t EXTERNAL=%s ARGS=%d PHYSICAL=%s", java, known, actual.externalName, len(actual.arguments), physical)
	if !known || actual.externalName != "java.util.stream.Stream" || actual.nominalScope != nil || actual.parameter != nil || actual.arrayRank != 0 || len(actual.arguments) != 1 || actual.arguments[0].parameter != expected.Declaration {
		t.Fatal("EXPLICIT_STREAM_STRUCTURED_RESULT_LOST_AFTER_WRITTEN_PHYSICAL_BINDER_PROOF")
	}
	var inherited inferredJavaTypeOrigin
	if _, known := inferExprJavaType(invocationArgumentNode(node, 0), ctx, h.File.Source, &inherited); !known || !sameJavaTypeOrigin(inherited, actual.arguments[0]) {
		t.Fatal("inherited field result was recaptured by reordered child binder")
	}
	if actual.arguments[0].parameter == ctx.currentClass.TypeParameters[1].Declaration {
		t.Fatal("other same-family binder donated result origin")
	}
}

func TestExplicitStreamFactoryResultCanonicalScalarsAndOuterTarget(t *testing.T) {
	for _, item := range []string{"java.lang.String", "java.lang.Object", "Item"} {
		source := `class Item{}class Use{Object run(){return java.util.stream.Stream.<` + item + `>of(null,null,null);}}`
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
		expected := declaredJavaTypeOrigin(symbol.JavaType{Original: item}, ctx)
		ctx.expectedType = "java.util.stream.Stream<Missing>"
		ctx.expectedTypeRoot = node
		var actual inferredJavaTypeOrigin
		java, known := inferExprJavaType(node, ctx, h.File.Source, &actual)
		if !known || !strings.HasPrefix(java, "java.util.stream.Stream<") || actual.externalName != "java.util.stream.Stream" || len(actual.arguments) != 1 || !sameJavaTypeOrigin(actual.arguments[0], expected) {
			t.Fatalf("actual canonical scalar replaced by outer target: %s", item)
		}
	}
}

func TestExplicitStreamFactoryResultSourceAuthority(t *testing.T) {
	source := `import java.util.stream.Stream;class Use{Object run(){return Stream.<Object>of(null,null,null);}Object other(){return Stream.<Object>of(null,null,null);}}`
	h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
	if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source); !known {
		t.Fatal("actual attached result positive failed")
	}
	graph := symbol.GlobalScope
	_, other, foreign := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
	symbol.GlobalScope = graph
	if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source); !known {
		t.Fatal("actual graph not restored before negatives")
	}
	wrong := ctx
	wrong.localScope = ctx.currentClass.FindMethodByName("other", nil)
	for _, row := range []struct {
		node   *sitter.Node
		ctx    Ctx
		source []byte
	}{{node, wrong, h.File.Source}, {node, other, h.File.Source}, {foreign, ctx, h.File.Source}, {node, ctx, nil}, {node, ctx, h.File.Source[:1]}, {node, ctx, append([]byte(nil), h.File.Source...)}} {
		sentinel := inferredJavaTypeOrigin{externalName: "sentinel"}
		if _, known := explicitStreamFactoryResultType(row.node, row.ctx, row.source, &sentinel); known || sentinel.externalName != "sentinel" {
			t.Fatal("foreign/copy/truncated/wrong executable donated structured result or changed output on refusal")
		}
	}
	saved := ctx.localScope.DeclarationNode
	ctx.localScope.DeclarationNode = other.localScope.DeclarationNode
	if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source); known {
		t.Fatal("equal-span rebound executable donated result")
	}
	ctx.localScope.DeclarationNode = saved
}

func TestExplicitStreamFactoryResultOriginalNamespace(t *testing.T) {
	for _, source := range []string{
		`import java.util.stream.Stream;class Use{Object run(){class Stream<T>{static <T> Object of(Object a,Object b,Object c){return null;}}return Stream.<Object>of(null,null,null);}}`,
		`import java.util.stream.Stream;class Use{Object run(){class Object{}return Stream.<Object>of(null,null,null);}}`,
		`import java.util.stream.Stream;class Parent{static class Stream{static <T> Object of(Object a,Object b,Object c){return null;}}}class Use extends Parent{Object run(){return Stream.<Object>of(null,null,null);}}`,
	} {
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, source, "Use", "run")
		header := classHeaderTypeCtx(ctx.currentClass, ctx)
		header.localScope = ctx.localScope
		if _, known := explicitStreamFactoryResultType(node, header, h.File.Source); known {
			t.Fatal("original local/inherited owner/element namespace donated canonical Stream result")
		}
	}
	h, ctx, node := explicitStreamFactoryGuardInvocation(t, `import foreign.Stream;class Use{Object run(){return Stream.<Object>of(null,null,null);}}`, "Use", "run")
	ctx.currentFile.Imports["Stream"] = "java.util.stream"
	if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source); known {
		t.Fatal("mutable import metadata donated native Stream result")
	}
	h, ctx, node = explicitStreamFactoryGuardInvocation(t, `import java.util.stream.Stream;class Shadow{}class Use{Object run(){return Stream.<Object>of(null,null,null);}}`, "Use", "run")
	shadow := h.Ctx.currentFile.FindClassScope("Shadow")
	ctx.localClasses = map[string]*localClassInfo{"Stream": {structName: "Stream", scope: shadow}}
	ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{{start: node.StartByte(), end: node.EndByte()}: {structName: "Stream", scope: shadow}}
	if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source); !known {
		t.Fatal("stale emitted caches hid actual native declaration")
	}
	ctx.currentFile.BaseClass = nil
	if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source); known {
		t.Fatal("missing original import root donated native result")
	}
}

func TestExplicitStreamFactoryResultUnknownFallbackIsUntouched(t *testing.T) {
	for _, expression := range []string{`Stream.of(null,null)`, `Stream.<String[]>of(null,null)`, `Stream.<java.util.ArrayList<String>>of(null,null)`, `Stream.<Missing>of(null,null)`, `Stream.<String,Object>of(null,null)`} {
		h, ctx, node := explicitStreamFactoryGuardInvocation(t, `import java.util.stream.Stream;class Use{Object run(){return `+expression+`;}}`, "Use", "run")
		sentinel := inferredJavaTypeOrigin{externalName: "sentinel"}
		if _, known := explicitStreamFactoryResultType(node, ctx, h.File.Source, &sentinel); known || sentinel.externalName != "sentinel" {
			t.Fatalf("unproved factory changed structured result: %s", expression)
		}
	}
}

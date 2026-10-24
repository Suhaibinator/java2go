package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/format"
	"go/token"
	"strings"
	"testing"
)

func TestArrayMethodFrontierStaticProof(t *testing.T) {
	for _, row := range []struct {
		source, wantType string
		absent           bool
	}{
		{`class Probe{Thread[] values;void run(){values.interrupt();}}`, "Thread[]", true},
		{`class Probe{void run(int[] values){values.missing();}}`, "int[]", true},
		{`class Component{void missing(){}}class Probe{Component[] values;void run(){values.missing();}}`, "Component[]", true},
		{`class Probe<T>{T[][] values;void run(){values[0].missing();}}`, "T[]", true},
		{`class Probe{Object values;void run(){((int[])values).missing();}}`, "int[]", true},
		{`class Probe{void run(){new int[1].missing();}}`, "int[]", true},
		{`class Probe{int[] values(){return null;}void run(){values().missing();}}`, "int[]", true},
		{`class Component{void missing(){}}class Probe{Component[] values;void run(){values[0].missing();}}`, "Component", false},
		{`class Component{void missing(){}}class Probe<T extends Component>{T value;void run(){value.missing();}}`, "T", false},
	} {
		t.Run(row.source, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if node == nil {
				t.Fatal("missing actual Java invocation")
			}
			actual, known := inferExprJavaType(node.ChildByFieldName("object"), ctx, h.File.Source)
			if !known || actual != row.wantType {
				t.Fatalf("static receiver proof=%q/%v want %q", actual, known, row.wantType)
			}
			if got := knownArrayMethodIsAbsent(node, ctx, h.File.Source); got != row.absent {
				t.Fatalf("array absence=%v want %v", got, row.absent)
			}
		})
	}
}

func TestArrayMethodFrontierConsumerCannotSupplyArrayType(t *testing.T) {
	h := setupParseHelper(t, `class Probe{static<T>T choose(){return null;}void run(){choose().missing();}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
	ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
	node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	ctx.expectedType = "int[]"
	ctx.expectedTypeRoot = node.ChildByFieldName("object")
	contaminated, contaminatedKnown := inferExprJavaType(node.ChildByFieldName("object"), ctx, h.File.Source)
	if _, rank := javaArrayTypeParts(contaminated); !contaminatedKnown || rank == 0 {
		t.Fatalf("target-typing control did not expose contamination: %q/%v", contaminated, contaminatedKnown)
	}
	if knownArrayMethodIsAbsent(node, ctx, h.File.Source) {
		t.Fatal("assignment consumer fabricated a static array receiver")
	}
	if ctx.expectedType != "int[]" || ctx.expectedTypeRoot != node.ChildByFieldName("object") {
		t.Fatal("proof changed caller context")
	}
	unknown := Ctx{}
	if actual, known := inferExprJavaType(node.ChildByFieldName("object"), unknown, h.File.Source); known {
		t.Fatalf("unknown donor/default context unexpectedly proved %q", actual)
	}
	if knownArrayMethodIsAbsent(node, unknown, h.File.Source) {
		t.Fatal("unknown context fabricated array proof")
	}
}

func TestArrayMethodFrontierPreservesObjectNames(t *testing.T) {
	for _, element := range []string{"int", "Thread", "Component", "T", "int[]"} {
		for _, method := range []string{"clone", "getClass", "equals", "hashCode", "toString", "notify", "notifyAll", "wait", "finalize"} {
			h := setupParseHelper(t, `class Component{}class Probe<T>{`+element+`[] values;void run(){values.`+method+`();}}`)
			ctx := h.Ctx.Clone()
			ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			actual, known := inferExprJavaType(node.ChildByFieldName("object"), ctx, h.File.Source)
			_, rank := javaArrayTypeParts(actual)
			if !known || rank == 0 {
				t.Fatalf("missing positive static array proof for %s.%s", element, method)
			}
			if knownArrayMethodIsAbsent(node, ctx, h.File.Source) {
				t.Fatalf("known Object/clone name changed route: %s.%s", element, method)
			}
		}
	}
}

func TestArrayMethodFrontierDiagnostics(t *testing.T) {
	for _, row := range []struct{ source, invocation string }{
		{`class Probe{Thread[] values;void run(){values.interrupt();}}`, "values.interrupt()"},
		{`class Component{void missing(){}}class Probe{Component[] values;void run(){values.missing();}}`, "values.missing()"},
		{`class Probe{void run(int[] values){class Local{void read(){values.missing();}}}}`, "values.missing()"},
		{`interface Callback{void read();}class Probe{Callback run(int[] values){return new Callback(){public void read(){values.missing();}};}}`, "values.missing()"},
		{`class Probe{String[] values;void run(){values.charAt(0);}}`, "values.charAt(0)"},
		{`class Probe{Throwable[] values;void run(){values.getMessage();}}`, "values.getMessage()"},
		{`class Probe{java.io.Reader[] values;void run(){values.read();}}`, "values.read()"},
		{`abstract class Component extends java.io.Reader{}class Probe{Component[] values;void run(){values.read();}}`, "values.read()"},
		{`class Probe{java.util.List<?>[] values;void run(){values.size();}}`, "values.size()"},
	} {
		source := row.source
		for _, strict := range []bool{false, true} {
			t.Run(source+":"+map[bool]string{true: "strict", false: "permissive"}[strict], func(t *testing.T) {
				diagnostics.mu.Lock()
				previousStrict, previousItems := diagnostics.strict, diagnostics.items
				diagnostics.strict, diagnostics.items = strict, nil
				diagnostics.mu.Unlock()
				defer func() {
					diagnostics.mu.Lock()
					diagnostics.strict, diagnostics.items = previousStrict, previousItems
					diagnostics.mu.Unlock()
				}()
				var output string
				var failure any
				func() { defer func() { failure = recover() }(); output = renderGoFileFromJava(t, source) }()
				items := Diagnostics()
				if len(items) != 1 || items[0].Kind != "expression" || items[0].NodeType != "method_invocation" || items[0].Message != row.invocation {
					t.Fatalf("absence must report exactly original invocation once: %v", items)
				}
				if strict {
					got, ok := failure.(strictModeError)
					if !ok || got.diagnostic != items[0] {
						t.Fatalf("strict failure=%v want exact diagnostic %v", failure, items[0])
					}
				} else if failure != nil || !strings.Contains(output, "panic(") || !strings.Contains(output, "UNSUPPORTED: expression") {
					t.Fatalf("permissive shared placeholder=%v output=%s", failure, output)
				}
			})
		}
	}
}

func TestArrayMethodFrontierUnknownAndNonArrayKeepFallback(t *testing.T) {
	for _, source := range []string{`class Probe{void run(){unknown.missing();}}`, `class Probe{missing.Provider source;void run(){source.values().missing();}}`, `class Component{void missing(){}}class Probe{Component value;void run(){value.missing();}}`} {
		h := setupParseHelper(t, source)
		ctx := h.Ctx.Clone()
		ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
		ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
		node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
		if knownArrayMethodIsAbsent(node, ctx, h.File.Source) {
			t.Fatal("non-array/unknown route changed")
		}
		var out bytes.Buffer
		if err := format.Node(&out, token.NewFileSet(), ParseExpr(node, h.File.Source, ctx)); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "panic(") {
			t.Fatalf("non-array fallback rejected: %s", out.String())
		}
	}
}

func TestArrayMethodFrontierDonorArrayRank(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	donor := parsing.SourceFile{Name: "donor/Provider.java", Source: []byte(`package donor;public class Provider<T>{public T[] values(){return null;}public T item(){return null;}}`)}
	consumer := parsing.SourceFile{Name: "consumer/Probe.java", Source: []byte(`package consumer;class Component{void missing(){}}class Probe{donor.Provider<Component> source;void run(){source.values().missing();}void scalar(){source.item().missing();}}`)}
	for _, file := range []*parsing.SourceFile{&donor, &consumer} {
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
	}
	ResolveFile(donor)
	ResolveFile(consumer)
	ctx := Ctx{currentFile: consumer.Symbols, currentClass: consumer.Symbols.FindClassScope("Probe")}
	for _, row := range []struct {
		method string
		rank   int
		absent bool
	}{{"run", 1, true}, {"scalar", 0, false}} {
		ctx.localScope = ctx.currentClass.FindMethodByName(row.method, nil)
		node := findNode(ctx.localScope.DeclarationNode, "method_invocation")
		javaType, known := inferExprJavaType(node.ChildByFieldName("object"), ctx, consumer.Source)
		_, rank := javaArrayTypeParts(javaType)
		if !known || rank != row.rank {
			t.Fatalf("donor %s static proof=%q/%v rank%d want %d", row.method, javaType, known, rank, row.rank)
		}
		if knownArrayMethodIsAbsent(node, ctx, consumer.Source) != row.absent {
			t.Fatalf("donor %s lost array/non-array method distinction", row.method)
		}
	}
}

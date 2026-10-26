package transpiler

import (
	"bytes"
	"go/format"
	"go/token"
	"reflect"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func scalarOrigin3931Call(t *testing.T, sources map[string]string) (Ctx, *sitter.Node, []byte) {
	t.Helper()
	ctx := ordinaryDemandTestContext(t, sources, "app/Main.java")
	methods := ctx.currentClass.FindMethod().ByOriginalName("read")
	if len(methods) != 1 {
		t.Fatal("actual read declaration missing")
	}
	ctx.localScope = methods[0]
	var found *sitter.Node
	var visit func(*sitter.Node)
	visit = func(n *sitter.Node) {
		if n.Type() == "method_invocation" && (n.ChildByFieldName("name").Content(ctx.currentFile.Source) == "charAt" || n.ChildByFieldName("name").Content(ctx.currentFile.Source) == "intValue") {
			found = n
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(methods[0].DeclarationNode)
	if found == nil {
		t.Fatal("actual charAt invocation missing")
	}
	return ctx, found, ctx.currentFile.Source
}

func scalarOrigin3931Sources(provider, expression string) map[string]string {
	return map[string]string{
		"model/Provider.java": "package model; public class Provider" + provider,
		"foreign/Tokens.java": "package foreign; public class Tokens { public static class StringBuilder {} public static class StringBuffer {} }",
		"app/Main.java":       "package app; import foreign.Tokens.StringBuilder; import foreign.Tokens.StringBuffer; public class Main { static class java {static class lang {static class StringBuilder {} static class StringBuffer {}}} int read(model.Provider p) {return " + expression + ".charAt(0);} }",
	}
}

func TestNativeScalarDeclarationOrigin3931FieldsAndResults(t *testing.T) {
	for _, family := range []string{"StringBuilder", "StringBuffer"} {
		for _, kind := range []string{"field", "result"} {
			t.Run(family+kind, func(t *testing.T) {
				member := "{ public java.lang." + family + " field; public java.lang." + family + " result(){return field;} }"
				expression := "p.field"
				if kind == "result" {
					expression = "p.result()"
				}
				ctx, invocation, source := scalarOrigin3931Call(t, scalarOrigin3931Sources(member, expression))
				receiver := invocation.ChildByFieldName("object")
				donor := findQualifiedSourceClass("model.Provider")
				var origin inferredJavaTypeOrigin
				display, known := inferExprJavaType(receiver, ctx, source, &origin)
				if !known || origin.externalName != "java.lang."+family || origin.declaringOwner != donor || origin.nominalScope != nil || origin.parameter != nil || origin.unresolved || origin.arrayRank != 0 || origin.wildcard || len(origin.arguments) != 0 {
					t.Fatalf("actual origin lost: display=%s origin=%+v", display, origin)
				}
				proof := ctx.Clone()
				proof.currentFile = nil
				proof.syntheticTypeParameters = nil
				declaring := classScopeCtx(donor, proof)
				if declaring.currentFile == nil || findFileScopeForClassScope(donor, proof) != declaring.currentFile {
					t.Fatal("actual donor/file certificate missing")
				}
				if resolveClassScopeByQualifiedName(ctx, display) == nil {
					t.Fatal("caller TYPE-root collision absent")
				}
				before := ctx.Clone()
				imports := map[string]string{}
				for name, pkg := range ctx.currentFile.Imports {
					imports[name] = pkg
				}
				for repeat := 0; repeat < 3; repeat++ {
					selected, ok := intrinsicReceiverTypeName(receiver, ctx, source)
					if !ok || selected != family {
						t.Fatalf("certified donor receiver refused: got=%q/%v want=%s", selected, ok, family)
					}
					if !reflect.DeepEqual(ctx, before) || !reflect.DeepEqual(ctx.currentFile.Imports, imports) {
						t.Fatal("origin selection mutated caller context/imports")
					}
				}
				expr, rewritten := tryInstanceIntrinsic(receiver, "charAt", source, ctx)
				var rendered bytes.Buffer
				if !rewritten {
					t.Fatal("actual charAt callback declined")
				}
				if err := format.Node(&rendered, token.NewFileSet(), expr); err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(rendered.String(), ".CharAt(0)") || strings.Contains(rendered.String(), ".charAt(") {
					t.Fatalf("actual callback selector=%s", rendered.String())
				}
			})
		}
	}
}

func TestNativeScalarDeclarationOrigin3931SourceCapturedAndArrayShields(t *testing.T) {
	cases := []struct {
		name, provider, expression string
		array                      bool
		captured                   bool
	}{
		{"sourceSimple", "{public static class StringBuilder { public char charAt(int i){return 's';}} public StringBuilder field;}", "p.field", false, false},
		{"sourceQualified", "{public static class java {public static class lang {public static class StringBuilder {public char charAt(int i){return 's';}}}} public java.lang.StringBuilder field;}", "p.field", false, false},
		{"captured", "<StringBuilder> {public StringBuilder field;}", "p.field", false, true},
		{"array", "{public java.lang.StringBuilder[] field;}", "p.field", true, false},
		{"unknown", "{public java.io.UnknownReader field;}", "p.field", false, false},
	}
	for _, row := range cases {
		t.Run(row.name, func(t *testing.T) {
			ctx, invocation, source := scalarOrigin3931Call(t, scalarOrigin3931Sources(row.provider, row.expression))
			receiver := invocation.ChildByFieldName("object")
			var origin inferredJavaTypeOrigin
			_, known := inferExprJavaType(receiver, ctx, source, &origin)
			if !known {
				t.Fatal("actual shield origin absent")
			}
			if row.captured {
				donor := findQualifiedSourceClass("model.Provider")
				if origin.parameter != donor.TypeParameters[0].Declaration || origin.externalName != "" {
					t.Fatalf("captured donor identity lost: %+v", origin)
				}
			}
			key, ok := intrinsicReceiverTypeName(receiver, ctx, source)
			if row.array {
				if !ok || key != "Object" {
					t.Fatalf("array inheritance changed: %q/%v", key, ok)
				}
			} else if key == "StringBuilder" || key == "StringBuffer" {
				t.Fatalf("non-native donor acquired legacy intrinsic: %+v -> %q/%v", origin, key, ok)
			}
			if _, rewritten := tryInstanceIntrinsic(receiver, "charAt", source, ctx); rewritten {
				t.Fatal("shielded actual charAt acquired native callback")
			}
		})
	}
}

func TestNativeScalarDeclarationOrigin3931RegisteredOwner(t *testing.T) {
	sources := scalarOrigin3931Sources("{public java.lang.Integer field;}", "p.field")
	sources["app/Main.java"] = strings.ReplaceAll(strings.ReplaceAll(sources["app/Main.java"], "static class StringBuilder {}", "static class Integer {}"), ".charAt(0)", ".intValue()")
	ctx, invocation, source := scalarOrigin3931Call(t, sources)
	receiver := invocation.ChildByFieldName("object")
	var origin inferredJavaTypeOrigin
	_, known := inferExprJavaType(receiver, ctx, source, &origin)
	if !known || origin.externalName != "java.lang.Integer" || origin.declaringOwner != findQualifiedSourceClass("model.Provider") || origin.nominalScope != nil || origin.parameter != nil {
		t.Fatalf("registered actual donor origin lost: %+v", origin)
	}
	if !intrinsicOwners["Integer"][origin.externalName] {
		t.Fatal("expected exact registered owner missing")
	}
	key, ok := intrinsicReceiverTypeName(receiver, ctx, source)
	if !ok || key != intrinsicOwnerKey(origin.externalName) {
		t.Fatalf("registered certified origin refused: %q/%v", key, ok)
	}
}

package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func ci205RequireCanonicalFormal(t *testing.T, formal inferredJavaTypeOrigin, written string) {
	t.Helper()
	component, rank := javaArrayTypeParts(written)
	base, arguments := parseJavaTypeString(component)
	if formal.unresolved || formal.parameter != nil || formal.nominalScope != nil || formal.externalName != base || formal.arrayRank != rank || len(formal.arguments) != len(arguments) {
		t.Fatalf("canonical formal premise lost: %#v", formal)
	}
	for i, argument := range arguments {
		got := formal.arguments[i]
		if got.unresolved || got.parameter != nil || got.nominalScope != nil || got.externalName != argument || got.arrayRank != 0 || len(got.arguments) != 0 {
			t.Fatalf("canonical argument premise lost: %#v", got)
		}
	}
}

func TestCI205BoxedComparableDeclaredOrigin(t *testing.T) {
	h := setupParseHelper(t, `class Probe{}`)
	for _, tc := range []struct {
		name, primitive, formal string
		want                    bool
	}{
		{"int_raw", "int", "java.lang.Comparable", true},
		{"int_typed", "int", "java.lang.Comparable<java.lang.Integer>", true},
		{"long_typed", "long", "java.lang.Comparable<java.lang.Long>", true},
		{"boolean_typed", "boolean", "java.lang.Comparable<java.lang.Boolean>", true},
		{"char_typed", "char", "java.lang.Comparable<java.lang.Character>", true},
		{"wrong_wrapper", "int", "java.lang.Comparable<java.lang.Long>", false},
		{"invariant_supertype", "int", "java.lang.Comparable<java.lang.Number>", false},
		{"array_formal", "int", "java.lang.Comparable<java.lang.Integer>[]", false},
		{"ordinary_boxing", "int", "java.lang.Integer", true},
		{"forbidden_widening_then_boxing", "int", "java.lang.Long", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			formal := declaredJavaTypeOrigin(symbol.JavaType{Original: tc.formal}, h.Ctx)
			ci205RequireCanonicalFormal(t, formal, tc.formal)
			if _, ok := javaPrimitiveType(tc.primitive); !ok {
				t.Fatal("primitive premise lost")
			}
			cost, exact, applies, handled := declaredOriginConversion(inferredJavaTypeOrigin{javaType: tc.primitive}, formal, h.Ctx)
			if applies != tc.want || tc.want && (!handled || exact || cost < 2) {
				t.Fatalf("boxed conversion=(%d,%v,%v,%v), want loose=%v", cost, exact, applies, handled, tc.want)
			}
		})
	}
}

func TestCI205BoxedComparableDeclarationRefusals(t *testing.T) {
	for _, tc := range []struct{ name, source, formal string }{
		{"source_interface", `interface Comparable<T>{} class Probe{}`, "Comparable<java.lang.Integer>"},
		{"source_wrapper_argument", `class Integer{} class Probe{}`, "java.lang.Comparable<Integer>"},
		{"source_qualified_prefix", `class java{static class lang{interface Comparable<T>{}}}class Probe{}`, "java.lang.Comparable<Integer>"},
		{"foreign_interface", `class Probe{}`, "foreign.Comparable<java.lang.Integer>"},
		{"bound_caller_binder", `class Probe<Comparable extends java.lang.Comparable<java.lang.Integer>>{}`, "Comparable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, tc.source)
			probe := h.File.Symbols.FindClassScope("Probe")
			ctx := classScopeCtx(probe, h.Ctx)
			formal := declaredJavaTypeOrigin(symbol.JavaType{Original: tc.formal, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
			switch tc.name {
			case "source_interface", "source_qualified_prefix":
				base, _ := parseJavaTypeString(tc.formal)
				scope := resolveClassScopeByQualifiedName(ctx, base)
				if scope == nil || formal.nominalScope != scope || formal.parameter != nil || formal.externalName != "" || formal.unresolved || formal.arrayRank != 0 {
					t.Fatal("exact source Comparable premise lost")
				}
			case "source_wrapper_argument":
				integer := h.File.Symbols.FindClassScope("Integer")
				if integer == nil || formal.externalName != "java.lang.Comparable" || formal.unresolved || formal.nominalScope != nil || formal.parameter != nil || formal.arrayRank != 0 || len(formal.arguments) != 1 || formal.arguments[0].nominalScope != integer || formal.arguments[0].unresolved || formal.arguments[0].externalName != "" {
					t.Fatal("exact source Integer argument premise lost")
				}
			case "foreign_interface":
				if !formal.unresolved || formal.parameter != nil || formal.nominalScope != nil || formal.externalName != "" {
					t.Fatal("unknown foreign declaration premise lost")
				}
			case "bound_caller_binder":
				if len(probe.TypeParameters) != 1 || probe.TypeParameters[0].Declaration == nil || formal.parameter != probe.TypeParameters[0].Declaration || formal.unresolved || formal.nominalScope != nil || formal.externalName != "" || formal.arrayRank != 0 {
					t.Fatal("exact caller binder premise lost")
				}
			}
			if _, _, applies, _ := declaredOriginConversion(inferredJavaTypeOrigin{javaType: "int"}, formal, ctx); applies {
				t.Fatal("primitive boxing acquired a source, foreign, or caller-binder declaration")
			}
		})
	}
}

func TestCI205BoxedComparableMethodSelection(t *testing.T) {
	for _, formal := range []string{"Comparable", "Comparable<Integer>"} {
		t.Run(formal, func(t *testing.T) {
			source := `interface Action{Integer apply(` + formal + ` receiver,Integer other);}class Probe{static Integer run(Action action){return action.apply(3,2);}}`
			h := setupParseHelper(t, source)
			scope := h.File.Symbols.FindClassScope("Probe")
			ctx := classScopeCtx(scope, h.Ctx)
			ctx.localScope = scope.FindMethod().ByOriginalName("run")[0]
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if call == nil || call.ChildByFieldName("name").Content(h.File.Source) != "apply" {
				t.Fatal("actual apply invocation premise lost")
			}
			object := call.ChildByFieldName("object")
			action := h.File.Symbols.FindClassScope("Action")
			if object == nil || object.Content(h.File.Source) != "action" || action == nil || !action.IsInterface {
				t.Fatal("source Action receiver premise lost")
			}
			target := resolveInvocationTarget(object, ctx, h.File.Source)
			if target == nil || target.classScope != action {
				t.Fatal("exact Action invocation target premise lost")
			}
			methods := action.FindMethod().ByOriginalName("apply")
			if len(methods) != 1 || methods[0].DeclarationNode == nil || len(methods[0].Parameters) != 2 {
				t.Fatal("exact declared apply method premise lost")
			}
			formals := methodFormalOrigins(methods[0], action, nil, ctx)
			if len(formals) != 2 {
				t.Fatal("apply formal count premise lost")
			}
			canonical := "java.lang.Comparable"
			if formal != "Comparable" {
				canonical += "<java.lang.Integer>"
			}
			ci205RequireCanonicalFormal(t, formals[0], canonical)
			ci205RequireCanonicalFormal(t, formals[1], "java.lang.Integer")
			selected, _ := findBestMethodForInvocationTarget(target, "apply", call.ChildByFieldName("arguments"), true, true, ctx, h.File.Source)
			if selected == nil || selected.owner != action || selected.def != methods[0] {
				t.Fatal("boxed Comparable declaration selection absent")
			}
		})
	}
}

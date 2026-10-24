package transpiler

import (
	"go/ast"
	"testing"
)

func TestDeclaredStreamResultShellPreservesElementIdentity(t *testing.T) {
	for _, tc := range []struct{ name, declarations, typeParameters, receiver, owner, element string }{
		{"factory-source-shadow", "static class Stream {}", "", `java.util.stream.Stream.of("x")`, "java.util.stream.Stream", "java.lang.String"},
		{"factory-live-binder-element", "static class Stream {}", "<Stream>", `java.util.stream.Stream.of(item)`, "java.util.stream.Stream", "item"},
		{"factory-source-element", "static class String {} static String item;", "", `java.util.stream.Stream.of(item)`, "java.util.stream.Stream", "item"},
		{"array-source-shadow", "static class Stream {} static java.lang.String[] items;", "", `java.util.Arrays.stream(items)`, "java.util.stream.Stream", "java.lang.String"},
		{"primitive-array-source-shadow", "static class IntStream {} static int[] items;", "", `java.util.Arrays.stream(items)`, "java.util.stream.IntStream", ""},
		{"collection-source-shadow", "static class Stream {} static java.util.List<java.lang.String> items;", "", `items.stream()`, "java.util.stream.Stream", "java.lang.String"},
		{"optional-source-shadow", "static class Stream {} static java.util.Optional<java.lang.String> item;", "", `item.stream()`, "java.util.stream.Stream", "java.lang.String"},
		{"mapped-source-shadow", "static class Stream {}", "", `java.util.stream.IntStream.of(1).mapToObj(value -> "x")`, "java.util.stream.Stream", "java.lang.String"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parameters := ""
			if tc.typeParameters != "" {
				parameters = "Stream item"
			}
			helper := setupParseHelper(t, "class StreamShellProbe {"+tc.declarations+" static "+tc.typeParameters+" boolean run("+parameters+"){return "+tc.receiver+".anyMatch(value -> true);}}")
			methods := helper.Ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("missing run declaration")
			}
			ctx := helper.Ctx.Clone()
			ctx.localScope = methods[0]
			invocation := findNode(helper.File.Ast, "method_invocation")
			receiver := invocation.ChildByFieldName("object")
			got, known := inferExprJavaType(receiver, ctx, helper.File.Source)
			owner, elements := parseJavaTypeString(got)
			if !known || owner != tc.owner {
				t.Fatalf("declared result shell = %q/%t, want %s", got, known, tc.owner)
			}
			expectedElement := tc.element
			if expectedElement == "item" {
				var ok bool
				expectedElement, ok = inferIdentifierJavaType("item", ctx)
				if !ok {
					t.Fatal("missing declared element identity")
				}
			}
			if expectedElement == "" {
				if len(elements) != 0 {
					t.Fatalf("primitive stream acquired elements %v", elements)
				}
			} else if len(elements) != 1 || elements[0] != expectedElement {
				t.Fatalf("element identity = %v, want %q", elements, expectedElement)
			}
			if result, ok := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source); !ok || result != "boolean" {
				t.Fatalf("canonical match result = %q/%t, want boolean", result, ok)
			}
		})
	}
}

func TestInvocationReceiverBinderPrecedesSourceClass(t *testing.T) {
	for _, tc := range []struct {
		name, declarations, typeParameters, parameter, expected string
		witness                                                 bool
	}{
		{"same-name", "static class Stream extends Counter {}", "<Stream extends Counter>", "Stream", "Counter", true},
		{"renamed-control", "static class Stream extends Counter {}", "<Value extends Counter>", "Value", "Counter", true},
		{"transitive-collision", "static class Stream extends Counter {}", "<Stream extends Counter, Value extends Stream>", "Value", "Counter", true},
		{"source-nominal-control", "static class Stream extends Counter {}", "", "Stream", "Stream", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, "class BoundReceiverProbe {static class Counter {String count(){return \"source\";}} "+tc.declarations+" static "+tc.typeParameters+" String run("+tc.parameter+" value){return value.count();}}")
			methods := helper.Ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("missing run declaration")
			}
			ctx := helper.Ctx.Clone()
			ctx.localScope = methods[0]
			ctx.dependentTypeWitnesses = planConcreteDependentTypeWitnesses(ctx.localScope, helper.File.Source, ctx)
			invocation := findNode(helper.File.Ast, "method_invocation")
			receiver := invocation.ChildByFieldName("object")
			target := resolveInvocationTarget(receiver, ctx, helper.File.Source)
			if target == nil || target.classScope == nil || target.classScope.Class.OriginalName != tc.expected {
				t.Fatalf("receiver target = %#v, want declared %s view", target, tc.expected)
			}
			resolution, _ := findInstanceMethodForInvocationTarget(target, "count", 0, ctx)
			if resolution == nil || resolution.owner == nil || resolution.owner.Class.OriginalName != "Counter" {
				t.Fatal("inherited count declaration was not selected")
			}
			projected := projectDependentTypeParameterReceiver(&ast.Ident{Name: "value"}, receiver, target.classScope, ctx, helper.File.Source)
			_, usesWitness := projected.(*ast.CallExpr)
			if usesWitness != tc.witness {
				t.Fatalf("bound projection uses hidden witness = %t, want %t", usesWitness, tc.witness)
			}
			if result, ok := inferIntrinsicMethodResultType(invocation, ctx, helper.File.Source); ok {
				t.Fatalf("source count captured intrinsic %q", result)
			}
		})
	}
}

func TestSourceStreamMatchDeclinesIntrinsic(t *testing.T) {
	helper := setupParseHelper(t, `class SourceMatchProbe {static class Stream {String anyMatch(java.util.function.Predicate<String> predicate){return "source";}} static Stream value; static String run(){return value.anyMatch(item -> true);}}`)
	invocation := findNode(helper.File.Ast, "method_invocation")
	if result, ok := inferIntrinsicMethodResultType(invocation, helper.Ctx, helper.File.Source); ok {
		t.Fatalf("source match captured intrinsic %q", result)
	}
	if result, ok := inferExprJavaType(invocation, helper.Ctx, helper.File.Source); !ok || !isBuiltinJavaString(result, helper.Ctx) {
		t.Fatalf("source match declaration result = %q/%t", result, ok)
	}
}

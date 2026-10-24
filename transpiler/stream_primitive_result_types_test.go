package transpiler

import (
	"testing"
)

func TestStreamPrimitiveTerminalResultTypes(t *testing.T) {
	for _, stream := range []string{"Stream<java.lang.String>", "IntStream", "LongStream", "DoubleStream"} {
		for _, terminal := range []struct{ name, args, result string }{
			{"count", "", "long"},
			{"anyMatch", "value -> true", "boolean"},
			{"allMatch", "value -> true", "boolean"},
			{"noneMatch", "value -> false", "boolean"},
		} {
			t.Run(stream+"-"+terminal.name, func(t *testing.T) {
				helper := setupParseHelper(t, "class TerminalTypes {static java.util.stream."+stream+" values; static Object run(){return values."+terminal.name+"("+terminal.args+");}}")
				invocation := findNode(helper.File.Ast, "method_invocation")
				got, known := inferExprJavaType(invocation, helper.Ctx, helper.File.Source)
				if !known || got != terminal.result {
					t.Fatalf("terminal Java result = %q/%t, want %q", got, known, terminal.result)
				}
			})
		}
	}
}

func TestStreamNumericTerminalResultControls(t *testing.T) {
	for _, tc := range []struct{ stream, want string }{{"IntStream", "int"}, {"LongStream", "long"}, {"DoubleStream", "double"}} {
		t.Run(tc.stream, func(t *testing.T) {
			helper := setupParseHelper(t, "class NumericTypes {static java.util.stream."+tc.stream+" values; static Object run(){return values.sum();}}")
			invocation := findNode(helper.File.Ast, "method_invocation")
			got, known := inferExprJavaType(invocation, helper.Ctx, helper.File.Source)
			if !known || got != tc.want {
				t.Fatalf("numeric Java result = %q/%t, want %q", got, known, tc.want)
			}
		})
	}
}

func TestStreamPrimitiveTerminalSourceOwnerControls(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"source-shadow", `class TerminalOwner {static class Stream {String count(){return "source";}} static Stream values; static String run(){return values.count();}}`},
		{"live-binder", `class TerminalOwner {static class Counter {String count(){return "source";}} static <Stream extends Counter> String run(Stream values){return values.count();}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			methods := helper.Ctx.currentClass.FindMethod().ByName("run")
			if len(methods) != 1 {
				t.Fatal("missing caller run declaration")
			}
			helper.Ctx.localScope = methods[0]
			invocation := findNode(helper.File.Ast, "method_invocation")
			got, known := inferExprJavaType(invocation, helper.Ctx, helper.File.Source)
			if !known || !isBuiltinJavaString(got, helper.Ctx) {
				t.Fatalf("source terminal captured as primitive = %q/%t, want canonical Java String", got, known)
			}
			if result, intrinsic := inferIntrinsicMethodResultType(invocation, helper.Ctx, helper.File.Source); intrinsic {
				t.Fatalf("source owner selected intrinsic result %q", result)
			}
		})
	}
}

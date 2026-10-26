package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestReflectionTypedLongMissingOwnedCallerStrictlyRefuses(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"getter", `class Probe {long run(java.lang.reflect.Field member,Object receiver)throws Exception{return member.getLong(receiver);}}`},
		{"setter", `class Probe {void run(java.lang.reflect.Field member,Object receiver)throws Exception{member.setLong(receiver,1L);}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := setupParseHelper(t, tc.source)
			ctx := h.Ctx
			ctx.currentClass = ctx.currentFile.FindClassScope("Probe")
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			ctx.runtimeLocationProfile = sourceRuntimeLocationClasspath
			ctx.executionContextName = "execution"
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if id, owned := reflectionSourceCallerID(invocation, ctx); owned || id != "" {
				t.Fatal("missing owned frame acquired caller ID")
			}
			strictRoutingState(t)
			var result ast.Expr
			var failure any
			func() { defer func() { failure = recover() }(); result = ParseExpr(invocation, h.File.Source, ctx) }()
			rejected, ok := failure.(strictModeError)
			if !ok || rejected.diagnostic.Kind != "intrinsic invocation" || result != nil {
				t.Fatalf("typed access bypassed caller boundary: %T %v %T", failure, failure, result)
			}
		})
	}
}

func TestReflectionTypedLongSourceHomonymRetainsDispatch(t *testing.T) {
	generated := renderGoFileFromJava(t, `public class TypedFieldOwnerControl {
 static class Field {public long getLong(Object value){return 7L;}public void setLong(Object value,long next){}}
 public static long run(){Field f=new Field();f.setLong(null,9L);return f.getLong(null);}
 }`)
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
				found[selector.Sel.Name] = true
			}
		}
		return true
	})
	if !found["GetLongJava2goExecution"] || !found["SetLongJava2goExecution"] {
		t.Fatalf("source homonym lost its own value contract:\n%s", generated)
	}
	if found["GetLongWithCallerExecution"] || found["SetLongWithCallerExecution"] {
		t.Fatalf("source homonym borrowed native reflection:\n%s", generated)
	}
}

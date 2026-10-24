package transpiler

import (
	"go/ast"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func capturedLocalLambdaContext(t *testing.T, nested bool) (Ctx, *localClassInfo, *sitter.Node, *sitter.Node) {
	t.Helper()
	body := `return () -> consume(value);`
	if nested {
		body = `return () -> { Runnable inner=() -> consume(value);inner.run(); };`
	}
	h := setupParseHelper(t, `class Capture {static void consume(Object value){} static Runnable make(){class Local{} Local value=new Local();`+body+`} static Runnable other(){return () -> {};} }`)
	ctx := classScopeCtx(h.File.Symbols.FindClassScope("Capture"), h.Ctx)
	ctx.localScope = ctx.currentClass.FindMethodByName("make", nil)
	if ctx.localScope == nil {
		t.Fatal("actual declaring method missing")
	}
	var declarations []ast.Decl
	counter := 0
	ctx.className = ctx.currentClass.Class.Name
	ctx.hoistedDecls, ctx.anonClassCounter = &declarations, &counter
	ctx.localClasses = map[string]*localClassInfo{}
	ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{}
	ctx.executionContextName = "callerExecution"
	local := findNode(ctx.localScope.DeclarationNode, "class_declaration")
	hoistLocalClass(local, ctx.currentFile.Source, ctx)
	info := ctx.localClasses["Local"]
	if info == nil || localClassInDeclaration("Local", ctx) != info {
		t.Fatal("actual local hoist was not admitted in its declaring method")
	}
	outer := findNode(ctx.localScope.DeclarationNode, "lambda_expression")
	ctx = contextWithLambdaParameters(ctx, outer.ChildByFieldName("parameters"), nil, "void", ctx.currentFile.Source)
	if nested {
		inner := findNode(outer.ChildByFieldName("body"), "lambda_expression")
		ctx = contextWithLambdaParameters(ctx, inner.ChildByFieldName("parameters"), nil, "void", ctx.currentFile.Source)
	}
	return ctx, info, local, outer
}

func TestCapturedLocalClassLambdaStaticExecutionLookup(t *testing.T) {
	for _, nested := range []bool{false, true} {
		name := "direct"
		if nested {
			name = "nested"
		}
		t.Run(name, func(t *testing.T) {
			ctx, info, _, _ := capturedLocalLambdaContext(t, nested)
			if localClassInDeclaration("Local", ctx) != info {
				t.Error("captured actual local class lost inside its lexical lambda")
			}
			call := findNode(ctx.localBindingBody, "method_invocation")
			if call == nil || call.ChildByFieldName("name").Content(ctx.currentFile.Source) != "consume" {
				t.Fatal("actual captured-value call missing")
			}
			args := call.ChildByFieldName("arguments")
			var actual inferredJavaTypeOrigin
			got, known := inferExprJavaType(args.NamedChild(0), ctx, ctx.currentFile.Source, &actual)
			if !known || actual.unresolved || actual.nominalScope != info.scope {
				t.Errorf("captured argument origin=%q known=%v unresolved=%v owner=%p want=%p", got, known, actual.unresolved, actual.nominalScope, info.scope)
			}
			selected := findBestMethodInHierarchy(ctx.currentClass, "consume", args, false, true, ctx, ctx.currentFile.Source)
			if selected == nil {
				t.Fatal("source consume(Object) rejected its captured local-class reference")
			}
			if selected.def != ctx.currentClass.FindMethodByName("consume", nil) || executionMethodCallName(selected.def, selected.owner, ctx) == selected.def.Name {
				t.Fatal("selected source callback did not retain its caller-execution implementation")
			}
		})
	}
}

func TestCapturedLocalClassLambdaVisibilityShields(t *testing.T) {
	for _, name := range []string{"missing-synthetic-method", "missing-body-anchor", "ordinary-block-is-not-lambda", "foreign-executable-lambda", "foreign-executable-method"} {
		t.Run(name, func(t *testing.T) {
			ctx, _, _, outer := capturedLocalLambdaContext(t, false)
			switch name {
			case "missing-synthetic-method":
				ctx.localScope = nil
			case "missing-body-anchor":
				ctx.localBindingBody = nil
			case "ordinary-block-is-not-lambda":
				ctx.localBindingBody = ctx.currentClass.FindMethodByName("make", nil).DeclarationNode.ChildByFieldName("body")
			case "foreign-executable-lambda":
				other := ctx.currentClass.FindMethodByName("other", nil)
				ctx.localBindingBody = findNode(other.DeclarationNode, "lambda_expression").ChildByFieldName("body")
			case "foreign-executable-method":
				ctx.localScope = ctx.currentClass.FindMethodByName("other", nil)
				ctx.localBindingBody = outer.ChildByFieldName("body")
			}
			if localClassInDeclaration("Local", ctx) != nil {
				t.Fatal("unrelated or incomplete lexical context donated local-class visibility")
			}
		})
	}
}

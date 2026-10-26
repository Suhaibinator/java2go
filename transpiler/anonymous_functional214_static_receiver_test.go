package transpiler

import (
	"bytes"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestCI214StaticIntrinsicReceiverRoleBoundary(t *testing.T) {
	rows := []struct {
		name, source, kind, owner string
		accepted                  bool
		importNode                bool
	}{
		{"method_lambda", `class Main{void run(java.util.function.IntUnaryOperator unary){unary.andThen(x->x*2).applyAsInt(5);}}`, "method_invocation", "", false, false},
		{"parenthesized", `class Main{void run(java.util.function.IntUnaryOperator unary){(unary).applyAsInt(1);}}`, "parenthesized_expression", "", false, false},
		{"new_expression", `class Main{void run(){new java.util.Random().nextInt(1);}}`, "object_creation_expression", "", false, false},
		{"field_invocation_root", `class Main{static Main make(){return null;}java.util.function.IntUnaryOperator value;void run(){make().value.applyAsInt(1);}}`, "field_access", "", false, false},
		{"simple_identifier", `class Main{void run(){Math.abs(1);}}`, "identifier", "Math", true, false},
		{"qualified_field_access", `class Main{void run(){java.lang.Math.abs(1);}}`, "field_access", "Math", true, false},
		{"explicit_method_type_arguments", `class Main{void run(){java.util.Collections.<String>emptyList();}}`, "field_access", "Collections", true, false},
		{"qualified_scoped_identifier", `import java.util.Collections;class Main{void run(){Collections.emptyList();}}`, "scoped_identifier", "Collections", true, true},
		{"value_shadow", `class Main{void run(int Math){Math.abs(1);}}`, "identifier", "", false, false},
		{"binder_shadow", `class Main{<Math>void run(){Math.abs(1);}}`, "identifier", "", false, false},
		{"source_type_shadow", `class Math{static int abs(int value){return value;}}class Main{void run(){Math.abs(1);}}`, "identifier", "", false, false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, row.source)
			ctx := helper.Ctx.Clone()
			ctx.currentClass = ctx.currentFile.FindClassScope("Main")
			if ctx.currentClass == nil {
				t.Fatal("actual caller class missing")
			}
			methods := ctx.currentClass.FindMethod().ByOriginalName("run")
			if len(methods) != 1 {
				t.Fatal("actual caller executable missing")
			}
			ctx.localScope = methods[0]
			invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if invocation == nil {
				t.Fatal("actual receiver invocation missing")
			}
			receiver := invocation.ChildByFieldName("object")
			if row.importNode {
				receiver = findNode(helper.File.Ast, "scoped_identifier")
			}
			if receiver == nil || receiver.Type() != row.kind {
				t.Fatalf("actual AST receiver kind=%v want=%q", receiver, row.kind)
			}
			logger := log.StandardLogger()
			previous := logger.Out
			var output bytes.Buffer
			logger.SetOutput(&output)
			owner, accepted := func() (string, bool) {
				defer logger.SetOutput(previous)
				return intrinsicStaticClassName(receiver, ctx, helper.File.Source)
			}()
			if output.Len() != 0 {
				t.Fatalf("expression receiver entered type parser: %s", output.String())
			}
			if accepted != row.accepted || owner != row.owner {
				t.Fatalf("class qualifier=%q,%t want=%q,%t", owner, accepted, row.owner, row.accepted)
			}
		})
	}
}

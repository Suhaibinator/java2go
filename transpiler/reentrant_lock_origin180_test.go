package transpiler

import (
	"go/ast"
	"strings"
	"testing"
)

func TestReentrantLock180BinderAndQualifiedShields(t *testing.T) {
	const source = `interface Weight { void lock(); }
public class Use<ReentrantLock extends Weight> {
 void use(ReentrantLock value) { value.lock(); }
 java.util.concurrent.locks.ReentrantLock canonical = new java.util.concurrent.locks.ReentrantLock();
 void canonicalUse() { canonical.lock(); canonical.unlock(); }
}`
	generated := renderGoFileFromJava(t, source)
	if strings.Contains(generated, "value *stdjava.JavaReentrantLock") {
		t.Fatalf("source type binder was replaced by canonical native type:\n%s", generated)
	}
	if !strings.Contains(generated, "stdjava.NewReentrantLock()") || !strings.Contains(generated, "stdjava.ReentrantLockLockExecution(") {
		t.Fatalf("fully qualified canonical type was hidden by unrelated simple binder:\n%s", generated)
	}
}

func TestReentrantLock180ConstructorScopeRefusals(t *testing.T) {
	for _, tc := range []struct {
		source string
		args   []ast.Expr
		want   bool
	}{
		{`import java.util.concurrent.locks.ReentrantLock; class Use { Object make(){return new ReentrantLock();}}`, nil, true},
		{`import java.util.concurrent.locks.*; class Use { Object make(){return new ReentrantLock();}}`, nil, true},
		{`import other.ReentrantLock; class Use { Object make(){return new ReentrantLock();}}`, nil, false},
		{`class ReentrantLock {} class Use { Object make(){return new ReentrantLock();}}`, nil, false},
		{`import java.util.concurrent.locks.ReentrantLock; class Use { Object make(){return new ReentrantLock(true);}}`, []ast.Expr{ast.NewIdent("true")}, false},
	} {
		t.Run(tc.source, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			ctx := helper.Ctx.Clone()
			if use := helper.File.Symbols.FindClassScope("Use"); use != nil {
				ctx.currentClass = use
			}
			node := findNode(helper.File.Ast, "object_creation_expression")
			if got := reentrantLockConstructorApplicable(node, nil, tc.args, ctx, []byte(tc.source)); got != tc.want {
				t.Fatalf("canonical noarg constructor admission=%t want=%t", got, tc.want)
			}
		})
	}
}

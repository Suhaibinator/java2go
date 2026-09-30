package transpiler

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

func TestCtxClonePreservesExecutionContext(t *testing.T) {
	original := Ctx{executionContextName: "execution"}
	if got := original.Clone().executionContextName; got != "execution" {
		t.Fatalf("cloned execution context = %q, want execution", got)
	}
}

func TestBuildExecutionAwareFuncDeclsPreservesPublicVariadicABI(t *testing.T) {
	declaration := &ast.FuncDecl{
		Name: &ast.Ident{Name: "Sum"},
		Recv: &ast.FieldList{List: []*ast.Field{{
			Names: []*ast.Ident{{Name: "wk"}},
			Type:  &ast.StarExpr{X: &ast.Ident{Name: "Worker"}},
		}}},
		Type: &ast.FuncType{
			Params: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: "values"}},
				Type:  &ast.Ellipsis{Elt: &ast.Ident{Name: "int32"}},
			}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "int32"}}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{
			&ast.BasicLit{Kind: token.INT, Value: "7"},
		}}}},
	}
	ctx := Ctx{importAliases: map[string]string{}, usedImports: map[string]bool{}}
	declarations := buildExecutionAwareFuncDecls(
		declaration,
		"SumJava2goExecution",
		"__java2goExecution",
		ctx,
	)
	if len(declarations) != 2 {
		t.Fatalf("execution-aware declarations = %d, want public wrapper + implementation", len(declarations))
	}

	var rendered bytes.Buffer
	for _, generated := range declarations {
		if err := printer.Fprint(&rendered, token.NewFileSet(), generated); err != nil {
			t.Fatalf("rendering generated declaration: %v", err)
		}
		rendered.WriteByte('\n')
	}
	output := rendered.String()
	for _, want := range []string{
		"func (wk *Worker) Sum(values ...int32) int32",
		"return wk.SumJava2goExecution(stdjava.NewExecution(), values...)",
		"func (wk *Worker) SumJava2goExecution(__java2goExecution *stdjava.Execution, values ...int32) int32",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("execution-aware declaration missing %q:\n%s", want, output)
		}
	}
}

func TestExecutionAwareConstructorNamesAvoidSourceTypeCollisions(t *testing.T) {
	src := `
class Target {
    Target(Object lock, int[] counter) {
        synchronized (lock) {
            counter[0]++;
        }
    }
}

class NewTargetJava2goExecution {
}

interface TargetFactory {
    Target make(Object lock, int[] counter);
}

public class ConstructorExecutionCollision {
    static final Object LOCK = new Object();

    public static int run() {
        int[] counter = {0};
        synchronized (LOCK) {
            new Target(LOCK, counter);
            TargetFactory factory = Target::new;
            factory.make(LOCK, counter);
            return counter[0];
        }
    }
}
`

	out := renderGoFileFromJava(t, src)
	if !strings.Contains(out, "newTargetJava2goExecution1") {
		t.Fatalf("constructor execution name did not avoid source type collision:\n%s", out)
	}
	runGeneratedWithStdjava(t, out, `
package main

import "testing"

func TestConstructorExecutionCollision(t *testing.T) {
	if got := Run(); got != 2 {
		t.Fatalf("Run() = %d, want 2", got)
	}
}
`)
}

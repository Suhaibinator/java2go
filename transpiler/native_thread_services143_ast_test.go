package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestNativeThreadServices143ParsedExecutionAndQualifierAST(t *testing.T) {
	source := `class Probe{static int effects;static Thread next(){effects++;return null;}static boolean observe(){return next().isInterrupted();}static boolean clear(){return next().interrupted();}static void hint(){next().onSpinWait();}static void typed(){java.lang.Thread.onSpinWait();}}`
	rendered := renderGoFileFromJava(t, source)
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", rendered, 0)
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx.Clone()
	owner := resolveClassScopeByQualifiedName(ctx, "Probe")
	ctx.currentClass = owner
	nextName := executionImplementationName(owner.FindMethodByName("next", nil), owner, ctx)
	logicalNames := map[string]string{}
	for _, name := range []string{"observe", "clear", "hint", "typed"} {
		logicalNames[executionImplementationName(owner.FindMethodByName(name, nil), owner, ctx)] = name
	}
	checks := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !strings.HasSuffix(fn.Name.Name, "Java2goExecution") {
			continue
		}
		services, nextCalls, discard := 0, 0, 0
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if assign, ok := node.(*ast.AssignStmt); ok && len(assign.Lhs) == 1 {
				if id, ok := assign.Lhs[0].(*ast.Ident); ok && id.Name == "_" {
					discard++
				}
			}
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == nextName {
				nextCalls++
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "ThreadIsInterruptedExecution" || sel.Sel.Name == "ThreadInterruptedExecution" || sel.Sel.Name == "ThreadOnSpinWaitExecution") {
				services++
				if len(call.Args) == 0 {
					t.Fatal("missing Execution")
				}
				id, ok := call.Args[0].(*ast.Ident)
				if !ok || len(fn.Type.Params.List) == 0 || len(fn.Type.Params.List[0].Names) != 1 || id.Name != fn.Type.Params.List[0].Names[0].Name {
					t.Fatal("service manufactured/replaced Execution")
				}
				if sel.Sel.Name == "ThreadIsInterruptedExecution" && len(call.Args) != 2 {
					t.Fatal("observer receiver lost")
				}
				if sel.Sel.Name != "ThreadIsInterruptedExecution" && len(call.Args) != 1 {
					t.Fatal("static qualifier became receiver")
				}
			}
			return true
		})
		if services == 0 {
			continue
		}
		checks++
		if services != 1 {
			t.Fatal("duplicated service", fn.Name.Name)
		}
		typed := logicalNames[fn.Name.Name] == "typed"
		if typed {
			if nextCalls != 0 || discard != 0 {
				t.Fatal("TYPE qualifier evaluated")
			}
		} else {
			if nextCalls != 1 {
				t.Fatal("qualifier/receiver not evaluated exactly once", fn.Name.Name, nextCalls)
			}
		}
		if !typed && logicalNames[fn.Name.Name] != "observe" && discard != 1 {
			t.Fatal("value static qualifier not discarded exactly once", fn.Name.Name, discard)
		}
	}
	if checks != 4 {
		t.Fatal("missing actual service implementations", checks)
	}
}

func TestNativeThreadServices143RequiresCallerExecution(t *testing.T) {
	for _, source := range []string{`class Probe{boolean run(Thread t){return t.isInterrupted();}}`, `class Probe{boolean run(){return Thread.interrupted();}}`, `class Probe{void run(){Thread.onSpinWait();}}`} {
		for _, mode := range []struct {
			name   string
			strict bool
		}{{"permissive", false}, {"strict", true}} {
			t.Run(source+"/"+mode.name, func(t *testing.T) {
				strictRoutingState(t)
				setStrictMode(mode.strict)
				helper := setupParseHelper(t, source)
				ctx := helper.Ctx.Clone()
				ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Probe")
				ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
				ctx.executionContextName = ""
				invocation := findNode(ctx.localScope.DeclarationNode, "method_invocation")
				want := Diagnostic{Kind: "intrinsic invocation", NodeType: "method_invocation", Message: invocation.Content(helper.File.Source), ClassName: ctx.className, Line: invocation.StartPoint().Row + 1}
				var expression ast.Expr
				var observed any
				func() {
					defer func() { observed = recover() }()
					expression = ParseExpr(invocation, helper.File.Source, ctx)
				}()
				items := Diagnostics()
				if len(items) != 1 || items[0] != want {
					t.Fatalf("missing caller diagnostic=%+v, want exact %+v", items, want)
				}
				if mode.strict {
					failure, ok := observed.(strictModeError)
					if !ok || failure.diagnostic != want || expression != nil {
						t.Fatalf("strict missing-caller refusal=%T/%v expression=%T", observed, observed, expression)
					}
					return
				}
				if observed != nil || expression == nil {
					t.Fatalf("permissive missing-caller result=%T panic=%v", expression, observed)
				}
				placeholders := 0
				ast.Inspect(expression, func(node ast.Node) bool {
					if call, ok := node.(*ast.CallExpr); ok {
						if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" && len(call.Args) == 1 {
							if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING && strings.Contains(lit.Value, "unsupported intrinsic invocation") && strings.Contains(lit.Value, invocation.Content(helper.File.Source)) {
								placeholders++
							}
						}
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "ThreadIsInterruptedExecution" || sel.Sel.Name == "ThreadInterruptedExecution" || sel.Sel.Name == "ThreadOnSpinWaitExecution" || sel.Sel.Name == "NewExecution") {
							t.Fatal("missing caller Execution lowered to service or manufactured token")
						}
					}
					return true
				})
				if placeholders != 1 {
					t.Fatalf("permissive unsupported placeholders=%d want1", placeholders)
				}
			})
		}
	}
}

func TestNativeThreadServices143WrongArityStrictRefusal(t *testing.T) {
	for _, source := range []string{`class Probe{boolean run(Thread t){return t.isInterrupted(7);}}`, `class Probe{boolean run(){return Thread.interrupted(7);}}`, `class Probe{void run(){Thread.onSpinWait(7);}}`, `class Probe{boolean run(){return Thread.isInterrupted();}}`, `import static java.lang.Thread.interrupted;class Probe{boolean run(){return interrupted(7);}}`, `import static java.lang.Thread.onSpinWait;class Probe{void run(){onSpinWait(7);}}`} {
		t.Run(source, func(t *testing.T) {
			strictRoutingState(t)
			defer func() {
				if recover() == nil || len(Diagnostics()) == 0 {
					t.Fatal("invalid method signature escaped strict refusal")
				}
			}()
			renderGoFileFromJava(t, source)
		})
	}
}

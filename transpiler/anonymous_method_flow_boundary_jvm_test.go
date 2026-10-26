package transpiler

import (
	"embed"
	"fmt"
	"go/ast"
	"io/fs"
	"strings"
	"testing"
)

//go:embed testdata/anonymous_method_flow_boundary
var anonymousMethodFlowInputs embed.FS

// The expected bytes were captured from nine actual seeded JDK21 executions.
// This keeps the entire four-Java application, main, and observation streams.
func TestAnonymousMethodFlowBoundaryJVM(t *testing.T) {
	const root = "testdata/anonymous_method_flow_boundary"
	files := map[string]string{}
	if err := fs.WalkDir(anonymousMethodFlowInputs, root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".java") && path != root+"/pom.xml" {
			return nil
		}
		value, err := anonymousMethodFlowInputs.ReadFile(path)
		if err != nil {
			return err
		}
		files[strings.TrimPrefix(path, root+"/")] = string(value)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	observations := []campaignCompilerProjectObservation{}
	for _, seed := range []int{17, 41, 97} {
		stdout, err := anonymousMethodFlowInputs.ReadFile(fmt.Sprintf("%s/oracles/seed%d.stdout", root, seed))
		if err != nil {
			t.Fatal(err)
		}
		stderr, err := anonymousMethodFlowInputs.ReadFile(fmt.Sprintf("%s/oracles/seed%d.stderr", root, seed))
		if err != nil {
			t.Fatal(err)
		}
		for repeat := 1; repeat <= 3; repeat++ {
			observations = append(observations, campaignCompilerProjectObservation{
				name: fmt.Sprintf("seed%d-repeat%d", seed, repeat), args: []string{fmt.Sprint(seed)},
				stdout: string(stdout), stderr: string(stderr),
			})
		}
	}
	runCampaignCompilerStrictProjectObservations(t, files, "flow.app.Main", observations)
}

// This is an emitted-source contract, not a Java oracle: creation and invocation
// deliberately have different synthetic Execution names. Each callback must use
// its own hidden invocation parameter for its monitor operations.
func TestAnonymousMethodCallbackExecutionOwnership(t *testing.T) {
	body := `public int read() {
        try { synchronized(lock) { counter[0]++; return seed; } }
        finally { synchronized(lock) { counter[0]++; } }
    }`
	cases := []struct{ name, creation string }{
		{"sam", "return new Action(){" + body + "};"},
		{"structured", "return new Action(){int marker=seed;" + body + "};"},
		{"local", "class Work implements Action {int marker=seed;" + body + "} return new Work();"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, `public class ExecutionBoundaryProbe {
                interface Action { int read(); }
                static Action factory(final Object lock, final int seed, final int[] counter) {`+test.creation+`}
            }`)
			ctx := helper.Ctx.Clone()
			for _, method := range ctx.currentClass.Methods {
				if method.OriginalName == "factory" {
					ctx.localScope = method
				}
			}
			if ctx.localScope == nil {
				t.Fatal("factory source declaration missing")
			}
			const factoryToken = "factoryExecutionSentinel"
			ctx.executionContextName = factoryToken
			ctx.tryReturnTarget = &tryReturnTarget{FlagName: "factoryFlowFlag", ValueName: "factoryReturnValue", HasValue: true}
			var hoisted []ast.Decl
			var ordinal int
			ctx.hoistedDecls = &hoisted
			ctx.anonClassCounter = &ordinal
			ctx.localClasses = make(map[string]*localClassInfo)
			ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
			lowered := ParseStmt(ctx.localScope.DeclarationNode.ChildByFieldName("body"), helper.File.Source, ctx)
			var callbackType *ast.FuncType
			var callbackBody *ast.BlockStmt
			if test.name == "sam" {
				ast.Inspect(lowered, func(node ast.Node) bool {
					literal, ok := node.(*ast.FuncLit)
					if !ok || literal.Type.Params == nil || len(literal.Type.Params.List) == 0 {
						return true
					}
					field := literal.Type.Params.List[0]
					pointer, ok := field.Type.(*ast.StarExpr)
					if !ok {
						return true
					}
					selector, ok := pointer.X.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "Execution" {
						callbackType, callbackBody = literal.Type, literal.Body
						return false
					}
					return true
				})
			} else {
				for _, node := range hoisted {
					method, ok := node.(*ast.FuncDecl)
					if ok && strings.Contains(method.Name.Name, "ReadJava2goExecution") {
						callbackType, callbackBody = method.Type, method.Body
					}
				}
			}
			if callbackType == nil || callbackBody == nil || callbackType.Params == nil || len(callbackType.Params.List) == 0 {
				t.Fatal("independent execution-aware read callback missing")
			}
			parameter := callbackType.Params.List[0]
			if len(parameter.Names) != 1 || parameter.Names[0].Name == factoryToken {
				t.Fatal("callback did not bind its own invocation Execution")
			}
			invocationToken := parameter.Names[0].Name
			monitors, captures := 0, map[string]bool{}
			ast.Inspect(callbackBody, func(node ast.Node) bool {
				if id, ok := node.(*ast.Ident); ok {
					captures[id.Name] = true
					if id.Name == factoryToken || id.Name == "factoryFlowFlag" || id.Name == "factoryReturnValue" {
						t.Errorf("callback body borrowed enclosing factory state %q", id.Name)
					}
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if selector.Sel.Name == "NewExecution" {
					t.Error("callback allocated a fresh Execution")
				}
				if selector.Sel.Name == "MonitorEnterExecution" {
					monitors++
					if len(call.Args) == 0 {
						t.Error("monitor has no caller Execution")
						return true
					}
					token, ok := call.Args[0].(*ast.Ident)
					if !ok || token.Name != invocationToken {
						t.Errorf("monitor uses %T, want callback parameter %s", call.Args[0], invocationToken)
					}
				}
				return true
			})
			if monitors < 2 {
				t.Fatalf("inner try and finally monitor operations missing: %d", monitors)
			}
			for _, name := range []string{"lock", "seed", "counter"} {
				if !captures[name] {
					t.Errorf("source capture %s missing from callback", name)
				}
			}
			t.Logf("creation token=%s; callback parameter=%s; monitor operations=%d", factoryToken, invocationToken, monitors)
		})
	}
}

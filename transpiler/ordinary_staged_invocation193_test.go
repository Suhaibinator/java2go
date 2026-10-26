package transpiler

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

type ci193OrdinaryStageProof struct {
	method string
	types  []string
	result string
}

func ci193ASTText(node ast.Node) string {
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		return ""
	}
	return out.String()
}

// Resolve only a complete, immediately executed receiver/argument staging
// sequence. The returned AST is a test-side description; generated code is never
// modified. Unrecognized closures stay opaque and cannot borrow this proof.
func ci193OrdinaryExpression(expr ast.Expr, aliases map[string]bool) (ast.Expr, []ci193OrdinaryStageProof) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return expr, nil
	}
	if literal, ok := call.Fun.(*ast.FuncLit); ok {
		if len(call.Args) != 0 || call.Ellipsis.IsValid() || literal.Type.Params != nil && len(literal.Type.Params.List) != 0 || literal.Body == nil || len(literal.Body.List) < 3 {
			return expr, nil
		}
		body := literal.Body.List
		receiver, ok := body[0].(*ast.AssignStmt)
		if !ok || receiver.Tok != token.DEFINE || len(receiver.Lhs) != 1 || len(receiver.Rhs) != 1 {
			return expr, nil
		}
		receiverName, ok := receiver.Lhs[0].(*ast.Ident)
		if !ok {
			return expr, nil
		}
		names := map[string]bool{receiverName.Name: true}
		var arguments []ast.Expr
		var types []string
		var stagedNames []string
		for _, statement := range body[1 : len(body)-2] {
			declaration, ok := statement.(*ast.DeclStmt)
			if !ok {
				return expr, nil
			}
			group, ok := declaration.Decl.(*ast.GenDecl)
			if !ok || group.Tok != token.VAR || len(group.Specs) != 1 {
				return expr, nil
			}
			value, ok := group.Specs[0].(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 || value.Type == nil || names[value.Names[0].Name] {
				return expr, nil
			}
			names[value.Names[0].Name] = true
			stagedNames = append(stagedNames, value.Names[0].Name)
			arguments = append(arguments, value.Values[0])
			types = append(types, ci193ASTText(value.Type))
		}
		check, ok := body[len(body)-2].(*ast.ExprStmt)
		if !ok {
			return expr, nil
		}
		guard, ok := check.X.(*ast.CallExpr)
		if !ok || len(guard.Args) != 2 || guard.Ellipsis.IsValid() {
			return expr, nil
		}
		guardMethod, ok := guard.Fun.(*ast.SelectorExpr)
		if !ok || guardMethod.Sel.Name != "RequireInvocationReceiver" {
			return expr, nil
		}
		alias, ok := guardMethod.X.(*ast.Ident)
		if !ok || !aliases[alias.Name] {
			return expr, nil
		}
		checked, ok := guard.Args[0].(*ast.Ident)
		if !ok || checked.Name != receiverName.Name {
			return expr, nil
		}
		message, ok := guard.Args[1].(*ast.BasicLit)
		if !ok || message.Kind != token.STRING {
			return expr, nil
		}
		text, err := strconv.Unquote(message.Value)
		if err != nil || text == "" {
			return expr, nil
		}
		var final ast.Expr
		switch statement := body[len(body)-1].(type) {
		case *ast.ReturnStmt:
			if len(statement.Results) != 1 {
				return expr, nil
			}
			final = statement.Results[0]
		case *ast.ExprStmt:
			final = statement.X
		default:
			return expr, nil
		}
		invocation, ok := final.(*ast.CallExpr)
		if !ok || len(invocation.Args) != len(arguments)+1 || invocation.Ellipsis.IsValid() {
			return expr, nil
		}
		method, ok := invocation.Fun.(*ast.SelectorExpr)
		if !ok {
			return expr, nil
		}
		selected, ok := method.X.(*ast.Ident)
		if !ok || selected.Name != receiverName.Name {
			return expr, nil
		}
		execution, ok := invocation.Args[0].(*ast.Ident)
		if !ok || execution.Name != "__java2goExecution" {
			return expr, nil
		}
		for i, name := range stagedNames {
			argument, ok := invocation.Args[i+1].(*ast.Ident)
			if !ok || argument.Name != name {
				return expr, nil
			}
		}
		result := ""
		if literal.Type.Results != nil {
			if len(literal.Type.Results.List) != 1 {
				return expr, nil
			}
			result = ci193ASTText(literal.Type.Results.List[0].Type)
		}
		normalized := &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver.Rhs[0], Sel: method.Sel}, Args: append([]ast.Expr{invocation.Args[0]}, arguments...)}
		return normalized, []ci193OrdinaryStageProof{{method: method.Sel.Name, types: types, result: result}}
	}
	if selection, ok := call.Fun.(*ast.SelectorExpr); ok {
		receiver, proof := ci193OrdinaryExpression(selection.X, aliases)
		copy := *call
		copy.Fun = &ast.SelectorExpr{X: receiver, Sel: selection.Sel}
		return &copy, proof
	}
	return expr, nil
}

func ci193OrdinaryCallMatches(out, function, expected string, argumentTypes []string, stagedResult string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		return false
	}
	want, err := parser.ParseExpr(expected)
	if err != nil {
		return false
	}
	text := ci193ASTText(want)
	expectedCall, ok := want.(*ast.CallExpr)
	if !ok {
		return false
	}
	expectedMethod, ok := expectedCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	aliases := map[string]bool{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "github.com/NickyBoy89/java2go/stdjava" {
			name := "stdjava"
			if spec.Name != nil {
				name = spec.Name.Name
			}
			aliases[name] = true
		}
	}
	count := 0
	for _, declaration := range file.Decls {
		declared, ok := declaration.(*ast.FuncDecl)
		if !ok || declared.Name.Name != function {
			continue
		}
		ast.Inspect(declared.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			normalized, proofs := ci193OrdinaryExpression(call, aliases)
			if ci193ASTText(normalized) != text {
				return true
			}
			for _, proof := range proofs {
				if proof.method == expectedMethod.Sel.Name && strings.Join(proof.types, "|") != strings.Join(argumentTypes, "|") {
					return true
				}
			}
			if stagedResult != "" && len(proofs) > 0 {
				found := false
				for _, proof := range proofs {
					found = found || proof.result == stagedResult
				}
				if !found {
					return true
				}
			}
			count++
			return true
		})
	}
	return count == 1
}

func ci193AssertOrdinaryCall(t *testing.T, out, function, expected string, argumentTypes []string, stagedResult string) {
	t.Helper()
	if !ci193OrdinaryCallMatches(out, function, expected, argumentTypes, stagedResult) {
		t.Fatalf("ordinary invocation lost receiver/argument/Execution/type/once dataflow in %s: want %s\n%s", function, expected, out)
	}
}

func ci193AssertFallbackBranch(t *testing.T, out, function string, operation string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, declaration := range file.Decls {
		declared, ok := declaration.(*ast.FuncDecl)
		if !ok || declared.Name.Name != function {
			continue
		}
		ast.Inspect(declared.Body, func(n ast.Node) bool {
			branch, ok := n.(*ast.BranchStmt)
			if ok && branch.Tok.String() == operation && branch.Label != nil && branch.Label.Name != "" {
				found = true
			}
			return true
		})
	}
	if !found {
		t.Fatalf("ordinary fallback lost labeled %s in %s", operation, function)
	}
}

func TestCI193OrdinaryInvocationProofRejectsChangedDataflow(t *testing.T) {
	const original = `package main
import stdjava "github.com/NickyBoy89/java2go/stdjava"
func Run(grid *Grid,__java2goExecution *stdjava.Execution) int32 {
 return func() int32 {
  receiver := grid
  var arg0 int32 = next()
  var arg1 int32 = 0
  stdjava.RequireInvocationReceiver(receiver,"source diagnostic")
  return receiver.getJava2goExecution(__java2goExecution,arg0,arg1)
 }()
}`
	const want = "grid.getJava2goExecution(__java2goExecution, next(), 0)"
	for _, row := range []struct{ name, old, new string }{
		{"exact", "", ""}, {"receiver", "receiver := grid", "receiver := other"},
		{"type", "var arg0 int32", "var arg0 int64"}, {"duplicated-evaluation", "= next()", "= next()+next()"},
		{"argument-order", "__java2goExecution,arg0,arg1", "__java2goExecution,arg1,arg0"},
		{"execution", "(__java2goExecution,arg0,arg1)", "(otherExecution,arg0,arg1)"},
		{"callee", "receiver.getJava2goExecution", "receiver.otherJava2goExecution"},
		{"guard-alias", "stdjava.RequireInvocationReceiver", "other.RequireInvocationReceiver"},
		{"eager-guard", "var arg1 int32 = 0\n  stdjava.RequireInvocationReceiver(receiver,\"source diagnostic\")", "stdjava.RequireInvocationReceiver(receiver,\"source diagnostic\")\n  var arg1 int32 = 0"},
	} {
		t.Run(row.name, func(t *testing.T) {
			source := original
			if row.old != "" {
				source = strings.Replace(source, row.old, row.new, 1)
			}
			got := ci193OrdinaryCallMatches(source, "Run", want, []string{"int32", "int32"}, "")
			if got != (row.name == "exact") {
				t.Fatalf("proof=%t for %s", got, row.name)
			}
		})
	}
}

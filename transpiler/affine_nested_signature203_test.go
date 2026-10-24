package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// Derive the emitted owner and formal from their declarations. Neither the
// collision suffix nor the emitted private type spelling is an oracle.
func ci203AffineNestedSignatureMatches(source string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, 0)
	if err != nil {
		return false
	}
	runtimeAlias := ""
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err == nil && path == "github.com/NickyBoy89/java2go/stdjava" {
			if runtimeAlias != "" {
				return false
			}
			runtimeAlias = "stdjava"
			if spec.Name != nil {
				runtimeAlias = spec.Name.Name
			}
			if runtimeAlias == "_" || runtimeAlias == "." {
				return false
			}
		}
	}
	if runtimeAlias == "" {
		return false
	}
	ident := func(expr ast.Expr) *ast.Ident { value, _ := expr.(*ast.Ident); return value }
	isInt := func(expr ast.Expr, value string) bool {
		literal, ok := expr.(*ast.BasicLit)
		return ok && literal.Kind == token.INT && literal.Value == value
	}
	runtimeSelector := func(expr ast.Expr, name string) bool {
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return false
		}
		qualifier := ident(selector.X)
		return qualifier != nil && qualifier.Name == runtimeAlias && qualifier.Obj == nil
	}
	oneResult := func(function *ast.FuncType) ast.Expr {
		if function == nil || function.Results == nil || len(function.Results.List) != 1 {
			return nil
		}
		return function.Results.List[0].Type
	}
	funcs := map[string]*ast.FuncDecl{}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil {
			if funcs[function.Name.Name] != nil {
				return false
			}
			funcs[function.Name.Name] = function
		}
	}
	factory, target := funcs["GridJava2goExecution"], funcs["ReceiverTypeJava2goExecution"]
	if factory == nil || target == nil || target.Body == nil || target.Type.Params == nil || len(target.Type.Params.List) != 2 {
		return false
	}
	ownerPointer, ok := oneResult(factory.Type).(*ast.StarExpr)
	if !ok {
		return false
	}
	owner := ident(ownerPointer.X)
	if owner == nil || owner.Obj == nil || owner.Obj.Kind != ast.Typ {
		return false
	}
	executionField, valueField := target.Type.Params.List[0], target.Type.Params.List[1]
	if len(executionField.Names) != 1 || len(valueField.Names) != 1 {
		return false
	}
	execution, formal := executionField.Names[0], valueField.Names[0]
	executionType, ok := executionField.Type.(*ast.StarExpr)
	if !ok || !runtimeSelector(executionType.X, "Execution") || execution.Obj == nil || formal.Obj == nil || execution.Obj == formal.Obj || formal.Name == owner.Name {
		return false
	}
	formalType, ok := valueField.Type.(*ast.StarExpr)
	if !ok || ident(formalType.X) == nil || ident(formalType.X).Obj != owner.Obj || ci193ASTText(oneResult(target.Type)) != "int32" {
		return false
	}
	origin, method := 0, 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) != 1 {
			continue
		}
		receiverType, ok := function.Recv.List[0].Type.(*ast.StarExpr)
		if !ok || ident(receiverType.X) == nil || ident(receiverType.X).Obj != owner.Obj {
			continue
		}
		switch function.Name.Name {
		case "JavaDynamicTypeID":
			if function.Body == nil || len(function.Body.List) != 1 {
				return false
			}
			returned, ok := function.Body.List[0].(*ast.ReturnStmt)
			if !ok || len(returned.Results) != 1 {
				return false
			}
			call, ok := returned.Results[0].(*ast.CallExpr)
			if !ok || !runtimeSelector(call.Fun, "TypeID") || len(call.Args) != 1 {
				return false
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return false
			}
			name, err := strconv.Unquote(literal.Value)
			if err != nil || name != "TypeShadowGrid" {
				return false
			}
			origin++
		case "getJava2goExecution":
			if function.Type.Params == nil || len(function.Type.Params.List) != 3 || ci193ASTText(oneResult(function.Type)) != "int32" {
				return false
			}
			parameter := function.Type.Params.List[0]
			pointer, ok := parameter.Type.(*ast.StarExpr)
			if !ok || !runtimeSelector(pointer.X, "Execution") || ci193ASTText(function.Type.Params.List[1].Type) != "int32" || ci193ASTText(function.Type.Params.List[2].Type) != "int32" {
				return false
			}
			method++
		}
	}
	if origin != 1 || method != 1 || len(target.Body.List) != 2 {
		return false
	}
	loop, ok := target.Body.List[0].(*ast.ForStmt)
	if !ok || loop.Body == nil || len(loop.Body.List) != 1 {
		return false
	}
	init, ok := loop.Init.(*ast.AssignStmt)
	if !ok || init.Tok != token.DEFINE || len(init.Lhs) != 1 || len(init.Rhs) != 1 || ident(init.Lhs[0]) == nil || ci193ASTText(init.Rhs[0]) != "int32(0)" {
		return false
	}
	once := ident(init.Lhs[0])
	condition, ok := loop.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.LSS || ident(condition.X) == nil || ident(condition.X).Obj != once.Obj || !isInt(condition.Y, "1") {
		return false
	}
	post, ok := loop.Post.(*ast.IncDecStmt)
	if !ok || post.Tok != token.INC || ident(post.X) == nil || ident(post.X).Obj != once.Obj {
		return false
	}
	tail, ok := target.Body.List[1].(*ast.ReturnStmt)
	if !ok || len(tail.Results) != 1 || ci193ASTText(tail.Results[0]) != "-1" {
		return false
	}
	returned, ok := loop.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return false
	}
	immediate, ok := returned.Results[0].(*ast.CallExpr)
	if !ok || len(immediate.Args) != 0 || immediate.Ellipsis.IsValid() {
		return false
	}
	literal, ok := immediate.Fun.(*ast.FuncLit)
	if !ok || literal.Body == nil || len(literal.Body.List) != 5 || ci193ASTText(oneResult(literal.Type)) != "int32" || literal.Type.Params != nil && len(literal.Type.Params.List) != 0 {
		return false
	}
	body := literal.Body.List
	staged, ok := body[0].(*ast.AssignStmt)
	if !ok || staged.Tok != token.DEFINE || len(staged.Lhs) != 1 || len(staged.Rhs) != 1 {
		return false
	}
	receiver, sourceFormal := ident(staged.Lhs[0]), ident(staged.Rhs[0])
	if receiver == nil || receiver.Obj == nil || sourceFormal == nil || sourceFormal.Obj != formal.Obj {
		return false
	}
	arguments := make([]*ast.Ident, 0, 2)
	for _, statement := range body[1:3] {
		declaration, ok := statement.(*ast.DeclStmt)
		if !ok {
			return false
		}
		group, ok := declaration.Decl.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR || len(group.Specs) != 1 {
			return false
		}
		value, ok := group.Specs[0].(*ast.ValueSpec)
		if !ok || len(value.Names) != 1 || value.Names[0].Obj == nil || value.Names[0].Obj == receiver.Obj || len(value.Values) != 1 || !isInt(value.Values[0], "0") || ci193ASTText(value.Type) != "int32" {
			return false
		}
		for _, earlier := range arguments {
			if earlier.Obj == value.Names[0].Obj {
				return false
			}
		}
		arguments = append(arguments, value.Names[0])
	}
	guardStatement, ok := body[3].(*ast.ExprStmt)
	if !ok {
		return false
	}
	guard, ok := guardStatement.X.(*ast.CallExpr)
	if !ok || !runtimeSelector(guard.Fun, "RequireInvocationReceiver") || len(guard.Args) != 2 || guard.Ellipsis.IsValid() || ident(guard.Args[0]) == nil || ident(guard.Args[0]).Obj != receiver.Obj {
		return false
	}
	message, ok := guard.Args[1].(*ast.BasicLit)
	if !ok || message.Kind != token.STRING {
		return false
	}
	diagnostic, err := strconv.Unquote(message.Value)
	if err != nil || diagnostic == "" {
		return false
	}
	finalReturn, ok := body[4].(*ast.ReturnStmt)
	if !ok || len(finalReturn.Results) != 1 {
		return false
	}
	call, ok := finalReturn.Results[0].(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() || len(call.Args) != 3 {
		return false
	}
	selection, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selection.Sel.Name != "getJava2goExecution" || ident(selection.X) == nil || ident(selection.X).Obj != receiver.Obj {
		return false
	}
	for index, binding := range []*ast.Ident{execution, arguments[0], arguments[1]} {
		argument := ident(call.Args[index])
		if argument == nil || argument.Obj != binding.Obj {
			return false
		}
	}
	calls, affine := 0, false
	ast.Inspect(target.Body, func(node ast.Node) bool {
		invocation, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selected, ok := invocation.Fun.(*ast.SelectorExpr)
		if ok {
			if selected.Sel.Name == "getJava2goExecution" {
				calls++
			}
			affine = affine || strings.HasPrefix(selected.Sel.Name, "Java2goAffineView")
		}
		return true
	})
	return calls == 1 && !affine
}

func ci203AssertAffineNestedSignature(t *testing.T, source string) {
	t.Helper()
	if !ci203AffineNestedSignatureMatches(source) {
		t.Fatalf("ordinary fallback lost actual owner/formal/Execution/typed arguments/receiver guard/order/once binding:\n%s", source)
	}
}

func TestCI203AffineNestedSignatureBindingGuards(t *testing.T) {
	const original = `package main
import runtime "github.com/NickyBoy89/java2go/stdjava"
type emittedGridOwner struct{}
var foreign *emittedGridOwner
var otherExecution *runtime.Execution
func GridJava2goExecution(e *runtime.Execution) *emittedGridOwner {return new(emittedGridOwner)}
func(g *emittedGridOwner) JavaDynamicTypeID() runtime.TypeID{return runtime.TypeID("TypeShadowGrid")}
func(g *emittedGridOwner) getJava2goExecution(e *runtime.Execution,row int32,column int32)int32{return 0}
func ReceiverTypeJava2goExecution(execution *runtime.Execution, observedFormal *emittedGridOwner) int32 {
 for loop := int32(0); loop < 1; loop++ {
  return func() int32 {
   stagedReceiver := observedFormal
   var first int32 = 0
   var second int32 = 0
   runtime.RequireInvocationReceiver(stagedReceiver,"original Java invocation")
   return stagedReceiver.getJava2goExecution(execution,first,second)
  }()
 }
 return -1
}`
	for _, row := range []struct {
		name, old, replacement string
		accept                 bool
	}{
		{"exact", "", "", true},
		{"different-formal-spelling", "observedFormal", "unrelatedGeneratedFormalName", true},
		{"different-owner-spelling", "emittedGridOwner", "anotherEmittedPrivateOwner", true},
		{"wrong-owner", "observedFormal *emittedGridOwner", "observedFormal *OtherOwner", false},
		{"wrong-Java-origin", `TypeID("TypeShadowGrid")`, `TypeID("OtherJavaOwner")`, false},
		{"wrong-formal-source", "stagedReceiver := observedFormal", "stagedReceiver := foreign", false},
		{"shadowed-formal", "stagedReceiver := observedFormal", "observedFormal := foreign\n   stagedReceiver := observedFormal", false},
		{"wrong-guard-receiver", "RequireInvocationReceiver(stagedReceiver,", "RequireInvocationReceiver(foreign,", false},
		{"missing-guard", "runtime.RequireInvocationReceiver(stagedReceiver,\"original Java invocation\")", "", false},
		{"wrong-Execution", "getJava2goExecution(execution,first,second)", "getJava2goExecution(otherExecution,first,second)", false},
		{"argument-order", "getJava2goExecution(execution,first,second)", "getJava2goExecution(execution,second,first)", false},
		{"wrong-argument-type", "var first int32 = 0", "var first int64 = 0", false},
		{"wrong-argument-value", "var first int32 = 0", "var first int32 = 1", false},
		{"early-null-check", "var second int32 = 0\n   runtime.RequireInvocationReceiver(stagedReceiver,\"original Java invocation\")", "runtime.RequireInvocationReceiver(stagedReceiver,\"original Java invocation\")\n   var second int32 = 0", false},
		{"changed-callee", "return stagedReceiver.getJava2goExecution", "return stagedReceiver.otherJava2goExecution", false},
		{"duplicated-receiver-evaluation", "stagedReceiver := observedFormal", "stagedReceiver := GridJava2goExecution(execution)", false},
		{"wrong-loop-count", "loop < 1", "loop < 2", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			source := original
			if row.old != "" {
				if !strings.Contains(source, row.old) {
					t.Fatal("invalid negative control")
				}
				source = strings.ReplaceAll(source, row.old, row.replacement)
			}
			if got := ci203AffineNestedSignatureMatches(source); got != row.accept {
				t.Fatalf("binding proof=%t, want %t", got, row.accept)
			}
		})
	}
}

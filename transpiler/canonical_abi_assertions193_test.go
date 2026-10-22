package transpiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

type ciABI193Reporter interface {
	Helper()
	Fatalf(string, ...any)
}

// These assertions inspect the current reference ABI without changing the Java
// fixture or treating generated text as an application oracle.
func ciABI193Parse(t ciABI193Reporter, source string) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, 0)
	if err != nil {
		t.Fatalf("parse generated Go: %v", err)
	}
	return file
}

func ciABI193Text(t ciABI193Reporter, node ast.Node) string {
	t.Helper()
	var text bytes.Buffer
	if err := format.Node(&text, token.NewFileSet(), node); err != nil {
		t.Fatalf("format generated AST: %v", err)
	}
	return text.String()
}

func ciABI193RequireExpr(t ciABI193Reporter, actual ast.Expr, expected string) {
	t.Helper()
	want, err := parser.ParseExpr(expected)
	if err != nil {
		t.Fatalf("invalid assertion expression: %v", err)
	}
	if got, want := ciABI193Text(t, actual), ciABI193Text(t, want); got != want {
		t.Fatalf("generated expression = %s, want %s", got, want)
	}
}

func ciABI193Function(t ciABI193Reporter, file *ast.File, name string) *ast.FuncDecl {
	t.Helper()
	var found *ast.FuncDecl
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Name.Name != name {
			continue
		}
		if found != nil {
			t.Fatalf("ambiguous generated function %s", name)
		}
		found = method
	}
	if found == nil {
		t.Fatalf("missing generated function %s", name)
	}
	return found
}

func ciABI193Local(t ciABI193Reporter, method *ast.FuncDecl, name string) ast.Expr {
	t.Helper()
	var found ast.Expr
	count := 0
	ast.Inspect(method.Body, func(node ast.Node) bool {
		switch declaration := node.(type) {
		case *ast.AssignStmt:
			if declaration.Tok != token.DEFINE || len(declaration.Lhs) != 1 || len(declaration.Rhs) != 1 {
				break
			}
			if identifier, ok := declaration.Lhs[0].(*ast.Ident); ok && identifier.Name == name {
				found, count = declaration.Rhs[0], count+1
			}
		case *ast.ValueSpec:
			if len(declaration.Names) == 1 && declaration.Names[0].Name == name && len(declaration.Values) == 1 {
				found, count = declaration.Values[0], count+1
			}
		}
		return true
	})
	if count != 1 {
		t.Fatalf("local %s has %d initializers, want exactly one", name, count)
	}
	return found
}

func ciABI193RequireReturn(t ciABI193Reporter, method *ast.FuncDecl, expected string) {
	t.Helper()
	if method.Body == nil || len(method.Body.List) != 1 {
		t.Fatalf("%s must have exactly one return statement", method.Name.Name)
	}
	result, ok := method.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatalf("%s has no single-expression return", method.Name.Name)
	}
	ciABI193RequireExpr(t, result.Results[0], expected)
}

func ciABI193RequireCallCount(t ciABI193Reporter, method *ast.FuncDecl, expected string, count int) {
	t.Helper()
	want, err := parser.ParseExpr(expected)
	if err != nil {
		t.Fatalf("invalid assertion expression: %v", err)
	}
	wantText := ciABI193Text(t, want)
	actual := 0
	ast.Inspect(method.Body, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && ciABI193Text(t, call) == wantText {
			actual++
		}
		return true
	})
	if actual != count {
		t.Fatalf("call %s occurs %d times, want %d", expected, actual, count)
	}
}

func ciABI193RequireMethod(t ciABI193Reporter, file *ast.File, name, receiver, result string) {
	t.Helper()
	method := ciABI193Function(t, file, name)
	if method.Recv == nil || len(method.Recv.List) != 1 || ciABI193Text(t, method.Recv.List[0].Type) != receiver ||
		method.Type.Params == nil || len(method.Type.Params.List) != 0 || method.Type.Results == nil || len(method.Type.Results.List) != 1 {
		t.Fatalf("%s lost its receiver/zero-argument/reference-result signature", name)
	}
	ciABI193RequireExpr(t, method.Type.Results.List[0].Type, result)
}

func ciABI193Field(t ciABI193Reporter, file *ast.File, owner, name string) ast.Expr {
	t.Helper()
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.TYPE {
			continue
		}
		for _, specification := range group.Specs {
			typ, ok := specification.(*ast.TypeSpec)
			if !ok || typ.Name.Name != owner {
				continue
			}
			body, ok := typ.Type.(*ast.StructType)
			if !ok {
				t.Fatalf("%s is not a generated struct", owner)
			}
			for _, field := range body.Fields.List {
				for _, identifier := range field.Names {
					if identifier.Name == name {
						return field.Type
					}
				}
			}
			return nil
		}
	}
	t.Fatalf("missing generated type %s", owner)
	return nil
}

func ciABI193RequireMethodAbsent(t ciABI193Reporter, file *ast.File, name string) {
	t.Helper()
	for _, declaration := range file.Decls {
		if method, ok := declaration.(*ast.FuncDecl); ok && (method.Name.Name == name || method.Name.Name == name+"Java2goExecution") {
			t.Fatalf("excluded method remains: %s", method.Name.Name)
		}
	}
}

func ciABI193RequireFieldAbsent(t ciABI193Reporter, file *ast.File, owner, name string) {
	t.Helper()
	if ciABI193Field(t, file, owner, name) != nil {
		t.Fatalf("excluded field remains: %s.%s", owner, name)
	}
}

type ciABI193AssertionFailure string

type ciABI193RejectReporter struct{}

func (*ciABI193RejectReporter) Helper() {}
func (*ciABI193RejectReporter) Fatalf(message string, args ...any) {
	panic(ciABI193AssertionFailure(fmt.Sprintf(message, args...)))
}

// Exercise the same assertions used by the migrated parents, including their
// failure path. Unexpected parser/panic errors cannot count as a rejection.
func ciABI193Reject(t *testing.T, message string, check func(ciABI193Reporter)) {
	t.Helper()
	rejected := false
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				failure, ok := recovered.(ciABI193AssertionFailure)
				if !ok || !strings.Contains(string(failure), message) {
					t.Fatalf("unexpected assertion failure: %v", recovered)
				}
				rejected = true
			}
		}()
		check(&ciABI193RejectReporter{})
	}()
	if !rejected {
		t.Fatal("incorrect generated ABI was accepted")
	}
}

func TestCI193CanonicalABIExpressionGuards(t *testing.T) {
	const literal = `stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.StringTypeID, stdjava.JavaStringLiteralUTF16([]uint16{120}), stdjava.JavaStringLiteralUTF16([]uint16{121}))`
	for _, row := range []struct{ name, actual, expected string }{
		{"descriptor", strings.Replace(literal, "StringTypeID", "ObjectTypeID", 1), literal},
		{"genericComponent", strings.Replace(literal, "*stdjava.JavaString", "any", 1), literal},
		{"UTF16Units", strings.Replace(literal, "{120}", "{122}", 1), literal},
		{"elementOrder", strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(literal, "{120}", "{0}"), "{121}", "{120}"), "{0}", "{121}"), literal},
		{"boxing", `stdjava.ReferenceArrayLiteralOf[any](stdjava.ObjectTypeID, int32(1))`, `stdjava.ReferenceArrayLiteralOf[any](stdjava.ObjectTypeID, stdjava.BoxInteger(int32(1)))`},
		{"nominalCast", `stdjava.ReferenceCheckCast[*Box](value, stdjava.ObjectTypeID)`, `stdjava.ReferenceCheckCast[*Box](value, stdjava.TypeID("Box"))`},
		{"missingCast", `value`, `stdjava.ReferenceCheckCast[*Box](value, stdjava.TypeID("Box"))`},
	} {
		t.Run(row.name, func(t *testing.T) {
			actual, err := parser.ParseExpr(row.actual)
			if err != nil {
				t.Fatal(err)
			}
			ciABI193RequireExpr(t, actual, row.actual)
			ciABI193Reject(t, "generated expression", func(report ciABI193Reporter) {
				ciABI193RequireExpr(report, actual, row.expected)
			})
		})
	}
}

func TestCI193CanonicalABIMemberGuards(t *testing.T) {
	positive := ciABI193Parse(t, `package p; type UserService struct { token *stdjava.JavaString }; func (*UserService) InternalName() *stdjava.JavaString { return nil }`)
	ciABI193RequireMethod(t, positive, "InternalName", "*UserService", "*stdjava.JavaString")
	ciABI193RequireExpr(t, ciABI193Field(t, positive, "UserService", "token"), "*stdjava.JavaString")
	for _, row := range []struct{ name, source, message string }{
		{"wrongResult", `package p; type UserService struct{}; func (*UserService) InternalName() string { return "" }`, "generated expression"},
		{"wrongReceiver", `package p; type UserService struct{}; func (*Other) InternalName() *stdjava.JavaString { return nil }`, "receiver/zero-argument"},
		{"wrongArity", `package p; type UserService struct{}; func (*UserService) InternalName(v int) *stdjava.JavaString { return nil }`, "receiver/zero-argument"},
	} {
		t.Run(row.name, func(t *testing.T) {
			file := ciABI193Parse(t, row.source)
			ciABI193Reject(t, row.message, func(report ciABI193Reporter) {
				ciABI193RequireMethod(report, file, "InternalName", "*UserService", "*stdjava.JavaString")
			})
		})
	}
	absent := ciABI193Parse(t, `package p; type UserService struct{}; type Other struct { token *stdjava.JavaString }; func (*UserService) PublicName() *stdjava.JavaString { return nil }`)
	ciABI193RequireFieldAbsent(t, absent, "UserService", "token")
	ciABI193RequireMethodAbsent(t, absent, "InternalName")
	ciABI193RequireMethod(t, absent, "PublicName", "*UserService", "*stdjava.JavaString")
	for _, typ := range []string{"string", "*stdjava.JavaString", "int"} {
		t.Run("retainedField/"+typ, func(t *testing.T) {
			file := ciABI193Parse(t, "package p; type UserService struct { token "+typ+" }")
			ciABI193Reject(t, "excluded field remains", func(report ciABI193Reporter) {
				ciABI193RequireFieldAbsent(report, file, "UserService", "token")
			})
		})
	}
	for _, name := range []string{"InternalName", "InternalNameJava2goExecution"} {
		t.Run("retainedMethod/"+name, func(t *testing.T) {
			file := ciABI193Parse(t, "package p; type UserService struct{}; func (*UserService) "+name+"() {}")
			ciABI193Reject(t, "excluded method remains", func(report ciABI193Reporter) {
				ciABI193RequireMethodAbsent(report, file, "InternalName")
			})
		})
	}
}

func TestCI193CanonicalABIStructureGuards(t *testing.T) {
	file := ciABI193Parse(t, `package p; func Run() int { x := F(); return x }`)
	method := ciABI193Function(t, file, "Run")
	ciABI193RequireExpr(t, ciABI193Local(t, method, "x"), "F()")
	ciABI193RequireCallCount(t, method, "F()", 1)
	ciABI193Reject(t, "missing generated function", func(report ciABI193Reporter) { ciABI193Function(report, file, "Absent") })
	ciABI193Reject(t, "local absent has 0", func(report ciABI193Reporter) { ciABI193Local(report, method, "absent") })
	ciABI193Reject(t, "occurs 1 times, want 2", func(report ciABI193Reporter) { ciABI193RequireCallCount(report, method, "F()", 2) })
	ciABI193Reject(t, "exactly one return", func(report ciABI193Reporter) { ciABI193RequireReturn(report, method, "x") })
	duplicate := ciABI193Parse(t, `package p; func Run() { x := F(); { x := F(); _ = x }; _ = x }`)
	ciABI193Reject(t, "local x has 2", func(report ciABI193Reporter) { ciABI193Local(report, ciABI193Function(report, duplicate, "Run"), "x") })
	stream := ciABI193Parse(t, `package p; func Run() int64 { return stdjava.NewStream[*stdjava.JavaString]().Count() }`)
	ciABI193RequireReturn(t, ciABI193Function(t, stream, "Run"), "stdjava.NewStream[*stdjava.JavaString]().Count()")
	ciABI193Reject(t, "generated expression", func(report ciABI193Reporter) {
		ciABI193RequireReturn(report, ciABI193Function(report, stream, "Run"), "stdjava.NewStream[*stdjava.JavaString]()")
	})
}

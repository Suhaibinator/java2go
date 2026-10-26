package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"testing"
)

// Check the current nominal-array contract without depending on printer spacing.
// The Java semantic/negative controls remain separate and unchanged.
func ci206AssertCompassValues(t ciABI193Reporter, source string) {
	t.Helper()
	file, err := ci199ParseContract(source)
	if err != nil {
		t.Fatalf("enum values runtime/parse contract: %v", err)
		return
	}
	wrapper := ciABI193Function(t, file, "CompassValues")
	implementation := ciABI193Function(t, file, "CompassValuesJava2goExecution")
	for _, fn := range []*ast.FuncDecl{wrapper, implementation} {
		arity := 0
		if fn == implementation {
			arity = 1
		}
		if fn.Recv != nil || fn.Type.TypeParams != nil || fn.Type.Params == nil || fn.Type.Params.NumFields() != arity || fn.Type.Results == nil || fn.Type.Results.NumFields() != 1 || fn.Body == nil {
			t.Fatalf("%s lost its static values signature", fn.Name.Name)
		}
		ciABI193RequireExpr(t, fn.Type.Results.List[0].Type, "*stdjava.ReferenceArray")
		if !ci199NoShadow(fn, "stdjava", "compass", "_compassValues", "CompassValuesJava2goExecution", "compassJava2goEnsureInitialized") {
			t.Fatalf("%s shadows enum array contract bindings", fn.Name.Name)
		}
	}
	ciABI193RequireReturn(t, wrapper, "CompassValuesJava2goExecution(stdjava.NewExecution())")
	parameter := implementation.Type.Params.List[0]
	if len(parameter.Names) != 1 || parameter.Names[0].Name == "_" {
		t.Fatalf("values implementation lost its execution parameter")
	}
	execution := parameter.Names[0].Name
	ciABI193RequireExpr(t, parameter.Type, "*stdjava.Execution")
	body := implementation.Body.List
	if len(body) != 5 {
		t.Fatalf("values requires initialization, guard, allocation, ordered copy, and return")
	}
	ensure, ok := body[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("values initialization is not the first statement")
	}
	ciABI193RequireExpr(t, ensure.X, "compassJava2goEnsureInitialized("+execution+")")
	guard, ok := body[1].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 1 {
		t.Fatalf("values lost its single uninitialized-storage guard")
	}
	ciABI193RequireExpr(t, guard.Cond, "_compassValues == nil")
	panicStatement, ok := guard.Body.List[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("values storage guard must panic")
	}
	panicCall, ok := panicStatement.X.(*ast.CallExpr)
	if !ok || len(panicCall.Args) != 1 {
		t.Fatalf("values storage guard must panic with one throwable")
	}
	ciABI193RequireExpr(t, panicCall.Fun, "panic")
	nullException, ok := panicCall.Args[0].(*ast.CallExpr)
	if !ok || len(nullException.Args) != 1 {
		t.Fatalf("values storage guard lost canonical null exception")
	}
	ciABI193RequireExpr(t, nullException.Fun, "stdjava.NewJavaNullPointerExceptionMessage")
	allocation, ok := body[2].(*ast.AssignStmt)
	if !ok || allocation.Tok != token.DEFINE || len(allocation.Lhs) != 1 || len(allocation.Rhs) != 1 {
		t.Fatalf("values lost its fresh local array allocation")
	}
	array, ok := allocation.Lhs[0].(*ast.Ident)
	if !ok || array.Name == "_" || array.Name == execution {
		t.Fatalf("values lost its distinct allocated array binding")
	}
	ciABI193RequireExpr(t, allocation.Rhs[0], `stdjava.NewReferenceArrayOf[*compass](len(_compassValues), stdjava.TypeID("Compass"))`)
	copyLoop, ok := body[3].(*ast.RangeStmt)
	if !ok || copyLoop.Tok != token.DEFINE || len(copyLoop.Body.List) != 1 {
		t.Fatalf("values lost its ordered element copy")
	}
	ciABI193RequireExpr(t, copyLoop.X, "_compassValues")
	index, indexOK := copyLoop.Key.(*ast.Ident)
	value, valueOK := copyLoop.Value.(*ast.Ident)
	if !indexOK || !valueOK || index.Name == "_" || value.Name == "_" || index.Name == value.Name || index.Name == array.Name || value.Name == array.Name {
		t.Fatalf("values copy bindings are not distinct index and element")
	}
	if index.Name == "stdjava" || value.Name == "stdjava" {
		t.Fatalf("values copy shadows runtime package")
	}
	store, ok := copyLoop.Body.List[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("values lost its ordered element store")
	}
	ciABI193RequireExpr(t, store.X, fmt.Sprintf("stdjava.ReferenceArraySet(%s, %s, %s)", array.Name, index.Name, value.Name))
	result, ok := body[4].(*ast.ReturnStmt)
	if !ok || len(result.Results) != 1 {
		t.Fatalf("values must return its fresh array")
	}
	ciABI193RequireExpr(t, result.Results[0], array.Name)
}

// Detached parsed Go mutants test the same assertions used by the old parent.
// They do not modify or execute the compiler's generated implementation.
func TestCI206EnumValuesASTGuards(t *testing.T) {
	const original = `package main
import stdjava "github.com/NickyBoy89/java2go/stdjava"
func CompassValues() *stdjava.ReferenceArray { return CompassValuesJava2goExecution(stdjava.NewExecution()) }
func CompassValuesJava2goExecution(e *stdjava.Execution) *stdjava.ReferenceArray {
 compassJava2goEnsureInitialized(e)
 if _compassValues == nil { panic(stdjava.NewJavaNullPointerExceptionMessage(nil)) }
 out := stdjava.NewReferenceArrayOf[*compass](len(_compassValues), stdjava.TypeID("Compass"))
 for index, element := range _compassValues { stdjava.ReferenceArraySet(out, index, element) }
 return out
}`
	ci206AssertCompassValues(t, original)
	for _, row := range []struct{ name, old, replacement, message string }{
		{"sliceResult", "func CompassValues() *stdjava.ReferenceArray", "func CompassValues() []*compass", "generated expression"},
		{"wrapperDelegate", "return CompassValuesJava2goExecution(stdjava.NewExecution())", "return nil", "generated expression"},
		{"wrapperExecution", "stdjava.NewExecution()", "nil", "generated expression"},
		{"executionType", "e *stdjava.Execution", "e any", "generated expression"},
		{"missingEnsure", "compassJava2goEnsureInitialized(e)\n", "", "initialization, guard, allocation"},
		{"wrongEnsureExecution", "compassJava2goEnsureInitialized(e)", "compassJava2goEnsureInitialized(stdjava.NewExecution())", "generated expression"},
		{"wrongGuard", "_compassValues == nil", "_compassValues != nil", "generated expression"},
		{"wrongException", "stdjava.NewJavaNullPointerExceptionMessage(nil)", "stdjava.NewJavaIllegalStateExceptionMessage(nil)", "generated expression"},
		{"genericComponent", "NewReferenceArrayOf[*compass]", "NewReferenceArrayOf[any]", "generated expression"},
		{"runtimeComponent", `stdjava.TypeID("Compass")`, "stdjava.ObjectTypeID", "generated expression"},
		{"length", "len(_compassValues)", "len(_compassValues)-1", "generated expression"},
		{"reuseBacking", `stdjava.NewReferenceArrayOf[*compass](len(_compassValues), stdjava.TypeID("Compass"))`, "_compassValues", "generated expression"},
		{"wrongCopySource", "range _compassValues", "range otherValues", "generated expression"},
		{"copyOrder", "ReferenceArraySet(out, index, element)", "ReferenceArraySet(out, element, index)", "generated expression"},
		{"indexShadowsRuntime", "for index, element := range _compassValues { stdjava.ReferenceArraySet(out, index, element) }", "for stdjava, element := range _compassValues { stdjava.ReferenceArraySet(out, stdjava, element) }", "copy shadows runtime package"},
		{"elementShadowsRuntime", "for index, element := range _compassValues { stdjava.ReferenceArraySet(out, index, element) }", "for index, stdjava := range _compassValues { stdjava.ReferenceArraySet(out, index, stdjava) }", "copy shadows runtime package"},
		{"missingCopy", "stdjava.ReferenceArraySet(out, index, element)", "", "ordered element copy"},
		{"returnBacking", "return out", "return _compassValues", "generated expression"},
		{"runtimeBinding", "github.com/NickyBoy89/java2go/stdjava", "example.invalid/runtime", "runtime/parse contract"},
	} {
		t.Run(row.name, func(t *testing.T) {
			if strings.Count(original, row.old) != 1 || row.old == row.replacement {
				t.Fatal("ambiguous or unchanged mutation premise")
			}
			mutant := strings.Replace(original, row.old, row.replacement, 1)
			ciABI193Parse(t, mutant) // A parse error cannot count as a rejection.
			ciABI193Reject(t, row.message, func(report ciABI193Reporter) {
				ci206AssertCompassValues(report, mutant)
			})
		})
	}
}

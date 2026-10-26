package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fix the canonical Add boundary and consumed return expression: each original
// field result is used once in Java order, with the current Execution and the
// String checkcast at the map consumer. The consumed Parser dispatch must keep
// its owned branches and terminal returns, not just unreachable matching calls.
const ci208FieldReturn = `return stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.ConcatJavaStrings(stdjava.JavaStringTextOperandExecution(__java2goExecution, fs.values.Get(0)), stdjava.JavaStringTextOperandExecution(__java2goExecution, stdjava.ObjectView[*stdjava.JavaString](stdjava.MapGetExecution(__java2goExecution, fs.byKey, stdjava.JavaStringLiteralUTF16([]uint16{119, 111, 114, 107, 101, 114})), stdjava.StringTypeID)))), stdjava.JavaStringValueOfInt(int32(fs.values.Size())))`

// Preserve the source-owned Parser value flow at the Add argument boundary.
const ci208FieldAdd = `stdjava.CollectionAddExecution(__java2goExecution, stdjava.EvaluationValue(fs.values), func() *stdjava.JavaString {
		__java2goInvocationReceiver := fs.parser
		__java2goVirtualInterfaceReceiver, __java2goHasVirtualInterfaceReceiver := interface {
		}(__java2goInvocationReceiver).(interface {
			Java2goInterfaceVirtual_7061726974792E616E616C79746963732E6669656C64732E50617273657200706172736501537472696E67(__java2goExecution *stdjava.Execution) *stdjava.JavaString
		})
		if __java2goHasVirtualInterfaceReceiver {
			return __java2goVirtualInterfaceReceiver.Java2goInterfaceVirtual_7061726974792E616E616C79746963732E6669656C64732E50617273657200706172736501537472696E67(__java2goExecution)
		}
		__java2goExecutionReceiver, __java2goHasExecutionReceiver := interface {
		}(__java2goInvocationReceiver).(ParserJava2goExecution)
		if __java2goHasExecutionReceiver {
			return __java2goExecutionReceiver.ParseJava2goExecution(__java2goExecution)
		}
		return interface {
		}(stdjava.ReferenceRequireNonNull(__java2goInvocationReceiver)).(Parser).Parse()
	}())`

const ci208FieldPut = `stdjava.MapPutExecution(__java2goExecution, stdjava.EvaluationValue(fs.byKey), stdjava.JavaStringLiteralUTF16([]uint16{119, 111, 114, 107, 101, 114}), fs.worker.NameJava2goExecution(__java2goExecution))`

func ci208FieldReceiverContract(out string) error {
	file, err := ci199ParseContract(out)
	if err != nil {
		return err
	}
	fn, err := ci199Function(file, "UseJava2goExecution")
	if err != nil {
		return err
	}
	expected := ci199ExpectedFunction(`func (fs *FieldBackedCalls) UseJava2goExecution(__java2goExecution *stdjava.Execution) *stdjava.JavaString {` + ci208FieldPut + `;` + ci208FieldReturn + `}`)
	if fn.Recv == nil || len(fn.Recv.List) != 1 || len(fn.Recv.List[0].Names) != 1 || fn.Recv.List[0].Names[0].Name != "fs" || ci199ASTText(fn.Recv.List[0].Type) != ci199ASTText(expected.Recv.List[0].Type) || ci199ASTText(fn.Type) != ci199ASTText(expected.Type) {
		return ci199Reject("field receiver, current Execution formal or String result changed")
	}
	if !ci199NoShadow(fn, "stdjava") || !ci208FieldReceiverBodyNoShadow(fn) {
		return ci199Reject("field receiver or runtime binding shadowed")
	}
	if fn.Body == nil || len(fn.Body.List) == 0 {
		return ci199Reject("missing field receiver body")
	}
	// Unlike an interface method signature, an assignment/range/value declaration
	// can replace the current Execution used by the original invocation.
	rebound := false
	counts := map[string]int{}
	fields := map[string]int{}
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		check := func(expr ast.Expr) {
			if name, ok := expr.(*ast.Ident); ok && (name.Name == "__java2goExecution" || name.Name == "fs") {
				rebound = true
			}
		}
		switch node := node.(type) {
		case *ast.AssignStmt:
			for _, left := range node.Lhs {
				check(left)
			}
		case *ast.RangeStmt:
			check(node.Key)
			check(node.Value)
		case *ast.ValueSpec:
			for _, name := range node.Names {
				check(name)
			}
		case *ast.CallExpr:
			if selector, ok := node.Fun.(*ast.SelectorExpr); ok {
				counts[selector.Sel.Name]++
			}
		case *ast.SelectorExpr:
			if owner, ok := node.X.(*ast.Ident); ok && owner.Name == "fs" {
				fields[node.Sel.Name]++
			}
		}
		return true
	})
	if rebound {
		return ci199Reject("current Execution or field receiver rebound")
	}
	for _, name := range []string{"CollectionAddExecution", "MapPutExecution", "Get", "MapGetExecution", "Size"} {
		if counts[name] != 1 {
			return ci199Reject("field operation missing or evaluated more than once: " + name)
		}
	}
	if counts["Add"] != 0 || fields["values"] != 3 || fields["byKey"] != 2 || fields["worker"] != 1 || fields["parser"] != 1 {
		return ci199Reject("legacy Add or duplicate/missing original field evaluation")
	}
	add, put := -1, -1
	for i, statement := range fn.Body.List {
		expression, ok := statement.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expression.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		if ci208CanonicalFieldAdd(call) {
			add = i
		}
		if ci199ASTText(statement) == ci199ASTText(expected.Body.List[0]) {
			put = i
		}
	}
	result, ok := fn.Body.List[len(fn.Body.List)-1].(*ast.ReturnStmt)
	if add < 0 || put <= add || put >= len(fn.Body.List)-1 || !ok || ci199ASTText(result) != ci199ASTText(expected.Body.List[1]) {
		return ci199Reject("field Add/Put order, map receiver/key/checkcast, return consumption or evaluation order changed")
	}
	return nil
}

// Pin this fixture's complete owned call, including its Parser argument flow.
// Counting matching calls alone would accept an early return or repeated parse.
func ci208CanonicalFieldAdd(call *ast.CallExpr) bool {
	expected := ci199ExpectedFunction("func add() {" + ci208FieldAdd + "}")
	return ci199ASTText(call) == ci199ASTText(expected.Body.List[0].(*ast.ExprStmt).X)
}

// Preserve the parsed declaration's Name, Type and Body for ast.Walk. Removing
// only the receiver keeps that formal out of the body-shadow check without
// manufacturing an incomplete FuncDecl or changing the original declaration.
func ci208FieldReceiverBodyNoShadow(fn *ast.FuncDecl) bool {
	bodyFunction := *fn
	bodyFunction.Recv = nil
	return ci199NoShadow(&bodyFunction, "fs")
}

func TestCI208FieldReceiverContractNegativeControls(t *testing.T) {
	// Pure AST construction regression: this must not panic, the receiver
	// formal is allowed, an inner body shadow is rejected, and fn is unchanged.
	probe := ci199ExpectedFunction(`func (fs *FieldBackedCalls) probe(__java2goExecution *stdjava.Execution) { _ = fs }`)
	before := ci199ASTText(probe)
	if !ci208FieldReceiverBodyNoShadow(probe) || ci199ASTText(probe) != before {
		t.Fatal("valid receiver formal rejected or original parsed function changed")
	}
	shadow := ci199ExpectedFunction(`func (fs *FieldBackedCalls) probe(__java2goExecution *stdjava.Execution) { { fs := new(FieldBackedCalls); _ = fs } }`)
	if ci208FieldReceiverBodyNoShadow(shadow) {
		t.Fatal("body receiver shadow escaped the pure construction guard")
	}
	t.Log("pure guard construction control passed")
	out := convertJavaProjectDir(t, filepath.Join("testdata", "ci208_field_receiver"))["FieldBackedCalls.go"]
	const fn = "UseJava2goExecution"
	const key = "stdjava.JavaStringLiteralUTF16([]uint16{119, 111, 114, 107, 101, 114})"
	const get = "stdjava.MapGetExecution(__java2goExecution, fs.byKey, " + key + ")"
	const list = "stdjava.JavaStringTextOperandExecution(__java2goExecution, fs.values.Get(0))"
	const size = "stdjava.JavaStringValueOfInt(int32(fs.values.Size()))"
	positions := token.NewFileSet()
	file, err := parser.ParseFile(positions, "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	method, err := ci199Function(file, fn)
	if err != nil {
		t.Fatal(err)
	}
	statement := method.Body.List[len(method.Body.List)-1]
	tail := out[positions.Position(statement.Pos()).Offset:positions.Position(statement.End()).Offset]
	reordered := strings.Replace(tail, list, "__ci208Swap", 1)
	reordered = strings.Replace(reordered, size, list, 1)
	reordered = strings.Replace(reordered, "__ci208Swap", size, 1)
	var addStatement, putStatement ast.Stmt
	var addArgument ast.Expr
	for _, statement := range method.Body.List {
		if expression, ok := statement.(*ast.ExprStmt); ok {
			if call, ok := expression.X.(*ast.CallExpr); ok {
				switch ci199ASTText(call.Fun) {
				case "stdjava.CollectionAddExecution":
					addStatement, addArgument = statement, call.Args[2]
				case "stdjava.MapPutExecution":
					putStatement = statement
				}
			}
		}
	}
	if addStatement == nil || putStatement == nil || addArgument == nil {
		t.Fatal("missing canonical field Add/Put mutation preimages")
	}
	region := func(node ast.Node) string {
		return out[positions.Position(node.Pos()).Offset:positions.Position(node.End()).Offset]
	}
	const addPrefix = "stdjava.CollectionAddExecution(__java2goExecution, stdjava.EvaluationValue(fs.values),"
	addPut := out[positions.Position(addStatement.Pos()).Offset:positions.Position(putStatement.End()).Offset]
	ci199CheckNegatives(t, out, ci208FieldReceiverContract, []ci199Mutation{
		{"wrong-map-operation", fn, get, strings.Replace(get, "MapGetExecution", "MapRemoveExecution", 1)},
		{"wrong-current-execution", fn, get, strings.Replace(get, "__java2goExecution", "stdjava.NewExecution()", 1)},
		{"wrong-map-receiver", fn, get, strings.Replace(get, "fs.byKey", "stdjava.NewMap[*stdjava.JavaString, *stdjava.JavaString]()", 1)},
		{"wrong-key", fn, get, strings.Replace(get, "101, 114}", "101, 115}", 1)},
		{"wrong-string-view", fn, "stdjava.ObjectView[*stdjava.JavaString]", "stdjava.ObjectView[any]"},
		{"wrong-nominal-view", fn, "stdjava.StringTypeID", "stdjava.ObjectTypeID"},
		{"wrong-evaluation-order", fn, tail, reordered},
		{"duplicate-map-evaluation", fn, tail, "_ = " + get + "\n" + tail},
		{"rebound-current-execution", fn, ci208FieldPut, "__java2goExecution = stdjava.NewExecution()\n" + ci208FieldPut},
		{"rebound-field-receiver", fn, ci208FieldPut, "fs = new(FieldBackedCalls)\n" + ci208FieldPut},
		{"wrong-put-receiver", fn, ci208FieldPut, strings.Replace(ci208FieldPut, "stdjava.EvaluationValue(fs.byKey)", "stdjava.NewMap[*stdjava.JavaString, *stdjava.JavaString]()", 1)},
		{"wrong-add-operation", fn, addPrefix, strings.Replace(addPrefix, "CollectionAddExecution", "CollectionRemoveExecution", 1)},
		{"wrong-add-current-execution", fn, addPrefix, strings.Replace(addPrefix, "__java2goExecution", "stdjava.NewExecution()", 1)},
		{"wrong-add-runtime-owner", fn, addPrefix, strings.Replace(addPrefix, "stdjava.CollectionAddExecution", "other.CollectionAddExecution", 1)},
		{"wrong-add-arity", fn, addPrefix, strings.Replace(addPrefix, "__java2goExecution, ", "", 1)},
		{"wrong-add-receiver", fn, addPrefix, strings.Replace(addPrefix, "stdjava.EvaluationValue(fs.values)", "stdjava.NewArrayList[*stdjava.JavaString]()", 1)},
		{"unstaged-add-receiver", fn, addPrefix, strings.Replace(addPrefix, "stdjava.EvaluationValue(fs.values)", "fs.values", 1)},
		{"duplicate-add-evaluation", fn, region(addStatement), region(addStatement) + "\n" + region(addStatement)},
		{"duplicate-parser-receiver", fn, "__java2goInvocationReceiver := fs.parser", "_ = fs.parser\n__java2goInvocationReceiver := fs.parser"},
		{"wrong-add-value", fn, region(addArgument), "stdjava.JavaStringLiteralUTF16([]uint16{120})"},
		{"wrong-parser-current-execution", fn, ".ParseJava2goExecution(__java2goExecution)", ".ParseJava2goExecution(stdjava.NewExecution())"},
		{"wrong-add-put-order", fn, addPut, region(putStatement) + "\n" + region(addStatement)},
		{"early-parser-return", fn, "__java2goInvocationReceiver := fs.parser", "__java2goInvocationReceiver := fs.parser\nreturn stdjava.JavaStringLiteralUTF16([]uint16{120})"},
		{"duplicate-parser-call", fn, "return __java2goExecutionReceiver.ParseJava2goExecution(__java2goExecution)", "_ = __java2goExecutionReceiver.ParseJava2goExecution(__java2goExecution)\nreturn __java2goExecutionReceiver.ParseJava2goExecution(__java2goExecution)"},
		{"wrong-parser-null-check", fn, "stdjava.ReferenceRequireNonNull(__java2goInvocationReceiver)", "stdjava.EvaluationValue(__java2goInvocationReceiver)"},
		{"runtime-import-spoof", "", `"github.com/NickyBoy89/java2go/stdjava"`, `"example.invalid/stdjava"`},
	})
}

// oracle.stdout is a separately captured JDK21 observation of these frozen
// complete sources. This test repeats that observation before any Go comparison.
func TestCI208FieldReceiverFullProjectJDK21(t *testing.T) {
	fixture := filepath.Join("testdata", "ci208_field_receiver")
	want, err := os.ReadFile(filepath.Join(fixture, "oracle.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"pom.xml": `<project><groupId>parity</groupId><artifactId>fieldreceiver</artifactId><version>1</version></project>`}
	for _, name := range []string{"Parser.java", "Worker.java", "FieldBackedCalls.java", "FieldReceiverProbe.java", "FieldReceiverDriver.java"} {
		data, err := os.ReadFile(filepath.Join(fixture, name))
		if err != nil {
			t.Fatal(err)
		}
		files["src/main/java/parity/analytics/fields/"+name] = string(data)
	}
	runCampaignCompilerStrictProjectOracle(t, files, "parity.analytics.fields.FieldReceiverDriver", string(want))
}

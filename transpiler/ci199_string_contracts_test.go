package transpiler

import (
	"bytes"
	"errors"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

const ci199OriginalGenericReceiverJava = `
class Box<T> {
    private T value;
    Box(T value) { this.value = value; }
    public T value() { return this.value; }
}

public class App {
    public static boolean blocked(Box<String> box) {
        return box.value().startsWith("BLOCK:");
    }
    public static String run() {
        return blocked(new Box<String>("BLOCK:value")) + ":" + blocked(new Box<String>("allow"));
    }
}
`
const ci199OriginalBuiltinArrayJava = `
import java.util.ArrayList;
import java.util.concurrent.atomic.AtomicInteger;

public class BuiltinReferenceArrays {
    interface Marker { }
    static class Marked implements Marker { }

    static class ClosingResource implements AutoCloseable {
        String message;
        ClosingResource(String message) { this.message = message; }
        public void close() { throw new IllegalStateException(message); }
    }

    static <T> T first(T[] values) {
        return values[0];
    }

    static Throwable[] suppressedValues() {
        try (ClosingResource first = new ClosingResource("alpha");
             ClosingResource second = new ClosingResource("beta")) {
            throw new Exception("primary");
        } catch (Exception primary) {
            return primary.getSuppressed();
        }
    }

    public static int run() {
        Object token = new Object();
        Object[] objects = new Object[] {
            token,
            new ArrayList<String>(),
            new AtomicInteger(3),
            new RuntimeException("runtime")
        };
        int score = 0;
        for (Object value : objects) {
            if (value != null) score++;
        }
        if (objects[0] == token) score += 10;

        Throwable[] suppressed = suppressedValues();
        for (Throwable value : suppressed) {
			if (value.getMessage() == "beta") score = score * 10 + 4;
			if (value.getMessage() == "alpha") score = score * 10 + 5;
        }

        Throwable[] nullable = new Throwable[] { null, new RuntimeException("z") };
        if (nullable[0] == null) score += 100;
        if (nullable[1].getMessage() == "z") score += 1000;

        Marked[] marked = new Marked[1];
        Object[] markedView = marked;
        try {
            markedView[0] = new Object();
            score += 10000;
        } catch (ArrayStoreException expected) {
            score += 20000;
        }

        Marker[] markers = new Marker[1];
        Object[] markerView = markers;
        try {
            markerView[0] = new Object();
            score += 100000;
        } catch (ArrayStoreException expected) {
            score += 200000;
        }

        String genericNull = first(new String[] { null });
        if (genericNull == null) score += 1000000;
        return score;
    }
}
`
const ci199OriginalBuiltinArrayGoChild = `
package main

import "testing"

func TestBuiltinReferenceArrayParity(t *testing.T) {
	if got := Run(); got != 1222545 {
		t.Fatalf("Run() = %d, want 1222545", got)
    }
}
`

// These fixtures describe only the two original programs' exact canonical ABI.
// Negative controls operate on detached strings, never on generated programs
// used as behavioral evidence.
const ci199ExpectedGenericReceiver = `func BlockedJava2goExecution(__java2goExecution *stdjava.Execution, boxJava2goLocal *box[*stdjava.JavaString]) bool {
 return func() bool {
  __java2goInvocationReceiver := stdjava.ObjectView[*stdjava.JavaString](func() any {
   __java2goInvocationReceiver := boxJava2goLocal
   stdjava.RequireInvocationReceiver(__java2goInvocationReceiver, "Cannot invoke \"Box.value()\" because \"<parameter1>\" is null")
   return __java2goInvocationReceiver.ValueJava2goExecution(__java2goExecution)
  }(), stdjava.StringTypeID)
  __java2goInvocationArg0 := stdjava.JavaStringLiteralUTF16([]uint16{66, 76, 79, 67, 75, 58})
  return stdjava.JavaStringStartsWith(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)
 }()
}`

const ci199ExpectedFirst = `func firstJava2goExecution[T any](__java2goExecution *stdjava.Execution, values *stdjava.ReferenceArray) T {
 return stdjava.ReferenceArrayGet[T](values, 0, stdjava.ObjectTypeID)
}`
const ci199ExpectedNullAssignment = `genericNull := firstJava2goExecution[*stdjava.JavaString](__java2goExecution, stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.StringTypeID, (*stdjava.JavaString)(nil)))`
const ci199ExpectedNullUse = `if stdjava.JavaReferenceEqual(genericNull, nil) { score += 1000000 }`

type ci199ContractError struct {
	parse bool
	why   string
}

func (e *ci199ContractError) Error() string { return "CI199_STRING_CONTRACT: " + e.why }
func ci199Reject(why string) error          { return &ci199ContractError{why: why} }
func ci199ASTText(node ast.Node) string {
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		return "invalid AST: " + err.Error()
	}
	return out.String()
}

func ci199ParseContract(out string) (*ast.File, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		return nil, &ci199ContractError{parse: true, why: err.Error()}
	}
	runtimeImports := 0
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return nil, ci199Reject("invalid import")
		}
		if path == "github.com/NickyBoy89/java2go/stdjava" {
			if spec.Name != nil && spec.Name.Name != "stdjava" {
				return nil, ci199Reject("runtime binding changed")
			}
			runtimeImports++
		} else if spec.Name != nil && spec.Name.Name == "stdjava" {
			return nil, ci199Reject("runtime binding spoofed")
		}
	}
	if runtimeImports != 1 {
		return nil, ci199Reject("exact runtime import missing")
	}
	return file, nil
}

func ci199Function(file *ast.File, name string) (*ast.FuncDecl, error) {
	var found *ast.FuncDecl
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if ok && fn.Name.Name == name {
			if found != nil {
				return nil, ci199Reject("duplicate function " + name)
			}
			found = fn
		}
	}
	if found == nil {
		return nil, ci199Reject("missing function " + name)
	}
	return found, nil
}

func ci199ExpectedFunction(text string) *ast.FuncDecl {
	file, err := parser.ParseFile(token.NewFileSet(), "expected.go", "package main\n"+text, 0)
	if err != nil {
		panic(err)
	}
	return file.Decls[0].(*ast.FuncDecl)
}

func ci199NoShadow(fn *ast.FuncDecl, reserved ...string) bool {
	names := map[string]bool{}
	for _, name := range reserved {
		names[name] = true
	}
	ok := true
	ast.Inspect(fn, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, left := range node.Lhs {
					if name, isName := left.(*ast.Ident); isName && names[name.Name] {
						ok = false
					}
				}
			}
		case *ast.ValueSpec:
			for _, name := range node.Names {
				if names[name.Name] {
					ok = false
				}
			}
		case *ast.TypeSpec:
			if names[node.Name.Name] {
				ok = false
			}
		case *ast.Field:
			for _, name := range node.Names {
				if names[name.Name] {
					ok = false
				}
			}
		}
		return ok
	})
	return ok
}

func ci199GenericReceiverContract(out string) error {
	file, err := ci199ParseContract(out)
	if err != nil {
		return err
	}
	fn, err := ci199Function(file, "BlockedJava2goExecution")
	if err != nil {
		return err
	}
	if !ci199NoShadow(fn, "stdjava") {
		return ci199Reject("runtime shadowed")
	}
	if ci199ASTText(fn) != ci199ASTText(ci199ExpectedFunction(ci199ExpectedGenericReceiver)) {
		return ci199Reject("generic receiver owner/ABI/Execution/UTF16 prefix/once/null/staged call changed")
	}
	return nil
}

func ci199StringArrayContract(out string) error {
	file, err := ci199ParseContract(out)
	if err != nil {
		return err
	}
	first, err := ci199Function(file, "firstJava2goExecution")
	if err != nil {
		return err
	}
	if ci199ASTText(first) != ci199ASTText(ci199ExpectedFunction(ci199ExpectedFirst)) {
		return ci199Reject("generic first owner/ABI/index/component/result changed")
	}
	run, err := ci199Function(file, "RunJava2goExecution")
	if err != nil {
		return err
	}
	wantRun := ci199ExpectedFunction("func RunJava2goExecution(__java2goExecution *stdjava.Execution) int32 {}")
	if run.Recv != nil || ci199ASTText(run.Type) != ci199ASTText(wantRun.Type) {
		return ci199Reject("original Run ABI changed")
	}
	if !ci199NoShadow(run, "stdjava", "firstJava2goExecution") {
		return ci199Reject("array call binding shadowed")
	}
	if run.Body == nil || len(run.Body.List) < 3 {
		return ci199Reject("original null result statements missing")
	}
	wantTail := ci199ExpectedFunction("func check() {" + ci199ExpectedNullAssignment + "\n" + ci199ExpectedNullUse + "\nreturn score\n}").Body.List
	tail := run.Body.List[len(run.Body.List)-3:]
	for index := range tail {
		if ci199ASTText(tail[index]) != ci199ASTText(wantTail[index]) {
			return ci199Reject("canonical String generic argument/array identity/null/component/Execution/result use changed")
		}
	}
	calls, uses := 0, 0
	ast.Inspect(run.Body, func(node ast.Node) bool {
		if name, ok := node.(*ast.Ident); ok && name.Name == "genericNull" {
			uses++
		}
		if call, ok := node.(*ast.CallExpr); ok {
			fun := call.Fun
			if indexed, ok := fun.(*ast.IndexExpr); ok {
				fun = indexed.X
			}
			if name, ok := fun.(*ast.Ident); ok && name.Name == "firstJava2goExecution" {
				calls++
			}
		}
		return true
	})
	if calls != 1 || uses != 2 {
		return ci199Reject("generic first call or bound null value duplicated/ignored")
	}
	return nil
}

func ci199AssertGenericReceiverContract(t *testing.T, out string) {
	t.Helper()
	if err := ci199GenericReceiverContract(out); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Keep the original fragment-list assertion as well as the stronger AST proof.
func ci199RequireStringArrayContract(t *testing.T, out string) string {
	t.Helper()
	if err := ci199StringArrayContract(out); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	return "firstJava2goExecution[*stdjava.JavaString]"
}

func TestCI199GenericReceiverASTGuard(t *testing.T) {
	ci199AssertGenericReceiverContract(t, renderGoFileFromJava(t, ci199OriginalGenericReceiverJava))
}

func TestCI199StringArrayASTGuard(t *testing.T) {
	ci199RequireStringArrayContract(t, renderGoFileFromJava(t, ci199OriginalBuiltinArrayJava))
}

type ci199Mutation struct{ name, function, before, after string }

func ci199Mutate(t *testing.T, original string, mutation ci199Mutation) string {
	t.Helper()
	start, end := 0, len(original)
	if mutation.function != "" {
		set := token.NewFileSet()
		file, err := parser.ParseFile(set, "original.go", original, 0)
		if err != nil {
			t.Fatal(err)
		}
		fn, err := ci199Function(file, mutation.function)
		if err != nil {
			t.Fatal(err)
		}
		start, end = set.Position(fn.Pos()).Offset, set.Position(fn.End()).Offset
	}
	region := original[start:end]
	if strings.Count(region, mutation.before) != 1 {
		t.Fatalf("mutation %s has no unique preimage", mutation.name)
	}
	return original[:start] + strings.Replace(region, mutation.before, mutation.after, 1) + original[end:]
}

func ci199CheckNegatives(t *testing.T, out string, guard func(string) error, mutations []ci199Mutation) {
	t.Helper()
	if err := guard(out); err != nil {
		t.Fatalf("positive baseline rejected: %v", err)
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			mutant := ci199Mutate(t, out, mutation)
			if _, err := parser.ParseFile(token.NewFileSet(), "detached-negative.go", mutant, 0); err != nil {
				t.Fatalf("negative control is not syntactically valid: %v", err)
			}
			err := guard(mutant)
			var rejected *ci199ContractError
			if !errors.As(err, &rejected) || rejected.parse {
				t.Fatalf("negative control escaped contract-only rejection: %v", err)
			}
		})
	}
}

func TestCI199GenericReceiverNegativeControls(t *testing.T) {
	fn := "BlockedJava2goExecution"
	const call = "stdjava.JavaStringStartsWith(stdjava.RequireJavaString(__java2goInvocationReceiver), __java2goInvocationArg0)"
	ci199CheckNegatives(t, renderGoFileFromJava(t, ci199OriginalGenericReceiverJava), ci199GenericReceiverContract, []ci199Mutation{
		{"host-string-view", fn, "stdjava.ObjectView[*stdjava.JavaString]", "stdjava.ObjectView[string]"},
		{"wrong-nominal-view", fn, "stdjava.StringTypeID", "stdjava.ObjectTypeID"},
		{"wrong-owner-ABI", fn, "*box[*stdjava.JavaString]", "any"},
		{"wrong-method", fn, "ValueJava2goExecution(__java2goExecution)", "OtherJava2goExecution(__java2goExecution)"},
		{"wrong-execution", fn, "ValueJava2goExecution(__java2goExecution)", "ValueJava2goExecution(stdjava.NewExecution())"},
		{"duplicate-value-evaluation", fn, "return __java2goInvocationReceiver.ValueJava2goExecution(__java2goExecution)", "_ = __java2goInvocationReceiver.ValueJava2goExecution(__java2goExecution)\nreturn __java2goInvocationReceiver.ValueJava2goExecution(__java2goExecution)"},
		{"wrong-prefix", fn, "[]uint16{66, 76, 79, 67, 75, 58}", "[]uint16{66, 76, 79, 67, 75, 59}"},
		{"wrong-string-intrinsic", fn, "stdjava.JavaStringStartsWith", "stdjava.JavaStringEndsWith"},
		{"constant-return", fn, call, "false"},
		{"swapped-arguments", fn, call, "stdjava.JavaStringStartsWith(__java2goInvocationArg0, stdjava.RequireJavaString(__java2goInvocationReceiver))"},
		{"missing-string-null-check", fn, "stdjava.RequireJavaString(__java2goInvocationReceiver)", "__java2goInvocationReceiver"},
		{"runtime-import-spoof", "", "\"github.com/NickyBoy89/java2go/stdjava\"", "\"example.invalid/stdjava\""},
	})
}

func TestCI199StringArrayNegativeControls(t *testing.T) {
	const run = "RunJava2goExecution"
	const array = "stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.StringTypeID, (*stdjava.JavaString)(nil))"
	const call = "firstJava2goExecution[*stdjava.JavaString](__java2goExecution, " + array + ")"
	ci199CheckNegatives(t, renderGoFileFromJava(t, ci199OriginalBuiltinArrayJava), ci199StringArrayContract, []ci199Mutation{
		{"host-string-instantiation", run, "firstJava2goExecution[*stdjava.JavaString]", "firstJava2goExecution[string]"},
		{"wrong-execution", run, call, "firstJava2goExecution[*stdjava.JavaString](stdjava.NewExecution(), " + array + ")"},
		{"wrong-component", run, array, "stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.ObjectTypeID, (*stdjava.JavaString)(nil))"},
		{"wrong-array-element-type", run, array, "stdjava.ReferenceArrayLiteralOf[any](stdjava.StringTypeID, (*stdjava.JavaString)(nil))"},
		{"null-replaced-nonnull", run, array, "stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.StringTypeID, stdjava.JavaStringLiteralUTF16([]uint16{}))"},
		{"empty-array", run, array, "stdjava.ReferenceArrayLiteralOf[*stdjava.JavaString](stdjava.StringTypeID)"},
		{"duplicate-first-call", run, "genericNull := " + call, "genericNull := " + call + "\n_ = " + call},
		{"wrong-callee", run, "firstJava2goExecution[*stdjava.JavaString]", "otherJava2goExecution[*stdjava.JavaString]"},
		{"ignored-result", run, "stdjava.JavaReferenceEqual(genericNull, nil)", "stdjava.JavaReferenceEqual(nil, nil)"},
		{"constant-null-test", run, "stdjava.JavaReferenceEqual(genericNull, nil)", "true"},
		{"wrong-behavior-score", run, "score += 1000000", "score += 1"},
		{"wrong-get-index", "firstJava2goExecution", "values, 0, stdjava.ObjectTypeID", "values, 1, stdjava.ObjectTypeID"},
		{"wrong-generic-bound-descriptor", "firstJava2goExecution", "stdjava.ObjectTypeID", "stdjava.StringTypeID"},
		{"runtime-import-spoof", "", "\"github.com/NickyBoy89/java2go/stdjava\"", "\"example.invalid/stdjava\""},
	})
}

func TestCI199OriginalGenericReceiverBehavior(t *testing.T) {
	want := campaignRuntimeJavaOracle(t, "App", ci199OriginalGenericReceiverJava)
	assertGeneratedLocalConstructorResult(t, ci199OriginalGenericReceiverJava, want)
}

func TestCI199OriginalBuiltinArrayBehavior(t *testing.T) {
	runGeneratedWithStdjava(t, renderGoFileFromJava(t, ci199OriginalBuiltinArrayJava), ci199OriginalBuiltinArrayGoChild)
}

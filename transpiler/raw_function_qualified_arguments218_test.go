package transpiler

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

// The complete type spelling is copied from the legal stage69 lifecycle source.
const qualifiedFunction218Source = `import java.util.function.Function;
class Main {
 static Function<java.lang.String,java.lang.String> make() {
  return new java.util.function.Function<java.lang.String,java.lang.String>() {
   class String {}
   public java.lang.String apply(java.lang.String value) { return value + "!"; }
  };
 }
}`

func qualifiedFunction218Logs(t *testing.T, call func()) []byte {
	t.Helper()
	logger := log.StandardLogger()
	oldOutput, oldFormatter, oldLevel := logger.Out, logger.Formatter, logger.GetLevel()
	var output bytes.Buffer
	logger.SetOutput(&output)
	logger.SetFormatter(&log.JSONFormatter{DisableTimestamp: true})
	logger.SetLevel(log.WarnLevel)
	func() {
		defer logger.SetOutput(oldOutput)
		defer logger.SetFormatter(oldFormatter)
		defer logger.SetLevel(oldLevel)
		call()
	}()
	return append([]byte(nil), output.Bytes()...)
}

func TestCI218RawFunctionQualifiedArgumentBoundary(t *testing.T) {
	rows := []struct{ name, spelling string }{
		{"qualified_arguments", "Function<java.lang.String,java.lang.String>"},
		{"nested_arguments", "Function<java.util.List<java.lang.String>,java.lang.String>"},
		{"array_qualified_arguments", "Function<java.lang.String,java.lang.String>[]"},
		{"qualified_owner_arguments", "java.util.function.Function<java.lang.String,java.lang.String>"},
		{"wildcard_qualified_bound", "Function<? extends java.lang.String,java.lang.String>"},
		{"simple_parameterized", "Function<String,String>"},
		{"raw_array", "Function[]"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, "import java.util.function.Function;class Main{"+row.spelling+" value;void body(){}}")
			ctx := classScopeCtx(h.File.Symbols.FindClassScope("Main"), h.Ctx)
			ctx.localScope = ctx.currentClass.FindMethodByName("body", nil)
			field := findNode(h.File.Ast, "field_declaration")
			if field == nil || field.ChildByFieldName("type") == nil {
				t.Fatal("actual declared target type missing")
			}
			written := field.ChildByFieldName("type").Content(h.File.Source)
			if written != row.spelling {
				t.Fatal("actual declared target differs from source spelling")
			}
			var plan *rawFunctionTargetABI
			output := qualifiedFunction218Logs(t, func() { plan = rawCanonicalFunctionTarget(written, ctx) })
			if plan != nil {
				t.Fatal("parameterized or array target acquired a raw Function plan")
			}
			if len(output) != 0 {
				t.Fatal("raw-only eligibility traversed qualified argument prefixes before refusing target")
			}
		})
	}
}

func TestCI218TypedFunctionSAMAnonymousRootNoRawProbe(t *testing.T) {
	h := setupParseHelper(t, qualifiedFunction218Source)
	ctx := classScopeCtx(h.File.Symbols.FindClassScope("Main"), h.Ctx)
	ctx.localScope = ctx.currentClass.FindMethodByName("make", nil)
	creation := findNode(h.File.Ast, "object_creation_expression")
	if ctx.localScope == nil || creation == nil {
		t.Fatal("actual typed anonymous consumer missing")
	}
	written := ctx.localScope.DeclarationNode.ChildByFieldName("type").Content(h.File.Source)
	ctx.expectedType, ctx.expectedTypeRoot = written, creation
	output := qualifiedFunction218Logs(t, func() {
		if rawCanonicalFunctionLambdaTarget(written, ctx) != nil {
			t.Fatal("anonymous allocation acquired a raw lambda plan")
		}
		method, bindings := resolveFunctionalInterfaceMethod(ctx, written)
		if method == nil || len(method.Parameters) != 1 || substituteJavaTypeParams(method.OriginalType, bindings) != "java.lang.String" || substituteJavaTypeParams(method.Parameters[0].OriginalType, bindings) != "java.lang.String" {
			t.Fatal("typed Function SAM slots changed while refusing raw eligibility")
		}
	})
	if len(output) != 0 {
		t.Fatal("typed anonymous SAM resolution entered raw qualified-prefix parsing")
	}
}

func TestCI218RawFunctionMalformedTargetDiagnostic(t *testing.T) {
	rows := []struct{ name, spelling, message string }{
		{"unclosed", "Function<java.lang.String", "Unbalanced angle brackets in type string: unclosed '<'"},
		{"stray_close", "Function<java.lang.String>>", "Unbalanced angle brackets in type string: too many '>'"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, `import java.util.function.Function;class Main{}`)
			ctx := classScopeCtx(h.File.Symbols.FindClassScope("Main"), h.Ctx)
			output := qualifiedFunction218Logs(t, func() {
				if rawCanonicalFunctionTarget(row.spelling, ctx) != nil {
					t.Fatal("malformed target acquired a raw Function plan")
				}
			})
			var record map[string]any
			if json.Unmarshal(output, &record) != nil || len(record) != 3 || record["level"] != "warning" || record["msg"] != row.message || record["typeStr"] != strings.TrimSpace(row.spelling) {
				t.Fatal("malformed complete target did not retain exactly its original parser diagnostic")
			}
		})
	}
}

// The RED plan isolates eligibility and typed resolution. The GREEN plan must
// additionally execute this real lowering/coercion path and the original strict
// lifecycle CLI case, under the unchanged no-warning contract.
func TestCI218QualifiedFunctionAnonymousCoercionNoDiagnostic(t *testing.T) {
	output := qualifiedFunction218Logs(t, func() {
		ctx, expression, _, info := anonymousFunctional209Lower(t, qualifiedFunction218Source, "Function<java.lang.String,java.lang.String>")
		creation := info.scope.Class.DeclarationNode
		if expression == nil || creation == nil || coerceArgumentToExpectedType(expression, creation, ctx.expectedType, ctx, ctx.currentFile.Source) == nil {
			t.Fatal("actual anonymous Function lowering or consuming coercion missing")
		}
	})
	if len(output) != 0 {
		t.Fatal("qualified Function lowering/coercion produced a diagnostic")
	}
}

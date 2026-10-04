package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func anonymousReceiverContainsIdentifier(expression ast.Expr, name string) bool {
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && identifier.Name == name {
			found = true
		}
		return true
	})
	return found
}

// The actual prepared constructor carries the caller's Execution and captured
// local. Reusing it after a lexical/emission/ABI change leaks those old values.
func TestAnonymousReceiverContextIsolationTDD(t *testing.T) {
	source := `public class AnonymousCacheProbe {static class Base {Base(){}} static int run(int seed){return (new Base(){volatile int flag=seed;}).flag;}}`
	helper := setupParseHelper(t, source)
	ctx := helper.Ctx
	ctx.className = ctx.currentClass.Class.Name
	for _, method := range ctx.currentClass.Methods {
		if method.OriginalName == "run" {
			ctx.localScope = method
		}
	}
	if ctx.localScope == nil || len(ctx.localScope.Parameters) != 1 {
		t.Fatal("original cache witness parameter missing")
	}
	var declarations []ast.Decl
	var counter int
	ctx.hoistedDecls = &declarations
	ctx.anonClassCounter = &counter
	ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
	ctx.localClasses = make(map[string]*localClassInfo)
	ctx.executionContextName = "executionA"
	field := findNode(helper.File.Ast, "field_access")
	if field == nil {
		t.Fatal("source anonymous member selector missing")
	}
	prepared := prepareAnonymousFieldReceiver(field, helper.File.Source, ctx)
	receiver := field.ChildByFieldName("object")
	original, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, prepared)
	if !ok || !anonymousReceiverContainsIdentifier(original, "executionA") || !anonymousReceiverContainsIdentifier(original, "seed") {
		t.Fatal("actual constructor did not capture original execution/local")
	}
	assertSingleConstruction := func(t *testing.T, expression ast.Expr) {
		base := resolveClassScopeByQualifiedName(prepared, "Base")
		if base == nil {
			t.Fatal("written Base constructor declaration missing")
		}
		names := map[string]bool{}
		for _, method := range base.Methods {
			if method.Constructor {
				names[executionConstructorImplementationName(symbol.GoIdentifier(method.Name), base)] = true
				names[executionConstructorWithSelfImplementationName(symbol.GoIdentifier(method.Name), base)] = true
			}
		}
		constructors := 0
		ast.Inspect(expression, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok && names[volatileTDDCallName(call)] {
				constructors++
			}
			return true
		})
		if constructors != 1 {
			t.Fatalf("one field evaluation must stage exactly one base construction, got %d", constructors)
		}
	}
	t.Run("same_context", func(t *testing.T) {
		if got, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, prepared.Clone()); !ok || got != original {
			t.Fatal("same scoped receiver was unnecessarily reconstructed")
		}
	})
	t.Run("intrinsic_generator_local_context", func(t *testing.T) {
		withIntrinsic := prepared
		withIntrinsic.intrinsicTypeArgs = []ast.Expr{ast.NewIdent("int32")}
		if got, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, withIntrinsic); !ok || got != original {
			t.Fatal("generator-local intrinsic type arguments invalidated an unchanged receiver")
		}
		previousCounter, previousDeclarations := counter, len(declarations)
		expression := ParseExpr(field, helper.File.Source, withIntrinsic.Clone())
		assertSingleConstruction(t, expression)
		again := prepareAnonymousFieldReceiver(field, helper.File.Source, withIntrinsic.Clone())
		cached, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, again)
		if !ok || cached != original || counter != previousCounter || len(declarations) != previousDeclarations {
			t.Fatal("generator-local intrinsic type arguments reconstructed an unchanged receiver")
		}
	})
	t.Run("post_parse_memo", func(t *testing.T) {
		previousCounter, previousDeclarations := counter, len(declarations)
		expression := ParseExpr(field, helper.File.Source, prepared)
		again := prepareAnonymousFieldReceiver(field, helper.File.Source, prepared.Clone())
		cached, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, again)
		if !ok || cached != original || counter != previousCounter || len(declarations) != previousDeclarations {
			t.Fatal("post-parse memo reconstructed/hoisted receiver after mutable collector advancement")
		}
		assertSingleConstruction(t, expression)
	})
	t.Run("changed_execution", func(t *testing.T) {
		changed := prepared.Clone()
		changed.executionContextName = "executionB"
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("cached constructor leaks executionA")
		}
		actual := ParseExpr(receiver, helper.File.Source, changed)
		if !anonymousReceiverContainsIdentifier(actual, "executionB") || anonymousReceiverContainsIdentifier(actual, "executionA") {
			t.Fatal("re-emitted constructor did not propagate executionB")
		}
	})
	t.Run("physical_abi", func(t *testing.T) {
		changed := prepared.Clone()
		changed.rawGenericParameterTypes = map[string]string{"T": "Object"}
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("cached physical ABI survived a changed type binding")
		}
	})
	t.Run("mutable_local_binding", func(t *testing.T) {
		parameter := prepared.localScope.Parameters[0]
		previous := parameter.Name
		parameter.Name = "seedB"
		defer func() { parameter.Name = previous }()
		changed := prepared.Clone()
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("cached captured local ignored renamed source binding")
		}
		actual := ParseExpr(receiver, helper.File.Source, changed)
		if !anonymousReceiverContainsIdentifier(actual, "seedB") || anonymousReceiverContainsIdentifier(actual, "seed") {
			t.Fatal("re-emitted constructor leaked previous captured local")
		}
	})
	t.Run("source_file", func(t *testing.T) {
		changed := prepared.Clone()
		changed.currentFile = &symbol.FileScope{}
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("same source offsets from another file reused cached AST")
		}
	})
	t.Run("source_bytes", func(t *testing.T) {
		changed := append([]byte(nil), helper.File.Source...)
		changed = append(changed, '\n')
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, changed, prepared.Clone()); ok {
			t.Fatal("different source bytes reused cached AST")
		}
	})
	t.Run("emission", func(t *testing.T) {
		changed := prepared.Clone()
		var fresh []ast.Decl
		changed.hoistedDecls = &fresh
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("another declaration emission reused cached AST")
		}
	})
	t.Run("local_scope", func(t *testing.T) {
		changed := prepared.Clone()
		copy := *prepared.localScope
		changed.localScope = &copy
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("another lexical scope reused cached AST")
		}
	})
	t.Run("current_class", func(t *testing.T) {
		changed := prepared.Clone()
		copy := *prepared.currentClass
		changed.currentClass = &copy
		if _, ok := preparedAnonymousFieldReceiverExpr(receiver, helper.File.Source, changed); ok {
			t.Fatal("another declaring class reused cached AST")
		}
	})
}

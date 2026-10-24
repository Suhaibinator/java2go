package transpiler

import "testing"

// Demand must follow actual field declarations, including syntax not hoisted
// into the initial symbol graph. Non-declaration spellings supply no authority.
func TestReflectionMetadataDemand_VolatileSyntaxAndFreshGraphs(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"ordinary", `class Owner{int value;}`, false},
		{"string", `class Owner{String value="volatile";}`, false},
		{"comment", `class Owner{/* volatile */ int value;}`, false},
		{"identifier", `class Owner{int volatileValue;}`, false},
		{"named-field", `class Owner{volatile int value;}`, true},
		{"local-field", `class Owner{void make(){class Local{volatile int value;}}}`, true},
		{"anonymous-field", `class Owner{Object make(){return new Object(){volatile int value;};}}`, true},
		{"reflection-call", `class Owner{Class<?> type(){return getClass();}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, tc.source)
			scope := helper.File.Symbols.FindClassScope("Owner")
			if scope == nil {
				t.Fatal("source declaration missing")
			}
			if got := sourceUsesReflection(); got != tc.want {
				t.Fatalf("metadata demand=%v want=%v", got, tc.want)
			}
			if got := sourceClassMetadataStmt(scope, helper.Ctx) != nil; got != tc.want {
				t.Fatalf("descriptor emitted=%v want=%v", got, tc.want)
			}
		})
	}
	t.Run("escaped-modifier-preserves-scanner", func(t *testing.T) {
		helper := setupParseHelper(t, `class Owner{vo\u006catile int value;}`)
		scope := helper.File.Symbols.FindClassScope("Owner")
		if scope == nil {
			t.Fatal("escaped source owner missing")
		}
		// This guard repair does not expand parser Unicode-keyword support. Any
		// declaration recognized by the existing AST scanner must retain demand.
		want := sourceNodeDeclaresVolatileField(scope.Class.DeclarationNode)
		if got := sourceUsesReflection(); got != want {
			t.Fatalf("escaped source changed existing scanner demand: got=%v want=%v", got, want)
		}
		if got := sourceClassMetadataStmt(scope, helper.Ctx) != nil; got != want {
			t.Fatalf("escaped descriptor=%v want=%v", got, want)
		}
	})
	t.Run("fresh-graph", func(t *testing.T) {
		for _, tc := range []struct {
			source string
			want   bool
		}{
			{`class Owner{volatile int value;}`, true},
			{`class Owner{int value;}`, false},
			{`class Owner{Object make(){return AtomicIntegerFieldUpdater.newUpdater(Owner.class,"value");}int value;}`, true},
		} {
			helper := setupParseHelper(t, tc.source)
			scope := helper.File.Symbols.FindClassScope("Owner")
			if scope == nil || sourceUsesReflection() != tc.want || (sourceClassMetadataStmt(scope, helper.Ctx) != nil) != tc.want {
				t.Fatal("fresh graph retained previous metadata demand or lost current source syntax")
			}
		}
	})
}

// Symbol scopes assembled manually or synthetically may retain declaration
// syntax without source bytes. Missing text must not erase a proven modifier.
func TestReflectionMetadataDemand_MissingSourceRetainsVolatileAST(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source []byte
	}{
		{"nil-source", nil}, {"empty-source", []byte{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			helper := setupParseHelper(t, `class Owner{volatile int value;}`)
			scope := helper.File.Symbols.FindClassScope("Owner")
			if scope == nil || !sourceNodeDeclaresVolatileField(scope.Class.DeclarationNode) {
				t.Fatal("baseline volatile declaration AST absent")
			}
			file := findFileScopeForClassScope(scope)
			if file == nil {
				t.Fatal("existing source scope required")
			}
			originalSource := file.Source
			file.Source = tc.source
			if !sourceUsesReflection() {
				t.Error("VOLATILE_METADATA_MISSING_SOURCE_EXCLUDED: AST volatile field lost descriptor demand")
			}
			file.Source = originalSource
			if sourceClassMetadataStmt(scope, helper.Ctx) == nil {
				t.Error("VOLATILE_METADATA_MISSING_SOURCE_EXCLUDED: AST volatile field lost descriptor emission")
			}
		})
	}
}

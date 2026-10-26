package transpiler

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

const importASTMalformedDiagnostic = "unsupported Java import declaration: incomplete or invalid syntax"

// Invoke the existing production API. Controlled recovery turns an exception
// into an explicit failed contract assertion; it never supplies parser facts.
func importASTExistingSymbols(file *parsing.SourceFile) (scope *symbol.FileScope, exception any) {
	defer func() { exception = recover() }()
	scope = file.ParseSymbols()
	return scope, nil
}

func importASTFreshSymbols(t *testing.T) {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
}

func TestImportASTValidDeclarationsExistingAPIs(t *testing.T) {
	for _, row := range []struct {
		name     string
		explicit bool
	}{
		{"ordinary_control", true},
		{"leading_block", true},
		{"leading_line", true},
		{"before_first_dot", true},
		{"after_first_dot", true},
		{"before_terminal_name", true},
		{"multiple_named_extras", true},
		{"wildcard_kind_control", false},
		{"static_kind_control", false},
		{"static_wildcard_kind_control", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			importASTFreshSymbols(t)
			data, err := os.ReadFile(filepath.Join("testdata", "import_ast_comments", row.name, "ImportSyntaxProbe.java"))
			if err != nil {
				t.Fatal("IMPORT_AST_FIXTURE_PREMISE", err)
			}
			file := parsing.SourceFile{Name: "ImportSyntaxProbe.java", Source: data}
			if err := file.ParseAST(); err != nil || file.Ast == nil || file.Ast.HasError() {
				t.Fatal("IMPORT_AST_VALID_SYNTAX_PREMISE", err)
			}
			scope, exception := importASTExistingSymbols(&file)
			if exception != nil {
				t.Errorf("IMPORT_AST_VALID_EXCEPTION case=%s type=%T text=%q", row.name, exception, fmt.Sprint(exception))
				return
			}
			if scope == nil {
				t.Fatal("IMPORT_AST_SYMBOL_SCOPE_PREMISE")
			}
			want := map[string]string{}
			if row.explicit {
				want["Map"] = "java.util"
			}
			if !maps.Equal(scope.Imports, want) {
				t.Errorf("IMPORT_AST_VALID_MAP_MISMATCH case=%s actual=%q expected=%q", row.name, fmt.Sprint(scope.Imports), fmt.Sprint(want))
			}
			symbol.AddSymbolsToPackage(scope)
			if !sourceArrayListConstructorFileNamespace(scope) {
				t.Errorf("IMPORT_AST_VALID_NAMESPACE_REJECTED case=%s", row.name)
			}
		})
	}
}

func TestImportASTMalformedDeclarationsExistingAPIContract(t *testing.T) {
	for _, row := range []struct{ name, source string }{
		{"ordinary_without_owner", `import Map;class Invalid{}`},
		{"static_without_member_owner", `import static Map;class Invalid{}`},
		{"static_commented_without_member_owner", `import static /* malformed */ Map;class Invalid{}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			importASTFreshSymbols(t)
			file := parsing.SourceFile{Name: "Invalid.java", Source: []byte(row.source)}
			if err := file.ParseAST(); err != nil || file.Ast == nil {
				t.Fatal("IMPORT_AST_MALFORMED_PARSE_PREMISE", err)
			}
			declaration := findNode(file.Ast, "import_declaration")
			if declaration == nil {
				t.Fatal("IMPORT_AST_MALFORMED_DECLARATION_PREMISE")
			}
			_, exception := importASTExistingSymbols(&file)
			if exception == nil {
				t.Errorf("IMPORT_AST_MALFORMED_NO_REJECTION case=%s", row.name)
				return
			}
			if actual := fmt.Sprint(exception); actual != importASTMalformedDiagnostic {
				t.Errorf("IMPORT_AST_MALFORMED_WRONG_DIAGNOSTIC case=%s type=%T text=%q", row.name, exception, actual)
			}
		})
	}
}

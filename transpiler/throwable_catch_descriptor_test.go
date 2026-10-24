package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestThrowableCatchDescriptorsRespectOwnersAndBinders(t *testing.T) {
	old := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = old })
	parse := func(name, source string) parsing.SourceFile {
		t.Helper()
		file := parsing.SourceFile{Name: name, Source: []byte(source)}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		return file
	}
	parse("Shadow.java", `package shadows; class IllegalCharsetNameException extends RuntimeException {} class Throwable extends RuntimeException {}`)
	file := parse("Use.java", `package app; import shadows.IllegalCharsetNameException; import shadows.Throwable; class Use {}`)
	ctx := Ctx{currentFile: file.Symbols, currentClass: file.Symbols.BaseClass}
	for _, entry := range []struct {
		input, want string
		all         bool
	}{
		{"IllegalCharsetNameException", "shadows.IllegalCharsetNameException", false}, {"Throwable", "shadows.Throwable", false},
		{"java.nio.charset.IllegalCharsetNameException", "java.nio.charset.IllegalCharsetNameException", false}, {"java.nio.charset.UnsupportedCharsetException", "java.nio.charset.UnsupportedCharsetException", false},
		{"java.lang.Throwable", "java.lang.Throwable", true}, {"java.lang.Object", "java.lang.Object", true},
		{"foreign.UnsupportedCharsetException", "foreign.UnsupportedCharsetException", false}, {"java.lang.UnsupportedCharsetException", "java.lang.UnsupportedCharsetException", false},
	} {
		name, nominal, all := throwableCatchDescriptor(entry.input, ctx)
		if name != entry.want || !nominal || all != entry.all {
			t.Errorf("%s: %s nominal=%v all=%v", entry.input, name, nominal, all)
		}
	}
	for _, family := range []string{"IllegalCharsetNameException", "UnsupportedCharsetException", "CharacterCodingException", "UnmappableCharacterException"} {
		imported := Ctx{currentFile: &symbol.FileScope{Imports: map[string]string{family: "java.nio.charset"}}}
		name, nominal, all := throwableCatchDescriptor(family, imported)
		if name != "java.nio.charset."+family || !nominal || all {
			t.Errorf("imported family %s misclassified", family)
		}
		var out bytes.Buffer
		if err := format.Node(&out, token.NewFileSet(), catchConditionExpr([]string{family}, "recovered", imported)); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), `CaughtAsType(recovered, "java.nio.charset.`+family+`")`) {
			t.Errorf("nominal guard absent: %s", out.String())
		}
		imported.currentFile.Imports[family] = "foreign"
		name, nominal, all = throwableCatchDescriptor(family, imported)
		if name != "foreign."+family || !nominal || all {
			t.Errorf("foreign import %s borrowed runtime identity", family)
		}
		if isBuiltinExceptionType(family) {
			t.Errorf("catch metadata enabled nonexistent constructor: %s", family)
		}
	}
	for _, name := range []string{"Throwable", "UnsupportedCharsetException"} {
		bound := Ctx{currentClass: &symbol.ClassScope{TypeParameters: []symbol.TypeParam{{Name: name}}}}
		got, nominal, all := throwableCatchDescriptor(name, bound)
		if got != name || nominal || all {
			t.Errorf("binder %s interpreted as JDK descriptor", name)
		}
	}
	local := parse("Local.java", `class IllegalCharsetNameException extends RuntimeException {} class Local {}`)
	localCtx := Ctx{currentFile: local.Symbols, currentClass: local.Symbols.BaseClass}
	if name, nominal, all := throwableCatchDescriptor("IllegalCharsetNameException", localCtx); name != "IllegalCharsetNameException" || !nominal || all {
		t.Fatal("default-package source exception lost identity")
	}
}

func TestCampaignCodecCatchFamilyOrderJVMParity(t *testing.T) {
	codecNamedReferenceOracle(t, "CodecCatchFamilyOrder", `import java.nio.charset.Charset;import java.nio.charset.UnsupportedCharsetException;import java.io.IOException;
class IllegalCharsetNameException extends RuntimeException {}
public class CodecCatchFamilyOrder {
 static String lookup(String name){try{Charset.forName(name);return "ok";}catch(IllegalCharsetNameException e){return "source";}catch(java.nio.charset.IllegalCharsetNameException e){return "illegal";}catch(UnsupportedCharsetException e){return "unsupported";}catch(IllegalArgumentException e){return "parent";}catch(Exception e){return "exception";}}
 static String parent(String name){try{Charset.forName(name);return "ok";}catch(IllegalArgumentException e){return "parent";}catch(Exception e){return "exception";}}
 static String checked(String name){try{"A".getBytes(name);return "ok";}catch(IllegalArgumentException e){return "unchecked";}catch(IOException e){return "checked";}}
 public static String run(){StringBuilder out=new StringBuilder();String[] names={null,"!bad","UTF_8","UTF8"};for(String name:names)out.append(lookup(name)).append('/').append(parent(name)).append('|');out.append(checked("!bad")).append('/').append(checked("UTF_8"));try{throw new IllegalCharsetNameException();}catch(java.nio.charset.IllegalCharsetNameException e){out.append("/wrong");}catch(IllegalCharsetNameException e){out.append("/source");}return out.toString();}
}`)
}

// Require the actual catch-condition call, including its recovered receiver and
// canonical nominal target; an unused helper mention cannot satisfy this guard.
func requireNominalThrowableCatchGuard(t *testing.T, generated, expected string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "generated.go", generated, 0)
	if err != nil {
		t.Fatal(err)
	}
	matched := 0
	ast.Inspect(file, func(node ast.Node) bool {
		branch, ok := node.(*ast.IfStmt)
		if !ok {
			return true
		}
		ast.Inspect(branch.Cond, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			fun, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || fun.Sel.Name != "CaughtAsType" {
				return true
			}
			owner, ok := fun.X.(*ast.Ident)
			if !ok || owner.Name != "stdjava" {
				return true
			}
			recovered, ok := call.Args[0].(*ast.Ident)
			if !ok || !strings.HasPrefix(recovered.Name, "__java2goRecovered_") {
				t.Fatal("nominal catch uses a different panic value")
			}
			literal, ok := call.Args[1].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			target, err := strconv.Unquote(literal.Value)
			if err != nil {
				t.Fatal(err)
			}
			if target == expected {
				matched++
			}
			return true
		})
		return true
	})
	if matched != 1 {
		t.Fatalf("expected one condition calling CaughtAsType on recovered panic with nominal target %q, got %d\n%s", expected, matched, generated)
	}
}

func TestThrowableCatchOriginalASTGuardMigration(t *testing.T) {
	// Read the unchanged original Java bodies rather than maintaining a second
	// version of those fixtures. Their behavior assertions remain in the original
	// tests; this control verifies only the two approved emission expectations.
	source, err := os.ReadFile("exceptions_integration_test.go")
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "exceptions_integration_test.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"TestExceptions_CatchBySupertype": "java.lang.RuntimeException", "TestExceptions_ErrorNotCaughtByExceptionClause": "java.lang.Exception"}
	found := 0
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		target, selected := expected[fn.Name.Name]
		if !selected {
			continue
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			assignment, ok := node.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				return true
			}
			name, ok := assignment.Lhs[0].(*ast.Ident)
			if !ok || name.Name != "src" {
				return true
			}
			value, ok := assignment.Rhs[0].(*ast.BasicLit)
			if !ok || value.Kind != token.STRING {
				t.Fatal("original fixture is not a literal")
			}
			java, err := strconv.Unquote(value.Value)
			if err != nil {
				t.Fatal(err)
			}
			requireNominalThrowableCatchGuard(t, renderGoFileFromJava(t, java), target)
			found++
			return false
		})
	}
	if found != 2 {
		t.Fatalf("expected both original AST fixtures, got %d", found)
	}
}

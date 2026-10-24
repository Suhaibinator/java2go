package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestObserver128ClassificationsRespectDeclarations(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	parse := func(name, text string) parsing.SourceFile {
		f := parsing.SourceFile{Name: name, Source: []byte(text)}
		if err := f.ParseAST(); err != nil {
			t.Fatal(err)
		}
		f.ParseSymbols()
		symbol.AddSymbolsToPackage(f.Symbols)
		return f
	}
	parse("Shadow.java", `package shadows;class GeneralSecurityException{}class URISyntaxException{}class Exception{}class ClassNotFoundException{}class OrdinaryChild extends GeneralSecurityException{}class TrueChild extends java.security.GeneralSecurityException{}`)
	f := parse("Use.java", `package app;import shadows.GeneralSecurityException;import shadows.URISyntaxException;import shadows.Exception;import shadows.ClassNotFoundException;class Use{}`)
	ctx := Ctx{currentFile: f.Symbols, currentClass: f.Symbols.BaseClass}
	for _, name := range []string{"GeneralSecurityException", "URISyntaxException", "Exception", "ClassNotFoundException", "shadows.OrdinaryChild", "foreign.GeneralSecurityException", "foreign.URISyntaxException", "java.security.GeneralSecurityException[]", "java.net.URISyntaxException[]"} {
		if isExceptionJavaType(ctx, name) {
			t.Errorf("nonThrowable owner/array admitted: %s", name)
		}
	}
	for _, name := range []string{"java.security.GeneralSecurityException", "java.security.NoSuchAlgorithmException", "java.net.URISyntaxException", "java.util.concurrent.CancellationException", "java.lang.ClassNotFoundException", "shadows.TrueChild"} {
		if !isExceptionJavaType(ctx, name) {
			t.Errorf("canonical/source Throwable rejected: %s", name)
		}
	}
	for _, name := range []string{"GeneralSecurityException", "URISyntaxException", "Throwable", "ClassNotFoundException"} {
		for _, method := range []bool{false, true} {
			parameter := symbol.TypeParam{Name: name}
			bound := ctx.Clone()
			bound.currentClass = &symbol.ClassScope{TypeParameters: []symbol.TypeParam{parameter}}
			if method {
				bound.localScope = &symbol.Definition{IsStatic: true, TypeParameters: []symbol.TypeParam{parameter}}
			}
			if isExceptionJavaType(bound, name) {
				t.Errorf("binder spelling admitted: %s", name)
			}
		}
	}
	if scope := resolveClassScopeByQualifiedName(ctx, "shadows.OrdinaryChild"); exceptionSuperclassName(ctx, scope) != "" {
		t.Fatal("ordinary child acquired Throwable registration")
	}
	if scope := resolveClassScopeByQualifiedName(ctx, "shadows.TrueChild"); exceptionSuperclassName(ctx, scope) != "GeneralSecurityException" {
		t.Fatal("canonical child lost exact immediate parent registration")
	}
}
func TestObserver128CanonicalSuperPrintUsesCurrentReceiver(t *testing.T) {
	out := renderGoFileFromJava(t, `class Current extends java.security.GeneralSecurityException {void printBody(){super.printStackTrace();}}`)
	f, err := parser.ParseFile(token.NewFileSet(), "generated.go", out, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, decl := range f.Decls {
		method, ok := decl.(*ast.FuncDecl)
		if !ok || method.Recv == nil || !strings.HasPrefix(method.Name.Name, "printBody") {
			continue
		}
		receiver := method.Recv.List[0].Names[0].Name
		ast.Inspect(method.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "PrintStackTrace" {
				return true
			}
			if len(call.Args) != 1 {
				t.Fatal("wrong canonical body arity")
			}
			id, ok := call.Args[0].(*ast.Ident)
			if !ok || id.Name != receiver {
				t.Fatal("canonical super body projected receiver")
			}
			found = true
			return true
		})
	}
	if !found {
		t.Fatalf("canonical super body/current self not selected: %s", out)
	}
}

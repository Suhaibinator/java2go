package transpiler

import (
	"go/ast"
	"testing"
)

func TestBoundedSourceClassGoInstallerAncestorArguments(t *testing.T) {
	for _, test := range []struct{ name, parent string }{
		{"raw_multiedge", "p.Middle"},
		{"concrete_multiedge", "p.Middle<java.util.List<java.lang.String>>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, map[string]string{
				"p/Base.java": `package p;public class Base<T extends java.util.List<String>>{public T value;public int kind(){return 1;}}`,
				"p/Middle.java": `package p;public class Middle<U extends java.util.List<String>> extends Base<U>{}`,
				"app/Use.java": `package app;public class Use extends ` + test.parent + `{public int kind(){return 2;}}`,
			})
			ctx = classScopeCtx(findQualifiedSourceClass("app.Use"), ctx)
			paths := classSubobjectAncestorPaths(ctx.currentClass, ctx)
			if len(paths) != 2 { t.Fatalf("ancestor path count = %d, want 2", len(paths)) }
			var installer *ast.FuncDecl
			for _, declaration := range generateClassSubobjectInstallerDecls(ctx) {
				function := declaration.(*ast.FuncDecl)
				if function.Name.Name == classSubobjectInstallerName(findQualifiedSourceClass("p.Base")) { installer = function }
			}
			if installer == nil { t.Fatal("missing ancestor installer for overridden source method") }
			got := boundedArgumentTestGoType(t, installer.Type.Params.List[0].Type)
			want := "*p.Base[*stdjava.List[*stdjava.JavaString]]"
			if got != want { t.Errorf("Go installer ancestor argument representation = %s, want %s", got, want) }
		})
	}
}

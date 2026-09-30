package transpiler

import (
	"go/ast"
	"strings"
)

// Java resolves a bare type variable by its lexical declaration before source
// classes or JDK names. Preserve that identity before class lookup can replace
// the written name with an allocated Go class name. Qualified names still name
// actual classes; arrays retain the caller's existing representation choice.
func lexicalTypeParameterTypeExpr(base string, arguments, parameters []string, ctx Ctx) (ast.Expr, bool) {
	if strings.Contains(base, ".") || len(arguments) != 0 {
		return nil, false
	}
	if name, visible := visibleTypeParameterGoName(base, ctx); visible {
		return ast.NewIdent(name), true
	}
	// Some generated declaration contexts carry only emitted parameter names.
	for _, name := range parameters {
		if name == base {
			return ast.NewIdent(name), true
		}
	}
	return nil, false
}

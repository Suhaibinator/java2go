package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// Enclosing records lexical declarations for static members, inner classes,
// and synthesized local/anonymous classes. IsInner describes the instance ABI
// and does not determine Java nest membership.
func sourceReflectionNestHostTypeID(scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	host := scope
	for host != nil && host.Enclosing != nil {
		host = host.Enclosing
	}
	return javaTypeIDLiteral(sourceClassRuntimeTypeID(host, ctx), ctx)
}

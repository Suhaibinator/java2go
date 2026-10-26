package transpiler

import (
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// Reuse the ordinary source-class coordinator and member walker. A record's
// active constructor entry runs its initialization with the caller execution;
// a constructor reference alone remains passive. Coordinated static blocks
// must appear only in the coordinator, never also in an eager Go init function.
// Reflection's separate unprovided-static-state refusal remains upstream.
func buildRecordConstructorBodyDecls(body *sitter.Node, source []byte, ctx Ctx) []ast.Decl {
	declarations := buildLazyClassInitializationDecls(body, source, ctx)
	coordinated := classBodyNeedsOrderedStaticInitialization(body, ctx.currentClass, source)
	return append(declarations, parseClassBodyDeclarations(body, source, ctx, coordinated)...)
}

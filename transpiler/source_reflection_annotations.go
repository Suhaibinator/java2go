package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Methods and constructors without a written annotation retain their existing
// metadata path. Annotation factories still come from the owned declaration
// planner, including its runtime-retention and nominal source checks.
func sourceReflectionWrittenAnnotationDescriptors(node *sitter.Node, scope *symbol.ClassScope, ctx Ctx) ast.Expr {
	if node == nil {
		return nil
	}
	for _, modifiers := range nodeutil.NamedChildrenOf(node) {
		if modifiers.Type() != "modifiers" {
			continue
		}
		for _, annotation := range nodeutil.NamedChildrenOf(modifiers) {
			if annotation.Type() != "annotation" && annotation.Type() != "marker_annotation" {
				continue
			}
			values := sourceAnnotationDescriptors(node, scope, ctx)
			if literal, ok := values.(*ast.CompositeLit); ok && len(literal.Elts) != 0 {
				return values
			}
			return nil
		}
	}
	return nil
}

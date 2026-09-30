package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// A standalone switch whose expression arms all have the same type has that
// type. This deliberately leaves numeric promotion, mixed reference bounds and
// block-local yield inference to their own rules; it never guesses one arm's
// type for a heterogeneous switch.
func inferUniformSwitchResultJavaType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return "", false
	}
	result := ""
	for _, rule := range nodeutil.NamedChildrenOf(body) {
		if rule.Type() == "line_comment" || rule.Type() == "block_comment" {
			continue
		}
		if rule.Type() != "switch_rule" {
			return "", false
		}
		for _, arm := range nodeutil.NamedChildrenOf(rule) {
			switch arm.Type() {
			case "switch_label", "line_comment", "block_comment", "throw_statement":
				continue
			case "expression_statement":
				typ, known := inferExprJavaType(arm.NamedChild(0), ctx, source)
				if !known || (result != "" && !javaInferenceSameType(result, typ, ctx)) {
					return "", false
				}
				result = typ
			default:
				return "", false
			}
		}
	}
	return result, result != ""
}

// Each result arm occupies its own target-typed context, including a nested
// poly switch. The outer switch node is not the inner expression's target root.
func parseSwitchArmReturnValue(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	armCtx := ctx.Clone()
	armCtx.expectedTypeRoot = node
	return parseReturnValue(node, source, armCtx)
}

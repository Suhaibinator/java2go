package transpiler

import (
	"fmt"
	"go/ast"

	sitter "github.com/smacker/go-tree-sitter"
)

// lowerLabeledStatement preserves Java's broader labeled-break semantics. Go
// accepts `break label` only when label names a for, switch, or select, while
// Java permits a break to any labeled statement. Loop and switch labels retain
// their direct Go form. Other statements are placed in a lexical block with a
// synthetic end label, and matching labeled breaks become gotos to that end.
//
// An end-label lowering is important here: wrapping the statement in a
// single-iteration for loop would make an otherwise-unlabeled break or continue
// bind to the synthetic loop instead of the enclosing Java loop or switch.
func lowerLabeledStatement(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	if node == nil || node.NamedChildCount() < 2 {
		return &ast.BadStmt{}
	}

	bodyNode := node.NamedChild(1)
	label := &ast.Ident{Name: javaSourceLabelName(node)}

	// Java permits an unreferenced label, while Go rejects one. Share the target
	// marker with nested lowering so direct branches and try/finally replay both
	// report whether this source label is actually needed. This also preserves the
	// do-while lowering's special case: a labeled continue rewritten to the
	// condition-phase goto no longer needs the source label emitted in Go.
	labelCtx := ctx.Clone()
	labelCtx.javaLabelTargets = cloneJavaLabelTargets(ctx.javaLabelTargets)
	labelTarget := &javaLabelTarget{}
	if !javaStatementSupportsGoBreakLabel(bodyNode) {
		labelTarget.BreakLabel = fmt.Sprintf("__java2goLabelEnd_%d", node.StartByte())
	}
	if key, ok := javaControlKey(node); ok {
		labelCtx.javaLabelTargets[key] = labelTarget
	}
	body := parseLabeledStatementBody(bodyNode, source, labelCtx)
	if !labelTarget.NeedsGoLabel {
		return body
	}
	if javaStatementSupportsGoBreakLabel(bodyNode) {
		return &ast.LabeledStmt{Label: label, Stmt: body}
	}

	return &ast.BlockStmt{List: []ast.Stmt{
		body,
		&ast.LabeledStmt{
			Label: &ast.Ident{Name: labelTarget.BreakLabel},
			// A label immediately before the closing brace denotes Go's implicit
			// empty statement, keeping the synthetic target side-effect free.
			Stmt: &ast.EmptyStmt{Implicit: true},
		},
	}}
}

func parseLabeledStatementBody(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	if node == nil {
		return &ast.BadStmt{}
	}
	// Try lowering already has an abrupt-control channel that can replay a break
	// outside its generated func literal. Synchronized lowering also returns a
	// statement list, but does not yet have that channel; keep its existing
	// unsupported diagnostic instead of turning a labeled synchronized break into
	// an invalid cross-function goto.
	switch node.Type() {
	case "try_statement", "try_with_resources_statement":
		if statements, ok := ParseNode(node, source, ctx).([]ast.Stmt); ok {
			return &ast.BlockStmt{List: statements}
		}
	}
	return ParseStmt(node, source, ctx)
}

func javaStatementSupportsGoBreakLabel(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "for_statement", "enhanced_for_statement", "while_statement", "do_statement", "switch_statement", "switch_expression":
		return true
	default:
		return false
	}
}

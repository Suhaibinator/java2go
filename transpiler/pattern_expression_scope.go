package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Pattern variables belong to the matching control-flow branch. Keep local
// lookup metadata private as well as placing the generated binding in that
// branch, so another expression can reuse the same Java spelling.
func patternExpressionContext(ctx Ctx) Ctx {
	result := ctx.Clone()
	if ctx.localScope != nil {
		method := *ctx.localScope
		method.Children = make([]*symbol.Definition, len(ctx.localScope.Children))
		for index, child := range ctx.localScope.Children {
			if child != nil {
				copy := *child
				method.Children[index] = &copy
			}
		}
		result.localScope = &method
	}
	return result
}

func patternConditionHasBindings(node *sitter.Node, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	switch node.Type() {
	case "instanceof_expression":
		return node.ChildByFieldName("name") != nil
	case "unary_expression":
		return node.Child(0).Content(source) == "!" && patternConditionHasBindings(node.NamedChild(0), source)
	case "binary_expression":
		operator := node.ChildByFieldName("operator").Content(source)
		return (operator == "&&" || operator == "||") && (patternConditionHasBindings(node.ChildByFieldName("left"), source) || patternConditionHasBindings(node.ChildByFieldName("right"), source))
	}
	return false
}

// Type inference needs each branch's declared pattern type before inspecting
// its result. Valid Java can only refer to the variable on its matched path;
// this private context is used for type inference, never runtime evaluation.
func patternConditionInferenceContext(node *sitter.Node, source []byte, ctx Ctx) Ctx {
	if !patternConditionHasBindings(node, source) {
		return ctx.Clone()
	}
	result := patternExpressionContext(ctx)
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		node = unwrapParenthesizedExpressionNode(node)
		if node == nil {
			return
		}
		if pattern := instanceofPatternNode(node); pattern != nil {
			typ := pattern.ChildByFieldName("right").Content(source)
			physical := instanceofAssertTypeExpr(typ, result)
			if physical != nil {
				recordLocalVariableDefinition(result, pattern.ChildByFieldName("name").Content(source), typ, symbol.NodeToStr(physical))
			}
			return
		}
		if node.Type() == "binary_expression" {
			operator := node.ChildByFieldName("operator").Content(source)
			if operator == "&&" || operator == "||" {
				visit(node.ChildByFieldName("left"))
				visit(node.ChildByFieldName("right"))
			}
		} else if node.Type() == "unary_expression" && node.Child(0).Content(source) == "!" {
			visit(node.NamedChild(0))
		}
	}
	visit(node)
	return result
}

// Continuations render code in the scope guaranteed by the selected path.
// Nesting conditions retains Java's left-to-right, short-circuit evaluation
// while allowing later operands and selected results to use earlier bindings.
func lowerPatternExpressionCondition(node *sitter.Node, source []byte, ctx Ctx, matched, unmatched func(Ctx) ast.Stmt) ast.Stmt {
	node = unwrapParenthesizedExpressionNode(node)
	if node.Type() == "unary_expression" && node.Child(0).Content(source) == "!" {
		return lowerPatternExpressionCondition(node.NamedChild(0), source, ctx, unmatched, matched)
	}
	if node.Type() == "binary_expression" {
		left, right := node.ChildByFieldName("left"), node.ChildByFieldName("right")
		switch node.ChildByFieldName("operator").Content(source) {
		case "&&":
			return lowerPatternExpressionCondition(left, source, ctx, func(leftCtx Ctx) ast.Stmt {
				return lowerPatternExpressionCondition(right, source, leftCtx, matched, unmatched)
			}, unmatched)
		case "||":
			return lowerPatternExpressionCondition(left, source, ctx, matched, func(leftCtx Ctx) ast.Stmt {
				return lowerPatternExpressionCondition(right, source, leftCtx, matched, unmatched)
			})
		}
	}
	if pattern := instanceofPatternNode(node); pattern != nil {
		branchCtx := patternExpressionContext(ctx)
		init, _, branchCtx := lowerInstanceofPattern(pattern, source, branchCtx)
		assignment := init.(*ast.AssignStmt)
		binding := localBindingIdent(pattern.ChildByFieldName("name"), source, branchCtx)
		success := "__java2goPatternMatched"
		for ctx.currentFile != nil && strings.Contains(string(ctx.currentFile.Source), success) {
			success += "_"
		}
		assignment.Lhs = []ast.Expr{binding, ast.NewIdent(success)}
		condition := ast.NewIdent(success)
		body := patternExpressionBlock(matched(branchCtx))
		body.List = append([]ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{binding}}}, body.List...)
		return &ast.IfStmt{Init: init, Cond: condition, Body: body, Else: patternExpressionBlock(unmatched(ctx))}
	}
	return &ast.IfStmt{Cond: parseJavaBooleanExpr(node, source, ctx), Body: patternExpressionBlock(matched(ctx)), Else: patternExpressionBlock(unmatched(ctx))}
}

func patternExpressionBlock(statement ast.Stmt) *ast.BlockStmt {
	if block, ok := statement.(*ast.BlockStmt); ok {
		return block
	}
	if statement == nil {
		return &ast.BlockStmt{}
	}
	return &ast.BlockStmt{List: []ast.Stmt{statement}}
}

func buildPatternBooleanExpression(node *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	constant := func(value string) func(Ctx) ast.Stmt {
		return func(Ctx) ast.Stmt { return &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent(value)}} }
	}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}, Body: patternExpressionBlock(lowerPatternExpressionCondition(node, source, ctx, constant("true"), constant("false")))}}
}

func buildPatternTernaryExpression(node *sitter.Node, source []byte, ctx Ctx, resultJavaType string, resultGoType ast.Expr) ast.Expr {
	condition, consequence, alternative := ternaryExpressionParts(node)
	branch := func(node *sitter.Node) func(Ctx) ast.Stmt {
		return func(branchCtx Ctx) ast.Stmt {
			return &ast.ReturnStmt{Results: []ast.Expr{parseTernaryBranch(node, resultJavaType, source, branchCtx)}}
		}
	}
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: resultGoType}}}}, Body: patternExpressionBlock(lowerPatternExpressionCondition(condition, source, ctx, branch(consequence), branch(alternative)))}}
}

package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/astutil"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	log "github.com/sirupsen/logrus"
	sitter "github.com/smacker/go-tree-sitter"
)

// isStmtListNode reports whether a node type is one that ParseNode lowers into a
// list of statements (rather than a single statement). These are the constructs
// that must be expanded inline when filling a block body.
func isStmtListNode(nodeType string) bool {
	switch nodeType {
	case "try_statement", "try_with_resources_statement", "synchronized_statement":
		return true
	}
	return false
}

func ParseStmt(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	if stmt := TryParseStmt(node, source, ctx); stmt != nil {
		return stmt
	}

	diag := reportUnsupported("statement", node, source, ctx)
	// Emit a placeholder statement that still compiles, so the rest of the file
	// can be converted. The panic call preserves the diagnostic at runtime.
	return &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.Ident{Name: "panic"},
			Args: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf("%q", strings.TrimPrefix(unsupportedComment(diag), "// "))},
			},
		},
	}
}

func TryParseStmt(node *sitter.Node, source []byte, ctx Ctx) ast.Stmt {
	switch node.Type() {
	case "ERROR":
		log.WithFields(log.Fields{
			"parsed":    node.Content(source),
			"className": ctx.className,
		}).Warn("Statement parse error")
		return &ast.BadStmt{}
	case "comment", "line_comment", "block_comment":
		return &ast.BadStmt{}
	case "local_variable_declaration":
		variableType := astutil.ParseType(node.ChildByFieldName("type"), source)
		originalType := node.ChildByFieldName("type").Content(source)
		variableDeclarator := node.ChildByFieldName("declarator")

		// If a variable is being declared, but not set to a value
		// Ex: `int value;`
		if variableDeclarator.NamedChildCount() == 1 {
			return &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names: []*ast.Ident{identFromNode(variableDeclarator.ChildByFieldName("name"), source)},
							Type:  variableType,
						},
					},
				},
			}
		}

		ctx.lastType = variableType
		// Set expected type for diamond operator inference
		ctx.expectedType = node.ChildByFieldName("type").Content(source)

		declaration := ParseStmt(variableDeclarator, source, ctx).(*ast.AssignStmt)

		// Now, if a variable is assigned to `null`, we can't infer its type, so
		// don't throw out the type information associated with it
		var containsNull bool

		// Go through the values and see if there is a `null_literal`
		for _, child := range nodeutil.NamedChildrenOf(variableDeclarator) {
			if child.Type() == "null_literal" {
				containsNull = true
				break
			}
		}

		names := make([]*ast.Ident, len(declaration.Lhs))
		for ind, decl := range declaration.Lhs {
			ident := decl.(*ast.Ident)
			names[ind] = ident
			recordedOriginalType := originalType
			// When assigning from object creation, preserve the concrete type for later
			// call-site coercions (e.g. subclass value passed where superclass is expected).
			if variableDeclarator.NamedChildCount() == 2 {
				if inferredType, ok := inferExprJavaType(variableDeclarator.NamedChild(1), ctx, source); ok && strings.TrimSpace(inferredType) != "" {
					recordedOriginalType = inferredType
				}
			}
			recordLocalVariableDefinition(ctx, ident.Name, recordedOriginalType, symbol.NodeToStr(variableType))
		}

		// If the declaration contains null, declare it with the `var` keyword instead
		// of implicitly
		if containsNull {
			return &ast.DeclStmt{
				Decl: &ast.GenDecl{
					Tok: token.VAR,
					Specs: []ast.Spec{
						&ast.ValueSpec{
							Names:  names,
							Type:   variableType,
							Values: declaration.Rhs,
						},
					},
				},
			}
		}

		return declaration
	case "variable_declarator":
		var names, values []ast.Expr

		// If there is only one node, then that node is just a name
		if node.NamedChildCount() == 1 {
			names = append(names, identFromNode(node.NamedChild(0), source))
		}

		// Loop through every pair of name and value
		for ind := 0; ind < int(node.NamedChildCount())-1; ind += 2 {
			names = append(names, identFromNode(node.NamedChild(ind), source))
			values = append(values, ParseExpr(node.NamedChild(ind+1), source, ctx))
		}

		return &ast.AssignStmt{Lhs: names, Tok: token.DEFINE, Rhs: values}
	case "assignment_expression":
		assignVar := ParseExpr(node.Child(0), source, ctx)
		assignVal := ParseExpr(node.Child(2), source, ctx)

		// Unsigned right shift
		if node.Child(1).Content(source) == ">>>=" {
			return &ast.ExprStmt{X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "UnsignedRightShiftAssignment"},
				Args: []ast.Expr{assignVar, assignVal},
			}}
		}

		return &ast.AssignStmt{
			Lhs: []ast.Expr{assignVar},
			Tok: StrToToken(node.Child(1).Content(source)),
			Rhs: []ast.Expr{assignVal},
		}
	case "update_expression":
		if node.Child(0).IsNamed() {
			return &ast.IncDecStmt{
				X:   ParseExpr(node.Child(0), source, ctx),
				Tok: StrToToken(node.Child(1).Content(source)),
			}
		}

		return &ast.IncDecStmt{
			X:   ParseExpr(node.Child(1), source, ctx),
			Tok: StrToToken(node.Child(0).Content(source)),
		}
	case "resource_specification":
		return ParseStmt(node.NamedChild(0), source, ctx)
	case "resource":
		var offset int
		if node.NamedChild(0).Type() == "modifiers" {
			offset = 1
		}
		return &ast.AssignStmt{
			Lhs: []ast.Expr{ParseExpr(node.NamedChild(1+offset), source, ctx)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{ParseExpr(node.NamedChild(2+offset), source, ctx)},
		}
	case "method_invocation":
		return &ast.ExprStmt{X: ParseExpr(node, source, ctx)}
	case "constructor_body", "block":
		body := &ast.BlockStmt{}
		for _, line := range nodeutil.NamedChildrenOf(node) {
			if line.Type() == "comment" || line.Type() == "line_comment" || line.Type() == "block_comment" {
				continue
			}
			if stmt := TryParseStmt(line, source, ctx); stmt != nil {
				body.List = append(body.List, stmt)
			} else if isStmtListNode(line.Type()) {
				// Try and synchronized statements are lowered into a list of statements
				body.List = append(body.List, ParseNode(line, source, ctx).([]ast.Stmt)...)
			} else {
				// Anything else (including unsupported constructs) is converted via
				// ParseStmt, which emits an UNSUPPORTED placeholder rather than crashing.
				body.List = append(body.List, ParseStmt(line, source, ctx))
			}
		}
		return body
	case "expression_statement":
		if stmt := TryParseStmt(node.NamedChild(0), source, ctx); stmt != nil {
			return stmt
		}
		return &ast.ExprStmt{X: ParseExpr(node.NamedChild(0), source, ctx)}
	case "explicit_constructor_invocation":
		// This is when a constructor calls another constructor with the use of
		// something such as `this(args...)`
		argsNode := node.ChildByFieldName("arguments")
		if argsNode == nil && node.NamedChildCount() > 1 {
			argsNode = node.NamedChild(1)
		}
		var args []ast.Expr
		if argsNode != nil {
			args = ParseNode(argsNode, source, ctx).([]ast.Expr)
		}

		constructorNode := node.ChildByFieldName("constructor")
		if constructorNode == nil && node.NamedChildCount() > 0 {
			constructorNode = node.NamedChild(0)
		}
		if constructorNode != nil && constructorNode.Type() == "super" && ctx.currentClass != nil {
			superType := strings.TrimSpace(ctx.currentClass.Superclass)
			if superType != "" {
				base, superArgStrs := parseJavaTypeString(superType)
				if base != "" {
					superName := stripJavaQualifier(base)
					// A built-in exception superclass is constructed via the stdjava
					// runtime; the embedded field is named after the runtime type.
					if isBuiltinExceptionType(superName) && resolveClassScopeByQualifiedName(ctx, base) == nil {
						recvName := ctx.className
						if recvName == "" && ctx.currentClass.Class != nil {
							recvName = ctx.currentClass.Class.Name
						}
						if recvName != "" {
							return &ast.AssignStmt{
								Lhs: []ast.Expr{&ast.SelectorExpr{
									X:   &ast.Ident{Name: ShortName(recvName)},
									Sel: &ast.Ident{Name: superName},
								}},
								Tok: token.ASSIGN,
								Rhs: []ast.Expr{&ast.CallExpr{
									Fun:  stdjavaQualifiedExpr("New"+superName, ctx),
									Args: args,
								}},
							}
						}
					}
					if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil && scope.Class != nil && scope.Class.Name != "" {
						superName = scope.Class.Name
					}
					funExpr := ast.Expr(&ast.Ident{Name: "New" + superName})
					if len(superArgStrs) > 0 {
						scopeTypeParams := inScopeTypeParameters(ctx)
						typeArgs := make([]ast.Expr, 0, len(superArgStrs))
						for _, arg := range superArgStrs {
							typeArgs = append(typeArgs, javaTypeStringToGoTypeExpr(arg, scopeTypeParams, ctx))
						}
						funExpr = applyTypeArguments(funExpr, typeArgs)
					}
					recvName := ctx.className
					if recvName == "" && ctx.currentClass.Class != nil {
						recvName = ctx.currentClass.Class.Name
					}
					if recvName != "" {
						return &ast.AssignStmt{
							Lhs: []ast.Expr{&ast.SelectorExpr{
								X:   &ast.Ident{Name: ShortName(recvName)},
								Sel: &ast.Ident{Name: superName},
							}},
							Tok: token.ASSIGN,
							Rhs: []ast.Expr{&ast.CallExpr{Fun: funExpr, Args: args}},
						}
					}
				}
			}
		}

		return &ast.ExprStmt{
			X: &ast.CallExpr{
				Fun:  &ast.Ident{Name: "New" + ctx.className},
				Args: args,
			},
		}
	case "return_statement":
		if ctx.tryReturnTarget != nil {
			stmts := []ast.Stmt{}
			if ctx.tryReturnTarget.HasValue && node.NamedChildCount() > 0 {
				returnCtx := ctx.Clone()
				if ctx.localScope != nil && strings.TrimSpace(ctx.localScope.OriginalType) != "" {
					returnCtx.expectedType = ctx.localScope.OriginalType
				}
				stmts = append(stmts, &ast.AssignStmt{
					Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.ValueName}},
					Tok: token.ASSIGN,
					Rhs: []ast.Expr{ParseExpr(node.NamedChild(0), source, returnCtx)},
				})
			}
			stmts = append(stmts, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: ctx.tryReturnTarget.FlagName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: "true"}},
			})
			stmts = append(stmts, &ast.ReturnStmt{})
			return &ast.BlockStmt{List: stmts}
		}

		if node.NamedChildCount() < 1 {
			return &ast.ReturnStmt{Results: []ast.Expr{}}
		}
		returnCtx := ctx.Clone()
		if ctx.localScope != nil && strings.TrimSpace(ctx.localScope.OriginalType) != "" {
			returnCtx.expectedType = ctx.localScope.OriginalType
		}
		return &ast.ReturnStmt{Results: []ast.Expr{ParseExpr(node.NamedChild(0), source, returnCtx)}}
	case "labeled_statement":
		return &ast.LabeledStmt{
			Label: identFromNode(node.NamedChild(0), source),
			Stmt:  ParseStmt(node.NamedChild(1), source, ctx),
		}
	case "break_statement":
		if node.NamedChildCount() > 0 {
			return &ast.BranchStmt{Tok: token.BREAK, Label: identFromNode(node.NamedChild(0), source)}
		}
		return &ast.BranchStmt{Tok: token.BREAK}
	case "continue_statement":
		if node.NamedChildCount() > 0 {
			return &ast.BranchStmt{Tok: token.CONTINUE, Label: identFromNode(node.NamedChild(0), source)}
		}
		return &ast.BranchStmt{Tok: token.CONTINUE}
	case "throw_statement":
		return &ast.ExprStmt{X: &ast.CallExpr{
			Fun:  &ast.Ident{Name: "panic"},
			Args: []ast.Expr{ParseExpr(node.NamedChild(0), source, ctx)},
		}}
	case "if_statement":
		var other ast.Stmt
		if node.ChildByFieldName("alternative") != nil {
			other = ParseStmt(node.ChildByFieldName("alternative"), source, ctx)
		}

		// If the `if` statement is inline, replace the line with a full block
		body := ParseStmt(node.ChildByFieldName("consequence"), source, ctx)
		if _, ok := body.(*ast.BlockStmt); !ok {
			body = &ast.BlockStmt{List: []ast.Stmt{
				body,
			}}
		}

		return &ast.IfStmt{
			Cond: ParseExpr(node.ChildByFieldName("condition"), source, ctx),
			Body: body.(*ast.BlockStmt),
			Else: other,
		}
	case "enhanced_for_statement":
		// An enhanced for statement has the following fields:
		// variables for the variable being declared (ex: int n)
		// then the expression that is being ranged over
		// and finally, the block of the expression

		total := int(node.NamedChildCount())
		typeNode := node.ChildByFieldName("type")
		nameNode := node.ChildByFieldName("name")
		valueNode := node.ChildByFieldName("value")
		bodyNode := node.ChildByFieldName("body")

		// Fallback for grammars that don't provide named fields.
		if nameNode == nil && total >= 3 {
			nameNode = node.NamedChild(total - 3)
		}
		if valueNode == nil && total >= 2 {
			valueNode = node.NamedChild(total - 2)
		}
		if bodyNode == nil && total >= 1 {
			bodyNode = node.NamedChild(total - 1)
		}

		loopCtx := ctx.Clone()
		if nameNode != nil && typeNode != nil {
			recordLocalVariableDefinition(loopCtx, nameNode.Content(source), typeNode.Content(source), symbol.NodeToStr(astutil.ParseType(typeNode, source)))
		}

		rangeValue := ast.Expr(&ast.Ident{Name: "_"})
		if nameNode != nil {
			rangeValue = ParseExpr(nameNode, source, loopCtx)
		}
		rangeExpr := ast.Expr(&ast.BadExpr{})
		if valueNode != nil {
			rangeExpr = ParseExpr(valueNode, source, ctx)
		}
		rangeBody := &ast.BlockStmt{}
		if bodyNode != nil {
			rangeBody = ParseStmt(bodyNode, source, loopCtx).(*ast.BlockStmt)
		}

		return &ast.RangeStmt{
			// We don't need the type of the variable for the range expression
			Key:   &ast.Ident{Name: "_"},
			Value: rangeValue,
			Tok:   token.DEFINE,
			X:     rangeExpr,
			Body:  rangeBody,
		}
	case "for_statement":
		var init, post ast.Stmt
		if node.ChildByFieldName("init") != nil {
			init = ParseStmt(node.ChildByFieldName("init"), source, ctx)
		}
		if node.ChildByFieldName("update") != nil {
			post = ParseStmt(node.ChildByFieldName("update"), source, ctx)
		}
		var cond ast.Expr
		if node.ChildByFieldName("condition") != nil {
			cond = ParseExpr(node.ChildByFieldName("condition"), source, ctx)
		}

		return &ast.ForStmt{
			Init: init,
			Cond: cond,
			Post: post,
			Body: ParseStmt(node.ChildByFieldName("body"), source, ctx).(*ast.BlockStmt),
		}
	case "while_statement":
		return &ast.ForStmt{
			Cond: ParseExpr(node.NamedChild(0), source, ctx),
			Body: ParseStmt(node.NamedChild(1), source, ctx).(*ast.BlockStmt),
		}
	case "do_statement":
		// A do statement is handled as a blank for loop with the condition
		// inserted as a break condition in the final part of the loop
		body := ParseStmt(node.NamedChild(0), source, ctx).(*ast.BlockStmt)

		body.List = append(body.List, &ast.IfStmt{
			Cond: &ast.UnaryExpr{
				Op: token.NOT,
				X: &ast.ParenExpr{
					X: ParseExpr(node.NamedChild(1), source, ctx),
				},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.BranchStmt{Tok: token.BREAK}}},
		})

		return &ast.ForStmt{
			Body: body,
		}
	case "switch_statement", "switch_expression":
		// A classic switch statement parses as `switch_expression` in tree-sitter.
		// Children: a parenthesized tag expression followed by the switch_block.
		var tagNode, blockNode *sitter.Node
		for _, c := range nodeutil.NamedChildrenOf(node) {
			switch c.Type() {
			case "switch_block":
				blockNode = c
			default:
				if tagNode == nil {
					tagNode = c
				}
			}
		}
		if blockNode == nil {
			return &ast.SwitchStmt{Body: &ast.BlockStmt{}}
		}
		return &ast.SwitchStmt{
			Tag:  ParseExpr(tagNode, source, ctx),
			Body: parseSwitchBlock(blockNode, source, ctx),
		}
	case "switch_block":
		return parseSwitchBlock(node, source, ctx)
	}
	return nil
}

// parseSwitchBlock lowers a Java switch body into a Go switch body, translating
// Java's fallthrough-by-default semantics into Go's break-by-default. A group
// that ends with `break` becomes an ordinary Go case (the break is dropped); a
// non-terminal group that does not break gets an explicit `fallthrough`. Empty
// groups (a label with no statements) are merged into the next group's case
// list, matching Java's stacked `case` labels.
func parseSwitchBlock(node *sitter.Node, source []byte, ctx Ctx) *ast.BlockStmt {
	switchBlock := &ast.BlockStmt{}

	groups := []*sitter.Node{}
	for _, c := range nodeutil.NamedChildrenOf(node) {
		if c.Type() == "switch_block_statement_group" {
			groups = append(groups, c)
		}
	}

	// Labels carried forward from preceding empty groups (stacked cases).
	var pendingExprs []ast.Expr
	var pendingDefault bool

	for index, group := range groups {
		caseExprs, isDefault, bodyNodes := splitSwitchGroup(group, source, ctx)

		caseExprs = append(pendingExprs, caseExprs...)
		isDefault = isDefault || pendingDefault

		// An empty group (no statements) stacks its labels onto the next group.
		if len(bodyNodes) == 0 && index != len(groups)-1 {
			pendingExprs = caseExprs
			pendingDefault = isDefault
			continue
		}
		pendingExprs = nil
		pendingDefault = false

		clause := &ast.CaseClause{}
		if !isDefault {
			clause.List = caseExprs
		}

		body, terminatedByBreak := parseSwitchGroupBody(bodyNodes, source, ctx)
		clause.Body = body

		// Java cases fall through unless they break/return. Go does the opposite,
		// so add an explicit fallthrough when the group neither breaks nor is the
		// final group.
		if !terminatedByBreak && index != len(groups)-1 && len(body) > 0 {
			clause.Body = append(clause.Body, &ast.BranchStmt{Tok: token.FALLTHROUGH})
		}

		switchBlock.List = append(switchBlock.List, clause)
	}

	return switchBlock
}

// splitSwitchGroup separates a switch_block_statement_group into its case label
// expressions (empty when the group is `default`), whether it is the default
// group, and the statement nodes that make up its body.
func splitSwitchGroup(group *sitter.Node, source []byte, ctx Ctx) (caseExprs []ast.Expr, isDefault bool, bodyNodes []*sitter.Node) {
	for _, child := range nodeutil.NamedChildrenOf(group) {
		if child.Type() == "switch_label" {
			if child.NamedChildCount() == 0 {
				// A `default` label has no child expression.
				isDefault = true
			} else {
				caseExprs = append(caseExprs, ParseExpr(child.NamedChild(0), source, ctx))
			}
			continue
		}
		bodyNodes = append(bodyNodes, child)
	}
	return caseExprs, isDefault, bodyNodes
}

// parseSwitchGroupBody converts the statement nodes of a switch group, dropping a
// trailing `break` (Go cases break implicitly) and reporting whether the group
// was terminated by such a break so the caller can decide on fallthrough.
func parseSwitchGroupBody(bodyNodes []*sitter.Node, source []byte, ctx Ctx) (body []ast.Stmt, terminatedByBreak bool) {
	for _, stmtNode := range bodyNodes {
		if stmtNode.Type() == "break_statement" {
			// A plain `break` ends the case in Java; Go does this implicitly. A
			// labeled break is rare in switches and is preserved as-is.
			if stmtNode.NamedChildCount() == 0 {
				terminatedByBreak = true
				continue
			}
		}
		if stmts := TryParseStmts(stmtNode, source, ctx); stmts != nil {
			body = append(body, stmts...)
		} else {
			body = append(body, ParseStmt(stmtNode, source, ctx))
		}
	}
	return body, terminatedByBreak
}

func recordLocalVariableDefinition(ctx Ctx, name, originalType, parsedType string) {
	if ctx.localScope == nil || strings.TrimSpace(name) == "" {
		return
	}

	for _, existing := range ctx.localScope.Children {
		if existing == nil || existing.OriginalName != name {
			continue
		}
		existing.Type = parsedType
		existing.OriginalType = originalType
		return
	}

	ctx.localScope.Children = append(ctx.localScope.Children, &symbol.Definition{
		OriginalName: name,
		Name:         name,
		OriginalType: originalType,
		Type:         parsedType,
	})
}

func ParseStmts(node *sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	if stmts := TryParseStmts(node, source, ctx); stmts != nil {
		return stmts
	}
	panic(fmt.Errorf("unhandled stmts type: %v", node.Type()))
}

func TryParseStmts(node *sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	switch node.Type() {
	case "assignment_expression":
		if stmts, ok := ParseNode(node, source, ctx).([]ast.Stmt); ok {
			return stmts
		}
	case "try_statement":
		if stmts, ok := ParseNode(node, source, ctx).([]ast.Stmt); ok {
			return stmts
		}
	}
	return nil
}

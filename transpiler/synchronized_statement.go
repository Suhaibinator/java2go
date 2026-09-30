package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// lowerSynchronizedStatement keeps Java abrupt completion outside the generated
// monitor closure. Go cannot return from the enclosing method or branch to an
// enclosing loop from a func literal, so the body records those completions,
// returns from the closure (running MonitorExit), and replays them afterwards.
// A panic needs no channel: ordinary Go unwinding already runs the monitor
// defer before propagating the thrown Java exception.
func lowerSynchronizedStatement(node *sitter.Node, source []byte, ctx Ctx) []ast.Stmt {
	if node == nil {
		return nil
	}

	lockNode := node.NamedChild(0)
	blockNode := node.NamedChild(1)
	if lockNode == nil || blockNode == nil {
		return nil
	}

	suffix := fmt.Sprintf("_%d", node.StartByte())
	usedNameRoot := node
	if ctx.localScope != nil && ctx.localScope.DeclarationNode != nil {
		usedNameRoot = ctx.localScope.DeclarationNode
	}
	usedNames := affineLoopUsedNames(usedNameRoot, source, ctx)
	monitorName := synchronizedUniqueLocalName("__java2goMonitor"+suffix, usedNames)
	executionName := ctx.executionContextName
	var executionInit ast.Stmt
	if executionName == "" {
		executionName = synchronizedUniqueLocalName(executionParamBase+suffix, usedNames)
		executionInit = &ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: executionName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{newExecutionExpr(ctx)},
		}
	}
	shouldReturnName := synchronizedUniqueLocalName("__java2goSyncShouldReturn"+suffix, usedNames)
	returnValueName := synchronizedUniqueLocalName("__java2goSyncReturnValue"+suffix, usedNames)
	controlName := synchronizedUniqueLocalName("__java2goSyncControl"+suffix, usedNames)

	hasReturnValue := false
	var returnValueType ast.Expr
	if ctx.localScope != nil {
		returnType := strings.TrimSpace(ctx.localScope.OriginalType)
		if returnType != "" && returnType != "void" {
			hasReturnValue = true
			returnValueType = javaTypeStringToGoTypeExpr(returnType, inScopeTypeParameters(ctx), ctx)
			returnValueType = directOwnerTypeParameterMethodResultType(
				ctx.currentClass,
				ctx.localScope,
				returnValueType,
				ctx,
			)
		}
	}

	returnTarget := &tryReturnTarget{
		FlagName:    shouldReturnName,
		ValueName:   returnValueName,
		HasValue:    hasReturnValue,
		ControlName: controlName,
	}
	bodyCtx := ctx.Clone()
	bodyCtx.executionContextName = executionName
	lockCtx := ctx.Clone()
	lockCtx.executionContextName = executionName
	bodyCtx.tryReturnTarget = returnTarget
	bodyCtx.tryControlBoundary = blockNode
	body := ParseStmt(blockNode, source, bodyCtx).(*ast.BlockStmt)

	varSpecs := []ast.Spec{&ast.ValueSpec{
		Names: []*ast.Ident{{Name: shouldReturnName}},
		Type:  &ast.Ident{Name: "bool"},
	}}
	if hasReturnValue {
		varSpecs = append(varSpecs, &ast.ValueSpec{
			Names: []*ast.Ident{{Name: returnValueName}},
			Type:  returnValueType,
		})
	}
	if len(returnTarget.controlList) > 0 {
		varSpecs = append(varSpecs, &ast.ValueSpec{
			Names: []*ast.Ident{{Name: controlName}},
			Type:  &ast.Ident{Name: "int"},
		})
	}

	guarded := append([]ast.Stmt{
		&ast.AssignStmt{
			Lhs: []ast.Expr{&ast.Ident{Name: monitorName}},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{stdjavaCall(ctx, "MonitorEnterExecution",
				&ast.Ident{Name: executionName}, ParseExpr(lockNode, source, lockCtx))},
		},
		&ast.DeferStmt{
			Call: stdjavaCall(ctx, "MonitorExitExecution", &ast.Ident{Name: monitorName}),
		},
	}, body.List...)

	statements := []ast.Stmt{}
	if executionInit != nil {
		statements = append(statements, executionInit)
	}
	statements = append(statements,
		&ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: varSpecs}},
		&ast.ExprStmt{X: &ast.CallExpr{Fun: &ast.FuncLit{
			Type: &ast.FuncType{},
			Body: &ast.BlockStmt{List: guarded},
		}}},
	)
	statements = append(statements, synchronizedReturnReplay(returnTarget, ctx))
	statements = append(statements, synchronizedControlReplay(returnTarget, ctx)...)
	return statements
}

// synchronizedUniqueLocalName uses a separator before its collision suffix.
// Source offsets are decimal, so appending a bare digit could reach another
// synchronized statement's natural base name (for example _300 + 0 == _3000).
func synchronizedUniqueLocalName(base string, used map[string]struct{}) string {
	if _, exists := used[base]; !exists {
		used[base] = struct{}{}
		return base
	}
	for suffix := 0; ; suffix++ {
		candidate := fmt.Sprintf("%s_%d", base, suffix)
		if _, exists := used[candidate]; exists {
			continue
		}
		used[candidate] = struct{}{}
		return candidate
	}
}

func synchronizedReturnReplay(target *tryReturnTarget, ctx Ctx) ast.Stmt {
	body := []ast.Stmt{}
	if enclosing := ctx.tryReturnTarget; enclosing != nil {
		if target.HasValue && enclosing.HasValue {
			body = append(body, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: enclosing.ValueName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: target.ValueName}},
			})
		}
		if len(enclosing.controlList) > 0 {
			body = append(body, &ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: enclosing.ControlName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}},
			})
		}
		body = append(body,
			&ast.AssignStmt{
				Lhs: []ast.Expr{&ast.Ident{Name: enclosing.FlagName}},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{&ast.Ident{Name: "true"}},
			},
			&ast.ReturnStmt{},
		)
	} else {
		body = append(body, replayedMethodReturnStmt(ctx, target.HasValue, target.ValueName))
	}

	return &ast.IfStmt{
		Cond: &ast.Ident{Name: target.FlagName},
		Body: &ast.BlockStmt{List: body},
	}
}

func synchronizedControlReplay(target *tryReturnTarget, ctx Ctx) []ast.Stmt {
	statements := make([]ast.Stmt, 0, len(target.controlList))
	for _, transfer := range target.controlList {
		if transfer == nil {
			continue
		}

		body := []ast.Stmt{lowerJavaControlTransferStmt(transfer.Tok, transfer.Label, transfer.JavaTarget, ctx)}
		if enclosing := ctx.tryReturnTarget; enclosing != nil && !transferTargetInsideBoundary(transfer, ctx.tryControlBoundary) {
			enclosingTransfer := enclosing.registerControlTransfer(transfer.Tok, transfer.Label, transfer.JavaTarget)
			if enclosingTransfer != nil {
				body = []ast.Stmt{
					&ast.AssignStmt{
						Lhs: []ast.Expr{&ast.Ident{Name: enclosing.FlagName}},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{&ast.Ident{Name: "false"}},
					},
					&ast.AssignStmt{
						Lhs: []ast.Expr{&ast.Ident{Name: enclosing.ControlName}},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", enclosingTransfer.Code)}},
					},
					&ast.ReturnStmt{},
				}
			}
		}

		statements = append(statements, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  &ast.Ident{Name: target.ControlName},
				Op: token.EQL,
				Y:  &ast.BasicLit{Kind: token.INT, Value: fmt.Sprintf("%d", transfer.Code)},
			},
			Body: &ast.BlockStmt{List: body},
		})
	}
	return statements
}

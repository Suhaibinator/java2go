package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"go/token"
)

func fileChannelSourceSlotType(slot string, ctx Ctx) *ast.FuncType {
	params := &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("__java2goChannelExecution")}, Type: &ast.StarExpr{X: stdjavaQualifiedExpr("Execution", ctx)}}}}
	if slot != fileChannelSourceCloseSlot {
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent("__java2goChannelBuffer")}, Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ByteBuffer", ctx)}})
	}
	if slot == fileChannelSourceReadAtSlot || slot == fileChannelSourceWriteAtSlot {
		params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent("__java2goChannelPosition")}, Type: ast.NewIdent("int64")})
	}
	signature := &ast.FuncType{Params: params}
	if slot != fileChannelSourceCloseSlot {
		signature.Results = &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int32")}}}
	}
	return signature
}
func fileChannelSourceProtocolFields(ctx Ctx, view bool) []*ast.Field {
	var fields []*ast.Field
	if view {
		fields = append(fields, &ast.Field{Names: []*ast.Ident{ast.NewIdent(fileChannelSourceViewMethod)}, Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("FileChannel", ctx)}}}}}})
	}
	for _, slot := range []string{fileChannelSourceReadSlot, fileChannelSourceReadAtSlot, fileChannelSourceWriteSlot, fileChannelSourceWriteAtSlot, fileChannelSourceCloseSlot} {
		fields = append(fields, &ast.Field{Names: []*ast.Ident{ast.NewIdent(slot)}, Type: fileChannelSourceSlotType(slot, ctx)})
	}
	return fields
}
func fileChannelSourceReceiver(scope *symbol.ClassScope) *ast.FieldList {
	return &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("channelSource")}, Type: &ast.StarExpr{X: instantiateGenericType(scope.Class.Name, typeParamExprs(scope.GoTypeParameterNames()))}}}}
}
func fileChannelSourceStorageField(scope *symbol.ClassScope, ctx Ctx) *ast.Field {
	if !fileChannelSourceDirectRoot(scope, ctx) {
		return nil
	}
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(fileChannelSourceStorage)}, Type: &ast.StarExpr{X: stdjavaQualifiedExpr("FileChannel", ctx)}}
}
func fileChannelSourceBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil || fileChannelSourceRoot(scope, ctx) == nil {
		return nil
	}
	var result []ast.Decl
	if fileChannelSourceDirectRoot(scope, ctx) {
		result = append(result, &ast.FuncDecl{Name: ast.NewIdent(fileChannelSourceViewMethod), Recv: fileChannelSourceReceiver(scope), Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("FileChannel", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.IfStmt{Cond: &ast.BinaryExpr{X: ast.NewIdent("channelSource"), Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("nil")}}}}},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent("channelSource"), Sel: ast.NewIdent(fileChannelSourceStorage)}}},
		}}})
	}
	for _, method := range scope.Methods {
		slot := fileChannelSourceProtocolSlot(method, scope, ctx)
		if slot == "" {
			continue
		}
		signature := fileChannelSourceSlotType(slot, ctx)
		args := methodCallArgs(signature.Params)
		call := methodCall(ast.NewIdent("channelSource"), executionImplementationName(method, scope, ctx), args...)
		statement := ast.Stmt(&ast.ExprStmt{X: call})
		if signature.Results != nil {
			statement = &ast.ReturnStmt{Results: []ast.Expr{call}}
		}
		result = append(result, &ast.FuncDecl{Name: ast.NewIdent(slot), Recv: fileChannelSourceReceiver(scope), Type: signature, Body: &ast.BlockStmt{List: []ast.Stmt{statement}}})
	}
	return result
}
func fileChannelSourceCarrierInitStmt(scope *symbol.ClassScope, receiverName string, self ast.Expr, ctx Ctx) ast.Stmt {
	if !fileChannelSourceDirectRoot(scope, ctx) {
		return nil
	}
	if self == nil {
		self = ast.NewIdent(receiverName)
	}
	target := func() ast.Expr {
		return &ast.TypeAssertExpr{X: &ast.CallExpr{Fun: ast.NewIdent("any"), Args: []ast.Expr{self}}, Type: &ast.InterfaceType{Methods: &ast.FieldList{List: fileChannelSourceProtocolFields(ctx, false)}}}
	}
	methods := &ast.CompositeLit{Type: stdjavaQualifiedExpr("FileChannelSourceMethods", ctx), Elts: []ast.Expr{
		&ast.KeyValueExpr{Key: ast.NewIdent("Read"), Value: &ast.SelectorExpr{X: target(), Sel: ast.NewIdent(fileChannelSourceReadSlot)}},
		&ast.KeyValueExpr{Key: ast.NewIdent("ReadAt"), Value: &ast.SelectorExpr{X: target(), Sel: ast.NewIdent(fileChannelSourceReadAtSlot)}},
		&ast.KeyValueExpr{Key: ast.NewIdent("Write"), Value: &ast.SelectorExpr{X: target(), Sel: ast.NewIdent(fileChannelSourceWriteSlot)}},
		&ast.KeyValueExpr{Key: ast.NewIdent("WriteAt"), Value: &ast.SelectorExpr{X: target(), Sel: ast.NewIdent(fileChannelSourceWriteAtSlot)}},
		&ast.KeyValueExpr{Key: ast.NewIdent("ImplClose"), Value: &ast.SelectorExpr{X: target(), Sel: ast.NewIdent(fileChannelSourceCloseSlot)}},
	}}
	return &ast.AssignStmt{Lhs: []ast.Expr{&ast.SelectorExpr{X: ast.NewIdent(receiverName), Sel: ast.NewIdent(fileChannelSourceStorage)}}, Tok: token.ASSIGN, Rhs: []ast.Expr{stdjavaCall(ctx, "NewFileChannelSourceReference", &ast.SelectorExpr{X: ast.NewIdent(receiverName), Sel: ast.NewIdent("ObjectInfo")}, methods)}}
}

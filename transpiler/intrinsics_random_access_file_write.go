package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func registerRandomAccessFileWriteIntrinsics() {
	for _, method := range []string{"setLength", "write"} {
		registerInstanceNodeIntrinsic("RandomAccessFile", method, func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if !randomAccessFileWriteSignature(node, ctx, source) {
				return unsupportedRandomAccessFileWrite(node, ctx, source)
			}
			return lowerRandomAccessFileWrite(receiver, node, ctx, source)
		})
		registerInstanceIntrinsicDerivedResultType("RandomAccessFile", method, func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
			if randomAccessFileWriteSignature(node, ctx, source) {
				return "void", true
			}
			return "", false
		})
	}
}

func randomAccessFileWriteSignature(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || executionExpr(ctx) == nil || invocationArgumentCount(node) != 1 || !randomAccessFileNativeAllocation(node.ChildByFieldName("object"), ctx, source) {
		return false
	}
	proof := ctx.Clone()
	proof.expectedType, proof.expectedTypeRoot = "", nil
	argument := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	if argument == nil {
		return false
	}
	switch node.ChildByFieldName("name").Content(source) {
	case "setLength":
		return fileChannelReadLongApplicable(argument, proof, source)
	case "write":
		if argument.Type() == "null_literal" {
			return true
		}
		var origin inferredJavaTypeOrigin
		actual, known := inferExprJavaType(argument, proof, source, &origin)
		element, rank := javaArrayTypeParts(strings.TrimSpace(actual))
		return known && !origin.unresolved && origin.parameter == nil && origin.nominalScope == nil && element == "byte" && rank == 1
	}
	return false
}

func unsupportedRandomAccessFileWrite(node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	diagnostic := reportUnsupported("intrinsic invocation", node, source, ctx)
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(diagnostic.String())})}}}}}
}

func lowerRandomAccessFileWrite(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	used := affineLoopUsedNames(node, source, ctx)
	used[ShortName(ctx.className)] = struct{}{}
	used[ctx.executionContextName] = struct{}{}
	if ctx.currentClass != nil {
		for _, parameter := range ctx.currentClass.GoTypeParameterNames() {
			used[parameter] = struct{}{}
		}
	}
	if ctx.localScope != nil {
		for _, parameter := range ctx.localScope.GoTypeParameterNames() {
			used[parameter] = struct{}{}
		}
	}
	name := func(base string) *ast.Ident {
		value := synchronizedUniqueLocalName(base, used)
		used[value] = struct{}{}
		return ast.NewIdent(value)
	}
	assign := func(target *ast.Ident, value ast.Expr) ast.Stmt {
		return &ast.AssignStmt{Lhs: []ast.Expr{target}, Tok: token.DEFINE, Rhs: []ast.Expr{value}}
	}
	file, arg := name("__java2goRAF"), name("__java2goRAFArgument")
	proof := ctx.Clone()
	proof.expectedType, proof.expectedTypeRoot = "", nil
	argument := invocationArgumentNode(node, 0)
	var value ast.Expr
	if unwrapParenthesizedExpressionNode(argument).Type() == "null_literal" {
		arrayType := &ast.StarExpr{X: &ast.IndexExpr{X: stdjavaQualifiedExpr("PrimitiveArray", ctx), Index: ast.NewIdent("int8")}}
		value = &ast.CallExpr{Fun: &ast.ParenExpr{X: arrayType}, Args: []ast.Expr{ast.NewIdent("nil")}}
	} else {
		value = ParseExpr(argument, source, proof)
	}
	body := []ast.Stmt{assign(file, receiver)}
	facade := "RandomAccessFileWriteBytesExecution"
	if node.ChildByFieldName("name").Content(source) == "setLength" {
		facade = "RandomAccessFileSetLengthExecution"
		var origin inferredJavaTypeOrigin
		actual, _ := inferExprJavaType(argument, proof, source, &origin)
		if _, primitive := javaPrimitiveType(actual); !primitive {
			base, declared, _ := fileChannelReadOriginContext(actual, origin, proof)
			primitive, _ := builtinJavaWrapperPrimitive(base, declared)
			wrapper := ternaryBoxedJavaType(primitive)
			boxed := name("__java2goRAFLengthBox")
			detail, _ := fileChannelReadNullDescription(argument, proof, source)
			message := "Cannot invoke " + strconv.Quote("java.lang."+wrapper+"."+primitive+"Value()") + " because " + detail + " is null"
			failure := stdjavaCall(ctx, "NewJavaNullPointerExceptionMessage", stdjavaCall(ctx, "JavaStringFromHostUTF8", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(message)}))
			body = append(body, assign(boxed, value), &ast.IfStmt{Cond: &ast.BinaryExpr{X: boxed, Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", failure)}}}})
			value = stdjavaCall(ctx, "Unbox"+wrapper, boxed)
		}
		value = &ast.CallExpr{Fun: ast.NewIdent("int64"), Args: []ast.Expr{value}}
	}
	body = append(body, assign(arg, value), &ast.ExprStmt{X: stdjavaCall(ctx, facade, executionExpr(ctx), file, arg)})
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{}}, Body: &ast.BlockStmt{List: body}}}
}

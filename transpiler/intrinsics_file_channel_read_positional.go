package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// The declaration sidecar is authoritative before a caller's TYPE namespace.
// In particular, a source return type cannot be recaptured by a caller binder.
func fileChannelReadOriginContext(actual string, origin inferredJavaTypeOrigin, ctx Ctx) (string, Ctx, bool) {
	if origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return "", ctx, false
	}
	proof := ctx
	if origin.declaringOwner != nil {
		proof = ctx.Clone()
		proof.currentFile = nil
		proof.syntheticTypeParameters = nil
		proof = classScopeCtx(origin.declaringOwner, proof)
		if proof.currentFile == nil {
			return "", proof, false
		}
		actual = origin.javaType
	}
	element, rank := javaArrayTypeParts(strings.TrimSpace(actual))
	base, arguments := parseJavaTypeString(element)
	if rank != 0 || len(arguments) != 0 || visibleTypeParameterDeclarationForJavaType(base, proof) != nil || resolveClassScopeByQualifiedName(proof, base) != nil {
		return "", proof, false
	}
	return base, proof, true
}

func fileChannelReadExternalExpression(node *sitter.Node, owner string, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	node = unwrapParenthesizedExpressionNode(node)
	if node.Type() == "object_creation_expression" && objectCreationClassBody(node) != nil {
		return false
	}
	var origin inferredJavaTypeOrigin
	actual, known := inferExprJavaType(node, ctx, source, &origin)
	if !known {
		return false
	}
	base, proof, eligible := fileChannelReadOriginContext(actual, origin, ctx)
	if !eligible {
		return false
	}
	canonical, registered := canonicalIntrinsicOwner(base, proof)
	if registered && canonical == owner {
		if base == owner {
			return true
		}
		if proof.currentFile != nil {
			if pkg, imported := proof.currentFile.Imports[base]; imported {
				return pkg+"."+base == owner
			}
			for _, pkg := range intrinsicOnDemandImports(proof) {
				if pkg+"."+base == owner {
					return true
				}
			}
		}
	}
	// The registered final RAF member returns FileChannel even when no caller
	// import names that result. This is a declaration-return certificate, not
	// evidence that the outer virtual read should use a native backend.
	return owner == "java.nio.channels.FileChannel" && node.Type() == "method_invocation" && node.ChildByFieldName("name").Content(source) == "getChannel" && invocationArgumentCount(node) == 0 && fileChannelReadExternalExpression(node.ChildByFieldName("object"), "java.io.RandomAccessFile", ctx, source)
}

func fileChannelReadLongApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil {
		return false
	}
	var origin inferredJavaTypeOrigin
	actual, known := inferExprJavaType(node, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return false
	}
	primitive, numeric := javaPrimitiveType(actual)
	if !numeric {
		base, proof, eligible := fileChannelReadOriginContext(actual, origin, ctx)
		if !eligible {
			return false
		}
		primitive, numeric = builtinJavaWrapperPrimitive(base, proof)
	}
	if !numeric {
		return false
	}
	if _, primitiveType := javaPrimitiveType(actual); !primitiveType {
		if _, described := fileChannelReadNullDescription(node, ctx, source); !described {
			return false
		}
	}
	if primitive == "long" {
		return true
	}
	_, widening := javaPrimitiveWideningDistance(primitive, "long")
	return widening
}

func fileChannelReadReceiverSelection(object *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if object == nil || object.Parent() == nil || invocationArgumentCount(object.Parent()) != 2 {
		return "", false
	}
	if creation := unwrapParenthesizedExpressionNode(object); creation.Type() == "object_creation_expression" && objectCreationClassBody(creation) != nil {
		return "", true
	}
	var origin inferredJavaTypeOrigin
	if _, known := inferExprJavaType(object, ctx, source, &origin); known && (origin.nominalScope != nil || origin.parameter != nil) {
		// Actual source declarations and captured binders retain source lookup.
		return "", true
	}
	if fileChannelReadExternalExpression(object, "java.nio.channels.FileChannel", ctx, source) {
		return "FileChannel", true
	}
	return "", false
}

func fileChannelReadPositionalApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || executionExpr(ctx) == nil || invocationArgumentCount(node) != 2 || !fileChannelReadExternalExpression(node.ChildByFieldName("object"), "java.nio.channels.FileChannel", ctx, source) || !fileChannelReadLongApplicable(invocationArgumentNode(node, 1), ctx, source) {
		return false
	}
	buffer := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	if buffer == nil || buffer.Type() != "null_literal" && !fileChannelReadExternalExpression(buffer, "java.nio.ByteBuffer", ctx, source) {
		return false
	}
	_, described := fileChannelReadNullDescription(node.ChildByFieldName("object"), ctx, source)
	return described
}

// Helpful invocation-null text comes from the selected declaration and the
// supported bytecode expression shape. Unrepresentable local-slot provenance
// remains an unsupported call instead of inventing a Java diagnostic.
func fileChannelReadNullDescription(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if node == nil {
		return "", false
	}
	node = unwrapParenthesizedExpressionNode(node)
	if node.Type() == "identifier" {
		if description, bound, described := fileChannelReadLambdaIdentifierDescription(node, ctx, source); bound {
			return description, described
		}
		name := node.Content(source)
		if ctx.localScope != nil {
			for index, parameter := range ctx.localScope.Parameters {
				if parameter != nil && parameter.OriginalName == name {
					for parent := node.Parent(); parent != nil && parent.Type() != "method_declaration" && parent.Type() != "constructor_declaration"; parent = parent.Parent() {
						if parent.Type() == "lambda_expression" {
							return "", false
						}
					}
					return "\"<parameter" + strconv.Itoa(index+1) + ">\"", true
				}
			}
			if ctx.localScope.FindVariable(name) != nil {
				return "", false
			}
		}
		if field := findFieldResolutionInHierarchy(ctx.currentClass, name, ctx); field != nil && field.def != nil && !field.def.IsStatic {
			return strconv.Quote("this." + field.def.OriginalName), true
		}
	}
	if node.Type() == "field_access" {
		object, fieldName := node.ChildByFieldName("object"), node.ChildByFieldName("field")
		if object != nil && object.Type() == "this" && fieldName != nil {
			if field := findFieldResolutionInHierarchy(ctx.currentClass, fieldName.Content(source), ctx); field != nil && field.def != nil && !field.def.IsStatic {
				return strconv.Quote("this." + field.def.OriginalName), true
			}
		}
	}
	if node.Type() == "method_invocation" {
		object := node.ChildByFieldName("object")
		name := node.ChildByFieldName("name").Content(source)
		if name == "getChannel" && invocationArgumentCount(node) == 0 && fileChannelReadExternalExpression(object, "java.io.RandomAccessFile", ctx, source) {
			return "the return value of \"java.io.RandomAccessFile.getChannel()\"", true
		}
		if object != nil {
			if scope := resolveClassScopeByIdentifier(ctx, source, object); scope != nil {
				selected := findBestMethodInHierarchy(scope, name, node.ChildByFieldName("arguments"), false, true, ctx, source)
				if selected != nil && selected.def != nil && selected.def.IsStatic && len(selected.def.Parameters) == 0 {
					return "the return value of " + strconv.Quote(javaClassBinaryName(selected.owner)+"."+selected.def.OriginalName+"()"), true
				}
			}
		}
	}
	return "", false
}

func lowerFileChannelReadPositional(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	description, described := fileChannelReadNullDescription(node.ChildByFieldName("object"), ctx, source)
	if !described || executionExpr(ctx) == nil {
		return unsupportedIntrinsicValue(node, "int", source, ctx)
	}
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
	channel, buffer, position := name("__java2goChannel"), name("__java2goBuffer"), name("__java2goPosition")
	assign := func(target *ast.Ident, value ast.Expr) ast.Stmt {
		return &ast.AssignStmt{Lhs: []ast.Expr{target}, Tok: token.DEFINE, Rhs: []ast.Expr{value}}
	}
	nullCheck := func(value *ast.Ident, method, detail string) ast.Stmt {
		message := "Cannot invoke " + strconv.Quote(method) + " because " + detail + " is null"
		throw := stdjavaCall(ctx, "NewJavaNullPointerExceptionMessage", stdjavaCall(ctx, "JavaStringFromHostUTF8", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(message)}))
		return &ast.IfStmt{Cond: &ast.BinaryExpr{X: value, Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", throw)}}}}
	}
	argCtx := ctx.Clone()
	argCtx.expectedType = ""
	argCtx.expectedTypeRoot = nil
	first := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	var dst ast.Expr
	if first.Type() == "null_literal" {
		dst = &ast.CallExpr{Fun: &ast.ParenExpr{X: &ast.StarExpr{X: stdjavaQualifiedExpr("ByteBuffer", ctx)}}, Args: []ast.Expr{ast.NewIdent("nil")}}
	} else {
		dst = ParseExpr(invocationArgumentNode(node, 0), source, argCtx)
	}
	offsetNode := invocationArgumentNode(node, 1)
	offset := ParseExpr(offsetNode, source, argCtx)
	var origin inferredJavaTypeOrigin
	actual, _ := inferExprJavaType(offsetNode, argCtx, source, &origin)
	body := []ast.Stmt{assign(channel, receiver), assign(buffer, dst)}
	if _, primitive := javaPrimitiveType(actual); !primitive {
		base, proof, _ := fileChannelReadOriginContext(actual, origin, ctx)
		primitive, _ := builtinJavaWrapperPrimitive(base, proof)
		boxed := name("__java2goOffsetBox")
		detail, _ := fileChannelReadNullDescription(offsetNode, ctx, source)
		wrapper := ternaryBoxedJavaType(primitive)
		body = append(body, assign(boxed, offset), nullCheck(boxed, "java.lang."+wrapper+"."+primitive+"Value()", detail))
		offset = stdjavaCall(ctx, "Unbox"+wrapper, boxed)
	}
	offset = &ast.CallExpr{Fun: ast.NewIdent("int64"), Args: []ast.Expr{offset}}
	body = append(body, assign(position, offset), nullCheck(channel, "java.nio.channels.FileChannel.read(java.nio.ByteBuffer, long)", description), &ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, "FileChannelReadAtVirtualExecution", executionExpr(ctx), channel, buffer, position)}})
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int32")}}}},
		Body: &ast.BlockStmt{List: body},
	}}
}

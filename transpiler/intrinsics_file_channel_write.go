package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

func registerFileChannelWriteIntrinsics() {
	registerInstanceNodeIntrinsic("FileChannel", "write", func(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		if _, applicable := fileChannelWriteSignature(node, ctx, source); !applicable {
			return unsupportedIntrinsicValue(node, "int", source, ctx)
		}
		return lowerFileChannelWrite(receiver, node, ctx, source)
	})
	registerInstanceIntrinsicDerivedResultType("FileChannel", "write", func(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if _, applicable := fileChannelWriteSignature(node, ctx, source); applicable {
			return "int", true
		}
		return "", false
	})
}

func fileChannelWriteReceiverSelection(object *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if object == nil || object.Parent() == nil {
		return "", false
	}
	arity := invocationArgumentCount(object.Parent())
	if arity != 1 && arity != 2 {
		return "", false
	}
	if creation := unwrapParenthesizedExpressionNode(object); creation.Type() == "object_creation_expression" && objectCreationClassBody(creation) != nil {
		return "", true
	}
	var origin inferredJavaTypeOrigin
	if _, known := inferExprJavaType(object, ctx, source, &origin); known && (origin.nominalScope != nil || origin.parameter != nil) {
		return "", true
	}
	if fileChannelReadExternalExpression(object, "java.nio.channels.FileChannel", ctx, source) {
		return "FileChannel", true
	}
	return "", false
}

// The emitter and Java result metadata use the same declaration/signature proof.
// A canonical declared receiver is a virtual view, not proof of native backing.
func fileChannelWriteSignature(node *sitter.Node, ctx Ctx, source []byte) (int, bool) {
	if node == nil || executionExpr(ctx) == nil {
		return 0, false
	}
	arity := invocationArgumentCount(node)
	if arity != 1 && arity != 2 || !fileChannelReadExternalExpression(node.ChildByFieldName("object"), "java.nio.channels.FileChannel", ctx, source) {
		return 0, false
	}
	buffer := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	if buffer == nil {
		return 0, false
	}
	// Bare null also matches write(ByteBuffer[]); retain the overload ambiguity.
	if buffer.Type() == "null_literal" {
		if arity == 1 {
			return 0, false
		}
	} else if !fileChannelReadExternalExpression(buffer, "java.nio.ByteBuffer", ctx, source) {
		return 0, false
	}
	if arity == 2 && !fileChannelReadLongApplicable(invocationArgumentNode(node, 1), ctx, source) {
		return 0, false
	}
	if _, proved := fileChannelWriteReceiverProofFor(node.ChildByFieldName("object"), ctx, source); !proved {
		return 0, false
	}
	return arity, true
}

func lowerFileChannelWrite(receiver ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	arity, applicable := fileChannelWriteSignature(node, ctx, source)
	if !applicable {
		return unsupportedIntrinsicValue(node, "int", source, ctx)
	}
	receiverProof, _ := fileChannelWriteReceiverProofFor(node.ChildByFieldName("object"), ctx, source)
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
	nullCheck := func(value *ast.Ident, method, detail string) ast.Stmt {
		message := "Cannot invoke " + strconv.Quote(method) + " because " + detail + " is null"
		failure := stdjavaCall(ctx, "NewJavaNullPointerExceptionMessage", stdjavaCall(ctx, "JavaStringFromHostUTF8", &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(message)}))
		return &ast.IfStmt{Cond: &ast.BinaryExpr{X: value, Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: callIdent("panic", failure)}}}}
	}
	channel, buffer := name("__java2goChannel"), name("__java2goBuffer")
	argCtx := ctx.Clone()
	argCtx.expectedType = ""
	argCtx.expectedTypeRoot = nil
	first := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, 0))
	var src ast.Expr
	if first.Type() == "null_literal" {
		src = &ast.CallExpr{Fun: &ast.ParenExpr{X: &ast.StarExpr{X: stdjavaQualifiedExpr("ByteBuffer", ctx)}}, Args: []ast.Expr{ast.NewIdent("nil")}}
	} else {
		src = ParseExpr(invocationArgumentNode(node, 0), source, argCtx)
	}
	body := []ast.Stmt{assign(channel, receiver), assign(buffer, src)}
	facade, method := "FileChannelWriteVirtualExecution", "java.nio.channels.FileChannel.write(java.nio.ByteBuffer)"
	args := []ast.Expr{executionExpr(ctx), channel, buffer}
	if arity == 2 {
		offsetNode := invocationArgumentNode(node, 1)
		offset := ParseExpr(offsetNode, source, argCtx)
		var origin inferredJavaTypeOrigin
		actual, _ := inferExprJavaType(offsetNode, argCtx, source, &origin)
		if _, primitive := javaPrimitiveType(actual); !primitive {
			base, proof, _ := fileChannelReadOriginContext(actual, origin, ctx)
			primitive, _ := builtinJavaWrapperPrimitive(base, proof)
			boxed := name("__java2goOffsetBox")
			detail, _ := fileChannelReadNullDescription(offsetNode, ctx, source)
			wrapper := ternaryBoxedJavaType(primitive)
			body = append(body, assign(boxed, offset), nullCheck(boxed, "java.lang."+wrapper+"."+primitive+"Value()", detail))
			offset = stdjavaCall(ctx, "Unbox"+wrapper, boxed)
		}
		position := name("__java2goPosition")
		body = append(body, assign(position, &ast.CallExpr{Fun: ast.NewIdent("int64"), Args: []ast.Expr{offset}}))
		args = append(args, position)
		facade, method = "FileChannelWriteAtVirtualExecution", "java.nio.channels.FileChannel.write(java.nio.ByteBuffer, long)"
	}
	if receiverProof.kind == fileChannelWriteReceiverNullable {
		body = append(body, nullCheck(channel, method, receiverProof.description))
	}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, facade, args...)}})
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int32")}}}}, Body: &ast.BlockStmt{List: body}}}
}

// Extend invocation-null provenance only when the actual selected source static
// declaration has a known fixed descriptor. Caller expectations never supply it.
func fileChannelWriteReceiverDescription(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	if description, known := fileChannelReadNullDescription(node, ctx, source); known {
		return description, true
	}
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || node.Type() != "method_invocation" {
		return "", false
	}
	name := node.ChildByFieldName("name")
	if name == nil {
		return "", false
	}
	scope := ctx.currentClass
	if object := node.ChildByFieldName("object"); object != nil {
		scope = resolveClassScopeByIdentifier(ctx, source, object)
	}
	if scope == nil {
		return "", false
	}
	selected := findBestMethodInHierarchy(scope, name.Content(source), node.ChildByFieldName("arguments"), false, true, ctx, source)
	if selected == nil || selected.def == nil || selected.owner != scope || !selected.def.IsStatic || selected.def.Constructor || len(selected.def.TypeParameters) != 0 || !sourceReflectionDeclaredMethod(selected.owner, selected.def) {
		return "", false
	}
	// Clear caller ownership before constructing the genuine declaration context.
	cleared := ctx.Clone()
	cleared.currentFile = nil
	cleared.syntheticTypeParameters = nil
	declaring := classScopeCtx(selected.owner, cleared)
	file := findFileScopeForClassScope(selected.owner, cleared)
	if file == nil || declaring.currentFile != file {
		return "", false
	}
	declaring.localScope = selected.def
	declaring.expectedType = ""
	declaring.expectedTypeRoot = nil
	result, known := fileChannelWriteDeclaredDiagnosticType(symbol.JavaType{Original: selected.def.OriginalType, TypeParameterBindings: selected.def.TypeParameterBindings}, declaring)
	if !known || result != "java.nio.channels.FileChannel" {
		return "", false
	}
	parameters := func(method *symbol.Definition) ([]string, bool) {
		if method == nil || !method.IsStatic || method.Constructor || len(method.TypeParameters) != 0 || len(method.Parameters) != invocationArgumentCount(node) || len(method.Parameters) > 0 && executionParameterIsVariadic(method, len(method.Parameters)-1) {
			return nil, false
		}
		proof := declaring.Clone()
		proof.localScope = method
		names := make([]string, len(method.Parameters))
		for index, parameter := range method.Parameters {
			if parameter == nil || parameter.DirectTypeParameter != nil {
				return nil, false
			}
			formal, ok := fileChannelWriteDeclaredDiagnosticType(symbol.JavaType{Original: definitionParameterJavaSignatureType(method, index), TypeParameterBindings: parameter.TypeParameterBindings}, proof)
			if !ok {
				return nil, false
			}
			argument := unwrapParenthesizedExpressionNode(invocationArgumentNode(node, index))
			if argument == nil {
				return nil, false
			}
			if expected, primitive := javaPrimitiveType(formal); primitive {
				var origin inferredJavaTypeOrigin
				actual, known := inferExprJavaType(argument, ctx, source, &origin)
				value, numeric := javaPrimitiveType(actual)
				if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil || !numeric {
					return nil, false
				}
				if value != expected {
					if _, widening := javaPrimitiveWideningDistance(value, expected); !widening {
						return nil, false
					}
				}
			} else if argument.Type() != "null_literal" && !fileChannelReadExternalExpression(argument, formal, ctx, source) {
				return nil, false
			}
			names[index] = formal
		}
		return names, true
	}
	formals, applicable := parameters(selected.def)
	if !applicable {
		return "", false
	}
	// A second independently applicable fixed declaration makes this finite proof
	// insufficient. Generic/vararg descriptor ambiguity is also kept explicit.
	for _, candidate := range scope.Methods {
		if candidate == nil || candidate == selected.def || !candidate.IsStatic || candidate.OriginalName != selected.def.OriginalName || len(candidate.Parameters) != len(selected.def.Parameters) {
			continue
		}
		if len(candidate.TypeParameters) != 0 || len(candidate.Parameters) > 0 && executionParameterIsVariadic(candidate, len(candidate.Parameters)-1) {
			return "", false
		}
		if _, alsoApplicable := parameters(candidate); alsoApplicable {
			return "", false
		}
	}
	return "the return value of " + strconv.Quote(javaClassBinaryName(selected.owner)+"."+selected.def.OriginalName+"("+strings.Join(formals, ", ")+")"), true
}

func fileChannelWriteDeclaredDiagnosticType(javaType symbol.JavaType, ctx Ctx) (string, bool) {
	element, rank := javaArrayTypeParts(strings.TrimSpace(javaType.Original))
	base, arguments := parseJavaTypeString(element)
	if rank != 0 || len(arguments) != 0 || base == "" || base == "void" || javaType.TypeParameterBindings[base] != nil || visibleTypeParameterDeclarationForJavaType(base, ctx) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", false
	}
	if primitive, known := javaPrimitiveType(base); known {
		return primitive, true
	}
	qualified := qualifyDeclaredReferenceType(javaType, ctx)
	owner, canonical := canonicalIntrinsicOwner(qualified, ctx)
	if !canonical {
		return "", false
	}
	return owner, true
}

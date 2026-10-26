package transpiler

import (
	"go/ast"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func canonicalStringReaderReferenceOwner(base string, parameters []string, ctx Ctx) bool {
	first := strings.SplitN(strings.TrimSpace(base), ".", 2)[0]
	if visibleTypeParameterDeclarationForJavaType(first, ctx) != nil || containsString(parameters, first) || resolveClassScopeByQualifiedName(ctx, first) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	return known && owner == "java.io.StringReader" && intrinsicOwnerSupported(owner)
}

// The caller retains array dimensions; this function certifies the component.
func stringReaderReferenceRuntimeTypeExpr(javaType string, typeArgs, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	element, _ := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, writtenArguments := parseJavaTypeString(element)
	if !canonicalStringReaderReferenceOwner(base, typeParams, ctx) {
		return nil, false
	}
	if len(typeArgs) != 0 || len(writtenArguments) != 0 {
		return &ast.BadExpr{}, true
	}
	return &ast.StarExpr{X: stdjavaQualifiedExpr("StringReader", ctx)}, true
}

func stringReaderReferenceDeclaringContext(origin inferredJavaTypeOrigin, ctx Ctx) Ctx {
	if origin.declaringOwner == nil {
		return ctx
	}
	proof := ctx.Clone()
	proof.currentFile = nil
	proof.localScope = nil
	proof.localClasses = nil
	proof.localBindingBody = nil
	proof.syntheticTypeParameters = nil
	return classScopeCtx(origin.declaringOwner, proof)
}
func stringReaderReferenceReceiverApplicable(object *sitter.Node, ctx Ctx, source []byte) bool {
	if object == nil || object.Type() == "super" {
		return false
	}
	var origin inferredJavaTypeOrigin
	javaType, known := inferExprJavaType(object, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return false
	}
	element, rank := javaArrayTypeParts(javaType)
	base, arguments := parseJavaTypeString(element)
	if rank != 0 || len(arguments) != 0 {
		return false
	}
	return canonicalStringReaderReferenceOwner(base, nil, stringReaderReferenceDeclaringContext(origin, ctx))
}
func stringReaderReferenceStringArgumentApplicable(node *sitter.Node, ctx Ctx, source []byte) bool {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return false
	}
	if node.Type() == "null_literal" {
		return true
	}
	var origin inferredJavaTypeOrigin
	javaType, known := inferExprJavaType(node, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return false
	}
	element, rank := javaArrayTypeParts(javaType)
	_, arguments := parseJavaTypeString(element)
	return rank == 0 && len(arguments) == 0 && isBuiltinJavaString(javaType, stringReaderReferenceDeclaringContext(origin, ctx))
}
func stringReaderReferenceUnsupported(node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	reportUnsupported("canonical StringReader invocation", node, source, ctx)
	return &ast.BadExpr{}
}

func registerStringReaderReferenceIntrinsics() {
	registerIntrinsicOwner("java.io.StringReader", true)
	registerConstructorNodeIntrinsic("StringReader", func(typeArgs, args []ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
		typeNode := node.ChildByFieldName("type")
		if typeNode == nil || len(typeArgs) != 0 || len(args) != 1 || node.ChildByFieldName("body") != nil {
			return stringReaderReferenceUnsupported(node, ctx, source)
		}
		element, rank := javaArrayTypeParts(typeNode.Content(source))
		base, arguments := parseJavaTypeString(element)
		if rank != 0 || len(arguments) != 0 || !canonicalStringReaderReferenceOwner(base, nil, ctx) || sourceGenericViewLocalTypeShadowed(typeNode, base, source) || !stringReaderReferenceStringArgumentApplicable(invocationArgumentNode(node, 0), ctx, source) {
			return stringReaderReferenceUnsupported(node, ctx, source)
		}
		// Constructor arguments are already rendered by the original caller.
		return stdjavaCall(ctx, "NewStringReaderReference", args[0])
	})
	for _, method := range []string{"read", "skip", "ready", "markSupported", "mark", "reset", "close"} {
		method := method
		registerInstanceNodeIntrinsic("StringReader", method, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			// An actual selected source declaration owns its call even when its
			// retained spelling matches the canonical JDK owner.
			if target := resolveInvocationTarget(node.ChildByFieldName("object"), ctx, source); target != nil {
				if declaration, _ := findBestMethodForInvocationTarget(target, method, node.ChildByFieldName("arguments"), true, false, ctx, source); declaration != nil {
					return nil
				}
			}
			if !stringReaderReferenceReceiverApplicable(node.ChildByFieldName("object"), ctx, source) {
				return stringReaderReferenceUnsupported(node, ctx, source)
			}
			count := invocationArgumentCount(node)
			var expected []string
			runtimeName := ""
			switch method {
			case "read":
				switch count {
				case 0:
					runtimeName = "StringReaderReadCharExecution"
				case 1:
					runtimeName = "ReaderReadArrayExecution"
					expected = []string{"char[]"}
				case 3:
					runtimeName = "StringReaderReadCharsExecution"
					expected = []string{"char[]", "int", "int"}
				}
			case "skip":
				if count == 1 {
					runtimeName = "StringReaderSkipExecution"
					expected = []string{"long"}
				}
			case "mark":
				if count == 1 {
					runtimeName = "StringReaderMarkExecution"
					expected = []string{"int"}
				}
			case "ready":
				if count == 0 {
					runtimeName = "StringReaderReadyExecution"
				}
			case "markSupported":
				if count == 0 {
					runtimeName = "StringReaderMarkSupportedExecution"
				}
			case "reset":
				if count == 0 {
					runtimeName = "StringReaderResetExecution"
				}
			case "close":
				if count == 0 {
					runtimeName = "StringReaderCloseExecution"
				}
			}
			if runtimeName == "" {
				return stringReaderReferenceUnsupported(node, ctx, source)
			}
			for i, t := range expected {
				if !intrinsicInvocationConversionApplicable(invocationArgumentNode(node, i), t, ctx, source) {
					return stringReaderReferenceUnsupported(node, ctx, source)
				}
			}
			args := parseArgumentListWithExpectedTypes(node.ChildByFieldName("arguments"), source, ctx, expected)
			return stdjavaCall(ctx, runtimeName, append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, args...)...)
		})
	}
	registerInstanceIntrinsicResultType("StringReader", "read", "int")
	registerInstanceIntrinsicResultType("StringReader", "skip", "long")
	registerInstanceIntrinsicResultType("StringReader", "ready", "boolean")
	registerInstanceIntrinsicResultType("StringReader", "markSupported", "boolean")
}

func registerBufferedStringReaderReferenceIntrinsics() {
	for method, runtimeName := range map[string]string{"readLine": "BufferedStringReaderReadLineExecution", "close": "BufferedStringReaderCloseExecution"} {
		method, runtimeName := method, runtimeName
		registerInstanceNodeIntrinsic("BufferedReader", method, func(recv ast.Expr, node *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(node) != 0 {
				return stringReaderReferenceUnsupported(node, ctx, source)
			}
			// The existing canonical-owner receiver selector admits the native class;
			// actual source receivers retain their source dispatch, without projection.
			return stdjavaCall(ctx, runtimeName, intrinsicExecutionExpr(ctx), recv)
		})
	}
	registerInstanceIntrinsicResultType("BufferedReader", "readLine", "String")
}

// Component validity is checked before the existing common reference-array ABI.
// This does not choose an array representation or add component projections.
func stringReaderReferenceArrayTypeRefusal(javaType string, typeParams []string, ctx Ctx) (ast.Expr, bool) {
	element, rank := javaArrayTypeParts(javaType)
	if rank == 0 {
		return nil, false
	}
	base, arguments := parseJavaTypeString(element)
	if !canonicalStringReaderReferenceOwner(base, typeParams, ctx) {
		return nil, false
	}
	if len(arguments) == 0 {
		return nil, false
	}
	return &ast.BadExpr{}, true
}

package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func init() {
	registerInstanceNodeIntrinsic("Class", "cast", classCastIntrinsic)
}

// Receiver declaration provenance, rather than an assignment's expected type,
// owns Class<T>'s result. Literal targets need their own syntax origin because
// the general literal inference sidecar describes only the Class allocation.
func classCastReceiverOrigin(node *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return inferredJavaTypeOrigin{}, false
	}
	if node.Type() == "class_literal" {
		target, ok := classLiteralJavaType(node, source)
		if !ok {
			return inferredJavaTypeOrigin{}, false
		}
		var arg inferredJavaTypeOrigin
		recordSyntaxJavaTypeOrigin([]*inferredJavaTypeOrigin{&arg}, target, ctx)
		if !completeJavaTypeOrigin(arg) {
			return inferredJavaTypeOrigin{}, false
		}
		if arg.arrayRank == 0 {
			if _, primitive := javaPrimitiveType(target); primitive {
				// Class literals have wrapper type arguments, but retain primitive runtime
				// tokens. These are the JLS primitive wrapper declarations, not carriers.
				for _, wrapper := range []string{"Boolean", "Byte", "Short", "Character", "Integer", "Long", "Float", "Double"} {
					if primitive, ok := ternaryBoxedPrimitive(wrapper); ok && primitive == target {
						arg = intrinsicJavaTypeOrigin("java.lang." + wrapper)
						break
					}
				}
			} else if target == "void" {
				arg = intrinsicJavaTypeOrigin("java.lang.Void")
			}
		}
		result := intrinsicJavaTypeOrigin("java.lang.Class")
		result.arguments = []inferredJavaTypeOrigin{arg}
		return result, true
	}
	if result, ok := classCastGetClassReceiverOrigin(node, ctx, source); ok {
		return result, true
	}
	var origin inferredJavaTypeOrigin
	if _, ok := inferExprJavaType(node, ctx, source, &origin); !ok {
		return origin, false
	}
	origin, ok := classCastErasedParameterOrigin(origin, ctx, map[*symbol.TypeParamDeclaration]bool{})
	return origin, ok && completeJavaTypeOrigin(origin) && origin.arrayRank == 0 && origin.parameter == nil && origin.nominalScope == nil && origin.externalName == "java.lang.Class"
}

// Object.getClass is final. Its expression type is Class<? extends |X|>, where
// |X| is the erasure of the static receiver type, independently of the object
// subsequently passed to Class.cast. Explicit raw/Class<?> views stay erased.
func classCastGetClassReceiverOrigin(node *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	if node.Type() != "method_invocation" || invocationArgumentCount(node) != 0 {
		return inferredJavaTypeOrigin{}, false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(source) != "getClass" {
		return inferredJavaTypeOrigin{}, false
	}
	var origin inferredJavaTypeOrigin
	if receiver := node.ChildByFieldName("object"); receiver != nil {
		if _, ok := inferExprJavaType(receiver, ctx, source, &origin); !ok {
			return origin, false
		}
	} else {
		if implicitRuntimeClassMethod(node, nil, ctx, source) != "ObjectGetClass" {
			return origin, false
		}
		origin = inferredJavaTypeOrigin{nominalScope: ctx.currentClass}
	}
	origin, ok := classCastErasedParameterOrigin(origin, ctx, map[*symbol.TypeParamDeclaration]bool{})
	if !ok || !completeJavaTypeOrigin(origin) {
		return origin, false
	}
	if origin.nominalScope == nil && origin.externalName == "" {
		return origin, false
	}
	origin.arguments = nil
	result := intrinsicJavaTypeOrigin("java.lang.Class")
	result.arguments = []inferredJavaTypeOrigin{origin}
	return result, true
}

func classCastErasedParameterOrigin(origin inferredJavaTypeOrigin, ctx Ctx, seen map[*symbol.TypeParamDeclaration]bool) (inferredJavaTypeOrigin, bool) {
	if origin.unresolved {
		return origin, false
	}
	if origin.parameter == nil {
		return origin, completeJavaTypeOrigin(origin)
	}
	if seen[origin.parameter] {
		return origin, false
	}
	seen[origin.parameter] = true
	defer delete(seen, origin.parameter)
	for _, binding := range referenceTypeParameterBindings(ctx) {
		if binding.parameter.Declaration != origin.parameter {
			continue
		}
		erased := intrinsicJavaTypeOrigin("java.lang.Object")
		if len(binding.parameter.Bounds) > 0 {
			var ok bool
			erased, ok = classCastErasedParameterOrigin(declaredJavaTypeOrigin(binding.parameter.Bounds[0], binding.context), binding.context, seen)
			if !ok {
				return erased, false
			}
		}
		erased.arrayRank += origin.arrayRank
		return erased, true
	}
	return origin, false
}

func classCastResultOrigin(invocation *sitter.Node, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	if invocation == nil || invocation.Type() != "method_invocation" || invocationArgumentCount(invocation) != 1 {
		return inferredJavaTypeOrigin{}, false
	}
	name := invocation.ChildByFieldName("name")
	if name == nil || name.Content(source) != "cast" {
		return inferredJavaTypeOrigin{}, false
	}
	receiver, ok := classCastReceiverOrigin(invocation.ChildByFieldName("object"), ctx, source)
	if !ok {
		return receiver, false
	}
	if len(receiver.arguments) == 0 {
		return intrinsicJavaTypeOrigin("java.lang.Object"), true
	}
	if len(receiver.arguments) != 1 || !completeJavaTypeOrigin(receiver.arguments[0]) {
		return inferredJavaTypeOrigin{}, false
	}
	return receiver.arguments[0], true
}

func classCastOriginJavaType(origin inferredJavaTypeOrigin) string {
	var name string
	switch {
	case origin.nominalScope != nil:
		name = qualifiedSourceClassName(origin.nominalScope)
	case origin.parameter != nil:
		name = origin.parameter.GoName
		if name == "" {
			name = origin.parameter.SourceName
		}
	case origin.externalName != "":
		name = origin.externalName
	default:
		name, _ = javaArrayTypeParts(origin.javaType)
	}
	if len(origin.arguments) > 0 {
		args := make([]string, len(origin.arguments))
		for i, arg := range origin.arguments {
			args[i] = classCastOriginJavaType(arg)
		}
		name += "<" + strings.Join(args, ", ") + ">"
	}
	return name + strings.Repeat("[]", origin.arrayRank)
}

func classCastResultType(node *sitter.Node, ctx Ctx, source []byte, outputs ...*inferredJavaTypeOrigin) (string, bool) {
	origin, ok := classCastResultOrigin(node, ctx, source)
	if !ok {
		return "", false
	}
	setInferredJavaTypeOrigin(outputs, origin)
	return classCastOriginJavaType(origin), true
}

// Declaration lookup and output import contexts are separate. Source names
// are emitted from their exact nominal scope; method variables use their
// existing physical ABI slots, never a caller binder with the same spelling.
func classCastResultGoType(origin inferredJavaTypeOrigin, ctx Ctx) (ast.Expr, bool) {
	if !completeJavaTypeOrigin(origin) {
		return nil, false
	}
	if origin.arrayRank > 0 {
		if origin.arrayRank == 1 && origin.parameter == nil && origin.nominalScope == nil && origin.externalName == "" {
			if result, ok := primitiveArrayTypeExpr(classCastOriginJavaType(origin), ctx); ok {
				return result, true
			}
		}
		return reifiedReferenceArrayTypeExpr(ctx), true
	}
	if origin.parameter != nil {
		for _, binding := range referenceTypeParameterBindings(ctx) {
			if binding.parameter.Declaration != origin.parameter {
				continue
			}
			name := binding.parameter.EmittedName()
			// Only an active body representation can erase this declaration.
			// Ordinary static generic functions still return their Go variable T.
			physical := erasedAnonymousMethodJavaType(genericFamilyPhysicalJavaType(name, ctx), ctx)
			if physical != name {
				erased, ok := classCastErasedParameterOrigin(origin, ctx, map[*symbol.TypeParamDeclaration]bool{})
				if !ok {
					return nil, false
				}
				return classCastResultGoType(erased, ctx)
			}
			return ast.NewIdent(name), true
		}
		return nil, false
	}
	if scope := origin.nominalScope; scope != nil {
		var args []ast.Expr
		if len(origin.arguments) == 0 && len(scope.TypeParameters) > 0 {
			args = sourceClassGoTypeArgumentExprs(scope, nil, nil, nil, nil, ctx)
		} else {
			if len(origin.arguments) != len(scope.TypeParameters) {
				return nil, false
			}
			for _, arg := range origin.arguments {
				expr, ok := classCastResultGoType(arg, ctx)
				if !ok {
					return nil, false
				}
				args = append(args, expr)
			}
		}
		expr := classSubobjectPointerTypeWithGoArguments(scope, args, ctx)
		if scope.IsInterface {
			return expr.(*ast.StarExpr).X, true
		}
		return expr, true
	}
	if result, ok := canonicalResultPhysicalExpr(origin, ctx); ok {
		return result, true
	}
	if origin.externalName == "java.lang.Void" {
		return nil, false
	}
	if origin.externalName == "" {
		return nil, false
	}
	// The external name is already proved at its declaration. Clear lookup
	// namespaces only for representation mapping so caller source roots cannot
	// recapture that certificate. Output import maps remain the caller's maps.
	native := ctx.Clone()
	native.currentClass = nil
	native.currentFile = nil
	native.localScope = nil
	native.syntheticTypeParameters = nil
	return javaTypeStringToGoTypeExpr(origin.externalName, nil, native), true
}

func classCastIntrinsic(receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
	origin, ok := classCastResultOrigin(invocation, ctx, source)
	if !ok {
		return nil
	}
	target, ok := classCastResultGoType(origin, ctx)
	if !ok {
		return nil
	}
	receiver = snapshotJavaExpressionValue(receiver, ctx)
	argument := invocationArgumentNode(invocation, 0)
	value := ParseExpr(argument, source, ctx)
	value = coerceArgumentToExpectedType(value, argument, "java.lang.Object", ctx, source)
	return stdjavaGenericCall(ctx, "ClassCast", []ast.Expr{target}, []ast.Expr{receiver, value})
}

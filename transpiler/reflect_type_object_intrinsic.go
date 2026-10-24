package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func init() {
	for name := range reflectProtocolConstants {
		registerIntrinsicOwner("java.lang.reflect."+name, true)
		registerInstanceIntrinsic(name, "equals", func(receiver ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 1 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectEqualsExecution", intrinsicExecutionExpr(ctx), receiver, args[0])
		})
		registerInstanceIntrinsicResultType(name, "equals", "boolean")
		registerInstanceIntrinsic(name, "hashCode", func(receiver ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
			if len(args) != 0 {
				return nil
			}
			return stdjavaCall(ctx, "ObjectHashCodeExecution", intrinsicExecutionExpr(ctx), receiver)
		})
		registerInstanceIntrinsicResultType(name, "hashCode", "int")
	}
}

// The Java declaration, rather than a narrowed Go representation, determines
// whether the receiver exposes the reflection protocol's Object contract.
func reflectObjectReceiverProtocol(javaType string, ctx Ctx) (string, bool) {
	base, rank := javaArrayTypeParts(javaType)
	if rank != 0 {
		return "", false
	}
	if binding, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); bound {
		erased, _ := javaTypeParameterErasure(base, ctx)
		base = erased
		ctx = binding.context
	}
	owner, registered := canonicalIntrinsicOwner(base, ctx)
	if !registered || !intrinsicOwnerSupported(owner) || !strings.HasPrefix(owner, "java.lang.reflect.") {
		return "", false
	}
	name := stripJavaQualifier(owner)
	_, reflection := reflectProtocolConstants[name]
	return name, reflection
}

func reflectObjectInvocationReceiver(object *sitter.Node, method string, ctx Ctx, source []byte) (string, bool) {
	if object == nil || (method != "equals" && method != "hashCode") {
		return "", false
	}
	invocation := object.Parent()
	if invocation == nil || invocation.Type() != "method_invocation" {
		return "", false
	}
	count := invocationArgumentCount(invocation)
	if method == "equals" && count != 1 || method == "hashCode" && count != 0 {
		return "", false
	}
	javaType, known := inferExprJavaType(object, ctx, source)
	if object.Type() == "identifier" {
		if declared, ok := identifierJavaTypeBeforeRepresentationRewrite(object.Content(source), ctx); ok {
			javaType, known = declared, true
		}
	}
	if !known {
		return "", false
	}
	protocol, canonical := reflectObjectReceiverProtocol(javaType, ctx)
	if !canonical {
		return "", false
	}
	// Intersection bounds can contribute a more specific source overload. Keep
	// that declaration-selected call on the ordinary source resolution path.
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: javaType}, ctx); bound {
		target := resolveInvocationTarget(object, ctx, source)
		selected, _ := findBestMethodForInvocationTarget(target, method, invocation.ChildByFieldName("arguments"), true, false, ctx, source)
		if selected != nil && selected.def != nil && selected.def.DeclarationNode != nil {
			if method == "equals" && !sourceObjectEqualsOverride(selected.owner, selected.def, ctx) {
				return protocol, false
			}
			if method == "hashCode" && (selected.def.IsStatic || selected.def.IsPrivate || selected.def.Constructor || len(selected.def.Parameters) != 0 || strings.TrimSpace(selected.def.OriginalType) != "int") {
				return protocol, false
			}
		}
	}
	return protocol, true
}

package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// Java Map methods return erased values. A read checkcast belongs to the actual
// consumer after the operation, including a put/remove that has already mutated.
func init() {
	for _, owner := range []string{"java.util.Set", "java.util.HashSet", "java.util.LinkedHashSet", "java.util.TreeSet"} {
		registerIntrinsicOwner(owner, true)
	}

	for _, owner := range mapTypeNames {
		// Source AbstractMap defaults own their erased protocol dispatch separately.
		if owner == "AbstractMap" {
			continue
		}
		for _, operation := range []struct {
			java, runtime string
			arity         int
		}{
			{"get", "GetObject", 1}, {"getOrDefault", "GetOrDefaultObject", 2},
			{"put", "PutObject", 2}, {"putIfAbsent", "PutIfAbsentObject", 2},
			{"remove", "RemoveObject", 1},
		} {
			// Canonical Map.put owns the shared native/source operation protocol.
			if owner == "Map" && operation.java == "put" {
				continue
			}
			registerInstanceNodeIntrinsic(owner, operation.java, func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
				if invocationArgumentCount(invocation) != operation.arity {
					return nil
				}
				args := intrinsicArgs(invocation.ChildByFieldName("object"), operation.java, source, ctx)
				raw := methodCall(recv, operation.runtime, append(args, intrinsicExecutionExpr(ctx))...)
				if collectionResultIsErased(invocation, ctx, source) {
					return raw
				}
				elements := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
				if len(elements) != 2 {
					return raw
				}
				element := readableWildcardProjection(elements[1])
				physical := javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)
				descriptor, known := javaTypeDescriptorExpr(element, ctx)
				if !known {
					descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
				}
				return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{raw, descriptor})
			})
		}
	}
}

// Native sets and map key/entry views already expose the live erased iterator.
// Recognize the JDK owner without treating source names or binders as runtime sets.
func nativeSetIterationExpression(node *sitter.Node, ctx Ctx, source []byte) bool {
	typ, known := inferExprJavaType(node, ctx, source)
	if !known {
		return false
	}
	base, _ := parseJavaTypeString(typ)
	if !strings.Contains(base, ".") {
		if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); bound {
			return false
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return false
	}
	switch owner {
	case "java.util.Set", "java.util.HashSet", "java.util.LinkedHashSet", "java.util.TreeSet":
		return true
	}
	return false
}

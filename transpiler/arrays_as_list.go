package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

// Arrays.asList first participates in fixed-arity applicability: a reference
// array supplies T[], whereas a primitive array is a single reference element.
func arraysAsListElementType(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	element := ""
	if explicit := invocation.ChildByFieldName("type_arguments"); explicit != nil {
		args := nodeutil.NamedChildrenOf(explicit)
		if len(args) == 1 {
			element = args[0].Content(source)
		}
	}
	if element == "" {
		_, args := parseJavaTypeString(ctx.expectedType)
		if len(args) == 1 && expectedTypeTargetsExpression(ctx, invocation) {
			element = args[0]
		}
	}
	if invocationArgumentCount(invocation) == 1 {
		actual, _ := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, source)
		if component, _, reference := reifiedSourceReferenceArrayComponent(actual, ctx); reference {
			if element == "" {
				element = component
			}
			if javaInferenceTypeAssignable(component, element, ctx) {
				return element, true
			}
		}
		if actual == "null" {
			if element == "" {
				element = "Object"
			}
			return element, true
		}
	}
	if element == "" {
		element = intrinsicFactoryElementJavaType(invocation, ctx, source)
	}
	return element, false
}

func arraysAsListCall(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	element, array := arraysAsListElementType(invocation, ctx, source)
	types := []ast.Expr{javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)}
	if array {
		descriptor, ok := javaTypeDescriptorExpr(element, ctx)
		if !ok {
			return nil
		}
		return stdjavaGenericCall(ctx, "AsListArray", types, []ast.Expr{args[0], descriptor})
	}
	return stdjavaGenericCall(ctx, "AsList", types, args)
}

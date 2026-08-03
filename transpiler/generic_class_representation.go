package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// genericClassRepresentationPlan keeps Java's source type view separate from
// the type arguments used to instantiate a generated Go class value.
//
// sourceArguments is the complete Java view, including type parameters carried
// by a non-static member class. It remains available to plan the narrowing casts
// required at field and method use sites. runtimeArguments is canonical for the
// class declaration: every slot is that declaration's transitive first-bound
// erasure, regardless of whether the source reference was raw or parameterized.
// Consequently Box<First> and raw Box can share one generated Go object type
// without duplicating storage or changing reference identity.
type genericClassRepresentationPlan struct {
	sourceArguments  []string
	runtimeArguments []string
}

// planGenericClassRepresentation returns the source and runtime views of one
// generated Java class instantiation. sourceTypeArguments uses the same source
// arity accepted by normalizeClassTypeArguments: member-class arguments may omit
// leading enclosing-instance slots, which are recovered from the receiver view.
//
// This planner intentionally operates on Java type spellings. The eventual Go
// type converter remains responsible for qualifying a bound and for recursively
// instantiating an erased bound that is itself a generated generic class.
// Java intersection erasure retains only the first bound here; operations from
// later bounds still require source-view bridges. A parameterized F-bound such
// as Comparable<T> similarly plans Comparable, while a malformed bare-variable
// cycle conservatively falls back to Object in rawTypeParameterErasure.
//
// Raw-versus-diamond classification remains a syntax/call-site concern. This
// helper does not infer it from an empty sourceTypeArguments slice; it only
// guarantees that once a class reference is planned, its runtime representation
// is independent of the source arguments.
func planGenericClassRepresentation(
	scope *symbol.ClassScope,
	sourceTypeArguments []string,
	receiverScope *symbol.ClassScope,
	receiverTypeArguments []string,
) genericClassRepresentationPlan {
	plan := genericClassRepresentationPlan{
		sourceArguments: normalizeClassTypeArguments(
			scope,
			sourceTypeArguments,
			receiverScope,
			receiverTypeArguments,
		),
	}
	if scope == nil || len(scope.TypeParameters) == 0 {
		return plan
	}

	plan.runtimeArguments = make([]string, 0, len(scope.TypeParameters))
	for _, parameter := range scope.TypeParameters {
		plan.runtimeArguments = append(
			plan.runtimeArguments,
			rawTypeParameterErasure(parameter, scope.TypeParameters),
		)
	}
	return plan
}

// uncheckedRawToParameterizedSameGeneratedClassCast reports the Java cast
// whose source is a raw view of the exact generated class named by the
// parameterized target, for example Box -> Box<First>.
//
// Java does not reify the target's type arguments. Because the operand's static
// raw type already proves the erased Box check, this cast performs no runtime
// operation. Preserving the operand expression is also essential while a raw
// alias may carry any existing Go instantiation of Box; asserting it to either
// Box<First> or Box<Erasure> would incorrectly make Java aliasing depend on
// Go's invariant generic types.
func uncheckedRawToParameterizedSameGeneratedClassCast(
	sourceJavaType string,
	targetJavaType string,
	ctx Ctx,
) bool {
	sourceComponent, sourceRank := javaArrayTypeParts(sourceJavaType)
	targetComponent, targetRank := javaArrayTypeParts(targetJavaType)
	if sourceRank != 0 || targetRank != 0 {
		return false
	}

	sourceBase, sourceArguments := parseJavaTypeString(sourceComponent)
	targetBase, targetArguments := parseJavaTypeString(targetComponent)
	if len(sourceArguments) != 0 || len(targetArguments) == 0 {
		return false
	}
	// Member-class qualification is not yet identity-safe in the general class
	// resolver when two enclosing types declare the same nested simple name. A
	// false positive would erase a real class check, so keep this optimization to
	// unqualified names until that resolver is strengthened.
	if strings.Contains(sourceBase, ".") || strings.Contains(targetBase, ".") {
		return false
	}
	if !sameJavaRawType(sourceBase, targetBase) {
		return false
	}

	sourceScope := resolveClassScopeByQualifiedName(ctx, sourceBase)
	targetScope := resolveClassScopeByQualifiedName(ctx, targetBase)
	return sourceScope != nil && sourceScope == targetScope && len(sourceScope.TypeParameters) > 0
}

// rawGenericCastCanPreserveLocalRepresentation identifies an initializer that
// Go lowers with :=. Returning the operand there preserves its exact existing
// instantiation while Java source metadata still records the parameterized
// local view. Explicitly typed fields, returns, and arguments need the broader
// canonical-representation migration before they can safely do the same.
func rawGenericCastCanPreserveLocalRepresentation(node *sitter.Node) bool {
	if node == nil || node.Type() != "cast_expression" {
		return false
	}
	declarator := node.Parent()
	if declarator == nil || declarator.Type() != "variable_declarator" ||
		!sameSourceNode(node, declarator.ChildByFieldName("value")) {
		return false
	}
	declaration := declarator.Parent()
	return declaration != nil && declaration.Type() == "local_variable_declaration"
}

func castOperandSourceJavaType(node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil {
		return "", false
	}
	if node.Type() == "identifier" {
		if javaType, ok := identifierJavaTypeBeforeRepresentationRewrite(node.Content(source), ctx); ok {
			return javaType, true
		}
	}
	return inferExprJavaType(node, ctx, source)
}

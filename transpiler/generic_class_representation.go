package transpiler

import "github.com/NickyBoy89/java2go/symbol"

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

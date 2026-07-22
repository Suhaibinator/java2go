package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// normalizeClassTypeArguments converts Java's source-level type arguments into
// the complete generated Go ABI for a class. A member class declares only its
// own trailing parameters in source, but TypeParameters also contains the
// leading parameters carried from its enclosing instance.
//
// receiverTypeArguments is nil when the receiver is the current generic
// declaration (its parameter names are therefore the actual arguments). A
// non-nil slice supplies a concrete receiver view, as in
// outer<String>.new Child<Impl>().
func normalizeClassTypeArguments(
	scope *symbol.ClassScope,
	sourceTypeArguments []string,
	receiverScope *symbol.ClassScope,
	receiverTypeArguments []string,
) []string {
	if scope == nil || len(scope.TypeParameters) == 0 {
		return append([]string(nil), sourceTypeArguments...)
	}
	if len(sourceTypeArguments) == len(scope.TypeParameters) {
		return append([]string(nil), sourceTypeArguments...)
	}

	declared := scope.OwnTypeParameters()
	hiddenCount := len(scope.TypeParameters) - len(declared)
	if hiddenCount < 0 {
		hiddenCount = 0
	}

	available := receiverClassTypeArgumentBindings(receiverScope, receiverTypeArguments)
	result := make([]string, 0, len(scope.TypeParameters))

	// A partially-qualified member type may already supply some enclosing
	// arguments. Arguments beyond the class's declared arity belong to leading
	// carried slots; remaining hidden slots come from the receiver view.
	providedHidden := len(sourceTypeArguments) - len(declared)
	if providedHidden < 0 {
		providedHidden = 0
	}
	if providedHidden > hiddenCount {
		providedHidden = hiddenCount
	}
	for index := 0; index < hiddenCount; index++ {
		parameter := scope.TypeParameters[index]
		if index < providedHidden {
			result = append(result, sourceTypeArguments[index])
			continue
		}
		if argument := available[parameter.Name]; strings.TrimSpace(argument) != "" {
			result = append(result, argument)
			continue
		}
		result = append(result, rawTypeParameterErasure(parameter, scope.TypeParameters))
	}

	declaredOffset := providedHidden
	for index, parameter := range declared {
		argumentIndex := declaredOffset + index
		if argumentIndex < len(sourceTypeArguments) {
			result = append(result, sourceTypeArguments[argumentIndex])
			continue
		}
		result = append(result, rawTypeParameterErasure(parameter, scope.TypeParameters))
	}

	// Synthetic scopes can expose an older, non-prefix parameter layout. Keep
	// their ABI valid by erasing any slots not described by the declared view.
	for len(result) < len(scope.TypeParameters) {
		result = append(result, rawTypeParameterErasure(scope.TypeParameters[len(result)], scope.TypeParameters))
	}
	return result
}

func receiverClassTypeArgumentBindings(scope *symbol.ClassScope, arguments []string) map[string]string {
	if scope == nil {
		return nil
	}
	actual := arguments
	if actual == nil {
		actual = scope.TypeParameterNames()
	}
	bindings := make(map[string]string, len(scope.TypeParameters))
	for index, parameter := range scope.TypeParameters {
		if index < len(actual) && strings.TrimSpace(actual[index]) != "" {
			bindings[parameter.Name] = actual[index]
		}
	}
	return bindings
}

// rawTypeParameterErasure implements Java's first-bound erasure for a missing
// source argument. A parameter-to-parameter bound is followed transitively;
// an unbounded or cyclic parameter erases to Object.
func rawTypeParameterErasure(parameter symbol.TypeParam, visible []symbol.TypeParam) string {
	byName := make(map[string]symbol.TypeParam, len(visible))
	for _, candidate := range visible {
		byName[candidate.Name] = candidate
	}
	visiting := map[string]bool{}
	var erase func(symbol.TypeParam) string
	erase = func(current symbol.TypeParam) string {
		if visiting[current.Name] || len(current.Bounds) == 0 {
			return "Object"
		}
		visiting[current.Name] = true
		defer delete(visiting, current.Name)

		bound := strings.TrimSpace(current.Bounds[0].Original)
		if bound == "" {
			return "Object"
		}
		base, _ := parseJavaTypeString(bound)
		if next, ok := byName[stripJavaQualifier(base)]; ok {
			return erase(next)
		}
		// Java erasure drops the bound's own type arguments. The downstream type
		// converter will instantiate a generated generic raw class with its own
		// erased arguments when Go requires them.
		if strings.TrimSpace(base) != "" {
			return base
		}
		return "Object"
	}
	return erase(parameter)
}

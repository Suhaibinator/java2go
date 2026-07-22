package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// typeParameterLookup resolves Java source references without collapsing
// distinct same-named declarations. The source-name map is retained only as a
// compatibility fallback for older synthetic TypeParam values that do not yet
// carry declaration identity.
type typeParameterLookup struct {
	byDeclaration map[*symbol.TypeParamDeclaration]symbol.TypeParam
	byName        map[string]symbol.TypeParam
}

func newTypeParameterLookup(parameters []symbol.TypeParam) typeParameterLookup {
	lookup := typeParameterLookup{
		byDeclaration: make(map[*symbol.TypeParamDeclaration]symbol.TypeParam, len(parameters)),
		byName:        make(map[string]symbol.TypeParam, len(parameters)*2),
	}
	for _, parameter := range parameters {
		if parameter.Declaration != nil {
			lookup.byDeclaration[parameter.Declaration] = parameter
		}
		lookup.byName[parameter.Name] = parameter
		lookup.byName[parameter.EmittedName()] = parameter
	}
	return lookup
}

func (lookup typeParameterLookup) resolve(javaType symbol.JavaType, name string) (symbol.TypeParam, bool) {
	if declaration := javaType.TypeParameterBindings[name]; declaration != nil {
		parameter, found := lookup.byDeclaration[declaration]
		return parameter, found
	}
	parameter, found := lookup.byName[name]
	return parameter, found
}

type typeParameterIdentityKey struct {
	declaration *symbol.TypeParamDeclaration
	legacyName  string
}

func identityKeyForTypeParameter(parameter symbol.TypeParam) typeParameterIdentityKey {
	if parameter.Declaration != nil {
		return typeParameterIdentityKey{declaration: parameter.Declaration}
	}
	return typeParameterIdentityKey{legacyName: parameter.Name}
}

type receiverTypeArgumentBindings struct {
	byDeclaration  map[*symbol.TypeParamDeclaration]string
	byUniqueName   map[string]string
	ambiguousNames map[string]struct{}
}

func (bindings receiverTypeArgumentBindings) argumentFor(parameter symbol.TypeParam) (string, bool) {
	if parameter.Declaration != nil {
		if argument, found := bindings.byDeclaration[parameter.Declaration]; found {
			return argument, true
		}
	}
	if _, ambiguous := bindings.ambiguousNames[parameter.Name]; ambiguous {
		return "", false
	}
	argument, found := bindings.byUniqueName[parameter.Name]
	return argument, found
}

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
		if argument, found := available.argumentFor(parameter); found && strings.TrimSpace(argument) != "" {
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

func receiverClassTypeArgumentBindings(scope *symbol.ClassScope, arguments []string) receiverTypeArgumentBindings {
	if scope == nil {
		return receiverTypeArgumentBindings{}
	}
	actual := arguments
	if actual == nil {
		actual = scope.GoTypeParameterNames()
	}
	bindings := receiverTypeArgumentBindings{
		byDeclaration:  make(map[*symbol.TypeParamDeclaration]string, len(scope.TypeParameters)),
		byUniqueName:   make(map[string]string, len(scope.TypeParameters)),
		ambiguousNames: make(map[string]struct{}),
	}
	for index, parameter := range scope.TypeParameters {
		if index >= len(actual) || strings.TrimSpace(actual[index]) == "" {
			continue
		}
		argument := actual[index]
		if parameter.Declaration != nil {
			bindings.byDeclaration[parameter.Declaration] = argument
		}
		if _, ambiguous := bindings.ambiguousNames[parameter.Name]; ambiguous {
			continue
		}
		if _, duplicate := bindings.byUniqueName[parameter.Name]; duplicate {
			delete(bindings.byUniqueName, parameter.Name)
			bindings.ambiguousNames[parameter.Name] = struct{}{}
			continue
		}
		bindings.byUniqueName[parameter.Name] = argument
	}
	return bindings
}

// rawTypeParameterErasure implements Java's first-bound erasure for a missing
// source argument. A parameter-to-parameter bound is followed transitively;
// an unbounded or cyclic parameter erases to Object.
func rawTypeParameterErasure(parameter symbol.TypeParam, visible []symbol.TypeParam) string {
	lookup := newTypeParameterLookup(visible)
	visiting := map[typeParameterIdentityKey]bool{}
	var erase func(symbol.TypeParam) string
	erase = func(current symbol.TypeParam) string {
		identity := identityKeyForTypeParameter(current)
		if visiting[identity] || len(current.Bounds) == 0 {
			return "Object"
		}
		visiting[identity] = true
		defer delete(visiting, identity)

		bound := strings.TrimSpace(current.Bounds[0].Original)
		if bound == "" {
			return "Object"
		}
		base, arguments := parseJavaTypeString(bound)
		if next, ok := lookup.resolve(current.Bounds[0], strings.TrimSpace(base)); ok && len(arguments) == 0 {
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

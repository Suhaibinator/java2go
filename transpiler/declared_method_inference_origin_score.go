package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// An eligible generic argument conversion still carries its method inference
// declarations. Only a complete, unique actual origin can turn that deferred
// conversion into the same exact conversion used for a nongeneric formal.
func declaredMethodArgumentInferredOrigin(formal inferredJavaTypeOrigin, method *symbol.Definition, formals []inferredJavaTypeOrigin, arguments []*sitter.Node, fixedArray bool, ctx Ctx, source []byte) (inferredJavaTypeOrigin, bool) {
	if method == nil || len(method.TypeParameters) == 0 || len(arguments) == 0 {
		return inferredJavaTypeOrigin{}, false
	}
	parameters := make(map[*symbol.TypeParamDeclaration]bool, len(method.TypeParameters))
	for _, parameter := range method.TypeParameters {
		if parameter.Declaration == nil {
			return inferredJavaTypeOrigin{}, false
		}
		parameters[parameter.Declaration] = true
	}
	candidates := map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
	inference := ctx.Clone()
	inference.expectedType = ""
	inference.expectedTypeRoot = nil
	for index, argument := range arguments {
		parameterIndex := index
		last := len(formals) - 1
		if last >= 0 && executionParameterIsVariadic(method, last) && parameterIndex > last {
			parameterIndex = last
		}
		if parameterIndex < 0 || parameterIndex >= len(formals) {
			return inferredJavaTypeOrigin{}, false
		}
		var actual inferredJavaTypeOrigin
		typ, known := inferExprJavaType(argument, inference, source, &actual)
		actual.javaType = typ
		if !known || !declaredMethodExactInferenceOrigin(actual) {
			continue
		}
		parameter := formals[parameterIndex]
		if fixedArray && parameterIndex == last {
			parameter.arrayRank++
		}
		if declaredMethodInferenceHasWildcard(parameter) {
			continue
		}
		gatherMethodTypeOriginBindings(parameter, actual, parameters, candidates)
	}
	substitutions := make(map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin, len(parameters))
	invocation := arguments[0].Parent()
	if invocation != nil && invocation.Type() == "argument_list" {
		invocation = invocation.Parent()
	} else {
		invocation = nil
	}
	var explicit []*sitter.Node
	if invocation != nil && invocation.Type() == "method_invocation" {
		if syntax := invocation.ChildByFieldName("type_arguments"); syntax != nil {
			explicit = nodeutil.NamedChildrenOf(syntax)
			if len(explicit) != len(method.TypeParameters) {
				return inferredJavaTypeOrigin{}, false
			}
		}
	}
	for index, parameter := range method.TypeParameters {
		values := candidates[parameter.Declaration]
		if len(values) == 0 {
			continue
		}
		selected := values[0]
		if !declaredMethodExactInferenceOrigin(selected) {
			continue
		}
		for _, value := range values[1:] {
			if !sameJavaTypeOrigin(selected, value) {
				return inferredJavaTypeOrigin{}, false
			}
		}
		if len(explicit) > 0 {
			supplied := declaredJavaTypeOrigin(symbol.JavaType{Original: explicit[index].Content(source), TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(inference))}, inference)
			if !declaredMethodExactInferenceOrigin(supplied) || !sameJavaTypeOrigin(selected, supplied) {
				return inferredJavaTypeOrigin{}, false
			}
			selected = supplied
		}
		substitutions[parameter.Declaration] = selected
	}
	projected := substituteJavaTypeOrigin(formal, substitutions)
	return projected, declaredMethodExactInferenceOrigin(projected)
}

func declaredMethodExactInferenceOrigin(origin inferredJavaTypeOrigin) bool {
	if origin.unresolved || origin.wildcard || !completeJavaTypeOrigin(origin) || origin.javaType == "null" || origin.javaType == ternaryNullJavaType {
		return false
	}
	for _, argument := range origin.arguments {
		if !declaredMethodExactInferenceOrigin(argument) {
			return false
		}
	}
	return true
}

func declaredMethodInferenceHasWildcard(origin inferredJavaTypeOrigin) bool {
	if origin.wildcard {
		return true
	}
	for _, argument := range origin.arguments {
		if declaredMethodInferenceHasWildcard(argument) {
			return true
		}
	}
	return false
}

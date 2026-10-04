package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

// A method binder and a caller's nominal type may have the same Java spelling.
// Resolve each side in its own declaration context before substitution; the
// resulting spelling alone cannot tell an inferred argument from a free binder.
func inferredGenericInvocationResult(def *symbol.Definition, node *sitter.Node, ctx Ctx, source []byte) (string, bool) {
	declaring, _ := invocationMethodDeclarationContext(def, ctx)
	declaring.localScope = def
	declaring.syntheticTypeParameters = nil
	returnType := symbol.JavaType{Original: def.OriginalType, TypeParameterBindings: def.TypeParameterBindings}
	if returnType.TypeParameterBindings == nil && def.DirectTypeParameter != nil {
		returnType.TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{def.DirectTypeParameter.SourceName: def.DirectTypeParameter}
	}
	result := qualifyDeclaredReferenceType(returnType, declaring)
	bindings := genericArrayInvocationTypeBindings(def, node, ctx, source)
	qualified := make(map[string]string, len(bindings)*2)
	for _, parameter := range def.TypeParameters {
		actual, found := bindings[parameter.Name]
		if !found || strings.TrimSpace(actual) == "" {
			continue
		}
		// Live caller binders retain their declaration's emitted name; concrete
		// references use imports/source ownership at the invocation site.
		actual = qualifyDeclaredReferenceType(symbol.JavaType{Original: actual}, ctx)
		qualified[parameter.Name] = actual
		qualified[parameter.EmittedName()] = actual
	}
	base, _ := parseJavaTypeString(result)
	for _, parameter := range def.TypeParameters {
		if base == parameter.Name || base == parameter.EmittedName() {
			if _, resolved := qualified[parameter.Name]; !resolved {
				return "", false
			}
		}
	}
	return substituteJavaTypeParameters(result, qualified), true
}

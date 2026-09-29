package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Source declarations can extend or implement runtime-owned nominal types,
// which have no source ClassScope. Follow the declared ancestry before testing
// an external bound; method shape or the object's dynamic type is not evidence
// that its static Java type satisfies that bound.
func sourceRuntimeReferenceAssignable(actual *symbol.ClassScope, arguments []string, expected string, ctx Ctx) bool {
	expected = qualifyDeclaredReferenceType(symbol.JavaType{Original: expected}, ctx)
	arguments = append([]string(nil), arguments...)
	for index, argument := range arguments {
		arguments[index] = qualifyDeclaredReferenceType(symbol.JavaType{Original: argument}, ctx)
	}
	type ancestryView struct {
		scope     *symbol.ClassScope
		arguments []string
	}
	queue := []ancestryView{{scope: actual, arguments: arguments}}
	seen := map[*symbol.ClassScope]bool{}
	for len(queue) > 0 {
		view := queue[0]
		queue = queue[1:]
		if view.scope == nil || seen[view.scope] {
			continue
		}
		seen[view.scope] = true
		declaring := classScopeCtx(view.scope, ctx)
		declaring.syntheticTypeParameters = nil
		actualArguments := normalizeClassTypeArguments(view.scope, view.arguments, ctx.currentClass, nil)
		bindings := make(map[string]string, len(actualArguments)*2)
		for index, parameter := range view.scope.TypeParameters {
			if index < len(actualArguments) {
				bindings[parameter.Name] = actualArguments[index]
				bindings[parameter.EmittedName()] = actualArguments[index]
			}
		}
		parents := append([]string{view.scope.Superclass}, view.scope.ImplementedInterfaces...)
		for _, parent := range parents {
			if strings.TrimSpace(parent) == "" {
				continue
			}
			parent = qualifyDeclaredReferenceType(symbol.JavaType{Original: parent}, declaring)
			parent = substituteJavaTypeParameters(parent, bindings)
			base, parentArguments := parseJavaTypeString(parent)
			if scope := resolveClassScopeByQualifiedName(declaring, base); scope != nil {
				queue = append(queue, ancestryView{scope: scope, arguments: parentArguments})
				continue
			}
			if javaInferenceTypeAssignable(parent, expected, ctx) {
				return true
			}
		}
	}
	return false
}

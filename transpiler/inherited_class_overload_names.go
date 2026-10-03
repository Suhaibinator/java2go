package transpiler

import (
	"encoding/hex"
	"sort"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// resolveInheritedClassOverloadNames gives distinct Java overload families
// distinct Go selectors. A direct method hides every promoted Go method with
// that selector, even when Java inherits those other parameter signatures.
// Group genuine overrides before allocating names so covariance, interface
// dispatch and declaration order cannot split one virtual method family.
func resolveInheritedClassOverloadNames() {
	scopes := allSourceClassScopes()
	owners := map[*symbol.Definition]*symbol.ClassScope{}
	parent := map[*symbol.Definition]*symbol.Definition{}
	var methods []*symbol.Definition
	for _, scope := range scopes {
		if inheritedOverloadHasUnresolvedOuterBinding(scope) {
			continue
		}
		for _, method := range scope.Methods {
			if method == nil || method.IsStatic || method.Constructor || method.IsPrivate || len(method.TypeParameters) != 0 {
				continue
			}
			owners[method] = scope
			parent[method] = method
			methods = append(methods, method)
		}
	}
	var root func(*symbol.Definition) *symbol.Definition
	root = func(method *symbol.Definition) *symbol.Definition {
		if parent[method] != method {
			parent[method] = root(parent[method])
		}
		return parent[method]
	}
	ancestorSets := map[*symbol.ClassScope][]*symbol.ClassScope{}
	for _, scope := range scopes {
		ancestorSets[scope] = inheritedOverloadAncestors(scope)
		for _, ancestor := range ancestorSets[scope] {
			for _, method := range scope.Methods {
				if owners[method] == nil {
					continue
				}
				for _, inherited := range ancestor.Methods {
					if owners[inherited] == nil || !inheritedOverloadOverrides(scope, method, ancestor, inherited) {
						continue
					}
					parent[root(method)] = root(inherited)
				}
			}
		}
	}
	affected := map[*symbol.Definition]bool{}
	for _, scope := range scopes {
		visible := append([]*symbol.Definition(nil), scope.Methods...)
		for _, ancestor := range ancestorSets[scope] {
			visible = append(visible, ancestor.Methods...)
		}
		for index, left := range visible {
			if owners[left] == nil {
				continue
			}
			for _, right := range visible[index+1:] {
				if owners[right] == nil || owners[left] == owners[right] {
					// Ordinary resolution disambiguates a declaring class's own
					// overloads. Other files may not have reached that pass yet;
					// only cross-owner collisions can hide inherited selectors.
					continue
				}
				a, b := root(left), root(right)
				if a != b && left.Name == right.Name {
					affected[a] = true
					affected[b] = true
				}
			}
		}
	}
	groups := map[*symbol.Definition][]*symbol.Definition{}
	for _, method := range methods {
		r := root(method)
		if affected[r] {
			groups[r] = append(groups[r], method)
		}
	}
	for _, group := range groups {
		// Include the declaration owner so unrelated package-private methods cannot
		// acquire a shared selector merely because their formal types are identical.
		keys := make([]string, 0, len(group))
		for _, method := range group {
			owner := owners[method]
			keys = append(keys, javaClassBinaryName(owner)+";"+interfaceOverloadSignature(owner, method))
		}
		sort.Strings(keys)
		name := "Java2goInheritedOverload_" + hex.EncodeToString([]byte(keys[0]))
		for inheritedOverloadNameOccupied(name, scopes) {
			name += "_"
		}
		for _, method := range group {
			method.Name = name
		}
	}
}

func inheritedOverloadAncestors(scope *symbol.ClassScope) []*symbol.ClassScope {
	var result []*symbol.ClassScope
	seen := map[*symbol.ClassScope]bool{scope: true}
	queue := []*symbol.ClassScope{scope}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		ctx := classScopeCtx(current, Ctx{})
		parents := append([]*symbol.ClassScope{resolveSuperclassScopeInDeclaringContext(ctx, current)}, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		for _, parent := range parents {
			if parent == nil || seen[parent] {
				continue
			}
			seen[parent] = true
			result = append(result, parent)
			queue = append(queue, parent)
		}
	}
	return result
}

func inheritedOverloadOverrides(owner *symbol.ClassScope, method *symbol.Definition, ancestor *symbol.ClassScope, inherited *symbol.Definition) bool {
	if method.OriginalName != inherited.OriginalName || len(method.Parameters) != len(inherited.Parameters) {
		return false
	}
	if directOwnerOverrideBridgeVisibility(ancestor, inherited) == directOwnerOverrideBridgePackagePrivate && !directOwnerOverrideBridgeSamePackage(owner, ancestor) {
		return false
	}
	ctx := classScopeCtx(owner, Ctx{})
	args := mapClassTypeArgumentStringsToAncestor(owner, owner.GoTypeParameterNames(), ancestor, ctx)
	if len(args) != len(ancestor.TypeParameters) {
		return false
	}
	bindings := map[string]string{}
	for index, parameter := range ancestor.TypeParameters {
		argument := qualifyJavaTypeInDeclaringContext(args[index], owner)
		bindings[parameter.Name] = argument
		bindings[parameter.EmittedName()] = argument
	}
	for index := range method.Parameters {
		formal := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(inherited, index), ancestor)
		formal = substituteJavaTypeParameters(formal, bindings)
		actual := definitionParameterJavaSignatureType(method, index)
		if !overrideBridgeJavaTypesIdentical(formal, owner, actual, owner, ctx) {
			return false
		}
	}
	return true
}

func inheritedOverloadNameOccupied(name string, scopes []*symbol.ClassScope) bool {
	for _, scope := range scopes {
		if scope.Class != nil && (scope.Class.Name == name || scope.Class.OriginalName == name) {
			return true
		}
		for _, field := range scope.Fields {
			if field != nil && (field.Name == name || field.OriginalName == name) {
				return true
			}
		}
		for _, method := range scope.Methods {
			if method != nil && (method.OriginalName == name || strings.HasPrefix(method.HelperName, name)) {
				return true
			}
		}
	}
	return false
}

// A member-class superclass can carry enclosing type arguments not spelled in
// its extends clause. Until the ancestor mapper resolves that enclosing view,
// its fallback bound is not evidence that two Java signatures differ. Preserve
// the existing bridge gate for the whole affected hierarchy instead of turning
// a possible override into a newly named overload.
func inheritedOverloadHasUnresolvedOuterBinding(scope *symbol.ClassScope) bool {
	for _, candidate := range append([]*symbol.ClassScope{scope}, inheritedOverloadAncestors(scope)...) {
		if candidate.IsInner && len(candidate.TypeParameters) != len(candidate.OwnTypeParameters()) {
			return true
		}
	}
	return false
}

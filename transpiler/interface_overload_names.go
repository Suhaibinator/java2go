package transpiler

import (
	"encoding/hex"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// A Java interface may overload an inherited method. Go interface embedding
// requires a distinct name for each signature, and implementing classes must
// use those same names regardless of their declaration order or extra overloads.
func resolveInheritedInterfaceOverloadNames() {
	scopes := allSourceClassScopes()
	affected := map[string]map[*symbol.ClassScope]bool{}
	for _, scope := range scopes {
		if !scope.IsInterface || len(scope.TypeParameters) != 0 {
			continue
		}
		ancestors := transitiveImplementedInterfaceScopes(scope, classScopeCtx(scope, Ctx{}))
		owners := append([]*symbol.ClassScope{scope}, ancestors...)
		for _, owner := range owners {
			if len(owner.TypeParameters) != 0 {
				continue
			}
			for _, method := range owner.Methods {
				if method.IsStatic || method.IsPrivate || method.Constructor || len(method.TypeParameters) != 0 {
					continue
				}
				for _, otherOwner := range owners {
					if otherOwner == owner || len(otherOwner.TypeParameters) != 0 {
						continue
					}
					for _, other := range otherOwner.Methods {
						if other.IsStatic || other.IsPrivate || other.Constructor || len(other.TypeParameters) != 0 || other.OriginalName != method.OriginalName {
							continue
						}
						if interfaceOverloadSignature(owner, method) == interfaceOverloadSignature(otherOwner, other) {
							continue
						}
						if affected[method.OriginalName] == nil {
							affected[method.OriginalName] = map[*symbol.ClassScope]bool{}
						}
						affected[method.OriginalName][owner] = true
						affected[method.OriginalName][otherOwner] = true
					}
				}
			}
		}
	}
	if len(affected) == 0 {
		return
	}
	for _, scope := range scopes {
		ancestors := append([]*symbol.ClassScope{scope}, transitiveImplementedInterfaceScopes(scope, classScopeCtx(scope, Ctx{}))...)
		// A class can inherit its interface implementation through a superclass.
		for parent := resolveSuperclassScopeInDeclaringContext(classScopeCtx(scope, Ctx{}), scope); parent != nil; parent = resolveSuperclassScopeInDeclaringContext(classScopeCtx(parent, Ctx{}), parent) {
			ancestors = append(ancestors, parent)
			ancestors = append(ancestors, transitiveImplementedInterfaceScopes(parent, classScopeCtx(parent, Ctx{}))...)
		}
		for _, method := range scope.Methods {
			if method.IsStatic || method.IsPrivate || method.Constructor || len(method.TypeParameters) != 0 {
				continue
			}
			related := false
			for _, ancestor := range ancestors {
				if affected[method.OriginalName][ancestor] {
					related = true
					break
				}
			}
			if !related {
				continue
			}
			signature := interfaceOverloadSignature(scope, method)
			name := "Java2goOverload_" + hex.EncodeToString([]byte(signature))
			// Do not collide with a Java identifier intentionally resembling the helper.
			for {
				collision := false
				for _, otherScope := range scopes {
					for _, other := range otherScope.Methods {
						if other.OriginalName == name {
							collision = true
						}
					}
					for _, field := range otherScope.Fields {
						if field.Name == name {
							collision = true
						}
					}
				}
				if !collision {
					break
				}
				name += "_"
			}
			method.Name = name
		}
	}
}

func interfaceOverloadSignature(owner *symbol.ClassScope, method *symbol.Definition) string {
	parts := []string{method.OriginalName}
	for index := range method.Parameters {
		parts = append(parts, qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, index), owner))
	}
	return strings.Join(parts, ";")
}

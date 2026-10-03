package transpiler

import (
	"go/ast"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
)

// javac can declare a bridge in a class that declares no source method. An
// inherited generic implementation may satisfy a newly implemented interface
// in Java while its erased JVM descriptor differs from the interface entry.
// Keep that Java bridge distinct from both the inherited source declaration
// and any Go execution selectors used to execute its body.
func sourceReflectionInheritedInterfaceBridges(scope *symbol.ClassScope, ctx Ctx, existing []ast.Expr) []ast.Expr {
	if scope == nil || scope.IsInterface {
		return nil
	}
	superclass := resolveSuperclassScopeInDeclaringContext(ctx, scope)
	if superclass == nil {
		return nil
	}
	interfaces := resolveImplementedInterfaceScopesInDeclaringContext(ctx, scope)
	seenInterfaces := make(map[*symbol.ClassScope]bool)
	seenDescriptors := make(map[string]bool)
	for _, descriptor := range existing {
		if key, ok := sourceReflectionMethodDescriptorIdentity(descriptor); ok {
			seenDescriptors[key] = true
		}
	}
	var out []ast.Expr
	for len(interfaces) != 0 {
		iface := interfaces[0]
		interfaces = interfaces[1:]
		if iface == nil || seenInterfaces[iface] {
			continue
		}
		seenInterfaces[iface] = true
		interfaces = append(interfaces, resolveImplementedInterfaceScopesInDeclaringContext(ctx, iface)...)
		// An already implemented interface keeps its bridge in the superclass;
		// merely inheriting or redundantly naming it declares no new Java method.
		if sourceReflectionSuperclassImplements(superclass, iface, ctx) {
			continue
		}
		ifaceArguments := mapClassTypeArgumentStringsToAncestor(scope, scope.GoTypeParameterNames(), iface, ctx)
		for _, required := range iface.Methods {
			if !sourceReflectionDeclaredMethod(iface, required) || required.IsStatic || required.IsPrivate || len(required.TypeParameters) != 0 {
				continue
			}
			mappedParameters, mappedResult := sourceReflectionMappedDeclarationSignature(iface, scope, required, ifaceArguments, ctx)
			// A real source override is handled by sourceReflectionMethods, with
			// its own declaration and ordinary override bridges.
			declaredOverride := false
			for _, method := range scope.Methods {
				if sourceReflectionDeclaredMethod(scope, method) && sourceReflectionImplementationMatches(scope, method, scope, mappedParameters, mappedResult, required, ctx) {
					declaredOverride = true
					break
				}
			}
			if declaredOverride {
				continue
			}
			parameters, result, supported := sourceReflectionMethodSignature(iface, required, ctx)
			if !supported {
				continue
			}
			key := required.OriginalName + "\x00" + sourceReflectionDescriptorKey(parameters, result)
			if seenDescriptors[key] {
				continue
			}
			seenClasses := make(map[*symbol.ClassScope]bool)
			for implementation := superclass; implementation != nil && !seenClasses[implementation]; implementation = resolveSuperclassScopeInDeclaringContext(ctx, implementation) {
				seenClasses[implementation] = true
				args := mapClassTypeArgumentStringsToAncestor(scope, scope.GoTypeParameterNames(), implementation, ctx)
				matched := false
				for _, method := range implementation.Methods {
					if !sourceReflectionDeclaredMethod(implementation, method) || !sourceReflectionImplementationMatches(implementation, method, scope, mappedParameters, mappedResult, required, ctx, args...) {
						continue
					}
					matched = true
					physicalParameters, physicalResult, ok := sourceReflectionMethodSignature(implementation, method, ctx)
					if !ok || sourceReflectionDescriptorKey(physicalParameters, physicalResult) == sourceReflectionDescriptorKey(parameters, result) {
						break
					}
					descriptor := sourceReflectionMethodDescriptor(implementation, method, parameters, result, true, physicalParameters, ctx)
					seenDescriptors[key] = true
					out = append(out, descriptor)
					break
				}
				if matched {
					break
				}
			}
		}
	}
	return out
}

// Canonicalize concrete references in the source declaration before replacing
// class binders with the receiver's actual arguments. A same-named receiver
// import must not rebind an inherited method's parameter or result declaration.
func sourceReflectionMappedDeclarationSignature(owner, receiver *symbol.ClassScope, method *symbol.Definition, arguments []string, ctx Ctx) ([]string, string) {
	declaring := classScopeCtx(sourceReflectionJavaDeclarationScope(owner), ctx).Clone()
	declaring.localScope = method
	receiverContext := classScopeCtx(receiver, ctx)
	substitutions := make(map[string]string, len(owner.TypeParameters)*2)
	for index, parameter := range owner.TypeParameters {
		if index < len(arguments) {
			// The mapper supplies receiver-view arguments. Canonicalize those
			// bindings too, so implicit and qualified library names compare alike.
			argument := qualifyDeclaredReferenceType(symbol.JavaType{Original: arguments[index]}, receiverContext)
			substitutions[parameter.Name] = argument
			substitutions[parameter.EmittedName()] = argument
		}
	}
	qualify := func(original string, binders map[string]*symbol.TypeParamDeclaration) string {
		qualified := qualifyDeclaredReferenceType(symbol.JavaType{Original: original, TypeParameterBindings: binders}, declaring)
		return substituteJavaTypeParams(qualified, substitutions)
	}
	parameters := make([]string, len(method.Parameters))
	for index, parameter := range method.Parameters {
		parameters[index] = qualify(definitionParameterJavaSignatureType(method, index), parameter.TypeParameterBindings)
	}
	result := "void"
	if !javaMethodResultIsVoid(method) {
		result = qualify(method.OriginalType, method.TypeParameterBindings)
	}
	return parameters, result
}

func sourceReflectionImplementationMatches(declaring *symbol.ClassScope, method *symbol.Definition, receiver *symbol.ClassScope, parameters []string, result string, required *symbol.Definition, ctx Ctx, arguments ...string) bool {
	if method == nil || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper || len(method.TypeParameters) != 0 || method.OriginalName != required.OriginalName || len(method.Parameters) != len(parameters) || sourceReflectionMethodModifiers(declaring, method, false)&1 == 0 {
		return false
	}
	actualParameters, actualResult := sourceReflectionMappedDeclarationSignature(declaring, receiver, method, arguments, ctx)
	for index, parameter := range parameters {
		if !overrideBridgeJavaTypesIdentical(actualParameters[index], receiver, parameter, receiver, ctx) {
			return false
		}
	}
	return overrideBridgeResultCompatible(actualResult, receiver, result, receiver, ctx)
}

func sourceReflectionSuperclassImplements(scope, target *symbol.ClassScope, ctx Ctx) bool {
	queue := []*symbol.ClassScope{scope}
	seen := make(map[*symbol.ClassScope]bool)
	for len(queue) != 0 {
		current := queue[0]
		queue = queue[1:]
		if current == nil || seen[current] {
			continue
		}
		if current == target {
			return true
		}
		seen[current] = true
		queue = append(queue, resolveImplementedInterfaceScopesInDeclaringContext(ctx, current)...)
		queue = append(queue, resolveSuperclassScopeInDeclaringContext(ctx, current))
	}
	return false
}

func sourceReflectionMethodDescriptorIdentity(descriptor ast.Expr) (string, bool) {
	value, ok := descriptor.(*ast.CompositeLit)
	if !ok {
		return "", false
	}
	var name string
	var parameters []ast.Expr
	var result ast.Expr
	for _, field := range value.Elts {
		key, ok := field.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		identifier, ok := key.Key.(*ast.Ident)
		if !ok {
			continue
		}
		switch identifier.Name {
		case "Name":
			if text, ok := key.Value.(*ast.BasicLit); ok {
				name, _ = strconv.Unquote(text.Value)
			}
		case "ParameterTypes":
			if array, ok := key.Value.(*ast.CompositeLit); ok {
				parameters = array.Elts
			}
		case "Return":
			result = key.Value
		}
	}
	if name == "" || result == nil {
		return "", false
	}
	return name + "\x00" + sourceReflectionDescriptorKey(parameters, result), true
}

package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Reflection consumes Java declaration syntax, rather than the generated Go
// storage view. In particular, a canonical generic core still declares its
// original parameters, and a variable is a finite declaration reference.
func reflectiveTypeDescriptorExpr(javaType symbol.JavaType, ctx Ctx) (ast.Expr, bool) {
	expression, _, supported := reflectiveTypeDescriptorNode(javaType, ctx)
	return expression, supported
}

// The optional raw TypeID records a resolved Class node. It is not Java erasure
// and does not admit a new physical storage or family representation.
func reflectiveTypeDescriptorNode(javaType symbol.JavaType, ctx Ctx) (ast.Expr, ast.Expr, bool) {
	original := strings.TrimSpace(javaType.Original)
	if original == "" {
		return nil, nil, false
	}
	// Wildcard bounds may themselves end in array syntax. Resolve the prefix
	// first, so ? extends T[] remains a wildcard with an array upper bound.
	if strings.HasPrefix(original, "?") {
		return reflectiveWildcardDescriptorNode(javaType, ctx)
	}
	base, rank := javaArrayTypeParts(original)
	if rank != 0 {
		if base == "void" {
			return nil, nil, false
		}
		component, raw, supported := reflectiveTypeDescriptorNode(symbol.JavaType{
			Original: base + strings.Repeat("[]", rank-1), TypeParameterBindings: javaType.TypeParameterBindings,
		}, ctx)
		if !supported {
			return nil, nil, false
		}
		// JDK reflection returns an array Class only when its resolved component
		// is a Class. A parameterized List<?> component remains a generic array.
		if raw != nil {
			arrayRaw := stdjavaCall(ctx, "ArrayTypeID", raw)
			return reflectiveDescriptorLiteral("ReflectClassKind", ctx, metadataKey("Raw", arrayRaw)), arrayRaw, true
		}
		return reflectiveDescriptorLiteral("ReflectGenericArrayKind", ctx,
			metadataKey("Component", &ast.UnaryExpr{Op: token.AND, X: component})), nil, true
	}
	// This API receives source declaration syntax. Captured source bindings are
	// authoritative even when a generated Go binder has the same spelling as
	// a nominal import. Uncaptured syntax likewise sees source names only.
	bindings := javaType.TypeParameterBindings
	if bindings == nil {
		bindings = symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	}
	if declaration := bindings[base]; declaration != nil {
		if !reflectiveClassVariableAccessible(declaration, ctx) {
			return nil, nil, false
		}
		binding, found := resolveReferenceTypeParameter(symbol.JavaType{
			Original: original, TypeParameterBindings: bindings,
		}, ctx)
		if !found {
			return nil, nil, false
		}
		owner := reflectiveClassVariableOwner(binding)
		if owner == nil {
			// The agreed ABI has class declaration keys only. A method or
			// constructor variable must never acquire its enclosing class's key.
			return nil, nil, false
		}
		name := binding.parameter.Name
		if declaration := binding.parameter.Declaration; declaration != nil && declaration.SourceName != "" {
			name = declaration.SourceName
		}
		return reflectiveDescriptorLiteral("ReflectVariableKind", ctx,
			metadataKey("VariableDeclaration", javaTypeIDLiteral(sourceClassRuntimeTypeID(owner, ctx), ctx)),
			metadataKey("VariableName", metadataString(name))), nil, true
	}
	name, arguments, ok := reflectiveNamedTypeParts(base)
	if !ok {
		return nil, nil, false
	}
	raw, sourceScope, ok := reflectiveRawClassTypeID(name, ctx)
	if !ok {
		return nil, nil, false
	}
	if len(arguments) == 0 {
		return reflectiveDescriptorLiteral("ReflectClassKind", ctx, metadataKey("Raw", raw)), raw, true
	}
	if sourceScope != nil && len(arguments) != len(sourceScope.OwnTypeParameters()) {
		return nil, nil, false
	}
	if sourceScope == nil && canonicalMapEntryOwner(name, ctx) != "" {
		// Its raw class has an existing binary identity, but its parameterized
		// owner is outside the initial top-level control.
		return nil, nil, false
	}
	actual := reflectiveDescriptorSlice(ctx)
	for _, argument := range arguments {
		expression, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{
			Original: argument, TypeParameterBindings: javaType.TypeParameterBindings,
		}, ctx)
		if !supported {
			return nil, nil, false
		}
		actual.Elts = append(actual.Elts, expression)
	}
	values := []ast.Expr{metadataKey("Raw", raw), metadataKey("Arguments", actual)}
	if sourceScope != nil && sourceScope.Enclosing != nil {
		// A nested parameterized owner requires the segmented-owner control;
		// do not manufacture an owner from flattened physical Go arguments.
		return nil, nil, false
	}
	return reflectiveDescriptorLiteral("ReflectParameterizedKind", ctx, values...), nil, true
}

func reflectiveCanonicalObjectDescriptor(ctx Ctx) ast.Expr {
	return reflectiveDescriptorLiteral("ReflectClassKind", ctx, metadataKey("Raw", stdjavaQualifiedExpr("ObjectTypeID", ctx)))
}

func reflectiveWildcardDescriptorNode(javaType symbol.JavaType, ctx Ctx) (ast.Expr, ast.Expr, bool) {
	original := strings.TrimSpace(javaType.Original)
	upper, lower := reflectiveDescriptorSlice(ctx), reflectiveDescriptorSlice(ctx)
	upper.Elts = append(upper.Elts, reflectiveCanonicalObjectDescriptor(ctx))
	if original != "?" {
		remainder := strings.TrimSpace(strings.TrimPrefix(original, "?"))
		parts := strings.Fields(remainder)
		if len(parts) < 2 || (parts[0] != "extends" && parts[0] != "super") {
			return nil, nil, false
		}
		bound := strings.TrimSpace(strings.TrimPrefix(remainder, parts[0]))
		base, rank := javaArrayTypeParts(bound)
		_, primitive := javaPrimitiveType(base)
		if strings.HasPrefix(bound, "?") || rank == 0 && (primitive || base == "void") {
			return nil, nil, false
		}
		expression, _, supported := reflectiveTypeDescriptorNode(symbol.JavaType{
			Original: bound, TypeParameterBindings: javaType.TypeParameterBindings,
		}, ctx)
		if !supported {
			return nil, nil, false
		}
		if parts[0] == "extends" {
			upper.Elts = []ast.Expr{expression}
		} else {
			lower.Elts = []ast.Expr{expression}
		}
	}
	return reflectiveDescriptorLiteral("ReflectWildcardKind", ctx,
		metadataKey("UpperBounds", upper), metadataKey("LowerBounds", lower)), nil, true
}

// Use the carried declaration list rather than walking Enclosing: static
// member classes do not carry outer variables, while a nonstatic member may
// legitimately carry an outer declaration hidden by its own source spelling.
func reflectiveClassVariableAccessible(declaration *symbol.TypeParamDeclaration, ctx Ctx) bool {
	if ctx.currentClass == nil || ctx.localScope != nil && ctx.localScope.IsStatic {
		return false
	}
	for _, parameter := range ctx.currentClass.TypeParameters {
		if parameter.Declaration == declaration {
			return true
		}
	}
	return false
}

func reflectiveClassVariableOwner(binding referenceTypeParameterBinding) *symbol.ClassScope {
	declaration := binding.parameter.Declaration
	if declaration == nil {
		// A structural legacy spelling cannot establish a reflective declaration key.
		return nil
	}
	for owner := binding.context.currentClass; owner != nil; owner = owner.Enclosing {
		for _, parameter := range owner.OwnTypeParameters() {
			if parameter.Declaration == declaration {
				return owner
			}
		}
	}
	return nil
}

func reflectiveDescriptorLiteral(kind string, ctx Ctx, values ...ast.Expr) *ast.CompositeLit {
	return &ast.CompositeLit{Type: stdjavaQualifiedExpr("ReflectTypeDescriptor", ctx),
		Elts: append([]ast.Expr{metadataKey("Kind", stdjavaQualifiedExpr(kind, ctx))}, values...)}
}

func reflectiveDescriptorSlice(ctx Ctx) *ast.CompositeLit {
	return &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("ReflectTypeDescriptor", ctx)}}
}

// Preserve the distinction between a top-level type's own arguments and the
// arguments of an enclosing type. This initial exercised slice admits one
// argument group at the final name segment; other shapes fail explicitly.
func reflectiveNamedTypeParts(original string) (string, []string, bool) {
	depth, first, last := 0, -1, -1
	var arguments []string
	start := -1
	for index, character := range original {
		switch character {
		case '<':
			if depth == 0 {
				if first != -1 {
					return "", nil, false
				}
				first, start = index, index+1
			}
			depth++
		case '>':
			depth--
			if depth < 0 {
				return "", nil, false
			}
			if depth == 0 {
				argument := strings.TrimSpace(original[start:index])
				if argument == "" {
					return "", nil, false
				}
				arguments = append(arguments, argument)
				last = index
			}
		case ',':
			switch depth {
			case 1:
				argument := strings.TrimSpace(original[start:index])
				if argument == "" {
					return "", nil, false
				}
				arguments = append(arguments, argument)
				start = index + 1
			case 0:
				return "", nil, false
			}
		}
	}
	if depth != 0 || first != -1 && strings.TrimSpace(original[last+1:]) != "" {
		return "", nil, false
	}
	name := strings.TrimSpace(original)
	if first != -1 {
		name = strings.TrimSpace(original[:first])
	}
	if name == "" || strings.ContainsAny(name, " \t\r\n[]?@&") {
		return "", nil, false
	}
	return name, arguments, true
}

func reflectiveRawClassTypeID(name string, ctx Ctx) (ast.Expr, *symbol.ClassScope, bool) {
	if source := resolveClassScopeByQualifiedName(ctx, name); source != nil {
		id := sourceClassRuntimeTypeID(source, ctx)
		return javaTypeIDLiteral(id, ctx), source, id != ""
	}
	if primitive, ok := javaPrimitiveTypeIDExpr(name, ctx); ok && !strings.Contains(name, ".") {
		return primitive, nil, true
	}
	if canonicalMapEntryOwner(name, ctx) != "" {
		return stdjavaQualifiedExpr("JavaMapEntryTypeID", ctx), nil, true
	}
	// Qualification honors declaration-site imports and lexical source classes.
	// An arbitrary foreign.String must not enter a basename builtin fallback.
	qualified := qualifyDeclaredNominalReference(name, ctx)
	if !strings.Contains(qualified, ".") {
		return nil, nil, false
	}
	return javaTypeIDLiteral(qualified, ctx), nil, true
}

// Emit only variables actually written on this Java class, never carried ABI
// parameters. Bounds retain their captured declarations and header namespace.
func reflectiveTypeVariableDescriptorsExpr(scope *symbol.ClassScope, ctx Ctx) (ast.Expr, bool) {
	if scope == nil {
		return nil, false
	}
	declaring := classHeaderTypeCtx(scope, ctx)
	result := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("TypeVariableDescriptor", ctx)}}
	for _, parameter := range scope.OwnTypeParameters() {
		bounds := parameter.Bounds
		emittedBounds := reflectiveDescriptorSlice(ctx)
		if len(bounds) == 0 {
			// An implicit platform bound has no source type occurrence to resolve.
			emittedBounds.Elts = append(emittedBounds.Elts, reflectiveCanonicalObjectDescriptor(ctx))
		}
		for _, bound := range bounds {
			expression, supported := reflectiveTypeDescriptorExpr(bound, declaring)
			if !supported {
				return nil, false
			}
			emittedBounds.Elts = append(emittedBounds.Elts, expression)
		}
		name := parameter.Name
		if parameter.Declaration != nil && parameter.Declaration.SourceName != "" {
			name = parameter.Declaration.SourceName
		}
		result.Elts = append(result.Elts, &ast.CompositeLit{Type: stdjavaQualifiedExpr("TypeVariableDescriptor", ctx), Elts: []ast.Expr{
			metadataKey("Name", metadataString(name)), metadataKey("Bounds", emittedBounds),
		}})
	}
	return result, true
}

func reflectiveGenericSuperclassDescriptorExpr(scope *symbol.ClassScope, ctx Ctx) (ast.Expr, bool) {
	if scope == nil {
		return nil, false
	}
	if scope.IsInterface || javaClassBinaryName(scope) == "java.lang.Object" {
		return ast.NewIdent("nil"), true
	}
	if scope.IsEnum {
		return nil, false
	}
	superclass := strings.TrimSpace(scope.Superclass)
	if superclass == "" {
		return &ast.UnaryExpr{Op: token.AND, X: reflectiveCanonicalObjectDescriptor(ctx)}, true
	}
	expression, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{
		Original: superclass, TypeParameterBindings: symbol.VisibleTypeParamBindings(scope.TypeParameters),
	}, classHeaderTypeCtx(scope, ctx))
	if !supported {
		return nil, false
	}
	return &ast.UnaryExpr{Op: token.AND, X: expression}, true
}

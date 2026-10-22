package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Inference keeps its existing Java spelling, while invocation lookup also
// receives the declaration that owns it. Qualification cannot distinguish a
// default-package class from a caller binder with the same spelling.
type inferredJavaTypeOrigin struct {
	declaringOwner *symbol.ClassScope
	externalName string
	arrayRank    int
	wildcard     bool
	nominalScope *symbol.ClassScope
	parameter    *symbol.TypeParamDeclaration
	arguments    []inferredJavaTypeOrigin
	javaType     string
	unresolved   bool
}

func setInferredJavaTypeOrigin(outputs []*inferredJavaTypeOrigin, origin inferredJavaTypeOrigin) {
	if len(outputs) > 0 && outputs[0] != nil {
		*outputs[0] = origin
	}
}

func recordSyntaxJavaTypeOrigin(outputs []*inferredJavaTypeOrigin, javaType string, ctx Ctx) {
	if len(outputs) == 0 || outputs[0] == nil {
		return
	}
	setInferredJavaTypeOrigin(outputs, declaredJavaTypeOrigin(symbol.JavaType{Original: javaType, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx))
}

func recordNominalClassTypeOrigin(outputs []*inferredJavaTypeOrigin, scope *symbol.ClassScope) {
	if len(outputs) == 0 || outputs[0] == nil {
		return
	}
	setInferredJavaTypeOrigin(outputs, inferredJavaTypeOrigin{nominalScope: scope, arguments: classParameterTypeOrigins(scope)})
}

func recordCurrentClassFieldTypeOrigin(outputs []*inferredJavaTypeOrigin, scope *symbol.ClassScope, field *fieldResolution, ctx Ctx, javaType string) {
	if len(outputs) == 0 || outputs[0] == nil {
		return
	}
	recordFieldTypeOrigin(outputs, scope, classParameterTypeOrigins(scope), field, ctx, javaType)
}

func definitionDeclaredJavaType(def *symbol.Definition) symbol.JavaType {
	if def == nil {
		return symbol.JavaType{}
	}
	bindings := def.TypeParameterBindings
	if len(bindings) == 0 && def.DirectTypeParameter != nil {
		bindings = map[string]*symbol.TypeParamDeclaration{def.DirectTypeParameter.SourceName: def.DirectTypeParameter}
	}
	return symbol.JavaType{Original: def.OriginalType, TypeParameterBindings: bindings}
}

func declaredJavaTypeOrigin(javaType symbol.JavaType, declaring Ctx) inferredJavaTypeOrigin {
	original := strings.TrimSpace(javaType.Original)
	readType := readableWildcardProjection(original)
	base, rank := javaArrayTypeParts(readType)
	base, args := parseJavaTypeString(base)
	origin := inferredJavaTypeOrigin{javaType: readType, declaringOwner: declaring.currentClass, arrayRank: rank, wildcard: strings.HasPrefix(original, "?")}
	if origin.wildcard && !strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(original, "?")), "extends") {
		origin.externalName = "java.lang.Object"
		return origin
	}
	if declaration := javaType.TypeParameterBindings[base]; declaration != nil {
		origin.parameter = declaration
	} else if _, primitive := javaPrimitiveType(base); primitive || base == "void" || base == "null" || base == ternaryNullJavaType {
	} else {
		declaring = declaring.Clone()
		declaring.syntheticTypeParameters = nil
		origin.nominalScope = resolveClassScopeByQualifiedName(declaring, base)
		if origin.nominalScope == nil {
			origin.externalName = declaredExternalNominalOwner(base, javaType.TypeParameterBindings, declaring)
			origin.unresolved = origin.externalName == ""
		}
	}
	for _, arg := range args {
		origin.arguments = append(origin.arguments, declaredJavaTypeOrigin(symbol.JavaType{Original: arg, TypeParameterBindings: javaType.TypeParameterBindings}, declaring))
	}
	return declarationMemberTypeOrigins(origin, base, javaType, declaring)
}

func substituteJavaTypeOrigin(origin inferredJavaTypeOrigin, bindings map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin) inferredJavaTypeOrigin {
	visiting := map[*symbol.TypeParamDeclaration]bool{}
	var substitute func(inferredJavaTypeOrigin) inferredJavaTypeOrigin
	substitute = func(value inferredJavaTypeOrigin) inferredJavaTypeOrigin {
		if replacement, ok := bindings[value.parameter]; value.parameter != nil && ok && replacement.parameter != value.parameter && !visiting[value.parameter] {
			visiting[value.parameter] = true
			result := substitute(replacement)
			result.arrayRank += value.arrayRank
			delete(visiting, value.parameter)
			return result
		}
		result := value
		result.arguments = make([]inferredJavaTypeOrigin, len(value.arguments))
		for index, arg := range value.arguments {
			result.arguments[index] = substitute(arg)
		}
		return result
	}
	return substitute(origin)
}

func classParameterTypeOrigins(scope *symbol.ClassScope) []inferredJavaTypeOrigin {
	if scope == nil {
		return nil
	}
	origins := make([]inferredJavaTypeOrigin, len(scope.TypeParameters))
	for index, parameter := range scope.TypeParameters {
		origins[index] = inferredJavaTypeOrigin{parameter: parameter.Declaration, javaType: parameter.EmittedName()}
	}
	return origins
}

func sourceClassOriginBindings(start *symbol.ClassScope, arguments []inferredJavaTypeOrigin, owner *symbol.ClassScope, ctx Ctx) map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin {
	type originView struct {
		scope     *symbol.ClassScope
		arguments []inferredJavaTypeOrigin
	}
	queue := []originView{{scope: start, arguments: arguments}}
	seen := map[*symbol.ClassScope]bool{}
	for len(queue) > 0 {
		view := queue[0]
		queue = queue[1:]
		current := view.scope
		if current == nil || seen[current] {
			continue
		}
		seen[current] = true
		bindings := map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
		declaring := classHeaderTypeCtx(current, ctx)
		for index, parameter := range current.TypeParameters {
			if index < len(view.arguments) {
				bindings[parameter.Declaration] = view.arguments[index]
			} else if len(parameter.Bounds) > 0 {
				bindings[parameter.Declaration] = declaredJavaTypeOrigin(parameter.Bounds[0], declaring)
			}
		}
		if current == owner {
			return bindings
		}
		parents := make([]string, 0, 1+len(current.ImplementedInterfaces))
		if superclass := strings.TrimSpace(current.Superclass); superclass != "" {
			parents = append(parents, superclass)
		}
		parents = append(parents, current.ImplementedInterfaces...)
		for _, parentType := range parents {
			edge := declaredJavaTypeOrigin(symbol.JavaType{Original: parentType, TypeParameterBindings: symbol.VisibleTypeParamBindings(current.TypeParameters)}, declaring)
			edge = substituteJavaTypeOrigin(edge, bindings)
			if edge.nominalScope != nil {
				queue = append(queue, originView{scope: edge.nominalScope, arguments: edge.arguments})
			}
		}
	}
	return nil
}

func recordDefinitionTypeOrigin(outputs []*inferredJavaTypeOrigin, def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx, substitutions map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin, javaType string) {
	if len(outputs) == 0 || outputs[0] == nil || def == nil {
		return
	}
	declaring := ctx
	if owner != nil {
		declaring = classScopeCtx(owner, ctx)
	}
	origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(def), declaring)
	origin = substituteJavaTypeOrigin(origin, substitutions)
	origin.javaType = javaType
	setInferredJavaTypeOrigin(outputs, origin)
}

func recordFieldTypeOrigin(outputs []*inferredJavaTypeOrigin, start *symbol.ClassScope, arguments []inferredJavaTypeOrigin, field *fieldResolution, ctx Ctx, javaType string) {
	if field == nil || len(outputs) == 0 || outputs[0] == nil {
		return
	}
	recordDefinitionTypeOrigin(outputs, field.def, field.owner, ctx, sourceClassOriginBindings(start, arguments, field.owner, ctx), javaType)
}

func gatherMethodTypeOriginBindings(formal, actual inferredJavaTypeOrigin, methodParameters map[*symbol.TypeParamDeclaration]bool, candidates map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin) {
	if !completeJavaTypeOrigin(actual) || actual.arrayRank < formal.arrayRank {
		return
	}
	if methodParameters[formal.parameter] {
		actual.arrayRank -= formal.arrayRank
		base, _ := javaArrayTypeParts(actual.javaType)
		if _, primitive := javaPrimitiveType(base); primitive && actual.arrayRank == 0 {
			return
		}
		actual.javaType = base + strings.Repeat("[]", actual.arrayRank)
		candidates[formal.parameter] = append(candidates[formal.parameter], actual)
		return
	}
	if formal.nominalScope == nil && formal.parameter == nil && formal.externalName != "" && actual.externalName != formal.externalName && formal.arrayRank == actual.arrayRank {
		component := actual
		component.arrayRank = 0
		if projected, proved := projectJDK21GenericParentOrigin(component, formal.externalName); proved {
			projected.arrayRank = actual.arrayRank
			actual = projected
		}
	}
	if formal.nominalScope != actual.nominalScope || formal.externalName != actual.externalName || formal.arrayRank != actual.arrayRank {
		return
	}
	for i, arg := range formal.arguments {
		if i < len(actual.arguments) {
			gatherMethodTypeOriginBindings(arg, actual.arguments[i], methodParameters, candidates)
		}
	}
}

func sameJavaTypeOrigin(left, right inferredJavaTypeOrigin) bool {
	if left.externalName != right.externalName || left.arrayRank != right.arrayRank || left.wildcard != right.wildcard || left.nominalScope != right.nominalScope || left.parameter != right.parameter || left.unresolved != right.unresolved || len(left.arguments) != len(right.arguments) {
		return false
	}
	if left.nominalScope == nil && left.parameter == nil && left.externalName == "" && stripOriginArrayDisplay(left.javaType) != stripOriginArrayDisplay(right.javaType) {
		return false
	}
	for index, arg := range left.arguments {
		if !sameJavaTypeOrigin(arg, right.arguments[index]) {
			return false
		}
	}
	return true
}

// The expression's existing typing rule chooses its result. The sidecar only
// carries either their shared declaration or a nominal declaration selected by
// the existing common-reference proof. Unproved origins cannot be recaptured
// through the consumer's namespace.
func recordJoinedJavaTypeOrigin(outputs []*inferredJavaTypeOrigin, javaType string, values []inferredJavaTypeOrigin, ctx Ctx, selected ...inferredJavaTypeOrigin) {
	if len(outputs) == 0 || outputs[0] == nil {
		return
	}
	// Primitive promotion is selected by the existing expression typing rule;
	// it has no declaration namespace and cannot lose a nominal reference.
	if _, primitive := javaPrimitiveType(javaType); primitive {
		setInferredJavaTypeOrigin(outputs, inferredJavaTypeOrigin{javaType: javaType})
		return
	}
	var contributions []inferredJavaTypeOrigin
	for _, value := range values {
		if value.javaType != "null" && value.javaType != ternaryNullJavaType {
			contributions = append(contributions, value)
		}
	}
	// A null-only conditional has no nominal identity to lose. Keep its
	// intrinsic null proof so contextual reference conversion can remain deferred.
	if len(values) > 0 && len(contributions) == 0 {
		setInferredJavaTypeOrigin(outputs, inferredJavaTypeOrigin{javaType: ternaryNullJavaType})
		return
	}
	if joined, known := joinedJavaTypeOrigin(javaType, contributions, ctx); known {
		setInferredJavaTypeOrigin(outputs, joined)
	} else {
		setInferredJavaTypeOrigin(outputs, inferredJavaTypeOrigin{javaType: javaType, unresolved: true})
	}
}

// A joined source result is nominal only when every contributing declaration
// proves the selected source reference type. A caller spelling alone is not
// evidence of that identity.
func joinedJavaTypeOrigin(javaType string, values []inferredJavaTypeOrigin, ctx Ctx) (inferredJavaTypeOrigin, bool) {
	if len(values) == 0 {
		return inferredJavaTypeOrigin{}, false
	}
	first := values[0]
	if !completeJavaTypeOrigin(first) {
		return inferredJavaTypeOrigin{}, false
	}
	same := true
	for _, value := range values[1:] {
		if !completeJavaTypeOrigin(value) {
			return inferredJavaTypeOrigin{}, false
		}
		if !sameJavaTypeOrigin(first, value) {
			same = false
		}
	}
	if same {
		if javaType != "" {
			first.javaType = javaType
		}
		return first, true
	}
	// Common source classes are selected by actual ancestry, never result text.
	if first.nominalScope != nil {
		for scope := first.nominalScope; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
			selected := inferredJavaTypeOrigin{nominalScope: scope, arrayRank: first.arrayRank, javaType: javaType}
			all := true
			for _, value := range values {
				_, _, applicable, handled := declaredOriginConversion(value, selected, ctx)
				if !handled || !applicable {
					all = false
					break
				}
			}
			if all {
				return selected, true
			}
		}
	}
	// An unrelated reference join owns the canonical root only from complete
	// contributing declarations. Unknown/primitive/binder arms do not prove it.
	for _, value := range values {
		if value.parameter != nil || value.arrayRank == 0 && value.nominalScope == nil && value.externalName == "" {
			return inferredJavaTypeOrigin{}, false
		}
	}
	root := intrinsicJavaTypeOrigin("java.lang.Object")
	if javaType != "" {
		root.javaType = javaType
	}
	return root, true
}

// This receives the already selected source method. It never repeats the
// invocation's overload lookup to recover a nominal return declaration.
func recordMethodTypeOrigin(outputs []*inferredJavaTypeOrigin, resolution *methodResolution, target *invocationTargetInfo, node *sitter.Node, ctx Ctx, source []byte, javaType string) {
	if len(outputs) == 0 || outputs[0] == nil || resolution == nil || resolution.def == nil {
		return
	}
	def := resolution.def
	var substitutions map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin
	if target != nil {
		substitutions = sourceClassOriginBindings(target.classScope, target.classTypeArgumentOrigins, resolution.owner, ctx)
	} else {
		substitutions = sourceClassOriginBindings(ctx.currentClass, classParameterTypeOrigins(ctx.currentClass), resolution.owner, ctx)
	}
	if substitutions == nil {
		substitutions = map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin{}
	}
	if len(def.TypeParameters) > 0 {
		methodParameters := map[*symbol.TypeParamDeclaration]bool{}
		for _, parameter := range def.TypeParameters {
			if parameter.Declaration != nil {
				methodParameters[parameter.Declaration] = true
			}
		}
		candidates := map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin{}
		declaring, _ := invocationMethodDeclarationContext(def, ctx)
		for index, formal := range def.Parameters {
			argument := invocationArgumentNode(node, index)
			if argument == nil {
				continue
			}
			var actualOrigin inferredJavaTypeOrigin
			actualType, known := inferExprJavaType(argument, ctx, source, &actualOrigin)
			if !known {
				continue
			}
			actualOrigin.javaType = actualType
			formalType := definitionDeclaredJavaType(formal)

			gatherMethodTypeOriginBindings(declaredJavaTypeOrigin(formalType, declaring), actualOrigin, methodParameters, candidates)
		}
		inferred := genericJavaInvocationTypeBindings(def, node, ctx, source)
		explicit := node.ChildByFieldName("type_arguments")
		for index, parameter := range def.TypeParameters {
			if explicit != nil && index < int(explicit.NamedChildCount()) {
				syntax := explicit.NamedChild(index).Content(source)
				substitutions[parameter.Declaration] = declaredJavaTypeOrigin(symbol.JavaType{Original: syntax, TypeParameterBindings: symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))}, ctx)
				continue
			}
			values := candidates[parameter.Declaration]
			if len(values) == 0 {
				if len(parameter.Bounds) > 0 {
					substitutions[parameter.Declaration] = declaredJavaTypeOrigin(parameter.Bounds[0], declaring)
				}
				continue
			}
			if selected, known := joinedJavaTypeOrigin(inferred[parameter.Name], values, ctx); known {
				substitutions[parameter.Declaration] = selected
			}
		}
	}
	recordDefinitionTypeOrigin(outputs, def, resolution.owner, ctx, substitutions, javaType)
}

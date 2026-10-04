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
	nominalScope   *symbol.ClassScope
	parameter      *symbol.TypeParamDeclaration
	arguments      []inferredJavaTypeOrigin
	javaType       string
	unresolved     bool
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
	readType := readableWildcardProjection(javaType.Original)
	if original := strings.TrimSpace(javaType.Original); strings.HasPrefix(original, "?") && !strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(original, "?")), "extends") {
		readType = "java.lang.Object"
	}
	base, _ := javaArrayTypeParts(readType)
	base, args := parseJavaTypeString(base)
	origin := inferredJavaTypeOrigin{javaType: readType, declaringOwner: declaring.currentClass}
	if declaration := javaType.TypeParameterBindings[base]; declaration != nil {
		origin.parameter = declaration
	} else {
		// A missing captured binder is a nominal declaration, not an invitation to
		// reinterpret the signature in the consuming method's binder namespace.
		origin.nominalScope = resolveClassScopeByQualifiedName(declaring, base)
	}
	for _, arg := range args {
		origin.arguments = append(origin.arguments, declaredJavaTypeOrigin(symbol.JavaType{Original: arg, TypeParameterBindings: javaType.TypeParameterBindings}, declaring))
	}
	return origin
}

func substituteJavaTypeOrigin(origin inferredJavaTypeOrigin, bindings map[*symbol.TypeParamDeclaration]inferredJavaTypeOrigin) inferredJavaTypeOrigin {
	visiting := map[*symbol.TypeParamDeclaration]bool{}
	var substitute func(inferredJavaTypeOrigin) inferredJavaTypeOrigin
	substitute = func(value inferredJavaTypeOrigin) inferredJavaTypeOrigin {
		if replacement, ok := bindings[value.parameter]; value.parameter != nil && ok && replacement.parameter != value.parameter && !visiting[value.parameter] {
			visiting[value.parameter] = true
			result := substitute(replacement)
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

func boundReceiverTypeOrigin(parameter *symbol.TypeParamDeclaration, expected *symbol.ClassScope, ctx Ctx) inferredJavaTypeOrigin {
	bindings := referenceTypeParameterBindings(ctx)
	byDeclaration := map[*symbol.TypeParamDeclaration]referenceTypeParameterBinding{}
	for _, binding := range bindings {
		byDeclaration[binding.parameter.Declaration] = binding
	}
	seen := map[*symbol.TypeParamDeclaration]bool{}
	var visit func(*symbol.TypeParamDeclaration) (inferredJavaTypeOrigin, bool)
	visit = func(declaration *symbol.TypeParamDeclaration) (inferredJavaTypeOrigin, bool) {
		if declaration == nil || seen[declaration] {
			return inferredJavaTypeOrigin{}, false
		}
		seen[declaration] = true
		binding, known := byDeclaration[declaration]
		if !known {
			return inferredJavaTypeOrigin{}, false
		}
		for _, bound := range binding.parameter.Bounds {
			origin := declaredJavaTypeOrigin(bound, binding.context)
			if origin.parameter != nil {
				if result, ok := visit(origin.parameter); ok {
					return result, true
				}
			}
			if origin.nominalScope == expected {
				return origin, true
			}
		}
		return inferredJavaTypeOrigin{}, false
	}
	origin, _ := visit(parameter)
	return origin
}

func gatherMethodTypeOriginBindings(formal, actual inferredJavaTypeOrigin, methodParameters map[*symbol.TypeParamDeclaration]bool, candidates map[*symbol.TypeParamDeclaration][]inferredJavaTypeOrigin) {
	if methodParameters[formal.parameter] {
		candidates[formal.parameter] = append(candidates[formal.parameter], actual)
		return
	}
	for index, arg := range formal.arguments {
		if index < len(actual.arguments) {
			gatherMethodTypeOriginBindings(arg, actual.arguments[index], methodParameters, candidates)
		}
	}
}

func sameJavaTypeOrigin(left, right inferredJavaTypeOrigin) bool {
	if left.nominalScope != right.nominalScope || left.parameter != right.parameter || left.unresolved != right.unresolved || len(left.arguments) != len(right.arguments) {
		return false
	}
	if left.nominalScope == nil && left.parameter == nil && left.javaType != right.javaType {
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
	var contributions []inferredJavaTypeOrigin
	for _, value := range values {
		if value.javaType != "null" && value.javaType != ternaryNullJavaType {
			contributions = append(contributions, value)
		}
	}
	unresolved := inferredJavaTypeOrigin{javaType: javaType, unresolved: true}
	if len(contributions) == 0 || contributions[0].unresolved || contributions[0].nominalScope == nil && contributions[0].parameter == nil {
		setInferredJavaTypeOrigin(outputs, unresolved)
		return
	}
	if len(selected) > 0 && selected[0].nominalScope != nil {
		chosen := selected[0]
		for _, value := range contributions {
			if value.unresolved || value.nominalScope == nil || !javaReferenceTypeAssignable(value.nominalScope, chosen.nominalScope, ctx) {
				setInferredJavaTypeOrigin(outputs, unresolved)
				return
			}
		}
		_, arguments := parseJavaTypeString(javaType)
		if len(arguments) > 0 {
			// The chosen parameterized arm already owns its argument origins.
			// No absent generic arguments are invented from a common class name.
			for _, value := range contributions {
				if value.nominalScope == chosen.nominalScope && value.javaType == javaType {
					chosen.arguments = value.arguments
					break
				}
			}
			if len(chosen.arguments) != len(arguments) {
				setInferredJavaTypeOrigin(outputs, unresolved)
				return
			}
		}
		chosen.javaType = javaType
		setInferredJavaTypeOrigin(outputs, chosen)
		return
	}
	for _, value := range contributions[1:] {
		if !sameJavaTypeOrigin(contributions[0], value) {
			setInferredJavaTypeOrigin(outputs, unresolved)
			return
		}
	}
	if joined, known := joinedJavaTypeOrigin(javaType, contributions, ctx); known {
		setInferredJavaTypeOrigin(outputs, joined)
	} else {
		setInferredJavaTypeOrigin(outputs, unresolved)
	}
}

// A joined source result is nominal only when every contributing declaration
// proves the selected source reference type. A caller spelling alone is not
// evidence of that identity.
func joinedJavaTypeOrigin(javaType string, values []inferredJavaTypeOrigin, ctx Ctx) (inferredJavaTypeOrigin, bool) {
	if len(values) == 0 {
		return inferredJavaTypeOrigin{}, false
	}
	projected := readableWildcardProjection(javaType)
	base, arguments := parseJavaTypeString(projected)
	same := true
	for _, value := range values[1:] {
		if !sameJavaTypeOrigin(values[0], value) {
			same = false
			break
		}
	}
	firstBase, _ := parseJavaTypeString(values[0].javaType)
	if same && base == firstBase {
		result := values[0]
		result.javaType = javaType
		return result, true
	}
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if same && scope != nil && values[0].nominalScope == scope {
		result := values[0]
		result.javaType = javaType
		return result, true
	}
	if scope == nil {
		return inferredJavaTypeOrigin{}, false
	}
	for _, value := range values {
		if value.javaType == "null" || value.javaType == ternaryNullJavaType {
			continue
		}
		if value.nominalScope != nil && javaReferenceTypeAssignable(value.nominalScope, scope, ctx) {
			continue
		}
		if value.parameter != nil && boundReceiverTypeOrigin(value.parameter, scope, ctx).nominalScope == scope {
			continue
		}
		return inferredJavaTypeOrigin{}, false
	}
	result := inferredJavaTypeOrigin{nominalScope: scope, javaType: javaType}
	for index, argument := range arguments {
		var candidates []inferredJavaTypeOrigin
		for _, value := range values {
			if index < len(value.arguments) {
				candidates = append(candidates, value.arguments[index])
			}
		}
		origin, known := joinedJavaTypeOrigin(argument, candidates, ctx)
		if !known {
			origin = inferredJavaTypeOrigin{javaType: argument}
		}
		result.arguments = append(result.arguments, origin)
	}
	return result, true
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
			gatherMethodTypeOriginBindings(declaredJavaTypeOrigin(definitionDeclaredJavaType(formal), declaring), actualOrigin, methodParameters, candidates)
		}
		inferred := resolvedMethodInvocationTypeBindings(def, node, ctx, source)
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

package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
)

// A method reference has the target functional interface's parameter and result
// types. Its referenced method may have different primitive/reference types,
// so a method value alone is insufficient at an autoboxing boundary.
func methodReferenceJavaSignature(ctx Ctx) ([]string, string) {
	method, bindings := resolveFunctionalInterfaceMethod(ctx, ctx.expectedType)
	var parameters []string
	result := ""
	if method == nil && isExternalRunnableType(ctx.expectedType, ctx) {
		result = "void"
	}
	if method != nil {
		for index := range method.Parameters {
			parameters = append(parameters, substituteJavaTypeParams(definitionParameterJavaSignatureType(method, index), bindings))
		}
		result = substituteJavaTypeParams(method.OriginalType, bindings)
	}
	if ctx.lambdaParameterJavaTypes != nil {
		parameters = append([]string(nil), ctx.lambdaParameterJavaTypes...)
	}
	if ctx.lambdaResultJavaType != "" {
		result = ctx.lambdaResultJavaType
	}
	return parameters, result
}

func methodReferenceUsesExecutionSAM(ctx Ctx) bool {
	base, _ := parseJavaTypeString(ctx.expectedType)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	return scope != nil && scope.IsInterface || isExternalRunnableType(ctx.expectedType, ctx)
}

func methodReferenceResultConversion(value ast.Expr, actualType string, ctx Ctx) ast.Expr {
	_, resultType := methodReferenceJavaSignature(ctx)
	if resultType == "" || resultType == "void" || actualType == resultType {
		return value
	}
	if converted, ok := convertJavaValue(value, actualType, resultType, ctx); ok {
		return converted
	}
	return methodReferenceConvertedArgument(value, actualType, resultType, ctx)
}

func methodReferenceConvertedArgument(value ast.Expr, actualType, expectedType string, ctx Ctx) ast.Expr {
	if converted, ok := convertJavaValue(value, actualType, expectedType, ctx); ok {
		return converted
	}
	if expectedType != actualType && javaDependentTypeParameterAssignable(actualType, expectedType, ctx) {
		return dependentTypeParameterWideningExpr(value, actualType, expectedType, ctx)
	}
	actualBase, _ := parseJavaTypeString(actualType)
	expectedBase, _ := parseJavaTypeString(expectedType)
	actualScope := resolveClassScopeByQualifiedName(ctx, actualBase)
	expectedScope := resolveClassScopeByQualifiedName(ctx, expectedBase)
	if actualScope != nil && expectedScope != nil && expectedScope.Class != nil &&
		!expectedScope.IsInterface && !expectedScope.IsAbstract && classInheritsFrom(actualScope, expectedScope, ctx) {
		return &ast.SelectorExpr{X: value, Sel: &ast.Ident{Name: expectedScope.Class.Name}}
	}
	return value
}

func inferMethodReferenceResultJavaType(node *sitter.Node, parameters []string, ctx Ctx, source []byte) (string, bool) {
	if node == nil || node.Type() != "method_reference" || node.NamedChildCount() == 0 {
		return "", false
	}
	target := node.NamedChild(0)
	if node.NamedChildCount() < 2 {
		return target.Content(source), true
	}
	method := node.NamedChild(int(node.NamedChildCount()) - 1).Content(source)
	if class, ok := intrinsicStaticClassName(target, ctx, source); ok {
		if result := staticIntrinsicResultTypes[intrinsicKey{class, method}]; result != "" {
			return result, true
		}
		if result := instanceIntrinsicResultTypes[intrinsicKey{class, method}]; result != "" {
			return result, true
		}
	}
	if class := resolveClassScopeByTypeQualifier(ctx, target.Content(source)); class != nil {
		if selected := resolveMethodReferenceOverload(class, nil, method, parameters, true, false, ctx); selected != nil {
			definition := selected.def
			bindings := methodReferenceInferredBindings(definition, parameters, "", ctx)
			if explicit := node.ChildByFieldName("type_arguments"); explicit != nil && int(explicit.NamedChildCount()) == len(definition.TypeParameters) {
				for index, parameter := range definition.TypeParameters {
					bindings[parameter.Name] = explicit.NamedChild(index).Content(source)
				}
			}
			return substituteJavaTypeParams(definition.OriginalType, bindings), true
		}
		if resolution := findInstanceMethodInHierarchy(class, method, len(parameters)-1, ctx); resolution != nil && resolution.def != nil {
			return resolution.def.OriginalType, true
		}
	}
	if javaType, ok := inferExprJavaType(target, ctx, source); ok {
		base, _ := parseJavaTypeString(javaType)
		if result := instanceIntrinsicResultTypes[intrinsicKey{stripJavaQualifier(base), method}]; result != "" {
			return result, true
		}
	}
	if targetInfo := resolveInvocationTarget(target, ctx, source); targetInfo != nil {
		if resolution, selected := findInstanceMethodForInvocationTarget(targetInfo, method, len(parameters), ctx); resolution != nil && resolution.def != nil {
			result := resolution.def.OriginalType
			arguments := invocationOwnerTypeArguments(selected, resolution, ctx)
			if resolution.owner != nil && len(arguments) == len(resolution.owner.TypeParameters) {
				bindings := map[string]string{}
				for i, parameter := range resolution.owner.TypeParameters {
					bindings[parameter.Name] = arguments[i]
				}
				result = substituteJavaTypeParams(result, bindings)
			}
			return result, true
		}
	}
	return "", false
}

// lowerBuiltinMethodReference lowers runtime-backed class references using the
// ordinary intrinsic generators, but converts their arguments and result using
// the SAM signature before producing the closure.
func lowerBuiltinMethodReference(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	parameters, result := methodReferenceJavaSignature(ctx)
	if result == "" || node.NamedChildCount() == 0 {
		return nil, false
	}
	targetNode := node.NamedChild(0)
	constructor := node.NamedChildCount() < 2
	method := "new"
	if !constructor {
		method = node.NamedChild(int(node.NamedChildCount()) - 1).Content(source)
	}
	class, typeQualifier := intrinsicStaticClassName(targetNode, ctx, source)
	targetJavaType := class
	if typeQualifier {
		targetJavaType = targetNode.Content(source)
	}
	if !typeQualifier && targetNode.Type() == "generic_type" {
		base, _ := parseJavaTypeString(targetNode.Content(source))
		if resolveClassScopeByQualifiedName(ctx, base) == nil {
			class, targetJavaType, typeQualifier = stripJavaQualifier(base), targetNode.Content(source), true
		}
	}
	if !typeQualifier {
		var known bool
		targetJavaType, known = inferExprJavaType(targetNode, ctx, source)
		if !known {
			return nil, false
		}
		base, _ := parseJavaTypeString(targetJavaType)
		if resolveClassScopeByQualifiedName(ctx, base) != nil {
			return nil, false
		}
		class = stripJavaQualifier(base)
	}
	primitive, wrapper := builtinJavaWrapperPrimitive(targetJavaType, ctx)
	if constructor && !wrapper {
		return nil, false
	}
	staticGenerator := staticIntrinsics[intrinsicKey{class, method}]
	instanceGenerator := instanceIntrinsics[intrinsicKey{class, method}]
	static := typeQualifier && staticGenerator != nil
	if !constructor && !static && instanceGenerator == nil {
		return nil, false
	}

	functionType := &ast.FuncType{Params: &ast.FieldList{}}
	for index, javaType := range parameters {
		functionType.Params.List = append(functionType.Params.List, &ast.Field{
			Names: []*ast.Ident{{Name: "__java2goArg" + strconv.Itoa(index)}},
			Type:  javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx),
		})
	}
	if result != "void" {
		functionType.Results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(result, inScopeTypeParameters(ctx), ctx)}}}
	}
	bodyCtx := ctx.Clone()
	if methodReferenceUsesExecutionSAM(ctx) {
		name := executionParameterName(node, source, ctx)
		functionType.Params.List = append([]*ast.Field{executionParameterField(name, ctx)}, functionType.Params.List...)
		bodyCtx.executionContextName = name
	}
	arguments := make([]ast.Expr, len(parameters))
	for index := range parameters {
		arguments[index] = &ast.Ident{Name: "__java2goArg" + strconv.Itoa(index)}
	}
	argumentTypes := append([]string(nil), parameters...)
	boundReceiver := !typeQualifier && !constructor
	var receiver ast.Expr
	if !constructor && !static {
		if typeQualifier {
			if len(arguments) == 0 {
				return nil, false
			}
			receiver = methodReferenceConvertedArgument(arguments[0], argumentTypes[0], targetJavaType, bodyCtx)
			arguments, argumentTypes = arguments[1:], argumentTypes[1:]
		} else {
			receiver = &ast.Ident{Name: "__java2goMethodReferenceReceiver"}
		}
		// String keeps a sentinel ABI; this checks both null representations.
		if class == "String" {
			receiver = stdjavaCall(bodyCtx, "StringRequireNonNull", receiver)
		}
	}

	expected := append([]string(nil), argumentTypes...)
	actualResult := ""
	if constructor || static {
		actualResult = staticIntrinsicResultTypes[intrinsicKey{class, method}]
		if constructor {
			actualResult = targetJavaType
		} else if wrapper && actualResult == class {
			actualResult = targetJavaType
		}
		if wrapper {
			switch method {
			case "new", "valueOf":
				if len(expected) > 0 && !isJavaStringType(expected[0]) {
					expected[0] = primitive
					// Float(double) is its only narrowing constructor overload.
					if constructor && class == "Float" && (argumentTypes[0] == "double" || argumentTypes[0] == "Double") {
						expected[0] = "double"
					}
				}
				if len(expected) > 1 {
					expected[1] = "int"
				}
			case "parseByte", "parseShort", "parseInt", "parseLong", "parseFloat", "parseDouble", "parseBoolean":
				if len(expected) > 0 {
					expected[0] = "String"
				}
				if len(expected) > 1 {
					expected[1] = "int"
				}
			case "compare":
				for i := range expected {
					expected[i] = primitive
				}
			case "toString", "hashCode", "isNaN", "isInfinite":
				if len(expected) > 0 {
					expected[0] = primitive
				}
			}
		}
	} else {
		actualResult = instanceIntrinsicResultTypes[intrinsicKey{class, method}]
		switch method {
		case "equals":
			if len(expected) == 1 {
				expected[0] = "Object"
			}
		case "compareTo":
			if len(expected) == 1 {
				expected[0] = targetJavaType
				if class == "Comparable" {
					_, typeArguments := parseJavaTypeString(targetJavaType)
					expected[0] = "Object"
					if len(typeArguments) > 0 {
						expected[0] = typeArguments[0]
					}
				}
			}
		}
	}
	for i := range arguments {
		arguments[i] = methodReferenceConvertedArgument(arguments[i], argumentTypes[i], expected[i], bodyCtx)
	}
	var call ast.Expr
	if constructor {
		if len(arguments) != 1 {
			return nil, false
		}
		value := arguments[0]
		if isJavaStringType(expected[0]) {
			parser := map[string]string{"Boolean": "ParseBoolean", "Byte": "ParseByte", "Short": "ParseShort", "Integer": "ParseInt", "Long": "ParseLong", "Float": "ParseFloat", "Double": "ParseDouble"}[class]
			if parser == "" {
				return nil, false
			}
			value = stdjavaCall(bodyCtx, parser, value)
		}
		name := "New" + class
		if class == "Float" && expected[0] == "double" {
			name = "NewFloatFromDouble"
		}
		call = stdjavaCall(bodyCtx, name, value)
	} else if static {
		call = staticGenerator(nil, arguments, bodyCtx)
	} else {
		call = instanceGenerator(receiver, arguments, bodyCtx)
	}
	if call == nil {
		return nil, false
	}
	if actualResult != "" {
		call = methodReferenceResultConversion(call, actualResult, bodyCtx)
	}
	closure := &ast.FuncLit{
		Type: functionType,
		Body: &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, functionType.Results)}},
	}
	var expression ast.Expr = closure
	if boundReceiver {
		parsedReceiver := ParseExpr(targetNode, source, ctx)
		expression = &ast.CallExpr{Fun: &ast.FuncLit{
			Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: functionType}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: "__java2goMethodReferenceReceiver"}}, Tok: token.DEFINE,
					Rhs: []ast.Expr{stdjavaCall(ctx, "ReferenceRequireNonNull", parsedReceiver)}},
				&ast.ReturnStmt{Results: []ast.Expr{closure}},
			}},
		}}
	}
	if adapted := wrapLambdaWithFunctionalInterfaceAdapter(expression, ctx.expectedType, methodReferenceUsesExecutionSAM(ctx), ctx); adapted != nil {
		expression = adapted
	}
	return expression, true
}

// Adapt source method parameter conversions independently of varargs packing.
func methodReferenceConvertedArguments(args []ast.Expr, resolution *methodResolution, target *invocationTargetInfo, unbound bool, ctx Ctx) []ast.Expr {
	actualTypes, _ := methodReferenceJavaSignature(ctx)
	if unbound && len(actualTypes) > 0 {
		actualTypes = actualTypes[1:]
	}
	var ownerArguments []string
	if target != nil {
		ownerArguments = invocationOwnerTypeArguments(target, resolution, ctx)
	}
	expectedTypes := instantiatedMethodParameterTypes(resolution, ownerArguments)
	converted := append([]ast.Expr(nil), args...)
	for index := range converted {
		if index >= len(expectedTypes) || index >= len(actualTypes) ||
			executionParameterIsVariadic(resolution.def, index) {
			break
		}
		converted[index] = methodReferenceConvertedArgument(converted[index], actualTypes[index], expectedTypes[index], ctx)
	}
	return converted
}

func methodReferenceDeclaredResultType(resolution *methodResolution, target *invocationTargetInfo, ctx Ctx) string {
	result := resolution.def.OriginalType
	if resolution.owner == nil || target == nil {
		return result
	}
	arguments := invocationOwnerTypeArguments(target, resolution, ctx)
	if len(arguments) != len(resolution.owner.TypeParameters) {
		return result
	}
	bindings := map[string]string{}
	for i, parameter := range resolution.owner.TypeParameters {
		bindings[parameter.Name] = arguments[i]
	}
	return substituteJavaTypeParams(result, bindings)
}

func staticMethodReferenceNeedsConversion(definition *symbol.Definition, ctx Ctx) bool {
	if definition == nil {
		return false
	}
	parameters, result := methodReferenceJavaSignature(ctx)
	if result == "void" && (definition.Constructor || definition.OriginalType != "" && definition.OriginalType != "void") {
		return true
	}
	if result != "void" && definition.OriginalType != "" && !javaInferenceSameType(definition.OriginalType, result, ctx) {
		return true
	}
	for index, actual := range parameters {
		if index >= len(definition.Parameters) {
			break
		}
		expected := definitionParameterJavaSignatureType(definition, index)
		if !javaInferenceSameType(actual, expected, ctx) {
			return true
		}
	}
	return false
}

func methodReferenceInferredBindings(definition *symbol.Definition, actualTypes []string, resultType string, ctx Ctx) map[string]string {
	bindings := map[string]string{}
	if definition == nil || len(definition.TypeParameters) == 0 {
		return bindings
	}
	names := map[string]struct{}{}
	for _, parameter := range definition.TypeParameters {
		names[parameter.Name] = struct{}{}
	}
	lowerBounds := map[string][]string{}
	for index, parameter := range definition.Parameters {
		if index >= len(actualTypes) {
			break
		}
		if executionParameterIsVariadic(definition, index) {
			formal := parameter.OriginalType
			actualBase, actualRank := javaArrayTypeParts(actualTypes[index])
			_, formalRank := javaArrayTypeParts(formal)
			_, primitive := javaPrimitiveType(actualBase)
			if len(actualTypes) == len(definition.Parameters) && actualRank > formalRank && (!primitive || actualRank != formalRank+1) {
				collectGenericMethodInferenceBounds(formal+"[]", actualTypes[index], names, lowerBounds, ctx)
			} else {
				for _, actual := range actualTypes[index:] {
					collectGenericMethodInferenceBounds(formal, actual, names, lowerBounds, ctx)
				}
			}
			break
		}
		collectGenericMethodInferenceBounds(parameter.OriginalType, actualTypes[index], names, lowerBounds, ctx)
	}
	targetBounds := map[string][]string{}
	if resultType != "" && resultType != "void" {
		collectGenericMethodInferenceBounds(definition.OriginalType, resultType, names, targetBounds, ctx)
	}
	for name, bounds := range targetBounds {
		if len(lowerBounds[name]) == 0 {
			lowerBounds[name] = bounds
		}
	}
	for _, parameter := range definition.TypeParameters {
		inferred := javaInferenceLeastUpperBound(lowerBounds[parameter.Name], ctx)
		if inferred == "" {
			continue
		}
		erased := rawTypeParameterErasure(parameter, definition.TypeParameters)
		base, _ := parseJavaTypeString(erased)
		if bound := resolveClassScopeByQualifiedName(ctx, base); bound != nil && !bound.IsInterface && !methodTypeParameterRequiresConcreteWitness(definition, parameter.Declaration, ctx) {
			inferred = erased
		}
		bindings[parameter.Name] = inferred
	}
	for _, parameter := range definition.TypeParameters {
		if bindings[parameter.Name] != "" {
			continue
		}
		var candidates []string
		for _, dependent := range definition.TypeParameters {
			if methodTypeParameterDependsOn(dependent.Name, parameter.Name, definition.TypeParameters) && bindings[dependent.Name] != "" {
				candidates = append(candidates, bindings[dependent.Name])
			}
		}
		inferred := javaInferenceLeastUpperBound(candidates, ctx)
		if inferred == "" {
			inferred = rawTypeParameterErasure(parameter, definition.TypeParameters)
		}
		bindings[parameter.Name] = inferred
	}
	return bindings
}

func instantiateStaticMethodReference(reference ast.Expr, definition *symbol.Definition, node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, *symbol.Definition) {
	if definition == nil || len(definition.TypeParameters) == 0 {
		return reference, definition
	}
	parameters, result := methodReferenceJavaSignature(ctx)
	bindings := methodReferenceInferredBindings(definition, parameters, result, ctx)
	if explicit := node.ChildByFieldName("type_arguments"); explicit != nil && int(explicit.NamedChildCount()) == len(definition.TypeParameters) {
		for index, parameter := range definition.TypeParameters {
			bindings[parameter.Name] = explicit.NamedChild(index).Content(source)
		}
	}
	types := make([]ast.Expr, len(definition.TypeParameters))
	for index, parameter := range definition.TypeParameters {
		types[index] = javaTypeStringToGoTypeExpr(bindings[parameter.Name], inScopeTypeParameters(ctx), ctx)
	}
	specialized := *definition
	specialized.TypeParameters = nil
	specialized.OriginalType = substituteJavaTypeParams(definition.OriginalType, bindings)
	specialized.Parameters = make([]*symbol.Definition, len(definition.Parameters))
	for index, parameter := range definition.Parameters {
		copy := *parameter
		copy.OriginalType = substituteJavaTypeParams(parameter.OriginalType, bindings)
		copy.TypeParameterBindings = nil
		copy.DirectTypeParameter = nil
		specialized.Parameters[index] = &copy
	}
	// Erased result metadata still belongs to the original target method; this
	// specialized signature is only used for the reference's argument adapter.
	return applyTypeArguments(reference, types), &specialized
}

func convertedStaticMethodReference(reference ast.Expr, definition *symbol.Definition, owner *symbol.ClassScope, functionType *ast.FuncType, executionAware bool, ctx Ctx) ast.Expr {
	if definition == nil || functionType == nil {
		return reference
	}
	if !staticMethodReferenceNeedsConversion(definition, ctx) && methodReferenceUsesExecutionSAM(ctx) {
		return generatedStaticVarargsMethodReference(reference, definition, owner, functionType, executionAware, ctx)
	}
	parameters := cloneFieldList(functionType.Params)
	args := methodCallArgs(parameters)
	if len(args) == 0 {
		return reference
	}
	execution := args[0]
	resolution := &methodResolution{def: definition, owner: owner}
	javaArgs := methodReferenceConvertedArguments(args[1:], resolution, nil, false, ctx)
	javaArgs = generatedMethodReferenceVarargsArguments(resolution, nil, javaArgs, false, ctx)
	if executionAware {
		javaArgs = append([]ast.Expr{execution}, javaArgs...)
	}
	call := ast.Expr(&ast.CallExpr{Fun: reference, Args: javaArgs})
	actualResult := definition.OriginalType
	if definition.Constructor && owner != nil && owner.Class != nil {
		actualResult = owner.Class.OriginalName
	}
	call = methodReferenceResultConversion(call, actualResult, ctx)
	return &ast.FuncLit{
		Type: &ast.FuncType{Params: parameters, Results: cloneFieldList(functionType.Results)},
		Body: &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, functionType.Results)}},
	}
}

// A plain built-in SAM captures the caller's execution while a generated SAM
// receives it explicitly from its adapter. Stage the method value at reference
// creation, including the immediate null check for a bound receiver.
func plainSAMMethodReference(reference ast.Expr, functionType *ast.FuncType, ctx Ctx) ast.Expr {
	if reference == nil || methodReferenceUsesExecutionSAM(ctx) || functionType == nil || functionType.Params == nil || len(functionType.Params.List) == 0 {
		return reference
	}
	parameters := cloneFieldList(functionType.Params)
	parameters.List = parameters.List[1:]
	plainType := &ast.FuncType{Params: parameters, Results: cloneFieldList(functionType.Results)}
	execution := executionExpr(ctx)
	if execution == nil {
		execution = newExecutionExpr(ctx)
	}
	call := &ast.CallExpr{Fun: &ast.Ident{Name: "__java2goMethodReferenceTarget"}, Args: append([]ast.Expr{execution}, methodCallArgs(parameters)...)}
	return &ast.CallExpr{Fun: &ast.FuncLit{
		Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: plainType}}}},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.AssignStmt{Lhs: []ast.Expr{&ast.Ident{Name: "__java2goMethodReferenceTarget"}}, Tok: token.DEFINE, Rhs: []ast.Expr{reference}},
			&ast.ReturnStmt{Results: []ast.Expr{&ast.FuncLit{Type: plainType, Body: &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, plainType.Results)}}}}},
		}},
	}}
}

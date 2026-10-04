package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

const abstractMapStateAccessor = "AbstractMapDefaultsStateJava2go"

type abstractMapDefault struct {
	java, result, algorithm, dispatch, role string
	parameters                              []string
	rawResult                               bool
}

var abstractMapDefaults = []abstractMapDefault{
	{"size", "int", "AbstractMapSizeExecution", "MapSizeExecution", "MapSizeJava2goExecution", nil, false},
	{"isEmpty", "boolean", "AbstractMapIsEmptyExecution", "MapIsEmptyExecution", "MapIsEmptyJava2goExecution", nil, false},
	{"containsKey", "boolean", "AbstractMapContainsKeyExecution", "MapContainsKeyExecution", "MapContainsKeyJava2goExecution", []string{"java.lang.Object"}, false},
	{"containsValue", "boolean", "AbstractMapContainsValueExecution", "MapContainsValueExecution", "MapContainsValueJava2goExecution", []string{"java.lang.Object"}, false},
	{"get", "V", "AbstractMapGetExecution", "MapGetExecution", "MapGetJava2goExecution", []string{"java.lang.Object"}, true},
	{"remove", "V", "AbstractMapRemoveExecution", "MapRemoveExecution", "MapRemoveJava2goExecution", []string{"java.lang.Object"}, true},
	{"put", "V", "AbstractMapPutExecution", "MapPutExecution", "MapPutJava2goExecution", []string{"K", "V"}, true},
	{"clear", "void", "AbstractMapClearExecution", "MapClearExecution", "MapClearJava2goExecution", nil, false},
	{"keySet", "java.util.Set<K>", "AbstractMapKeySetExecution", "MapKeySetExecution", "MapKeySetJava2goExecution", nil, false},
	{"values", "java.util.Collection<V>", "AbstractMapValuesExecution", "MapValuesExecution", "MapValuesJava2goExecution", nil, false},
	{"equals", "boolean", "AbstractMapEqualsExecution", "", "", []string{"java.lang.Object"}, false},
	{"hashCode", "int", "AbstractMapHashCodeExecution", "", "", nil, false},
}

func canonicalAbstractMapOwner(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	if !strings.Contains(base, ".") {
		if _, found := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); found {
			return false
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	return known && owner == "java.util.AbstractMap"
}
func directAbstractMap(scope *symbol.ClassScope, ctx Ctx) bool {
	return scope != nil && !scope.IsInterface && canonicalAbstractMapOwner(scope.Superclass, classScopeCtx(scope, ctx))
}
func abstractMapSpecs(scope *symbol.ClassScope, ctx Ctx) []abstractMapDefault {
	if !directAbstractMap(scope, ctx) {
		return nil
	}
	_, args := parseJavaTypeString(scope.Superclass)
	bindings := map[string]string{"K": "java.lang.Object", "V": "java.lang.Object"}
	if len(args) == 2 {
		bindings["K"], bindings["V"] = args[0], args[1]
	}
	result := append([]abstractMapDefault(nil), abstractMapDefaults...)
	for index := range result {
		result[index].result = substituteJavaTypeParams(result[index].result, bindings)
		result[index].parameters = append([]string(nil), result[index].parameters...)
		for arg := range result[index].parameters {
			result[index].parameters[arg] = substituteJavaTypeParams(result[index].parameters[arg], bindings)
		}
	}
	return result
}
func prepareAbstractMapDefaults() {
	for _, scope := range allSourceClassScopes() {
		ctx := Ctx{currentClass: scope, currentFile: findFileScopeForClassScope(scope)}
		for _, spec := range abstractMapSpecs(scope, ctx) {
			if method, _ := sourceCollectionMethod(scope, spec.java, spec.parameters, ctx); method != nil {
				continue
			}
			method := &symbol.Definition{OriginalName: spec.java, Name: symbol.Uppercase(spec.java), OriginalType: spec.result, HasBody: true, RuntimeDefault: true}
			used := map[string]struct{}{}
			for _, name := range scope.GoTypeParameterNames() {
				used[name] = struct{}{}
			}
			for _, formal := range spec.parameters {
				name := synchronizedUniqueLocalName("__java2goMapArgument", used)
				used[name] = struct{}{}
				method.Parameters = append(method.Parameters, &symbol.Definition{OriginalName: name, Name: name, OriginalType: formal})
			}
			scope.Methods = append(scope.Methods, method)
		}
	}
}
func sourceHasAbstractMap(scope *symbol.ClassScope, ctx Ctx) bool {
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil && !seen[current]; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		seen[current] = true
		if directAbstractMap(current, ctx) {
			return true
		}
	}
	return false
}
func abstractMapDefaultSpec(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) (abstractMapDefault, bool) {
	if method == nil || !method.RuntimeDefault {
		return abstractMapDefault{}, false
	}
	for _, spec := range abstractMapSpecs(owner, ctx) {
		if spec.java == method.OriginalName {
			return spec, true
		}
	}
	return abstractMapDefault{}, false
}
func abstractMapReference(receiver ast.Expr, ctx Ctx) ast.Expr {
	return stdjavaCall(ctx, "MapReferenceView", receiver)
}
func abstractMapStateFieldName(scope *symbol.ClassScope) string {
	used := map[string]struct{}{}
	for _, field := range scope.Fields {
		used[field.Name] = struct{}{}
	}
	for _, method := range scope.Methods {
		used[method.Name] = struct{}{}
	}
	return synchronizedUniqueLocalName("__java2goAbstractMapDefaultsState", used)
}
func abstractMapStateField(ctx Ctx) *ast.Field {
	if !directAbstractMap(ctx.currentClass, ctx) {
		return nil
	}
	return &ast.Field{Names: []*ast.Ident{ast.NewIdent(abstractMapStateFieldName(ctx.currentClass))}, Type: stdjavaQualifiedExpr("AbstractMapDefaultsState", ctx)}
}
func abstractMapAlgorithmCall(spec abstractMapDefault, receiver ast.Expr, args []ast.Expr, execution ast.Expr, ctx Ctx) ast.Expr {
	arguments := append([]ast.Expr{execution, abstractMapReference(receiver, ctx)}, args...)
	if spec.java == "keySet" || spec.java == "values" {
		arguments = append(arguments, &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(abstractMapStateAccessor)}})
	}
	return stdjavaCall(ctx, spec.algorithm, arguments...)
}
func abstractMapResultProjection(raw ast.Expr, javaType string, ctx Ctx) ast.Expr {
	physical := javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
	if strings.TrimSpace(javaType) == "java.lang.Object" {
		return raw
	}
	if strings.TrimSpace(javaType) == "Object" {
		if _, binder := resolveReferenceTypeParameter(symbol.JavaType{Original: javaType}, ctx); !binder && qualifyDeclaredNominalReference(javaType, ctx) == "java.lang.Object" {
			return raw
		}
	}
	descriptor, known := javaTypeDescriptorExpr(javaType, ctx)
	if !known {
		descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{raw, descriptor})
}
func generateAbstractMapDefaultDecls(ctx Ctx) []ast.Decl {
	if !directAbstractMap(ctx.currentClass, ctx) {
		return nil
	}
	var declarations []ast.Decl
	receiverName := abstractCollectionReceiverName(ctx, nil)
	declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(abstractMapStateAccessor), Recv: abstractCollectionReceiver(ctx, receiverName), Type: &ast.FuncType{Params: &ast.FieldList{}, Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("AbstractMapDefaultsState", ctx)}}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiverName), &ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.SelectorExpr{X: ast.NewIdent(receiverName), Sel: ast.NewIdent(abstractMapStateFieldName(ctx.currentClass))}}}}}}})
	for _, method := range ctx.currentClass.Methods {
		spec, found := abstractMapDefaultSpec(method, ctx.currentClass, ctx)
		if !found {
			continue
		}
		params := &ast.FieldList{}
		var args []ast.Expr
		for _, param := range method.Parameters {
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(param.Name)}, Type: javaTypeStringToGoTypeExpr(param.OriginalType, inScopeTypeParameters(ctx), ctx)})
			args = append(args, ast.NewIdent(param.Name))
		}
		receiver := abstractCollectionReceiverName(ctx, params)
		execution := executionNameForParams(params, ctx.currentClass.GoTypeParameterNames()...)
		call := abstractMapAlgorithmCall(spec, ast.NewIdent(receiver), args, ast.NewIdent(execution), ctx)
		var results *ast.FieldList
		if spec.result != "void" {
			results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(method.OriginalType, inScopeTypeParameters(ctx), ctx)}}}
			if spec.rawResult {
				call = abstractMapResultProjection(call, method.OriginalType, ctx)
			}
		}
		decl := &ast.FuncDecl{Name: ast.NewIdent(method.Name), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Params: params, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiver), invocationClosureCallStatement(call, results)}}}
		declarations = append(declarations, buildExecutionAwareFuncDecls(decl, executionImplementationName(method, ctx.currentClass, ctx), execution, ctx)...)
	}
	return declarations
}

// Instantiate inherited declarations in the receiver's source view before
// matching overrides or projecting erased bridge parameters. Declaration-aware
// qualification preserves binders before their concrete substitution.
func abstractMapTypeInReceiver(javaType string, owner *symbol.ClassScope, ctx Ctx) string {
	qualified := qualifyDeclaredReferenceType(symbol.JavaType{Original: javaType}, classScopeCtx(owner, ctx))
	arguments := abstractMapAncestorArguments(ctx.currentClass, owner, ctx)
	return substituteJavaTypeParams(qualified, abstractMapArgumentBindings(owner, arguments))
}

func abstractMapArgumentBindings(owner *symbol.ClassScope, arguments []string) map[string]string {
	bindings := map[string]string{}
	for index, parameter := range owner.TypeParameters {
		if index >= len(arguments) {
			break
		}
		bindings[parameter.Name] = arguments[index]
		bindings[parameter.EmittedName()] = arguments[index]
	}
	return bindings
}

// Qualify each written superclass argument where it was declared, then carry
// the already qualified reference or binder through subsequent substitutions.
func abstractMapAncestorArguments(receiver, owner *symbol.ClassScope, ctx Ctx) []string {
	if receiver == nil || owner == nil {
		return nil
	}
	arguments := receiver.GoTypeParameterNames()
	seen := map[*symbol.ClassScope]bool{}
	for current := receiver; current != nil && !seen[current]; {
		if current == owner {
			return arguments
		}
		seen[current] = true
		parent := resolveSuperclassScopeInDeclaringContext(ctx, current)
		if parent == nil {
			return nil
		}
		_, written := parseJavaTypeString(current.Superclass)
		normalized := normalizeClassTypeArguments(parent, written, current, arguments)
		bindings := abstractMapArgumentBindings(current, arguments)
		next := make([]string, len(normalized))
		for index, argument := range normalized {
			declared := qualifyDeclaredReferenceType(symbol.JavaType{Original: argument}, classScopeCtx(current, ctx))
			next[index] = substituteJavaTypeParams(declared, bindings)
		}
		current, arguments = parent, next
	}
	return nil
}

func generateAbstractMapBridgeDecls(ctx Ctx) []ast.Decl {
	if ctx.currentClass == nil || ctx.currentClass.IsInterface || !sourceHasAbstractMap(ctx.currentClass, ctx) {
		return nil
	}
	contracts := append([]abstractMapDefault{{java: "entrySet", result: "java.util.Set", role: "MapEntrySetJava2goExecution"}}, abstractMapDefaults...)
	for current := ctx.currentClass; current != nil; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		if directAbstractMap(current, ctx) {
			specs := abstractMapSpecs(current, ctx)
			for index := range specs {
				for parameter := range specs[index].parameters {
					specs[index].parameters[parameter] = abstractMapTypeInReceiver(specs[index].parameters[parameter], current, ctx)
				}
			}
			contracts = append(contracts[:1], specs...)
			break
		}
	}
	var declarations []ast.Decl
	for _, contract := range contracts {
		if contract.role == "" {
			continue
		}
		var method *symbol.Definition
		var owner *symbol.ClassScope
		seen := map[*symbol.ClassScope]bool{}
		for current := ctx.currentClass; current != nil && !seen[current]; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
			seen[current] = true
			for _, candidate := range current.Methods {
				if candidate.OriginalName != contract.java || candidate.IsStatic || candidate.IsPrivate || !candidate.HasBody || len(candidate.Parameters) != len(contract.parameters) {
					continue
				}
				matching := true
				for index, formal := range contract.parameters {
					matching = matching && javaInferenceSameType(abstractMapTypeInReceiver(candidate.Parameters[index].OriginalType, current, ctx), formal, ctx)
				}
				if matching {
					method, owner = candidate, current
					break
				}
			}
			if method != nil {
				break
			}
		}
		if method == nil {
			continue
		}
		params := &ast.FieldList{}
		var args []ast.Expr
		used := map[string]struct{}{}
		for _, name := range ctx.currentClass.GoTypeParameterNames() {
			used[name] = struct{}{}
		}
		for _, parameter := range method.Parameters {
			name := synchronizedUniqueLocalName("__java2goMapArgument", used)
			used[name] = struct{}{}
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(name)}, Type: ast.NewIdent("any")})
			args = append(args, ast.NewIdent(name))
			if !method.RuntimeDefault && !javaInferenceSameType(abstractMapTypeInReceiver(parameter.OriginalType, owner, ctx), "java.lang.Object", ctx) {
				args[len(args)-1] = abstractMapResultProjection(args[len(args)-1], abstractMapTypeInReceiver(parameter.OriginalType, owner, ctx), ctx)
			}
		}
		receiver := abstractCollectionReceiverName(ctx, params)
		execution := executionNameForParams(params, ctx.currentClass.GoTypeParameterNames()...)
		params.List = append([]*ast.Field{executionParameterField(execution, ctx)}, params.List...)
		var call ast.Expr
		if spec, found := abstractMapDefaultSpec(method, owner, ctx); found {
			call = abstractMapAlgorithmCall(spec, ast.NewIdent(receiver), args, ast.NewIdent(execution), ctx)
		} else {
			var selected ast.Expr = ast.NewIdent(receiver)
			if classNeedsVirtualDispatch(owner, ctx) {
				selected = &ast.SelectorExpr{X: selected, Sel: ast.NewIdent(classDispatchFieldName(owner))}
			}
			call = &ast.CallExpr{Fun: &ast.SelectorExpr{X: selected, Sel: ast.NewIdent(executionImplementationName(method, owner, ctx))}, Args: append([]ast.Expr{ast.NewIdent(execution)}, args...)}
		}
		var results *ast.FieldList
		if contract.result != "void" {
			var result ast.Expr
			switch contract.java {
			case "get", "remove", "put":
				result = ast.NewIdent("any")
			case "entrySet", "keySet", "values":
				result = stdjavaQualifiedExpr("JavaIterable", ctx)
			default:
				result = javaTypeStringToGoTypeExpr(contract.result, inScopeTypeParameters(ctx), ctx)
			}
			results = &ast.FieldList{List: []*ast.Field{{Type: result}}}
		}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(contract.role), Recv: abstractCollectionReceiver(ctx, receiver), Type: &ast.FuncType{Params: params, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, results)}}})
	}
	return declarations
}
func abstractMapProtocolReservedSelector(name string) bool {
	if name == abstractMapStateAccessor {
		return true
	}
	for _, spec := range abstractMapDefaults {
		if spec.role != "" && spec.role == name {
			return true
		}
	}
	return name == "MapEntrySetJava2goExecution"
}

// Calls selected as inherited platform declarations bypass typed ABI shims.
// The result narrows only when its actual Java consumer requires that view.
func rewriteAbstractMapDefaultInvocation(invocation, object *sitter.Node, receiver ast.Expr, resolution *methodResolution, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	if resolution == nil || resolution.def == nil {
		return nil
	}
	spec, found := abstractMapDefaultSpec(resolution.def, resolution.owner, ctx)
	if !found || !spec.rawResult {
		return nil
	}
	var call ast.Expr
	if object != nil && object.Type() == "super" {
		call = abstractMapAlgorithmCall(spec, receiver, args, intrinsicExecutionExpr(ctx), ctx)
	} else {
		call = stdjavaCall(ctx, spec.dispatch, append([]ast.Expr{intrinsicExecutionExpr(ctx), abstractMapReference(receiver, ctx)}, args...)...)
	}
	logical, known := inferExprJavaType(invocation, ctx, source)
	if !known {
		logical = resolution.def.OriginalType
	}
	return mapErasedResultProjection(call, invocation, logical, ctx, source)
}

// One result convention is shared by source defaults and native adapters.
// Enclosing Object context never suppresses a nested receiver's own cast.
func mapErasedResultProjection(raw ast.Expr, invocation *sitter.Node, logicalResult string, ctx Ctx, source []byte) ast.Expr {
	parent := invocation.Parent()
	for parent != nil && parent.Type() == "parenthesized_expression" {
		parent = parent.Parent()
	}
	if parent != nil && parent.Type() == "expression_statement" {
		return raw
	}
	target := logicalResult
	if expectedTypeTargetsExpression(ctx, invocation) && strings.TrimSpace(ctx.expectedType) != "" {
		if _, primitive := javaPrimitiveType(ctx.expectedType); !primitive && !isVarKeywordType(ctx.expectedType) {
			target = ctx.expectedType
		}
	}
	if parent != nil && parent.Type() == "cast_expression" && parent.NamedChildCount() > 0 {
		castTarget := parent.NamedChild(0).Content(source)
		if _, primitive := javaPrimitiveType(castTarget); !primitive {
			target = castTarget
		}
	}
	// Primitive consumers still check the expression's boxed logical source view
	// before Java unboxing and widening. Reference consumers use their own target.
	return abstractMapResultProjection(raw, target, ctx)
}

func registerAbstractMapProtocolIntrinsics() {
	registerCanonicalMapPutIntrinsic()
	for _, spec := range abstractMapDefaults {
		if spec.dispatch == "" {
			continue
		}
		registerInstanceNodeIntrinsic("AbstractMap", spec.java, func(receiver ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(invocation) != len(spec.parameters) {
				return nil
			}
			args := intrinsicArgs(invocation.ChildByFieldName("object"), spec.java, source, ctx)
			call := stdjavaCall(ctx, spec.dispatch, append([]ast.Expr{intrinsicExecutionExpr(ctx), receiver}, args...)...)
			if !spec.rawResult {
				return call
			}
			logical, known := inferExprJavaType(invocation, ctx, source)
			if !known {
				logical = "java.lang.Object"
			}
			return mapErasedResultProjection(call, invocation, logical, ctx, source)
		})
	}
	registerInstanceIntrinsic("AbstractMap", "entrySet", func(receiver ast.Expr, args []ast.Expr, ctx Ctx) ast.Expr {
		if len(args) != 0 {
			return nil
		}
		return stdjavaCall(ctx, "MapEntrySetExecution", intrinsicExecutionExpr(ctx), receiver)
	})
}
func rewriteExplicitAbstractMapSuperInvocation(object *sitter.Node, methodName string, ctx Ctx, source []byte) ast.Expr {
	if object == nil || object.Type() != "super" || !directAbstractMap(ctx.currentClass, ctx) {
		return nil
	}
	for _, spec := range abstractMapSpecs(ctx.currentClass, ctx) {
		if spec.java != methodName {
			continue
		}
		invocation := object.Parent()
		if invocationArgumentCount(invocation) != len(spec.parameters) {
			return nil
		}
		definition := &symbol.Definition{OriginalName: spec.java, OriginalType: spec.result, RuntimeDefault: true, HasBody: true}
		for _, parameter := range spec.parameters {
			definition.Parameters = append(definition.Parameters, &symbol.Definition{OriginalType: parameter})
		}
		resolution := &methodResolution{def: definition, owner: ctx.currentClass}
		args, _ := parseResolvedInvocationArguments(resolution, invocation.ChildByFieldName("arguments"), source, ctx, spec.parameters, nil, invocation)
		call := abstractMapAlgorithmCall(spec, ast.NewIdent(ShortName(ctx.className)), args, intrinsicExecutionExpr(ctx), ctx)
		if !spec.rawResult {
			return call
		}
		return mapErasedResultProjection(call, invocation, spec.result, ctx, source)
	}
	return nil
}

package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

const (
	executionMethodSuffix = "Java2goExecution"
	executionParamBase    = "__java2goExecution"
)

// executionParameterName chooses a hidden parameter that cannot shadow any
// Java identifier in the declaration. The source-span name scan is shared with
// other lowering passes and includes parameters, locals, labels, and nested
// declarations.
func executionParameterName(node *sitter.Node, source []byte, ctx Ctx) string {
	root := node
	if ctx.localScope != nil && ctx.localScope.DeclarationNode != nil {
		root = ctx.localScope.DeclarationNode
	}
	used := affineLoopUsedNames(root, source, ctx)
	return synchronizedUniqueLocalName(executionParamBase, used)
}

func executionParameterField(name string, ctx Ctx) *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{{Name: name}},
		Type:  &ast.StarExpr{X: stdjavaQualifiedExpr("Execution", ctx)},
	}
}

func executionExpr(ctx Ctx) ast.Expr {
	if ctx.executionContextName == "" {
		return nil
	}
	return &ast.Ident{Name: ctx.executionContextName}
}

func newExecutionExpr(ctx Ctx) ast.Expr {
	return stdjavaCall(ctx, "NewExecution")
}

// executionImplementationName is stable across an ordinary override family.
// A concrete generic specialization is the one deliberate split: javac gives
// it both a narrow source body and an erased ancestor bridge, which Go cannot
// overload under one selector. Such a body receives its collision-safe exact
// name here; the bridge planner retains the ancestor's stable erased name.
// Checking source-level generated names globally keeps a user method such as
// FooJava2goExecution from colliding with either hidden implementation.
func executionImplementationName(def *symbol.Definition, owner *symbol.ClassScope) string {
	if def == nil {
		return ""
	}
	if selection, bridged := directOwnerSpecializedOverrideBridgeForMethod(owner, def, classScopeCtx(owner, Ctx{})); bridged {
		return directOwnerOverrideBridgeExactExecutionName(selection.bridge)
	}
	return collisionSafeExecutionIdentifier(def.Name+executionMethodSuffix, owner)
}

func collisionSafeExecutionIdentifier(base string, owner *symbol.ClassScope) string {
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += strconv.Itoa(suffix)
		}
		if !generatedIdentifierExists(candidate, owner) {
			return candidate
		}
	}
}

func generatedIdentifierExists(name string, owner *symbol.ClassScope) bool {
	matchesScope := func(scope *symbol.ClassScope) bool {
		if scope == nil {
			return false
		}
		if scope.Class != nil && scope.Class.Name == name {
			return true
		}
		for _, field := range scope.Fields {
			if field != nil && field.Name == name {
				return true
			}
		}
		for _, method := range scope.Methods {
			if method != nil && method.Name == name {
				return true
			}
		}
		for _, enumConstant := range scope.EnumConstants {
			if enumConstant.Name == name {
				return true
			}
		}
		return name == classDispatchTypeName(scope) ||
			name == classDispatchFieldName(scope) ||
			name == classSelfSetterName(scope) ||
			name == interfaceDefaultCarrierName(scope)
	}
	if matchesScope(owner) {
		return true
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			for _, top := range file.TopLevelClasses {
				if visitClassScopes(top, matchesScope) {
					return true
				}
			}
		}
	}
	return false
}

func prependExecutionArgument(ctx Ctx, args []ast.Expr) []ast.Expr {
	execution := executionExpr(ctx)
	if execution == nil {
		return args
	}
	result := make([]ast.Expr, 0, len(args)+1)
	result = append(result, execution)
	result = append(result, args...)
	return result
}

func prependExecutionMethodArgument(ctx Ctx, def *symbol.Definition, args []ast.Expr) []ast.Expr {
	if def == nil || def.DeclarationNode == nil {
		return args
	}
	return prependExecutionArgument(ctx, args)
}

func executionMethodCallName(def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	if def == nil {
		return ""
	}
	// Runtime/synthetic members (enum name/ordinal/valueOf, record accessors,
	// generated equality helpers) have no source declaration and are emitted only
	// in their public form. They contain no Java synchronized body to re-enter.
	if executionExpr(ctx) == nil || def.DeclarationNode == nil {
		return def.Name
	}
	return executionImplementationName(def, owner)
}

func executionConstructorImplementationName(name string, owner *symbol.ClassScope) string {
	return collisionSafeExecutionIdentifier(name+executionMethodSuffix, owner)
}

func executionConstructorWithSelfImplementationName(name string, owner *symbol.ClassScope) string {
	return collisionSafeExecutionIdentifier(constructorWithSelfName(name)+executionMethodSuffix, owner)
}

// constructorHasExecutionImplementation reports whether code generation emits
// a hidden execution-aware form for this constructor selection. Source-backed
// class constructors and synthesized default class constructors receive one.
// Synthetic constructors such as a record's canonical constructor are emitted
// public-only and must retain that ABI.
func constructorHasExecutionImplementation(def *symbol.Definition, scope *symbol.ClassScope) bool {
	if def != nil {
		return def.Constructor && def.DeclarationNode != nil && def.DeclarationNode.Type() == "constructor_declaration"
	}
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum || classHasExplicitConstructor(scope) {
		return false
	}
	declaration := scope.Class.DeclarationNode
	return declaration != nil && declaration.Type() == "class_declaration"
}

func executionFieldInitializerMethodName() string {
	return fieldInitMethodName + executionMethodSuffix
}

func executionStringMethodName(scope *symbol.ClassScope) string {
	return collisionSafeExecutionIdentifier("String"+executionMethodSuffix, scope)
}

func executionNameForParams(params *ast.FieldList, reservedNames ...string) string {
	used := make(map[string]struct{})
	for _, name := range reservedNames {
		if name != "" {
			used[name] = struct{}{}
		}
	}
	if params != nil {
		for _, field := range params.List {
			if field == nil {
				continue
			}
			for _, name := range field.Names {
				if name != nil {
					used[name.Name] = struct{}{}
				}
			}
		}
	}
	return synchronizedUniqueLocalName(executionParamBase, used)
}

func executionNameForClass(scope *symbol.ClassScope) string {
	used := make(map[string]struct{})
	if scope != nil {
		for _, typeParam := range scope.GoTypeParameterNames() {
			used[typeParam] = struct{}{}
		}
		for _, field := range scope.Fields {
			if field != nil {
				used[field.Name] = struct{}{}
			}
		}
		for _, method := range scope.Methods {
			if method != nil {
				used[method.Name] = struct{}{}
			}
		}
		if scope.IsInner {
			used[scope.EnclosingFieldName()] = struct{}{}
		}
	}
	return synchronizedUniqueLocalName(executionParamBase, used)
}

func executionMethodField(public *ast.Field, def *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) *ast.Field {
	if public == nil || def == nil {
		return nil
	}
	functionType, ok := public.Type.(*ast.FuncType)
	if !ok || functionType == nil {
		return nil
	}
	params := cloneFieldList(functionType.Params)
	if params == nil {
		params = &ast.FieldList{}
	}
	var reservedNames []string
	if owner != nil {
		reservedNames = append(reservedNames, owner.GoTypeParameterNames()...)
	}
	reservedNames = append(reservedNames, def.GoTypeParameterNames()...)
	executionName := executionNameForParams(params, reservedNames...)
	params.List = append([]*ast.Field{executionParameterField(executionName, ctx)}, params.List...)
	return &ast.Field{
		Names: []*ast.Ident{{Name: executionImplementationName(def, owner)}},
		Type: &ast.FuncType{
			Params:  params,
			Results: cloneFieldList(functionType.Results),
		},
	}
}

func executionParameterTypeExpr(def *symbol.Definition, index int, javaType string, typeParams []string, ctx Ctx) ast.Expr {
	typeExpr := javaTypeStringToGoTypeExpr(javaType, typeParams, ctx)
	if executionParameterIsVariadic(def, index) {
		return &ast.Ellipsis{Elt: typeExpr}
	}
	return typeExpr
}

func executionParameterIsVariadic(def *symbol.Definition, index int) bool {
	if def == nil || def.DeclarationNode == nil {
		return false
	}
	parameters := def.DeclarationNode.ChildByFieldName("parameters")
	if parameters != nil && index >= 0 && index < int(parameters.NamedChildCount()) {
		if parameter := parameters.NamedChild(index); parameter != nil && parameter.Type() == "spread_parameter" {
			return true
		}
	}
	return false
}

// definitionParameterJavaSignatureType restores the array dimension that Java
// assigns to a varargs declaration. Symbols intentionally keep the written
// element type (T for T...), while overload identity and fixed-arity invocation
// semantics use T[].
func definitionParameterJavaSignatureType(def *symbol.Definition, index int) string {
	if def == nil || index < 0 || index >= len(def.Parameters) || def.Parameters[index] == nil {
		return ""
	}
	javaType := def.Parameters[index].OriginalType
	if executionParameterIsVariadic(def, index) {
		javaType += "[]"
	}
	return javaType
}

// markVariadicForwardCall expands the final slice when a generated forwarding
// call targets a Java varargs declaration. The forwarded parameter is already
// represented as a Go slice in the wrapper or closure; omitting Ellipsis would
// pass that slice as one vararg element (and usually fail to type-check).
func markVariadicForwardCall(call *ast.CallExpr, def *symbol.Definition) {
	if call == nil || def == nil || len(def.Parameters) == 0 {
		return
	}
	if executionParameterIsVariadic(def, len(def.Parameters)-1) {
		call.Ellipsis = token.Pos(1)
	}
}

func executionCompanionInterfaceName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	base := scope.Class.Name + executionMethodSuffix
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += strconv.Itoa(suffix)
		}
		if !generatedIdentifierExists(candidate, scope) {
			return candidate
		}
	}
}

// generateExecutionCompanionInterface keeps hidden token methods out of the
// public Java interface ABI. Generated implementations satisfy this companion;
// interface calls use a type assertion and fall back to the public method for
// handwritten Go implementations.
func generateExecutionCompanionInterface(scope *symbol.ClassScope, ctx Ctx) ast.Decl {
	if scope == nil || scope.Class == nil || (!scope.IsInterface && !scope.IsAbstract) {
		return nil
	}
	typeParams := scope.TypeParameterNames()
	methods := &ast.FieldList{}
	for _, method := range scope.Methods {
		if method == nil || method.Constructor || method.IsStatic || method.IsPrivate || method.RequiresHelper {
			continue
		}
		params := &ast.FieldList{}
		for index, parameter := range method.Parameters {
			parameterType := executionParameterTypeExpr(method, index, parameter.OriginalType, typeParams, ctx)
			parameterType = rawUnboundReceiverParameterType(scope, method, index, parameter.OriginalType, parameterType, ctx)
			params.List = append(params.List, &ast.Field{
				Names: []*ast.Ident{{Name: parameter.Name}},
				Type:  parameterType,
			})
		}
		var results *ast.FieldList
		if method.OriginalType != "" && method.OriginalType != "void" {
			results = &ast.FieldList{List: []*ast.Field{{
				Type: javaTypeStringToGoTypeExpr(method.OriginalType, typeParams, ctx),
			}}}
		}
		public := &ast.Field{Type: &ast.FuncType{Params: params, Results: results}}
		if hidden := executionMethodField(public, method, scope, ctx); hidden != nil {
			methods.List = append(methods.List, hidden)
		}
	}
	if len(methods.List) == 0 {
		return nil
	}
	return genInterfaceInContext(executionCompanionInterfaceName(scope), methods, scope.TypeParameters, ctx)
}

func executionWrapperBody(fun ast.Expr, params, results *ast.FieldList, ctx Ctx) *ast.BlockStmt {
	args := append([]ast.Expr{newExecutionExpr(ctx)}, methodCallArgs(params)...)
	call := &ast.CallExpr{Fun: fun, Args: args}
	if params != nil && len(params.List) > 0 {
		if _, variadic := params.List[len(params.List)-1].Type.(*ast.Ellipsis); variadic {
			call.Ellipsis = token.Pos(1)
		}
	}
	if results != nil && len(results.List) > 0 {
		return &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}
	}
	return &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: call}}}
}

// buildExecutionAwareFuncDecls preserves the existing Go-facing declaration as
// a fresh-execution wrapper and emits a hidden implementation whose first
// parameter receives the caller's logical Java execution. This avoids changing
// the public ABI used by handwritten Go tests and runtime adapters.
func buildExecutionAwareFuncDecls(
	declaration *ast.FuncDecl,
	implementationName string,
	executionName string,
	ctx Ctx,
) []ast.Decl {
	if declaration == nil || declaration.Name == nil || declaration.Type == nil || implementationName == "" || executionName == "" {
		if declaration == nil {
			return nil
		}
		return []ast.Decl{declaration}
	}

	publicParams := cloneFieldList(declaration.Type.Params)
	publicResults := cloneFieldList(declaration.Type.Results)
	publicTypeParams := cloneFieldList(declaration.Type.TypeParams)
	implementationParams := cloneFieldList(declaration.Type.Params)
	if implementationParams == nil {
		implementationParams = &ast.FieldList{}
	}
	implementationParams.List = append(
		[]*ast.Field{executionParameterField(executionName, ctx)},
		implementationParams.List...,
	)

	implementation := &ast.FuncDecl{
		Name: &ast.Ident{Name: implementationName},
		Recv: cloneFieldList(declaration.Recv),
		Type: &ast.FuncType{
			TypeParams: cloneFieldList(declaration.Type.TypeParams),
			Params:     implementationParams,
			Results:    cloneFieldList(declaration.Type.Results),
		},
		Body: declaration.Body,
	}

	var implementationFun ast.Expr = &ast.Ident{Name: implementationName}
	if declaration.Recv != nil && len(declaration.Recv.List) > 0 && len(declaration.Recv.List[0].Names) > 0 {
		implementationFun = &ast.SelectorExpr{
			X:   &ast.Ident{Name: declaration.Recv.List[0].Names[0].Name},
			Sel: &ast.Ident{Name: implementationName},
		}
	}
	if typeArgs := fieldListNames(publicTypeParams); len(typeArgs) > 0 {
		implementationFun = applyTypeArguments(implementationFun, typeArgs)
	}

	wrapper := &ast.FuncDecl{
		Doc:  declaration.Doc,
		Name: &ast.Ident{Name: declaration.Name.Name},
		Recv: cloneFieldList(declaration.Recv),
		Type: &ast.FuncType{
			TypeParams: publicTypeParams,
			Params:     publicParams,
			Results:    publicResults,
		},
		Body: executionWrapperBody(implementationFun, publicParams, publicResults, ctx),
	}

	return []ast.Decl{wrapper, implementation}
}

func fieldListNames(fields *ast.FieldList) []ast.Expr {
	if fields == nil {
		return nil
	}
	var names []ast.Expr
	for _, field := range fields.List {
		if field == nil {
			continue
		}
		for _, name := range field.Names {
			if name != nil && name.Name != "" {
				names = append(names, &ast.Ident{Name: name.Name})
			}
		}
	}
	return names
}

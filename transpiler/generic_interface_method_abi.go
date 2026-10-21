package transpiler

import (
	"fmt"
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"strings"
)

// A Java method binder belongs to the declaration, not to a Go method set.
// All entry producers use the same declaration-bound descriptor and physical
// carrier proof. The Java types remain separate from their Go representations:
// a nested Java argument never authorizes a runtime check of that argument.
// Plans are local to an emission; they do not outlive the source symbol graph.
type genericMethodEntryABI struct {
	owner              *symbol.ClassScope
	method             *symbol.Definition
	methodDeclarations []*symbol.TypeParamDeclaration
	parameterJavaTypes []string
	resultJavaType     string
	parameterTypes     []ast.Expr
	resultType         ast.Expr
}

func planGenericMethodEntryABI(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *genericMethodEntryABI {
	if owner == nil || method == nil || !genericMethodHasErasedEntry(method) {
		return nil
	}
	declared := false
	for _, candidate := range owner.Methods {
		declared = declared || candidate == method
	}
	if !declared {
		return nil
	}
	declaring := classScopeCtx(owner, ctx).Clone()
	declaring.localScope = method
	plan := &genericMethodEntryABI{owner: owner, method: method}
	for _, parameter := range method.TypeParameters {
		plan.methodDeclarations = append(plan.methodDeclarations, parameter.Declaration)
	}
	for _, parameter := range method.Parameters {
		erased := genericMethodErasedJavaType(method, parameter.OriginalType)
		plan.parameterJavaTypes = append(plan.parameterJavaTypes, erased)
		var physical ast.Expr
		if plan.implicitObjectScalar(parameter) {
			physical = &ast.Ident{Name: "any"}
		} else {
			physical = javaTypeStringToGoTypeExpr(erased, owner.TypeParameterNames(), declaring)
		}
		plan.parameterTypes = append(plan.parameterTypes, physical)
	}
	plan.resultJavaType = genericMethodErasedJavaType(method, method.OriginalType)
	if plan.resultJavaType != "" && plan.resultJavaType != "void" {
		if plan.implicitObjectScalar(method) {
			plan.resultType = &ast.Ident{Name: "any"}
		} else {
			plan.resultType = javaTypeStringToGoTypeExpr(plan.resultJavaType, owner.TypeParameterNames(), declaring)
		}
	}
	return plan
}

func (plan *genericMethodEntryABI) apply(params, results *ast.FieldList) {
	if plan == nil {
		return
	}
	for index, typ := range plan.parameterTypes {
		params.List[index].Type = typ
	}
	if results != nil && len(results.List) > 0 && plan.resultType != nil {
		results.List[0].Type = plan.resultType
	}
}

// Only an exact method declaration can authorize the erased entry's nullable
// superclass argument view. All other argument conversions retain their normal
// path, and each operand is parsed and evaluated once in Java argument order.
func (plan *genericMethodEntryABI) parseInvocationArguments(argsNode *sitter.Node, source []byte, ctx Ctx, expectedTypes []string, invocation *sitter.Node) []ast.Expr {
	if argsNode == nil {
		return nil
	}
	nodes := nodeutil.SemanticNamedChildrenOf(argsNode)
	effects := javaLaterExpressionEffects(nodes, source, ctx)
	args := make([]ast.Expr, 0, len(nodes))
	for index, node := range nodes {
		argCtx := ctx.Clone()
		expected := ""
		if index < len(expectedTypes) {
			expected = expectedTypes[index]
		}
		if strings.TrimSpace(expected) != "" {
			argCtx.expectedType = expected
			argCtx.expectedTypeRoot = node
		}
		implicitObject := index < len(plan.method.Parameters) && plan.implicitObjectScalar(plan.method.Parameters[index])
		if implicitObject {
			// The canonical unwritten bound is a physical any consumer, not a
			// source-written Object target in this caller's lexical namespace.
			argCtx.expectedType = ""
			argCtx.expectedTypeRoot = nil
			argCtx.genericImplicitArgument = plan.implicitObjectArgument(index, node, argCtx)
		}
		value := ParseExpr(node, source, argCtx)
		projected := false
		var boundPhysical ast.Expr
		if implicitObject {
			if actualType, known := inferExprJavaType(node, argCtx, source); known {
				if primitive, ok := javaPrimitiveType(actualType); ok {
					value = javaBoxExpr(value, primitive, ctx)
				}
			}
			projected = true
		}
		if index < len(plan.method.Parameters) && index < len(plan.parameterJavaTypes) {
			boundPhysical = plan.sourceBoundPhysicalType(plan.method.Parameters[index], plan.parameterJavaTypes[index], plan.parameterTypes[index], invocation, source, ctx)
			declared := definitionDeclaredJavaType(plan.method.Parameters[index])
			base, rank := javaArrayTypeParts(strings.TrimSpace(declared.Original))
			binder := declared.TypeParameterBindings[base]
			methodOwned := false
			for _, declaration := range plan.methodDeclarations {
				methodOwned = methodOwned || binder != nil && binder == declaration
			}
			actualType, _ := inferExprJavaType(node, ctx, source)
			actual := resolveClassScopeByQualifiedName(ctx, actualType)
			expectedScope := resolveClassScopeByQualifiedName(ctx, expected)
			if !implicitObject && boundPhysical != nil && methodOwned && rank == 0 && expected == plan.parameterJavaTypes[index] && actual != nil && expectedScope != nil && !expectedScope.IsInterface && classInheritsFrom(actual, expectedScope, ctx) {
				value = nullableSuperclassEdgeView(value, actualType, expected, actual, expectedScope, ctx)
				value.(*ast.CallExpr).Fun.(*ast.FuncLit).Type.Results.List[0].Type = boundPhysical
				projected = true
			}
		}
		if boundPhysical != nil && isStaticallyNullReference(node) {
			value = zeroValueForType(boundPhysical)
			projected = true
		}
		if !projected {
			if expected != "" && usesNullableValueStorage(expected) && expressionAlwaysProducesNull(node) {
				if isJavaStringType(expected) {
					value = javaStringNullExpr(ctx)
				} else {
					value = zeroValueForType(javaTypeStringToGoTypeExpr(expected, inScopeTypeParameters(ctx), ctx))
				}
			}
			value = coerceArgumentToExpectedType(value, node, expected, ctx, source)
		}
		if effects[index] {
			if implicitObject && index < len(plan.parameterTypes) {
				if snapshotJavaExpressionValue(value, ctx) != value {
					value = stdjavaGenericCall(ctx, "EvaluationValue", []ast.Expr{plan.parameterTypes[index]}, []ast.Expr{value})
				}
			} else {
				valueType := expected
				if valueType == "" {
					valueType, _ = inferExprJavaType(node, ctx, source)
				}
				value = snapshotJavaExpressionValueForType(value, valueType, ctx)
			}
		}
		args = append(args, value)
	}
	return args
}

// The bound's nominal declaration is resolved in the callee, while its Go
// package reference and receiver substitutions belong to the caller. A written
// bound name must never be reparsed as an unrelated caller type variable.
func (plan *genericMethodEntryABI) sourceBoundPhysicalType(member *symbol.Definition, erasedJavaType string, physical ast.Expr, invocation *sitter.Node, source []byte, ctx Ctx) ast.Expr {
	if plan.implicitObjectScalar(member) {
		return &ast.Ident{Name: "any"}
	}
	declared := definitionDeclaredJavaType(member)
	base, rank := javaArrayTypeParts(strings.TrimSpace(declared.Original))
	if rank != 0 {
		return nil
	}
	binder := declared.TypeParameterBindings[base]
	ownedBound := false
	for _, parameter := range plan.method.TypeParameters {
		if binder != nil && binder == parameter.Declaration && len(parameter.Bounds) > 0 {
			ownedBound = true
		}
	}
	if !ownedBound {
		return nil
	}
	erasedBase, erasedRank := javaArrayTypeParts(erasedJavaType)
	if erasedRank != 0 {
		return nil
	}
	declaring := classScopeCtx(plan.owner, ctx)
	declaring.localScope = plan.method
	nominalBase, _ := parseJavaTypeString(erasedBase)
	bound := resolveClassScopeByQualifiedName(declaring, nominalBase)
	if bound == nil || bound.Class == nil || bound.IsInterface {
		return nil
	}
	pointer, ok := physical.(*ast.StarExpr)
	if !ok {
		return nil
	}
	head := pointer.X
	var args []ast.Expr
	switch indexed := head.(type) {
	case *ast.IndexExpr:
		head = indexed.X
		args = []ast.Expr{indexed.Index}
	case *ast.IndexListExpr:
		head = indexed.X
		args = append([]ast.Expr(nil), indexed.Indices...)
	}
	// The planned head must actually denote this source class. Other physical
	// services retain their existing representation and consumer path.
	matches := false
	switch named := head.(type) {
	case *ast.Ident:
		matches = named.Name == bound.Class.Name
	case *ast.SelectorExpr:
		matches = named.Sel.Name == bound.Class.Name
	}
	if !matches {
		return nil
	}
	if len(args) > 0 && len(plan.owner.TypeParameters) > 0 {
		var target *invocationTargetInfo
		if invocation != nil {
			if object := invocation.ChildByFieldName("object"); object != nil {
				target = resolveInvocationTarget(object, ctx, source)
			}
		}
		if target == nil && ctx.currentClass != nil {
			target = &invocationTargetInfo{classScope: ctx.currentClass, classTypeArgs: typeParamExprs(ctx.currentClass.GoTypeParameterNames())}
		}
		if target != nil {
			ownerArgs := target.classTypeArgs
			if target.classScope != plan.owner {
				ownerArgs = mapClassTypeArgsToAncestor(target.classScope, ownerArgs, plan.owner, ctx)
			}
			if len(ownerArgs) == len(plan.owner.TypeParameters) {
				bindings := map[string]ast.Expr{}
				for index, parameter := range plan.owner.TypeParameters {
					bindings[parameter.EmittedName()] = ownerArgs[index]
				}
				for index, argument := range args {
					args[index] = substituteSourceClassGoArgument(argument, bindings)
				}
			}
		}
	}
	return classSubobjectPointerTypeWithGoArguments(bound, args, ctx)
}

// An unwritten first bound belongs to Java's Object declaration regardless of
// caller/source names. Only the exact scalar method binder authorizes this ABI.
func (plan *genericMethodEntryABI) implicitObjectScalar(member *symbol.Definition) bool {
	declared := definitionDeclaredJavaType(member)
	base, rank := javaArrayTypeParts(strings.TrimSpace(declared.Original))
	if rank != 0 {
		return false
	}
	binder := declared.TypeParameterBindings[base]
	slots, _, known := declaredGenericMethodPhysicalSlots(plan.method)
	if !known {
		return false
	}
	for _, slot := range slots {
		if binder != nil && binder == slot.parameter.Declaration && slot.implicitObject {
			return true
		}
	}
	return false
}

// A polymorphic anonymous implementation owns distinct method declarations from
// its interface. This plan is confined to that body; it never supplies a binder
// to an unrelated callback, class method, or nested source method.
type genericInterfaceMethodBodyABI struct {
	entry          *genericMethodEntryABI
	implementation *symbol.Definition
	emissionOwner  *symbol.ClassScope
	declarations   map[*symbol.TypeParamDeclaration]bool
}

func planGenericInterfaceMethodBodyABI(entry *genericMethodEntryABI, implementation *symbol.Definition, declaration *sitter.Node, ctx Ctx) *genericInterfaceMethodBodyABI {
	if entry == nil || entry.owner == nil || !entry.owner.IsInterface || implementation == nil || implementation.DeclarationNode == nil || declaration == nil || !implementation.DeclarationNode.Equal(declaration) || declaration.Type() != "method_declaration" {
		return nil
	}
	if parent := declaration.Parent(); parent == nil || parent.Type() != "class_body" {
		return nil
	}
	owned := false
	for _, candidate := range entry.owner.Methods {
		owned = owned || candidate == entry.method
	}
	if !owned {
		return nil
	}
	if len(entry.method.TypeParameters) != len(implementation.TypeParameters) || len(entry.method.Parameters) != len(implementation.Parameters) {
		return nil
	}
	plan := &genericInterfaceMethodBodyABI{entry: entry, implementation: implementation, emissionOwner: ctx.currentClass, declarations: map[*symbol.TypeParamDeclaration]bool{}}
	for index, parameter := range entry.method.TypeParameters {
		actual := implementation.TypeParameters[index]
		if parameter.Declaration == nil || actual.Declaration == nil || len(parameter.Bounds) != 0 || len(actual.Bounds) != 0 {
			return nil
		}
		plan.declarations[actual.Declaration] = true
	}
	alphaType := func(member *symbol.Definition, parameters []symbol.TypeParam) string {
		declared := definitionDeclaredJavaType(member)
		bindings := map[string]string{}
		for name, declaration := range declared.TypeParameterBindings {
			for index, parameter := range parameters {
				if declaration != nil && declaration == parameter.Declaration {
					bindings[name] = fmt.Sprintf("__method_slot_%d", index)
				}
			}
		}
		return substituteJavaTypeParameters(declared.Original, bindings)
	}
	if alphaType(entry.method, entry.method.TypeParameters) != alphaType(implementation, implementation.TypeParameters) {
		return nil
	}
	for index, parameter := range entry.method.Parameters {
		if alphaType(parameter, entry.method.TypeParameters) != alphaType(implementation.Parameters[index], implementation.TypeParameters) {
			return nil
		}
	}

	return plan
}

func (plan *genericInterfaceMethodBodyABI) scalar(javaType string, ctx Ctx) bool {
	if plan == nil || ctx.erasedGenericMethodBody != plan.implementation || ctx.localScope != plan.implementation || ctx.currentClass != plan.emissionOwner {
		return false
	}
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	name, args := parseJavaTypeString(base)
	if rank != 0 || len(args) != 0 || strings.ContainsAny(name, ".?<>") || strings.TrimSpace(name) != strings.TrimSpace(javaType) {
		return false
	}
	declaration := visibleTypeParameterDeclarationForJavaType(name, ctx)
	return declaration != nil && plan.declarations[declaration]
}

func (plan *genericInterfaceMethodBodyABI) apply(params, results *ast.FieldList) {
	if plan == nil {
		return
	}
	for index, parameter := range plan.implementation.Parameters {
		declared := definitionDeclaredJavaType(parameter)
		base, rank := javaArrayTypeParts(strings.TrimSpace(declared.Original))
		if rank == 0 && plan.declarations[declared.TypeParameterBindings[base]] {
			params.List[index].Type = &ast.Ident{Name: "any"}
		}
	}
	declared := definitionDeclaredJavaType(plan.implementation)
	base, rank := javaArrayTypeParts(strings.TrimSpace(declared.Original))
	if results != nil && len(results.List) > 0 && rank == 0 && plan.declarations[declared.TypeParameterBindings[base]] {
		results.List[0].Type = &ast.Ident{Name: "any"}
	}
}

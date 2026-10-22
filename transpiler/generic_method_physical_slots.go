package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A Java method variable has one physical first-bound representation for its
// body. The inferred Java view belongs to each consumer, not to that body.
// Plans retain declaration identity and end with this source graph/emission.
type genericMethodPhysicalSlot struct {
	parameter      symbol.TypeParam
	javaType       string
	implicitObject bool
}

func declaredGenericMethodPhysicalSlots(def *symbol.Definition) ([]genericMethodPhysicalSlot, Ctx, bool) {
	declaring, known := genericMethodOwnerContext(def)
	if !known || def == nil || def.DeclarationNode == nil || len(def.TypeParameters) == 0 {
		return nil, declaring, false
	}
	declaring.localScope = def
	declaring.syntheticTypeParameters = nil
	parameters := qualifyTypeParameterBounds(def.TypeParameters, declaring)
	lookup := newTypeParameterLookup(parameters)
	visiting := map[*symbol.TypeParamDeclaration]bool{}
	var erase func(symbol.TypeParam) (string, bool, bool)
	erase = func(parameter symbol.TypeParam) (string, bool, bool) {
		if parameter.Declaration == nil || visiting[parameter.Declaration] || len(parameter.Bounds) > 1 {
			return "", false, false
		}
		if len(parameter.Bounds) == 0 {
			return "java.lang.Object", true, true
		}
		visiting[parameter.Declaration] = true
		defer delete(visiting, parameter.Declaration)
		bound := parameter.Bounds[0]
		base, args := parseJavaTypeString(strings.TrimSpace(bound.Original))
		if len(args) != 0 || strings.TrimSpace(base) == "" {
			return "", false, false
		}
		if next, found := lookup.resolve(bound, strings.TrimSpace(base)); found {
			return erase(next)
		}
		// A written bound retains its declaration context, even if named Object.
		return strings.TrimSpace(base), false, true
	}
	slots := make([]genericMethodPhysicalSlot, 0, len(parameters))
	for _, parameter := range parameters {
		javaType, implicit, ok := erase(parameter)
		if !ok {
			return nil, declaring, false
		}
		slots = append(slots, genericMethodPhysicalSlot{parameter: parameter, javaType: javaType, implicitObject: implicit})
	}
	return slots, declaring, true
}

func genericMethodPhysicalTypeArguments(def *symbol.Definition, ctx Ctx) ([]ast.Expr, []string, bool) {
	slots, declaring, known := declaredGenericMethodPhysicalSlots(def)
	if !known {
		return nil, nil, false
	}
	args := make([]ast.Expr, 0, len(slots))
	javaTypes := make([]string, 0, len(slots))
	for _, slot := range slots {
		javaTypes = append(javaTypes, slot.javaType)
		if slot.implicitObject {
			args = append(args, ast.NewIdent("any"))
			continue
		}
		physical := javaTypeStringToGoTypeExpr(slot.javaType, inScopeTypeParameters(declaring), declaring)
		if scope := resolveClassScopeByQualifiedName(declaring, slot.javaType); scope != nil && scope.Class != nil {
			if scope.IsInterface {
				// The slot belongs to the declaration, but its named Go type is
				// consumed in the caller's package. Preserve any existing physical
				// type arguments while qualifying that same owned interface.
				head := qualifiedNameExpr(scope.Class.Name, findJavaPackageForClassScope(scope), ctx)
				switch expression := physical.(type) {
				case *ast.Ident, *ast.SelectorExpr:
					physical = head
				case *ast.IndexExpr:
					physical = &ast.IndexExpr{X: head, Index: expression.Index}
				case *ast.IndexListExpr:
					physical = &ast.IndexListExpr{X: head, Indices: expression.Indices}
				}
			} else if pointer, ok := physical.(*ast.StarExpr); ok {
				var parameters []ast.Expr
				switch head := pointer.X.(type) {
				case *ast.IndexExpr:
					parameters = []ast.Expr{head.Index}
				case *ast.IndexListExpr:
					parameters = head.Indices
				}
				physical = classSubobjectPointerTypeWithGoArguments(scope, parameters, ctx)
			}
		}
		args = append(args, physical)
	}
	return args, javaTypes, true
}

// Static functions have no virtual helper eligibility. Prove their existing
// signature carriers separately; this does not widen instance entry admission.
func planStaticGenericMethodPhysicalABI(owner *symbol.ClassScope, def *symbol.Definition, ctx Ctx) *genericMethodEntryABI {
	if owner == nil || def == nil || !def.IsStatic || def.IsPrivate || def.Constructor || executionParameterIsVariadic(def, len(def.Parameters)-1) {
		return nil
	}
	declaring, known := genericMethodOwnerContext(def)
	if !known || declaring.currentClass != owner || !canonicalGenericMethodSignature(def, declaring) {
		return nil
	}
	if _, _, ok := declaredGenericMethodPhysicalSlots(def); !ok {
		return nil
	}
	declaring.localScope = def
	plan := &genericMethodEntryABI{owner: owner, method: def}
	for _, parameter := range def.TypeParameters {
		plan.methodDeclarations = append(plan.methodDeclarations, parameter.Declaration)
	}
	for _, parameter := range def.Parameters {
		erased := genericMethodErasedJavaType(def, parameter.OriginalType)
		plan.parameterJavaTypes = append(plan.parameterJavaTypes, erased)
		physical := javaTypeStringToGoTypeExpr(erased, inScopeTypeParameters(declaring), declaring)
		if plan.implicitObjectScalar(parameter) {
			physical = ast.NewIdent("any")
		}
		plan.parameterTypes = append(plan.parameterTypes, physical)
	}
	plan.resultJavaType = genericMethodErasedJavaType(def, def.OriginalType)
	if plan.resultJavaType != "" && plan.resultJavaType != "void" {
		plan.resultType = javaTypeStringToGoTypeExpr(plan.resultJavaType, inScopeTypeParameters(declaring), declaring)
		if plan.implicitObjectScalar(def) {
			plan.resultType = ast.NewIdent("any")
		}
	}
	return plan
}

func staticGenericPhysicalInvocation(resolution *methodResolution, invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte, qualifier ast.Expr) ast.Expr {
	if resolution == nil || planStaticGenericMethodPhysicalABI(resolution.owner, resolution.def, ctx) == nil {
		return nil
	}
	physical, javaTypes, ok := genericMethodPhysicalTypeArguments(resolution.def, ctx)
	if !ok {
		return nil
	}
	if count := len(concreteDependentTypeWitnessEdges(resolution.def, ctx)); count > 0 {
		if len(args) < count {
			return nil
		}
		witnesses := dependentTypeWitnessArgumentsForJavaTypes(resolution.def, javaTypes, ctx)
		if len(witnesses) != count {
			return nil
		}
		args = append(witnesses, args[count:]...)
	}
	fun := qualifiedNameExpr(executionMethodCallName(resolution.def, resolution.owner, ctx), findJavaPackageForClassScope(resolution.owner), ctx)
	call := ast.Expr(&ast.CallExpr{Fun: applyTypeArguments(fun, physical), Args: prependExecutionMethodArgument(ctx, resolution.def, args)})
	if qualifier != nil {
		if staged := stageStaticInvocationQualifier(invocation, qualifier, resolution, call, ctx, source); staged != nil {
			call = staged
		}
	}
	if parent := invocation.Parent(); parent != nil && parent.Type() == "expression_statement" {
		return call
	}
	if ctx.rawFunctionTarget.resultBoundary(invocation, ctx) {
		return call
	}
	bindings := genericJavaInvocationTypeBindings(resolution.def, invocation, ctx, source)
	if ctx.genericImplicitArgument.defaultResult(resolution.def, bindings, invocation, ctx) {
		return call
	}
	consuming := ""
	if expectedTypeTargetsExpression(ctx, invocation) {
		consuming = ctx.expectedType
	}
	if genericMethodImplicitDefaultResult(resolution.def, bindings, consuming, ctx) {
		return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{ast.NewIdent("any")}, []ast.Expr{call, stdjavaQualifiedExpr("ObjectTypeID", ctx)})
	}
	logical := make([]ast.Expr, len(resolution.def.TypeParameters))
	for index, parameter := range resolution.def.TypeParameters {
		javaType := bindings[parameter.Name]
		if javaType == "" {
			javaType = genericMethodErasedJavaType(resolution.def, parameter.EmittedName())
		}
		logical[index] = javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
	}
	return genericMethodProjectedResult(call, resolution.def, logical, bindings, consuming, ctx)
}

// Preserve checkcast at the Java body location. The method's scalar carrier
// is fixed by its declared bound, while a source subclass needs a nominal view
// rather than a Go pointer assertion on that carrier.
func genericMethodPhysicalScalarCast(value ast.Expr, target string, ctx Ctx) ast.Expr {
	def := ctx.localScope
	owner := ctx.currentClass
	if def == nil || owner == nil {
		return nil
	}
	if def.IsStatic {
		if planStaticGenericMethodPhysicalABI(owner, def, ctx) == nil {
			return nil
		}
	} else if planGenericMethodEntryABI(owner, def, ctx) == nil {
		return nil
	}
	base, rank := javaArrayTypeParts(target)
	_, arguments := parseJavaTypeString(base)
	if rank != 0 || len(arguments) != 0 {
		return nil
	}
	declaration := visibleTypeParameterDeclarationForJavaType(target, ctx)
	if declaration == nil {
		return nil
	}
	slots, declaring, ok := declaredGenericMethodPhysicalSlots(def)
	if !ok {
		return nil
	}
	for _, slot := range slots {
		if slot.parameter.Declaration != declaration {
			continue
		}
		descriptor := ast.Expr(stdjavaQualifiedExpr("ObjectTypeID", ctx))
		if !slot.implicitObject {
			var found bool
			descriptor, found = javaTypeDescriptorExpr(slot.javaType, declaring)
			if !found {
				return nil
			}
		}
		physical := javaTypeStringToGoTypeExpr(target, inScopeTypeParameters(ctx), ctx)
		return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{value, descriptor})
	}
	return nil
}

// Only a proven implicit method-slot argument supplies this canonical Object
// consumer. A default method result at its root may retain physical any; an
// explicit or inferred Java binding still performs its ordinary checkcast.
type genericMethodImplicitArgumentABI struct {
	entry     *genericMethodEntryABI
	parameter *symbol.Definition
	root      *sitter.Node
	file      *symbol.FileScope
	owner     *symbol.ClassScope
	scope     *symbol.Definition
}

func (plan *genericMethodEntryABI) implicitObjectArgument(index int, node *sitter.Node, ctx Ctx) *genericMethodImplicitArgumentABI {
	if plan == nil || plan.method == nil || index < 0 || index >= len(plan.method.Parameters) || node == nil || ctx.currentFile == nil || ctx.currentClass == nil || ctx.localScope == nil {
		return nil
	}
	parameter := plan.method.Parameters[index]
	if !plan.implicitObjectScalar(parameter) {
		return nil
	}
	return &genericMethodImplicitArgumentABI{entry: plan, parameter: parameter, root: node, file: ctx.currentFile, owner: ctx.currentClass, scope: ctx.localScope}
}

func (plan *genericMethodImplicitArgumentABI) defaultResult(def *symbol.Definition, bindings map[string]string, node *sitter.Node, ctx Ctx) bool {
	if plan == nil || ctx.currentFile != plan.file || ctx.currentClass != plan.owner || ctx.localScope != plan.scope || ctx.expectedType != "" || ctx.expectedTypeRoot != nil {
		return false
	}
	rootCtx := ctx.Clone()
	rootCtx.expectedTypeRoot = plan.root
	if !expectedTypeTargetsExpression(rootCtx, node) || plan.entry == nil || !plan.entry.implicitObjectScalar(plan.parameter) {
		return false
	}
	return genericMethodImplicitDefaultResult(def, bindings, "", ctx)
}

// A default inferred result of an admitted method variable uses its canonical
// implicit Object origin. An explicit/inferred binding (including a binding of
// its method-owned bound) or written consumer preserves the ordinary checkcast.
func genericMethodImplicitDefaultResult(def *symbol.Definition, bindings map[string]string, consuming string, ctx Ctx) bool {
	if strings.TrimSpace(consuming) != "" && !isVarKeywordType(consuming) {
		return false
	}
	declaring, known := genericMethodOwnerContext(def)
	if !known {
		return false
	}
	owner := declaring.currentClass
	if def.IsStatic {
		if planStaticGenericMethodPhysicalABI(owner, def, ctx) == nil {
			return false
		}
	} else if planGenericMethodEntryABI(owner, def, ctx) == nil {
		return false
	}
	declared := definitionDeclaredJavaType(def)
	base, rank := javaArrayTypeParts(declared.Original)
	declaration := declared.TypeParameterBindings[base]
	if rank != 0 || declaration == nil {
		return false
	}
	slots, _, known := declaredGenericMethodPhysicalSlots(def)
	if !known {
		return false
	}
	parameters := make([]symbol.TypeParam, 0, len(slots))
	byDeclaration := map[*symbol.TypeParamDeclaration]genericMethodPhysicalSlot{}
	for _, slot := range slots {
		parameters = append(parameters, slot.parameter)
		byDeclaration[slot.parameter.Declaration] = slot
	}
	lookup := newTypeParameterLookup(parameters)
	for declaration != nil {
		slot, found := byDeclaration[declaration]
		if !found || !slot.implicitObject || bindings[slot.parameter.Name] != "" || bindings[slot.parameter.EmittedName()] != "" {
			return false
		}
		if len(slot.parameter.Bounds) == 0 {
			return true
		}
		bound := slot.parameter.Bounds[0]
		name, _ := parseJavaTypeString(bound.Original)
		next, found := lookup.resolve(bound, name)
		if !found {
			return false
		}
		declaration = next.Declaration
	}
	return false
}

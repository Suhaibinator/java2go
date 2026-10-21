package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

type sourceNativeComparableReference struct {
	source    *symbol.ClassScope
	parameter symbol.TypeParam
	context   Ctx
	canonical string
}

type sourceNativeComparableView struct {
	scope    *symbol.ClassScope
	bindings map[*symbol.TypeParamDeclaration]sourceNativeComparableReference
}

type sourceNativeComparableBridgePlan struct {
	owner                       *symbol.ClassScope
	method                      *symbol.Definition
	castType, castID, transport ast.Expr
	selector, registration      string
}

// Declaration syntax uses source names and declaration pointers, never a caller
// binder or an emitted alias that happens to have the same spelling.
func sourceNativeComparableDeclarationCtx(scope *symbol.ClassScope, ctx Ctx) (Ctx, bool) {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return Ctx{}, false
	}
	for owner := scope; owner != nil; owner = owner.Enclosing {
		if file := findFileScopeForClassScope(owner, ctx); file != nil {
			declaring := classHeaderTypeCtx(scope, ctx)
			declaring.currentFile = file
			declaring.syntheticTypeParameters = nil
			return declaring, true
		}
	}
	return Ctx{}, false
}

func resolveSourceNativeComparableReference(javaType symbol.JavaType, ctx Ctx, bindings map[*symbol.TypeParamDeclaration]sourceNativeComparableReference) (sourceNativeComparableReference, bool) {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType.Original))
	base, arguments := parseJavaTypeString(component)
	if rank != 0 || len(arguments) != 0 || base == "" || strings.ContainsAny(base, "? ") {
		return sourceNativeComparableReference{}, false
	}
	lexical := referenceTypeParameterBindings(ctx)
	var parameter *referenceTypeParameterBinding
	for index := len(lexical) - 1; index >= 0; index-- {
		binding := &lexical[index]
		declaration := binding.parameter.Declaration
		if declaration == nil {
			continue
		}
		requested := javaType.TypeParameterBindings[base]
		if requested != nil && requested == declaration || requested == nil && declaration.SourceName == base {
			parameter = binding
			break
		}
	}
	if parameter != nil {
		if substituted, found := bindings[parameter.parameter.Declaration]; found {
			return substituted, true
		}
		return sourceNativeComparableReference{parameter: parameter.parameter, context: parameter.context}, true
	}
	if javaType.TypeParameterBindings[base] != nil {
		return sourceNativeComparableReference{}, false
	}
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		if scope.Class == nil || len(scope.TypeParameters) != 0 {
			return sourceNativeComparableReference{}, false
		}
		return sourceNativeComparableReference{source: scope, context: ctx}, true
	}
	first := strings.SplitN(base, ".", 2)[0]
	for _, binding := range lexical {
		if binding.parameter.Declaration != nil && binding.parameter.Declaration.SourceName == first {
			return sourceNativeComparableReference{}, false
		}
	}
	if strings.Contains(base, ".") {
		if resolveClassScopeByQualifiedName(ctx, first) != nil {
			return sourceNativeComparableReference{}, false
		}
		if _, imported := ctx.currentFile.Imports[first]; imported {
			return sourceNativeComparableReference{}, false
		}
		if !declaredInterfaceDemandPrefixAbsent(first, ctx) {
			return sourceNativeComparableReference{}, false
		}
	} else if _, explicit := ctx.currentFile.Imports[base]; !explicit {
		for _, demand := range ordinaryTypeOnDemandImports(ctx) {
			if demand == "java.lang" {
				continue
			}
			if demand == "java.util" {
				if entry, present := declaredJavaUtilTypes[base]; present && entry.public {
					return sourceNativeComparableReference{}, false
				}
				continue
			}
			if symbol.GlobalScope.FindPackage(demand) == nil && findQualifiedSourceClass(demand) == nil {
				return sourceNativeComparableReference{}, false
			}
		}
	}
	canonical := qualifyDeclaredNominalReference(base, ctx)
	// Only existing runtime representations can provide a concrete erased ABI.
	supported := canonical == "java.lang.Object" || canonical == "java.lang.String"
	if _, boxed := builtinJavaWrapperPrimitive(canonical, ctx); boxed {
		supported = true
	}
	if _, native := stdjavaRuntimeTypeExpr(canonical, nil, inScopeTypeParameters(ctx), ctx); native {
		supported = true
	}
	if !supported {
		return sourceNativeComparableReference{}, false
	}
	return sourceNativeComparableReference{canonical: canonical, context: ctx}, true
}

func sourceNativeComparableSameReference(left, right sourceNativeComparableReference) bool {
	if left.parameter.Declaration != nil || right.parameter.Declaration != nil {
		return left.parameter.Declaration != nil && left.parameter.Declaration == right.parameter.Declaration
	}
	if left.source != nil || right.source != nil {
		return left.source != nil && left.source == right.source
	}
	return left.canonical != "" && left.canonical == right.canonical
}

func sourceNativeComparableViews(scope *symbol.ClassScope, ctx Ctx) ([]sourceNativeComparableView, sourceNativeComparableReference, bool) {
	queue := []sourceNativeComparableView{{scope: scope, bindings: map[*symbol.TypeParamDeclaration]sourceNativeComparableReference{}}}
	visited := map[*symbol.ClassScope]sourceNativeComparableView{}
	var views []sourceNativeComparableView
	var contract sourceNativeComparableReference
	found := false
	for len(queue) > 0 {
		view := queue[0]
		queue = queue[1:]
		if prior, seen := visited[view.scope]; seen {
			for _, parameter := range view.scope.TypeParameters {
				l, lf := prior.bindings[parameter.Declaration]
				r, rf := view.bindings[parameter.Declaration]
				if lf != rf || lf && !sourceNativeComparableSameReference(l, r) {
					return nil, contract, false
				}
			}
			continue
		}
		declaring, anchored := sourceNativeComparableDeclarationCtx(view.scope, ctx)
		if !anchored {
			return nil, contract, false
		}
		visited[view.scope] = view
		views = append(views, view)
		parents := append([]string(nil), view.scope.ImplementedInterfaces...)
		if view.scope.Superclass != "" {
			parents = append([]string{view.scope.Superclass}, parents...)
		}
		for _, written := range parents {
			base, args := parseJavaTypeString(written)
			if declarationOwnedExternalInterface(written, view.scope, declaring) == "java.lang.Comparable" {
				argument := symbol.JavaType{Original: "java.lang.Object"}
				if len(args) > 1 {
					return nil, contract, false
				}
				if len(args) == 1 {
					argument.Original = args[0]
				}
				reference, ok := resolveSourceNativeComparableReference(argument, declaring, view.bindings)
				if !ok {
					return nil, contract, false
				}
				if found && !sourceNativeComparableSameReference(contract, reference) {
					return nil, contract, false
				}
				contract = reference
				found = true
				continue
			}
			first := strings.SplitN(base, ".", 2)[0]
			shadowed := false
			for _, binding := range referenceTypeParameterBindings(declaring) {
				if binding.parameter.Declaration != nil && binding.parameter.Declaration.SourceName == first {
					shadowed = true
				}
			}
			if shadowed {
				continue
			}
			parent := resolveClassScopeByQualifiedName(declaring, base)
			if parent == nil {
				continue
			}
			if parent.Class == nil || len(parent.TypeParameters) != len(args) {
				return nil, contract, false
			}
			next := sourceNativeComparableView{scope: parent, bindings: map[*symbol.TypeParamDeclaration]sourceNativeComparableReference{}}
			for index, parameter := range parent.TypeParameters {
				if parameter.Declaration == nil {
					return nil, contract, false
				}
				reference, ok := resolveSourceNativeComparableReference(symbol.JavaType{Original: args[index]}, declaring, view.bindings)
				if !ok {
					return nil, contract, false
				}
				next.bindings[parameter.Declaration] = reference
			}
			queue = append(queue, next)
		}
	}
	return views, contract, found
}

func sourceNativeComparableErasure(reference sourceNativeComparableReference) (sourceNativeComparableReference, bool) {
	seen := map[*symbol.TypeParamDeclaration]bool{}
	for reference.parameter.Declaration != nil {
		parameter := reference.parameter
		if seen[parameter.Declaration] {
			return sourceNativeComparableReference{}, false
		}
		seen[parameter.Declaration] = true
		if len(parameter.Bounds) == 0 {
			return sourceNativeComparableReference{canonical: "java.lang.Object", context: reference.context}, true
		}
		bound := parameter.Bounds[0]
		next, ok := resolveSourceNativeComparableReference(bound, reference.context, nil)
		if !ok {
			return sourceNativeComparableReference{}, false
		}
		reference = next
	}
	return reference, reference.source != nil || reference.canonical != ""
}

func sourceNativeComparableType(reference sourceNativeComparableReference, ctx Ctx) (ast.Expr, ast.Expr, bool) {
	if reference.parameter.Declaration != nil {
		return nil, nil, false
	}
	if reference.source != nil {
		scope := reference.source
		typ := qualifiedNameExpr(scope.Class.Name, findJavaPackageForClassScope(scope), ctx)
		if !scope.IsInterface {
			typ = &ast.StarExpr{X: typ}
		}
		return typ, javaTypeIDLiteral(sourceClassRuntimeTypeID(scope, ctx), ctx), true
	}
	// Reserve the runtime alias in the consuming file, then interpret the
	// proved Java owner solely in its declaration namespace. Import maps are
	// shared by the render; changing currentFile must not rebind its types.
	markStdjavaUsage(ctx)
	declaring := reference.context.Clone()
	declaring.localScope = nil
	id, known := javaTypeDescriptorExpr(reference.canonical, declaring)
	if !known {
		return nil, nil, false
	}
	// Supported external types use runtime names; caller binders cannot capture
	// these already proved nominal references.
	physical := javaTypeStringToGoTypeExpr(reference.canonical, nil, declaring)
	if reference.canonical == "java.lang.Object" {
		physical = ast.NewIdent("any")
		id = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	}
	return physical, id, known
}

func sourceNativeComparableAllocatedIdentifier(base string, scope *symbol.ClassScope, ctx Ctx) string {
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += strconv.Itoa(suffix)
		}
		if generatedIdentifierExists(candidate, scope) {
			continue
		}
		occupied := false
		for _, owner := range allSourceClassScopes() {
			for _, method := range owner.Methods {
				if method != nil && (method.HelperName == candidate || !method.Constructor && sourceNativeComparableExecutionNameMayEqual(method.Name, candidate) && executionImplementationName(method, owner, ctx) == candidate) {
					occupied = true
				}
			}
		}
		if !occupied {
			return candidate
		}
	}
}

// Every executionImplementationName branch retains the emitted method name,
// then adds one of these suffixes and an optional positive collision number.
// This only rejects impossible equality queries; possible names still require
// the complete declaration and ancestry proof. Keep the alternatives aligned
// with ordinary, specialized override, Map.Entry, and iteration body naming.
func sourceNativeComparableExecutionNameMayEqual(methodName, candidate string) bool {
	if methodName == "" || candidate == "" {
		return true
	}
	tail, matches := strings.CutPrefix(candidate, methodName)
	if !matches {
		return false
	}
	for _, suffix := range []string{
		executionMethodSuffix,
		"Java2goExactExecution",
		"Java2goEntryBodyExecution",
		"Java2goIterationBodyExecution",
	} {
		digits, matches := strings.CutPrefix(tail, suffix)
		if !matches {
			continue
		}
		if digits == "" {
			return true
		}
		if digits[0] < '1' || digits[0] > '9' {
			continue
		}
		decimal := true
		for index := 1; index < len(digits); index++ {
			if digits[index] < '0' || digits[index] > '9' {
				decimal = false
				break
			}
		}
		if decimal {
			return true
		}
	}
	return false
}

func planSourceNativeComparableBridge(ctx Ctx) *sourceNativeComparableBridgePlan {
	scope := ctx.currentClass
	if scope == nil || scope.Class == nil || scope.IsInterface || scope.IsEnum {
		return nil
	}
	views, contract, known := sourceNativeComparableViews(scope, ctx)
	if !known {
		return nil
	}
	var selected *symbol.Definition
	var owner *symbol.ClassScope
	var selectedView sourceNativeComparableView
	// Class bodies take precedence over all interface defaults. Nearer class
	// declarations are visited before their superclass declarations.
	for _, view := range views {
		if view.scope.IsInterface {
			continue
		}
		declaring, _ := sourceNativeComparableDeclarationCtx(view.scope, ctx)
		for _, method := range view.scope.Methods {
			if method == nil || method.OriginalName != "compareTo" || method.IsStatic || method.IsPrivate || method.Constructor || method.RequiresHelper || len(method.TypeParameters) != 0 || len(method.Parameters) != 1 || executionParameterIsVariadic(method, 0) || strings.TrimSpace(method.OriginalType) != "int" {
				continue
			}
			methodCtx := declaring.Clone()
			methodCtx.localScope = method
			parameter := method.Parameters[0]
			reference, ok := resolveSourceNativeComparableReference(symbol.JavaType{Original: parameter.OriginalType, TypeParameterBindings: parameter.TypeParameterBindings}, methodCtx, view.bindings)
			if !ok || !sourceNativeComparableSameReference(reference, contract) {
				continue
			}
			if !method.HasBody {
				return nil
			}
			selected = method
			owner = view.scope
			selectedView = view
			break
		}
		if selected != nil {
			break
		}
	}
	if selected == nil {
		for _, view := range views {
			if !view.scope.IsInterface {
				continue
			}
			declaring, _ := sourceNativeComparableDeclarationCtx(view.scope, ctx)
			for _, method := range view.scope.Methods {
				if method == nil || !method.HasBody || method.IsStatic || method.IsPrivate || method.OriginalName != "compareTo" || len(method.Parameters) != 1 || len(method.TypeParameters) != 0 || method.RequiresHelper || executionParameterIsVariadic(method, 0) || method.OriginalType != "int" {
					continue
				}
				methodCtx := declaring.Clone()
				methodCtx.localScope = method
				parameter := method.Parameters[0]
				reference, ok := resolveSourceNativeComparableReference(symbol.JavaType{Original: parameter.OriginalType, TypeParameterBindings: parameter.TypeParameterBindings}, methodCtx, view.bindings)
				if !ok || !sourceNativeComparableSameReference(reference, contract) {
					continue
				}
				if selected != nil {
					return nil
				} // unresolved competing defaults require proof
				selected = method
				owner = view.scope
				selectedView = view
			}
		}
	}
	if selected == nil {
		return nil
	}
	declaring, _ := sourceNativeComparableDeclarationCtx(owner, ctx)
	declaring.localScope = selected
	parameter := selected.Parameters[0]
	original, ok := resolveSourceNativeComparableReference(symbol.JavaType{Original: parameter.OriginalType, TypeParameterBindings: parameter.TypeParameterBindings}, declaring, nil)
	if !ok {
		return nil
	}
	erased, ok := sourceNativeComparableErasure(original)
	if !ok {
		return nil
	}
	castType, castID, ok := sourceNativeComparableType(erased, ctx)
	if !ok {
		return nil
	}
	var transport ast.Expr
	if original.parameter.Declaration != nil {
		_, physical := directOwnerMethodParameterInterfaceErasure(owner, selected, 0, declaring)
		if !physical && genericFamilyPhysicalJavaType(parameter.OriginalType, declaring) == parameter.OriginalType {
			target := original
			if mapped, present := selectedView.bindings[original.parameter.Declaration]; present {
				target = mapped
			}
			if target.parameter.Declaration != nil {
				// This conversion transports an already checked declared Item pointer into
				// an existing Go slot whose exact pointer constraint admits only Item. It
				// performs no Java cast to instantiated T and disappears with physical ABI.
				bounds := target.parameter.Bounds
				if len(bounds) != 1 || erased.source == nil {
					return nil
				}
				bound, proved := resolveSourceNativeComparableReference(bounds[0], target.context, nil)
				if !proved || !sourceNativeComparableSameReference(bound, erased) {
					return nil
				}
				transport = ast.NewIdent(target.parameter.EmittedName())
			} else if !sourceNativeComparableSameReference(target, erased) {
				return nil
			}
		}
	}
	return &sourceNativeComparableBridgePlan{owner: owner, method: selected, castType: castType, castID: castID, transport: transport,
		selector:     sourceNativeComparableAllocatedIdentifier("CompareToJava2goErasedExecution", scope, ctx),
		registration: sourceNativeComparableAllocatedIdentifier("__java2goErasedContractRegistration"+scope.Class.Name, scope, ctx)}
}

func generateSourceNativeComparableBridgeDecls(ctx Ctx) []ast.Decl {
	plan := planSourceNativeComparableBridge(ctx)
	if plan == nil {
		return nil
	}
	scope := ctx.currentClass
	execution := executionNameForClass(scope)
	used := map[string]struct{}{execution: {}}
	for _, name := range scope.GoTypeParameterNames() {
		used[name] = struct{}{}
	}
	receiver := ast.NewIdent(synchronizedUniqueLocalName("receiver", used))
	argument := ast.NewIdent(synchronizedUniqueLocalName("other", used))
	checked := ast.NewIdent(synchronizedUniqueLocalName("checked", used))
	body := []ast.Stmt{&ast.ExprStmt{X: stdjavaCall(ctx, "ReferenceRequireNonNull", receiver)},
		&ast.AssignStmt{Lhs: []ast.Expr{checked}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.CallExpr{Fun: &ast.IndexExpr{X: stdjavaQualifiedExpr("ObjectView", ctx), Index: plan.castType}, Args: []ast.Expr{argument, plan.castID}}}}}
	var callReceiver ast.Expr = receiver
	if !plan.owner.IsInterface {
		callReceiver = sourceClassViewExpr(scope, plan.owner, receiver, ctx)
		if callReceiver == nil {
			return nil
		}
		if classNeedsVirtualDispatch(plan.owner, ctx) {
			self := ast.NewIdent(synchronizedUniqueLocalName("self", used))
			body = append(body, &ast.AssignStmt{Lhs: []ast.Expr{self}, Tok: token.DEFINE, Rhs: []ast.Expr{&ast.SelectorExpr{X: callReceiver, Sel: ast.NewIdent(classDispatchFieldName(plan.owner))}}},
				&ast.IfStmt{Cond: &ast.BinaryExpr{X: self, Op: token.EQL, Y: ast.NewIdent("nil")}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.AssignStmt{Lhs: []ast.Expr{self}, Tok: token.ASSIGN, Rhs: []ast.Expr{callReceiver}}}}})
			callReceiver = self
		}
	}
	var passed ast.Expr = checked
	if plan.transport != nil {
		passed = &ast.CallExpr{Fun: plan.transport, Args: []ast.Expr{checked}}
	}
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: callReceiver, Sel: ast.NewIdent(executionImplementationName(plan.method, plan.owner, ctx))}, Args: []ast.Expr{ast.NewIdent(execution), passed}}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
	method := &ast.FuncDecl{Name: ast.NewIdent(plan.selector), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: &ast.StarExpr{X: instantiateGenericType(scope.Class.Name, typeParamExprs(scope.GoTypeParameterNames()))}}}},
		Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField(execution, ctx), {Names: []*ast.Ident{argument}, Type: ast.NewIdent("any")}}}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("int32")}}}}, Body: &ast.BlockStmt{List: body}}
	register := stdjavaCall(ctx, "RegisterSourceErasedContract", javaTypeIDLiteral(sourceClassRuntimeTypeID(scope, ctx), ctx), stdjavaQualifiedExpr("ComparableTypeID", ctx),
		&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote("compareTo(Ljava/lang/Object;)I")}, &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(plan.selector)})
	registration := &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&ast.ValueSpec{Names: []*ast.Ident{ast.NewIdent(plan.registration)}, Values: []ast.Expr{&ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("bool")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ExprStmt{X: register}, &ast.ReturnStmt{Results: []ast.Expr{ast.NewIdent("true")}}}}}}}}}}
	return []ast.Decl{method, registration}
}

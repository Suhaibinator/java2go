package transpiler

import (
	"fmt"
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A raw Java class has one runtime class regardless of its source type
// arguments. Go instantiations are invariant, however, so *Box[First] and
// *Box[Second] cannot occupy the receiver parameter of one unbound method
// reference. The generated view below exposes only erased, uniform entry
// points and is consequently implemented by every Go instantiation.
type rawUnboundReceiverMethod struct {
	owner  *symbol.ClassScope
	method *symbol.Definition
}

func rawUnboundReceiverViewTypeName(scope *symbol.ClassScope) string {
	if scope == nil || scope.Class == nil {
		return ""
	}
	base := "Java2goRawReceiverFrom" + rawUnboundHexIdentity(javaClassBinaryName(scope))
	return collisionSafeRawUnboundIdentifier(base, scope)
}

func rawUnboundReceiverEntryName(owner *symbol.ClassScope, method *symbol.Definition) string {
	if owner == nil || method == nil {
		return ""
	}
	identity := javaClassBinaryName(owner) + "#" + method.OriginalName + "("
	for index := range method.Parameters {
		if index > 0 {
			identity += ","
		}
		identity += definitionParameterJavaSignatureType(method, index)
	}
	identity += ")" + method.OriginalType
	base := "Java2goRawInvokeFrom" + rawUnboundHexIdentity(identity)
	return collisionSafeRawUnboundIdentifier(base, owner)
}

func rawUnboundHexIdentity(value string) string {
	var result strings.Builder
	result.Grow(len(value) * 2)
	for _, octet := range []byte(value) {
		_, _ = fmt.Fprintf(&result, "%02X", octet)
	}
	return result.String()
}

func collisionSafeRawUnboundIdentifier(base string, owner *symbol.ClassScope) string {
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += fmt.Sprintf("%d", suffix)
		}
		if !generatedIdentifierExists(candidate, owner) && !sourceMethodIdentifierExists(candidate) {
			return candidate
		}
	}
}

// rawUnboundReceiverMethods returns methods whose generated physical ABI is
// independent of the receiver's Go type arguments. A bare owner parameter is
// eligible only after the callable-erasure planner has moved it to its Java
// bound; nested generic/array uses remain excluded until they have a uniform
// physical representation.
func rawUnboundReceiverMethods(scope *symbol.ClassScope, ctx Ctx) []rawUnboundReceiverMethod {
	if scope == nil || scope.Class == nil || len(scope.TypeParameters) == 0 || scope.IsInterface || scope.IsEnum {
		return nil
	}
	declarationCtx := classScopeCtx(scope, ctx)
	var methods []rawUnboundReceiverMethod
	for _, method := range scope.Methods {
		if !ordinarySourceMethod(scope, method) {
			continue
		}
		_, changed, uniform := directOwnerCallablePhysicalSignature(scope, method, declarationCtx)
		if !uniform || (changed && !directOwnerCallableMethodEligible(scope, method, declarationCtx)) {
			continue
		}
		methods = append(methods, rawUnboundReceiverMethod{owner: scope, method: method})
	}
	return methods
}

func rawUnboundReceiverMethodEligible(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) bool {
	for _, candidate := range rawUnboundReceiverMethods(owner, ctx) {
		if candidate.method == method {
			return true
		}
	}
	return false
}

func rawUnboundEntryFuncType(owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) *ast.FuncType {
	if owner == nil || method == nil {
		return nil
	}
	declarationCtx := classScopeCtx(owner, ctx)
	executionName := executionParameterName(method.DeclarationNode, findFileScopeForClassScope(owner).Source, declarationCtx)
	params := &ast.FieldList{List: []*ast.Field{executionParameterField(executionName, declarationCtx)}}
	for index, parameter := range method.Parameters {
		parameterType := executionParameterTypeExpr(method, index, parameter.OriginalType, owner.TypeParameterNames(), declarationCtx)
		parameterType = directOwnerTypeParameterMethodParameterType(owner, method, index, parameterType, declarationCtx)
		params.List = append(params.List, &ast.Field{
			Names: []*ast.Ident{{Name: parameter.Name}},
			Type:  parameterType,
		})
	}
	var results *ast.FieldList
	if strings.TrimSpace(method.OriginalType) != "" && strings.TrimSpace(method.OriginalType) != "void" {
		resultType := javaTypeStringToGoTypeExpr(method.OriginalType, owner.TypeParameterNames(), declarationCtx)
		resultType = directOwnerTypeParameterMethodResultType(owner, method, resultType, declarationCtx)
		results = &ast.FieldList{List: []*ast.Field{{Type: resultType}}}
	}
	return &ast.FuncType{Params: params, Results: results}
}

func generateRawUnboundReceiverViewDecl(ctx Ctx) ast.Decl {
	scope := ctx.currentClass
	if !rawUnboundReceiverHasSourceReference(scope, ctx) {
		return nil
	}
	methods := rawUnboundReceiverMethods(scope, ctx)
	if len(methods) == 0 {
		return nil
	}
	fields := &ast.FieldList{}
	for _, method := range methods {
		fields.List = append(fields.List, &ast.Field{
			Names: []*ast.Ident{{Name: rawUnboundReceiverEntryName(method.owner, method.method)}},
			Type:  rawUnboundEntryFuncType(method.owner, method.method, ctx),
		})
	}
	return genInterfaceInContext(rawUnboundReceiverViewTypeName(scope), fields, nil, ctx)
}

func generateRawUnboundReceiverEntryDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if !rawUnboundReceiverHasSourceReference(scope, ctx) {
		return nil
	}
	methods := rawUnboundReceiverMethods(scope, ctx)
	if len(methods) == 0 {
		return nil
	}
	receiverName := ShortName(scope.Class.Name)
	receiverType := instantiateGenericType(scope.Class.Name, typeParamExprs(scope.GoTypeParameterNames()))
	var declarations []ast.Decl
	for _, candidate := range methods {
		functionType := rawUnboundEntryFuncType(candidate.owner, candidate.method, ctx)
		if functionType == nil || functionType.Params == nil || len(functionType.Params.List) == 0 {
			continue
		}
		arguments := methodCallArgs(functionType.Params)
		execution := arguments[0]
		javaArguments := arguments[1:]
		var callReceiver ast.Expr = &ast.Ident{Name: receiverName}
		if classNeedsVirtualDispatch(candidate.owner, ctx) {
			callReceiver = &ast.SelectorExpr{
				X:   callReceiver,
				Sel: &ast.Ident{Name: classDispatchFieldName(candidate.owner)},
			}
		}
		call := &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   callReceiver,
				Sel: &ast.Ident{Name: executionImplementationName(candidate.method, candidate.owner)},
			},
			Args: append([]ast.Expr{execution}, javaArguments...),
		}
		markVariadicForwardCall(call, candidate.method)
		body := []ast.Stmt{instanceMethodNilReceiverGuard(receiverName)}
		if functionType.Results != nil && len(functionType.Results.List) > 0 {
			body = append(body, &ast.ReturnStmt{Results: []ast.Expr{call}})
		} else {
			body = append(body, &ast.ExprStmt{X: call})
		}
		declarations = append(declarations, &ast.FuncDecl{
			Name: &ast.Ident{Name: rawUnboundReceiverEntryName(candidate.owner, candidate.method)},
			Recv: &ast.FieldList{List: []*ast.Field{{
				Names: []*ast.Ident{{Name: receiverName}},
				Type:  &ast.StarExpr{X: receiverType},
			}}},
			Type: functionType,
			Body: &ast.BlockStmt{List: body},
		})
	}
	return declarations
}

// rawUnboundReceiverViewScopeForSAMParameter recognizes the receiver slot of
// a source functional interface whose Java type is a raw generated generic
// class. Restricting the representation change to the first slot of a true SAM
// keeps ordinary raw parameters on their existing lowering path.
func rawUnboundReceiverViewScopeForSAMParameter(
	interfaceScope *symbol.ClassScope,
	method *symbol.Definition,
	parameterIndex int,
	javaType string,
	ctx Ctx,
) *symbol.ClassScope {
	if interfaceScope == nil || !interfaceScope.IsInterface || method == nil || parameterIndex != 0 {
		return nil
	}
	var sam *symbol.Definition
	for _, candidate := range interfaceScope.Methods {
		if candidate == nil || candidate.IsStatic || candidate.Constructor {
			continue
		}
		if sam != nil {
			return nil
		}
		sam = candidate
	}
	if sam != method {
		return nil
	}
	if rawUnboundSAMHasSourceImplementation(interfaceScope, ctx) {
		// A source implementation may dereference fields through the raw class
		// parameter. Moving that declaration to a method-only receiver interface
		// needs field accessor bridges as well; retain the established ABI until
		// that wider representation is available instead of breaking its body.
		return nil
	}
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments := parseJavaTypeString(component)
	if rank != 0 || len(arguments) != 0 || strings.TrimSpace(base) == "" {
		return nil
	}
	target := resolveClassScopeByTypeQualifier(ctx, base)
	if target == nil || len(target.TypeParameters) == 0 || len(rawUnboundReceiverMethods(target, ctx)) == 0 {
		return nil
	}
	if !rawUnboundReceiverHasSourceReference(target, ctx) {
		// Do not change an otherwise ordinary raw SAM declaration merely because
		// its first parameter happens to be generic. The view is an adaptation for
		// an observed raw unbound method-reference site.
		return nil
	}
	return target
}

func rawUnboundSAMHasSourceImplementation(interfaceScope *symbol.ClassScope, ctx Ctx) bool {
	if interfaceScope == nil {
		return false
	}
	implemented := false
	visitAllClassScopes(func(candidate *symbol.ClassScope) bool {
		if candidate == nil || candidate == interfaceScope || candidate.IsInterface {
			return false
		}
		candidateCtx := classScopeCtx(candidate, ctx)
		for _, javaType := range candidate.ImplementedInterfaces {
			base, _ := parseJavaTypeString(javaType)
			implementedScope := resolveClassScopeByQualifiedName(candidateCtx, base)
			if implementedScope == interfaceScope || rawUnboundInterfaceExtends(implementedScope, interfaceScope, candidateCtx, map[*symbol.ClassScope]struct{}{}) {
				implemented = true
				return true
			}
		}
		return false
	})
	if implemented {
		return true
	}

	// Anonymous and method-local implementations are not necessarily represented
	// in GlobalScope. Audit their syntax as well so changing the SAM descriptor
	// cannot invalidate a body that still expects the concrete raw class view.
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			fileCtx := ctx.Clone()
			fileCtx.currentFile = file
			var walk func(*symbol.ClassScope) bool
			walk = func(owner *symbol.ClassScope) bool {
				if owner == nil || owner.Class == nil || owner.Class.DeclarationNode == nil {
					return false
				}
				fileCtx.currentClass = owner
				var visit func(*sitter.Node) bool
				visit = func(node *sitter.Node) bool {
					if node == nil {
						return false
					}
					if node.Type() == "lambda_expression" {
						// Conservatively retain existing SAM descriptors whenever this
						// conversion contains a source lambda. Proving its full target type
						// through overload/return contexts belongs in a wider SAM-use plan;
						// a lambda body may legally read fields from the raw receiver.
						return true
					}
					if node.Type() == "class_declaration" && node != owner.Class.DeclarationNode {
						interfaces := node.ChildByFieldName("interfaces")
						if interfaces != nil {
							for _, typeNode := range collectTypeNodes(interfaces) {
								base, _ := parseJavaTypeString(typeNode.Content(file.Source))
								implementedScope := resolveClassScopeByQualifiedName(fileCtx, base)
								if implementedScope == interfaceScope || rawUnboundInterfaceExtends(implementedScope, interfaceScope, fileCtx, map[*symbol.ClassScope]struct{}{}) {
									return true
								}
							}
						}
					}
					if node.Type() == "object_creation_expression" {
						hasBody := false
						for _, child := range nodeutil.NamedChildrenOf(node) {
							hasBody = hasBody || child.Type() == "class_body"
						}
						if hasBody {
							if typeNode := node.ChildByFieldName("type"); typeNode != nil {
								base, _ := parseJavaTypeString(typeNode.Content(file.Source))
								implementedScope := resolveClassScopeByQualifiedName(fileCtx, base)
								if implementedScope == interfaceScope || rawUnboundInterfaceExtends(implementedScope, interfaceScope, fileCtx, map[*symbol.ClassScope]struct{}{}) {
									return true
								}
							}
						}
					}
					for _, child := range nodeutil.NamedChildrenOf(node) {
						if visit(child) {
							return true
						}
					}
					return false
				}
				if visit(owner.Class.DeclarationNode) {
					return true
				}
				for _, nested := range owner.Subclasses {
					if walk(nested) {
						return true
					}
				}
				return false
			}
			for _, top := range file.TopLevelClasses {
				if walk(top) {
					return true
				}
			}
		}
	}
	return implemented
}

func rawUnboundReceiverHasSourceReference(target *symbol.ClassScope, ctx Ctx) bool {
	if target == nil {
		return false
	}
	for _, pkg := range symbol.GlobalScope.Packages {
		for _, file := range pkg.Files {
			if file == nil {
				continue
			}
			for _, top := range file.TopLevelClasses {
				if top == nil || top.Class == nil || top.Class.DeclarationNode == nil {
					continue
				}
				declarationCtx := ctx.Clone()
				declarationCtx.currentFile = file
				declarationCtx.currentClass = top
				var visit func(*sitter.Node) bool
				visit = func(node *sitter.Node) bool {
					if node == nil {
						return false
					}
					if node.Type() == "method_reference" && node.NamedChildCount() >= 2 {
						qualifier := node.NamedChild(0)
						methodNode := node.NamedChild(1)
						if qualifier != nil && methodNode != nil &&
							resolveClassScopeByTypeQualifier(declarationCtx, qualifier.Content(file.Source)) == target &&
							javaTypeOmitsGenericArguments(qualifier.Content(file.Source), declarationCtx) {
							methodName := methodNode.Content(file.Source)
							// ParseExpr classifies a type-qualified method reference as
							// static first. Its first SAM parameter is then an ordinary
							// explicit argument, not an unbound receiver, and must retain
							// the established raw-parameter representation.
							if findStaticMethodByName(target, methodName) == nil {
								for _, candidate := range rawUnboundReceiverMethods(target, declarationCtx) {
									if candidate.method != nil && candidate.method.DeclarationNode != nil &&
										candidate.method.OriginalName == methodName {
										return true
									}
								}
							}
						}
					}
					for _, child := range nodeutil.NamedChildrenOf(node) {
						if visit(child) {
							return true
						}
					}
					return false
				}
				if visit(top.Class.DeclarationNode) {
					return true
				}
			}
		}
	}
	return false
}

func rawUnboundInterfaceExtends(candidate, target *symbol.ClassScope, ctx Ctx, seen map[*symbol.ClassScope]struct{}) bool {
	if candidate == nil || target == nil || !candidate.IsInterface {
		return false
	}
	if candidate == target {
		return true
	}
	if _, duplicate := seen[candidate]; duplicate {
		return false
	}
	seen[candidate] = struct{}{}
	declarationCtx := classScopeCtx(candidate, ctx)
	for _, javaType := range candidate.ImplementedInterfaces {
		base, _ := parseJavaTypeString(javaType)
		parent := resolveClassScopeByQualifiedName(declarationCtx, base)
		if parent == target || rawUnboundInterfaceExtends(parent, target, declarationCtx, seen) {
			return true
		}
	}
	return false
}

func rawUnboundReceiverParameterType(
	interfaceScope *symbol.ClassScope,
	method *symbol.Definition,
	parameterIndex int,
	javaType string,
	fallback ast.Expr,
	ctx Ctx,
) ast.Expr {
	target := rawUnboundReceiverViewScopeForSAMParameter(interfaceScope, method, parameterIndex, javaType, ctx)
	if target == nil {
		return fallback
	}
	return qualifiedNameExpr(
		rawUnboundReceiverViewTypeName(target),
		findJavaPackageForClassScope(target),
		ctx,
	)
}

func classScopeOwningMethodDefinition(method *symbol.Definition) *symbol.ClassScope {
	if method == nil {
		return nil
	}
	var owner *symbol.ClassScope
	visitAllClassScopes(func(scope *symbol.ClassScope) bool {
		for _, candidate := range scope.Methods {
			if candidate == method {
				owner = scope
				return true
			}
		}
		return false
	})
	return owner
}

func rawUnboundFunctionUsesReceiverView(functionType *ast.FuncType, target *symbol.ClassScope) bool {
	if functionType == nil || functionType.Params == nil || len(functionType.Params.List) < 2 || target == nil {
		return false
	}
	name := rawUnboundReceiverViewTypeName(target)
	switch parameterType := functionType.Params.List[1].Type.(type) {
	case *ast.Ident:
		return parameterType.Name == name
	case *ast.SelectorExpr:
		return parameterType.Sel != nil && parameterType.Sel.Name == name
	default:
		return false
	}
}

// resolveClassScopeByTypeQualifier resolves member-class qualification one
// owner at a time. Falling back directly to the last simple name would confuse
// A.Inner with B.Inner and can make a raw receiver view reference a different
// generated Go type than javac selected.
func resolveClassScopeByTypeQualifier(ctx Ctx, javaType string) *symbol.ClassScope {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return resolveClassScopeByQualifiedName(ctx, base)
	}

	descend := func(scope *symbol.ClassScope, nested []string) *symbol.ClassScope {
		for _, name := range nested {
			var next *symbol.ClassScope
			for _, child := range scope.Subclasses {
				if child != nil && child.Class != nil && child.Class.OriginalName == name {
					next = child
					break
				}
			}
			if next == nil {
				return nil
			}
			scope = next
		}
		return scope
	}
	findTop := func(pkg *symbol.PackageScope, name string) *symbol.ClassScope {
		if pkg == nil {
			return nil
		}
		for _, file := range pkg.Files {
			for _, top := range file.TopLevelClasses {
				if top != nil && top.Class != nil && top.Class.OriginalName == name {
					return top
				}
			}
		}
		return nil
	}

	// Fully-qualified package prefix, choosing the longest real package.
	for split := len(parts) - 1; split > 0; split-- {
		pkgName := strings.Join(parts[:split], ".")
		if pkg := symbol.GlobalScope.FindPackage(pkgName); pkg != nil {
			return descend(findTop(pkg, parts[split]), parts[split+1:])
		}
	}

	// Lexically visible top-level owner in the current file/package or imports.
	outerName := parts[0]
	if ctx.currentFile != nil {
		// A nested type may qualify a sibling member type of the same enclosing
		// declaration (the common Outer.Inner spelling inside a top-level host).
		// Walk lexical owners before considering unrelated same-simple-name types.
		for lexical := ctx.currentClass; lexical != nil; lexical = lexical.Enclosing {
			if lexical.Class != nil && lexical.Class.OriginalName == outerName {
				return descend(lexical, parts[1:])
			}
			for _, child := range lexical.Subclasses {
				if child != nil && child.Class != nil && child.Class.OriginalName == outerName {
					return descend(child, parts[1:])
				}
			}
		}
		for _, top := range ctx.currentFile.TopLevelClasses {
			if top != nil && top.Class != nil && top.Class.OriginalName == outerName {
				return descend(top, parts[1:])
			}
		}
		if visible := ctx.currentFile.FindClassScope(outerName); visible != nil {
			return descend(visible, parts[1:])
		}
		if pkg := symbol.GlobalScope.FindPackage(ctx.currentFile.Package); pkg != nil {
			if top := findTop(pkg, outerName); top != nil {
				return descend(top, parts[1:])
			}
		}
		if importedPackage, ok := ctx.currentFile.Imports[outerName]; ok {
			if top := findTop(symbol.GlobalScope.FindPackage(importedPackage), outerName); top != nil {
				return descend(top, parts[1:])
			}
		}
	}
	return nil
}

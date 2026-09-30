package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

func init() {
	for _, name := range []string{"Collection", "Set", "AbstractCollection", "AbstractSet", "AbstractMap"} {
		registerIntrinsicOwner("java.util."+name, true)
	}
}

func canonicalAbstractCollectionOwner(javaType string, ctx Ctx) string {
	base, _ := parseJavaTypeString(javaType)
	if !strings.Contains(base, ".") {
		if _, found := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); found {
			return ""
		}
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return ""
	}
	switch owner {
	case "java.util.Collection", "java.util.Set", "java.util.AbstractCollection", "java.util.AbstractSet", "java.util.AbstractMap":
		return owner
	}
	return ""
}

func builtinCollectionIterationElementType(javaType, target string, ctx Ctx) (string, bool) {
	owner := canonicalAbstractCollectionOwner(javaType, ctx)
	if owner == "" || owner == "java.util.AbstractMap" {
		return "", false
	}
	switch target {
	case "java.lang.Iterable", "java.util.Collection":
	case "java.util.Set":
		if owner != "java.util.Set" && owner != "java.util.AbstractSet" {
			return "", false
		}
	default:
		return "", false
	}
	_, args := parseJavaTypeString(javaType)
	if len(args) == 1 {
		return args[0], true
	}
	return "java.lang.Object", true
}

// Invocation applicability follows declared platform edges even when the
// source class's external superclass has no source symbol. Go protocol shape
// does not establish this Java nominal conversion.
func sourceAbstractCollectionAssignable(actual, expected string, candidateTypeParams []string, ctx Ctx) bool {
	owner := canonicalAbstractCollectionOwner(expected, ctx)
	if owner != "java.util.Collection" && owner != "java.util.Set" {
		return false
	}
	element, found := iterationElementType(actual, owner, ctx)
	if !found {
		return false
	}
	_, arguments := parseJavaTypeString(expected)
	return javaGenericArgumentsApplicable([]string{element}, arguments, candidateTypeParams)
}

type abstractCollectionDefault struct {
	java, result, runtime string
	parameters            []string
}

var abstractCollectionDefaults = []abstractCollectionDefault{
	{"isEmpty", "boolean", "AbstractCollectionIsEmptyExecution", nil},
	{"contains", "boolean", "AbstractCollectionContainsExecution", []string{"java.lang.Object"}},
	{"remove", "boolean", "AbstractCollectionRemoveExecution", []string{"java.lang.Object"}},
	{"clear", "void", "AbstractCollectionClearExecution", nil},
	{"containsAll", "boolean", "AbstractCollectionContainsAllExecution", []string{"java.util.Collection<?>"}},
}

var abstractSetDefaults = []abstractCollectionDefault{
	{"equals", "boolean", "AbstractSetEqualsExecution", []string{"java.lang.Object"}},
	{"hashCode", "int", "AbstractSetHashCodeExecution", nil},
	{"removeAll", "boolean", "AbstractSetRemoveAllExecution", []string{"java.util.Collection<?>"}},
}

func directAbstractCollectionDefaults(scope *symbol.ClassScope, ctx Ctx) []abstractCollectionDefault {
	if scope == nil || scope.IsInterface {
		return nil
	}
	owner := canonicalAbstractCollectionOwner(scope.Superclass, classScopeCtx(scope, ctx))
	if owner == "java.util.AbstractSet" {
		return append(append([]abstractCollectionDefault(nil), abstractCollectionDefaults...), abstractSetDefaults...)
	}
	if owner == "java.util.AbstractCollection" {
		return abstractCollectionDefaults
	}
	return nil
}

// Platform symbols retain their Java formals for normal source overload and
// override resolution, while their body is generated separately below.
func prepareAbstractCollectionDefaults() {
	for _, scope := range allSourceClassScopes() {
		ctx := Ctx{currentClass: scope, currentFile: findFileScopeForClassScope(scope)}
		for _, spec := range directAbstractCollectionDefaults(scope, ctx) {
			exists := false
			for _, method := range scope.Methods {
				if method.OriginalName != spec.java || method.IsStatic || len(method.Parameters) != len(spec.parameters) {
					continue
				}
				same := true
				for index, formal := range spec.parameters {
					same = same && javaInferenceSameType(method.Parameters[index].OriginalType, formal, ctx)
				}
				if same {
					exists = true
					break
				}
			}
			if exists {
				continue
			}
			method := &symbol.Definition{OriginalName: spec.java, Name: symbol.Uppercase(spec.java), OriginalType: spec.result, HasBody: true, RuntimeDefault: true}
			for index, formal := range spec.parameters {
				used := map[string]struct{}{}
				for _, parameter := range scope.GoTypeParameterNames() {
					used[parameter] = struct{}{}
				}
				name := synchronizedUniqueLocalName("__java2goCollectionArgument", used)
				if index != 0 {
					name = "other"
				}
				method.Parameters = append(method.Parameters, &symbol.Definition{OriginalName: name, Name: name, OriginalType: formal})
			}
			scope.Methods = append(scope.Methods, method)
		}
	}
}

func abstractCollectionReceiver(ctx Ctx, receiverName string) *ast.FieldList {
	return &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(receiverName)}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(ctx.currentClass.GoTypeParameterNames()))}}}}
}

func abstractCollectionReceiverName(ctx Ctx, params *ast.FieldList) string {
	used := map[string]struct{}{}
	for _, name := range inScopeTypeParameters(ctx) {
		used[name] = struct{}{}
	}
	if params != nil {
		for _, field := range params.List {
			for _, name := range field.Names {
				used[name.Name] = struct{}{}
			}
		}
	}
	return synchronizedUniqueLocalName("__java2goCollectionReceiver", used)
}

func abstractCollectionDynamicReference(ctx Ctx, receiverName string) ast.Expr {
	var receiver ast.Expr = ast.NewIdent(receiverName)
	if classNeedsVirtualDispatch(ctx.currentClass, ctx) {
		receiver = &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(classDispatchFieldName(ctx.currentClass))}
		return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{stdjavaQualifiedExpr("JavaIterable", ctx)}, []ast.Expr{receiver, stdjavaQualifiedExpr("CollectionTypeID", ctx)})
	}
	return receiver
}

func generateAbstractCollectionDefaultDecls(ctx Ctx) []ast.Decl {
	var declarations []ast.Decl
	for _, method := range ctx.currentClass.Methods {
		if !method.RuntimeDefault {
			continue
		}
		var selected *abstractCollectionDefault
		for _, spec := range directAbstractCollectionDefaults(ctx.currentClass, ctx) {
			if spec.java == method.OriginalName {
				copy := spec
				selected = &copy
				break
			}
		}
		if selected == nil {
			continue
		}
		params := &ast.FieldList{}
		var args []ast.Expr
		for _, parameter := range method.Parameters {
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: javaTypeStringToGoTypeExpr(parameter.OriginalType, inScopeTypeParameters(ctx), ctx)})
			args = append(args, ast.NewIdent(parameter.Name))
		}
		var results *ast.FieldList
		if method.OriginalType != "void" {
			results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(method.OriginalType, inScopeTypeParameters(ctx), ctx)}}}
		}
		receiverName := abstractCollectionReceiverName(ctx, params)
		executionName := executionNameForParams(params, ctx.currentClass.GoTypeParameterNames()...)
		args = append([]ast.Expr{ast.NewIdent(executionName), abstractCollectionDynamicReference(ctx, receiverName)}, args...)
		call := stdjavaCall(ctx, selected.runtime, args...)
		body := &ast.BlockStmt{List: []ast.Stmt{instanceMethodNilReceiverGuard(receiverName), invocationClosureCallStatement(call, results)}}
		declaration := &ast.FuncDecl{Name: ast.NewIdent(method.Name), Recv: abstractCollectionReceiver(ctx, receiverName), Type: &ast.FuncType{Params: params, Results: results}, Body: body}
		declarations = append(declarations, buildExecutionAwareFuncDecls(declaration, executionImplementationName(method, ctx.currentClass, ctx), executionName, ctx)...)
	}
	return declarations
}

func sourceCollectionMethod(scope *symbol.ClassScope, java string, parameters []string, ctx Ctx) (*symbol.Definition, *symbol.ClassScope) {
	seen := map[*symbol.ClassScope]bool{}
	for current := scope; current != nil && !seen[current]; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
		seen[current] = true
		for _, method := range current.Methods {
			if method.OriginalName == java && len(method.Parameters) == len(parameters) && !method.IsStatic && !method.IsPrivate && method.HasBody {
				matching := true
				for index, formal := range parameters {
					matching = matching && javaInferenceSameType(method.Parameters[index].OriginalType, formal, classScopeCtx(current, ctx))
				}
				if matching {
					return method, current
				}
			}
		}
	}
	return nil, nil
}

func sourceHasCollectionContract(scope *symbol.ClassScope, ctx Ctx, seen map[*symbol.ClassScope]bool) bool {
	if scope == nil || seen[scope] {
		return false
	}
	seen[scope] = true
	declaring := classScopeCtx(scope, ctx)
	for _, parent := range append(append([]string(nil), scope.ImplementedInterfaces...), scope.Superclass) {
		if owner := canonicalAbstractCollectionOwner(parent, declaring); owner != "" && owner != "java.util.AbstractMap" {
			return true
		}
		base, _ := parseJavaTypeString(parent)
		if sourceHasCollectionContract(resolveClassScopeByQualifiedName(declaring, base), declaring, seen) {
			return true
		}
	}
	return false
}

func generateAbstractCollectionBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.IsInterface || !sourceHasCollectionContract(scope, ctx, map[*symbol.ClassScope]bool{}) {
		return nil
	}
	if _, ok := sourceIterationContract(scope, "java.lang.Iterable", ctx); !ok {
		return nil
	}
	var declarations []ast.Decl
	for _, contract := range []struct {
		java, bridge string
		parameters   []string
	}{
		{"size", "CollectionSizeJava2goExecution", nil},
		{"isEmpty", "CollectionIsEmptyJava2goExecution", nil},
		{"contains", "CollectionContainsJava2goExecution", []string{"java.lang.Object"}},
		{"remove", "CollectionRemoveJava2goExecution", []string{"java.lang.Object"}},
		{"clear", "CollectionClearJava2goExecution", nil},
		{"containsAll", "CollectionContainsAllJava2goExecution", []string{"java.util.Collection<?>"}},
	} {
		method, owner := sourceCollectionMethod(scope, contract.java, contract.parameters, ctx)
		if method == nil {
			continue
		}
		params := &ast.FieldList{}
		var args []ast.Expr
		for _, parameter := range method.Parameters {
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(parameter.Name)}, Type: javaTypeStringToGoTypeExpr(parameter.OriginalType, inScopeTypeParameters(ctx), ctx)})
			args = append(args, ast.NewIdent(parameter.Name))
		}
		receiverName := abstractCollectionReceiverName(ctx, params)
		executionName := executionNameForParams(params, scope.GoTypeParameterNames()...)
		params.List = append([]*ast.Field{executionParameterField(executionName, ctx)}, params.List...)
		args = append([]ast.Expr{ast.NewIdent(executionName)}, args...)
		var receiver ast.Expr = ast.NewIdent(receiverName)
		if classNeedsVirtualDispatch(owner, ctx) {
			receiver = &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(classDispatchFieldName(owner))}
		}
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(executionImplementationName(method, owner, ctx))}, Args: args}
		var results *ast.FieldList
		if method.OriginalType != "void" {
			results = &ast.FieldList{List: []*ast.Field{{Type: javaTypeStringToGoTypeExpr(method.OriginalType, inScopeTypeParameters(ctx), ctx)}}}
		}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(contract.bridge), Recv: abstractCollectionReceiver(ctx, receiverName), Type: &ast.FuncType{Params: params, Results: results}, Body: &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, results)}}})
	}
	return declarations
}

func sourceAbstractCollectionInterfaceIDs(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	var result []ast.Expr
	for _, parent := range append(append([]string(nil), scope.ImplementedInterfaces...), scope.Superclass) {
		owner := canonicalAbstractCollectionOwner(parent, classScopeCtx(scope, ctx))
		constant := map[string]string{"java.util.Collection": "CollectionTypeID", "java.util.Set": "SetTypeID"}[owner]
		if constant != "" {
			result = append(result, stdjavaQualifiedExpr(constant, ctx))
		}
	}
	return result
}

func abstractCollectionSuperclassConstant(javaType string, ctx Ctx) string {
	return map[string]string{"java.util.AbstractCollection": "AbstractCollectionTypeID", "java.util.AbstractSet": "AbstractSetTypeID", "java.util.AbstractMap": "AbstractMapTypeID"}[canonicalAbstractCollectionOwner(javaType, ctx)]
}

func abstractCollectionProtocolReservedSelector(name string) bool {
	switch name {
	case "CollectionSizeJava2goExecution", "CollectionIsEmptyJava2goExecution", "CollectionContainsJava2goExecution", "CollectionRemoveJava2goExecution", "CollectionClearJava2goExecution", "CollectionContainsAllJava2goExecution":
		return true
	}
	return false
}

// An external abstract collection base has no alternative storage to embed.
// Its methods are inherited platform declarations generated on the source root.
func skipAbstractCollectionSuperclass(javaType string, ctx Ctx) bool {
	owner := canonicalAbstractCollectionOwner(javaType, ctx)
	return owner == "java.util.AbstractCollection" || owner == "java.util.AbstractSet" || owner == "java.util.AbstractMap"
}

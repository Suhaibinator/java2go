package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func init() {
	registerIntrinsicOwner("java.util.Map.Entry", true)
	for _, operation := range []struct {
		java, runtime string
		slot, arity   int
	}{
		{"getKey", "MapEntryGetKeyExecution", 0, 0},
		{"getValue", "MapEntryGetValueExecution", 1, 0},
		{"setValue", "MapEntrySetValueExecution", 1, 1},
	} {
		registerInstanceNodeIntrinsic("Entry", operation.java, func(recv ast.Expr, invocation *sitter.Node, ctx Ctx, source []byte) ast.Expr {
			if invocationArgumentCount(invocation) != operation.arity {
				return nil
			}
			args := append([]ast.Expr{intrinsicExecutionExpr(ctx), recv}, intrinsicArgs(invocation.ChildByFieldName("object"), operation.java, source, ctx)...)
			raw := stdjavaCall(ctx, operation.runtime, args...)
			if collectionResultIsErased(invocation, ctx, source) {
				return raw
			}
			elements := receiverElementJavaTypes(invocation.ChildByFieldName("object"), ctx, source)
			if operation.slot >= len(elements) {
				return raw
			}
			element := readableWildcardProjection(elements[operation.slot])
			physical := javaTypeStringToGoTypeExpr(element, inScopeTypeParameters(ctx), ctx)
			descriptor, known := javaTypeDescriptorExpr(element, ctx)
			if !known {
				descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
			}
			return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{raw, descriptor})
		})
	}
}

// Entry is an inherited member type of Map and AbstractMap, including when an
// enclosing source class inherits that member. Source declarations and binders
// always take precedence over the external nested declaration.
func canonicalMapEntryOwner(javaType string, ctx Ctx) string {
	base, _ := parseJavaTypeString(javaType)
	if stripJavaQualifier(base) != "Entry" || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return ""
	}
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); bound {
		return ""
	}
	if base == "java.util.Map.Entry" || base == "java.util.Map$Entry" {
		return "java.util.Map.Entry"
	}
	if base == "Map.Entry" && canonicalEntryMemberOwner("Map", ctx) {
		return "java.util.Map.Entry"
	}
	if base != "Entry" {
		return ""
	}
	if ctx.currentFile != nil {
		if pkg, imported := ctx.currentFile.Imports[base]; imported {
			if pkg == "java.util.Map" {
				return "java.util.Map.Entry"
			}
			return ""
		}
	}
	for _, pkg := range intrinsicOnDemandImports(ctx) {
		if pkg == "java.util.Map" {
			return "java.util.Map.Entry"
		}
	}
	for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
		seen := map[*symbol.ClassScope]bool{}
		for owner := scope; owner != nil && !seen[owner]; owner = resolveSuperclassScopeInDeclaringContext(ctx, owner) {
			seen[owner] = true
			declaring := classScopeCtx(owner, ctx)
			if canonicalEntryMemberOwner(owner.Superclass, declaring) {
				return "java.util.Map.Entry"
			}
			for _, parent := range owner.ImplementedInterfaces {
				if canonicalEntryMemberOwner(parent, declaring) {
					return "java.util.Map.Entry"
				}
			}
		}
	}
	return ""
}

// The canonical entry reference has one physical representation for every
// scalar key/value parameterization. Arrays need their independent array proof.
func mapEntryArgumentIndependent(javaType string, ctx Ctx) bool {
	component, rank := javaArrayTypeParts(javaType)
	if rank != 0 || canonicalMapEntryOwner(component, ctx) == "" {
		return false
	}
	_, arguments := parseJavaTypeString(component)
	return len(arguments) == 0 || len(arguments) == 2
}

func canonicalEntryMemberOwner(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(javaType)
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return false
	}
	if base == "java.util.Map" || base == "java.util.AbstractMap" {
		return true
	}
	if base != "Map" && base != "AbstractMap" {
		return false
	}
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: base}, ctx); bound {
		return false
	}
	if ctx.currentFile != nil {
		if pkg, imported := ctx.currentFile.Imports[base]; imported {
			return pkg == "java.util"
		}
	}
	for _, pkg := range intrinsicOnDemandImports(ctx) {
		if pkg == "java.util" {
			return true
		}
	}
	return false
}

func sourceMapEntryArguments(scope *symbol.ClassScope, ctx Ctx) ([]string, bool) {
	seen := map[*symbol.ClassScope]bool{}
	var visit func(*symbol.ClassScope) ([]string, bool)
	visit = func(owner *symbol.ClassScope) ([]string, bool) {
		if owner == nil || seen[owner] {
			return nil, false
		}
		seen[owner] = true
		declaring := classScopeCtx(owner, ctx)
		for _, parent := range append(append([]string(nil), owner.ImplementedInterfaces...), owner.Superclass) {
			base, arguments := parseJavaTypeString(parent)
			if canonicalMapEntryOwner(base, declaring) != "" {
				if len(arguments) == 2 {
					return arguments, true
				}
				if len(arguments) == 0 {
					return []string{"java.lang.Object", "java.lang.Object"}, true
				}
				return nil, false
			}
			if parentScope := resolveClassScopeByQualifiedName(declaring, base); parentScope != nil {
				if entries, ok := visit(parentScope); ok {
					return entries, true
				}
			}
		}
		return nil, false
	}
	return visit(scope)
}

func mapEntrySourceMethod(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	if method == nil || method.IsStatic || method.IsPrivate || len(method.TypeParameters) != 0 {
		return false
	}
	slots, entry := sourceMapEntryArguments(owner, ctx)
	if !entry {
		return false
	}
	switch method.OriginalName {
	case "getKey", "getValue":
		return len(method.Parameters) == 0
	case "setValue":
		if len(method.Parameters) != 1 {
			return false
		}
		declaring := classScopeCtx(owner, ctx)
		value := method.Parameters[0].OriginalType
		return javaInferenceTypeAssignable(value, slots[1], declaring) && javaInferenceTypeAssignable(slots[1], value, declaring)
	}
	return false
}

func mapEntrySourceImplementationName(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) string {
	if !mapEntrySourceMethod(method, owner, ctx) {
		return ""
	}
	return collisionSafeExecutionIdentifier(method.Name+"Java2goEntryBodyExecution", owner)
}

func mapEntryProtocolReservedSelector(name string) bool {
	return name == "GetKeyJava2goExecution" || name == "GetValueJava2goExecution" || name == "SetValueJava2goExecution"
}

func sourceMapEntryInterfaceIDs(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if scope == nil {
		return nil
	}
	declaring := classScopeCtx(scope, ctx)
	for _, parent := range scope.ImplementedInterfaces {
		if canonicalMapEntryOwner(parent, declaring) != "" {
			return []ast.Expr{stdjavaQualifiedExpr("JavaMapEntryTypeID", ctx)}
		}
	}
	return nil
}

func generateMapEntryBridgeDecls(ctx Ctx) []ast.Decl {
	scope := ctx.currentClass
	if scope == nil || scope.IsInterface || scope.Class == nil {
		return nil
	}
	if _, entry := sourceMapEntryArguments(scope, ctx); !entry {
		return nil
	}
	var declarations []ast.Decl
	for _, contract := range []struct {
		java, bridge string
		arity        int
	}{
		{"getKey", "GetKeyJava2goExecution", 0}, {"getValue", "GetValueJava2goExecution", 0}, {"setValue", "SetValueJava2goExecution", 1},
	} {
		var method *symbol.Definition
		var owner *symbol.ClassScope
		seen := map[*symbol.ClassScope]bool{}
		for current := scope; current != nil && !seen[current]; current = resolveSuperclassScopeInDeclaringContext(ctx, current) {
			seen[current] = true
			for _, candidate := range current.Methods {
				if candidate != nil && candidate.OriginalName == contract.java && mapEntrySourceMethod(candidate, current, ctx) {
					method, owner = candidate, current
					break
				}
			}
			if method != nil {
				break
			}
		}
		if method == nil || !method.HasBody {
			continue
		}
		receiver := ast.NewIdent("receiver")
		parameters := []*ast.Field{executionParameterField("execution", ctx)}
		args := []ast.Expr{ast.NewIdent("execution")}
		if contract.arity == 1 {
			parameters = append(parameters, &ast.Field{Names: []*ast.Ident{ast.NewIdent("value")}, Type: ast.NewIdent("any")})
			declaring := classScopeCtx(owner, ctx)
			valueType := method.Parameters[0].OriginalType
			if erasure, erased := directOwnerMethodParameterInterfaceErasure(owner, method, 0, declaring); erased {
				valueType = erasure
			}
			physical := javaTypeStringToGoTypeExpr(valueType, inScopeTypeParameters(declaring), declaring)
			descriptor, known := javaTypeDescriptorExpr(valueType, declaring)
			if !known {
				descriptor = stdjavaQualifiedExpr("ObjectTypeID", ctx)
			}
			value := ast.Expr(ast.NewIdent("value"))
			if valueType != "Object" && valueType != "java.lang.Object" {
				value = stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{physical}, []ast.Expr{value, descriptor})
			}
			args = append(args, value)
		}
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(executionImplementationName(method, owner, ctx))}, Args: args}
		declarations = append(declarations, &ast.FuncDecl{Name: ast.NewIdent(contract.bridge), Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{receiver}, Type: &ast.StarExpr{X: instantiateGenericType(ctx.className, typeParamExprs(scope.GoTypeParameterNames()))}}}}, Type: &ast.FuncType{Params: &ast.FieldList{List: parameters}, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent("any")}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}}})
	}
	return declarations
}

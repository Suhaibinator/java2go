package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/symbol"
)

// These canonical platform bases own algorithms, not embedded Go storage.
// AbstractMap has additional state/protocol requirements and is not admitted.
func anonymousAbstractCollectionBase(javaType string, ctx Ctx) bool {
	switch canonicalAbstractCollectionOwner(javaType, ctx) {
	case "java.util.AbstractCollection", "java.util.AbstractSet":
		return true
	}
	return false
}

func anonymousAbstractCollectionReservedSelectors(javaType string, ctx Ctx) map[string]struct{} {
	reserved := map[string]struct{}{}
	if !anonymousAbstractCollectionBase(javaType, ctx) {
		return reserved
	}
	specs := abstractCollectionDefaults
	if canonicalAbstractCollectionOwner(javaType, ctx) == "java.util.AbstractSet" {
		specs = append(append([]abstractCollectionDefault(nil), specs...), abstractSetDefaults...)
	}
	for _, spec := range specs {
		name := symbol.Uppercase(spec.java)
		reserved[name] = struct{}{}
		reserved[name+executionMethodSuffix] = struct{}{}
	}
	for _, name := range []string{
		"CollectionSizeJava2goExecution", "CollectionIsEmptyJava2goExecution",
		"CollectionContainsJava2goExecution", "CollectionRemoveJava2goExecution",
		"CollectionClearJava2goExecution", "CollectionContainsAllJava2goExecution",
		"IteratorJava2goExecution", "JavaDynamicTypeID",
	} {
		reserved[name] = struct{}{}
	}
	return reserved
}

func prepareAnonymousAbstractCollectionScope(scope *symbol.ClassScope, ctx Ctx) {
	if scope == nil || !anonymousAbstractCollectionBase(scope.Superclass, classScopeCtx(scope, ctx)) {
		return
	}
	prepareAbstractCollectionDefaultsForScope(scope, classScopeCtx(scope, ctx))
	// Source overloads were named when the anonymous scope was synthesized.
	// A late-added default must not collide with those methods or Java fields.
	used := map[string]struct{}{}
	for _, field := range scope.Fields {
		if field != nil {
			used[field.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method != nil && !method.RuntimeDefault {
			used[method.Name] = struct{}{}
		}
	}
	for _, method := range scope.Methods {
		if method == nil || !method.RuntimeDefault {
			continue
		}
		if _, collision := used[method.Name]; collision {
			method.Rename(nextSyntheticMemberName(method.Name, used))
		}
		used[method.Name] = struct{}{}
	}
}

func anonymousAbstractCollectionDecls(supertype string, ctx Ctx) []ast.Decl {
	if !anonymousAbstractCollectionBase(supertype, ctx) {
		return nil
	}
	declarations := generateAbstractCollectionDefaultDecls(ctx)
	declarations = append(declarations, generateAbstractCollectionBridgeDecls(ctx)...)
	return append(declarations, anonymousAbstractCollectionObjectDefaultBridges(ctx)...)
}

func anonymousAbstractCollectionIdentityDecls(structName, ownerID, supertype string, ctx Ctx) []ast.Decl {
	if !anonymousAbstractCollectionBase(supertype, ctx) {
		return nil
	}
	return syntheticReferenceIdentityDeclsWithTypeParams(
		structName, ownerID+"$"+structName,
		stdjavaQualifiedExpr(abstractCollectionSuperclassConstant(supertype, ctx), ctx), nil,
		ctx.currentClass.GoTypeParameterNames(), ctx,
	)
}

// A source overload may have occupied the ordinary Object method selector
// before inherited defaults are prepared. Preserve that overload and expose
// the exact inherited Object declaration through a distinct execution entry.
func anonymousAbstractCollectionObjectDefaultBridges(ctx Ctx) []ast.Decl {
	var declarations []ast.Decl
	for _, method := range ctx.currentClass.Methods {
		if method == nil || !method.RuntimeDefault || (method.OriginalName != "equals" && method.OriginalName != "hashCode") {
			continue
		}
		base := symbol.Uppercase(method.OriginalName) + executionMethodSuffix
		implementation := executionImplementationName(method, ctx.currentClass, ctx)
		if implementation == base {
			continue
		}
		entry := collisionSafeExecutionIdentifier(base, ctx.currentClass)
		for {
			occupied := false
			for _, candidate := range ctx.currentClass.Methods {
				if candidate != nil && executionImplementationName(candidate, ctx.currentClass, ctx) == entry {
					occupied = true
					break
				}
			}
			if !occupied && !generatedIdentifierExists(entry, ctx.currentClass) {
				break
			}
			entry += "1"
		}
		params := &ast.FieldList{}
		used := map[string]struct{}{}
		for _, name := range ctx.currentClass.GoTypeParameterNames() {
			used[name] = struct{}{}
		}
		receiver := synchronizedUniqueLocalName("receiver", used)
		execution := executionNameForParams(params, append(ctx.currentClass.GoTypeParameterNames(), receiver)...)
		params.List = append(params.List, executionParameterField(execution, ctx))
		args := []ast.Expr{ast.NewIdent(execution)}
		result := "int32"
		if method.OriginalName == "equals" {
			used := map[string]struct{}{receiver: {}, execution: {}}
			for _, name := range ctx.currentClass.GoTypeParameterNames() {
				used[name] = struct{}{}
			}
			other := synchronizedUniqueLocalName("other", used)
			params.List = append(params.List, &ast.Field{Names: []*ast.Ident{ast.NewIdent(other)}, Type: ast.NewIdent("any")})
			args = append(args, ast.NewIdent(other))
			result = "bool"
		}
		call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(receiver), Sel: ast.NewIdent(implementation)}, Args: args}
		declarations = append(declarations, &ast.FuncDecl{
			Name: ast.NewIdent(entry), Recv: abstractCollectionReceiver(ctx, receiver),
			Type: &ast.FuncType{Params: params, Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(result)}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{call}}}},
		})
	}
	return declarations
}

package transpiler

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
)

type sourceRecordComponentPlan struct {
	origin   symbol.RecordComponentOrigin
	typ      ast.Expr
	physical ast.Expr
}

// Java header lookup starts from positive declaration facts only. No caller
// binders, synthetic registries, traversal frames or Go layout aliases enter.
func sourceReflectionRecordDeclarationCtx(owner *symbol.ClassScope) Ctx {
	return Ctx{currentClass: owner, currentFile: recordConstructorSourceFile(owner)}
}

// Go declaration emission is separate from Java identity. Preserve only the
// file's import bookkeeping and existing physical generic layout protocol;
// caller body aliases/registries and function binders cannot alter this ABI.
func sourceReflectionRecordEmissionCtx(owner *symbol.ClassScope, ctx Ctx) Ctx {
	return Ctx{currentClass: owner, currentFile: recordConstructorSourceFile(owner),
		importAliases: ctx.importAliases, usedImports: ctx.usedImports,
		genericFamilies: ctx.genericFamilies, executionContextName: ctx.executionContextName}
}

type sourceRecordErasedIdentity struct {
	nominal   string
	primitive string
	rank      int
}

func sourceReflectionRecordIdentityExpr(identity sourceRecordErasedIdentity, ctx Ctx) ast.Expr {
	var result ast.Expr
	if identity.primitive != "" {
		result = stdjavaCall(ctx, "PrimitiveTypeID", metadataString(identity.primitive))
	} else if identity.nominal == "java.lang.Object" {
		result = stdjavaQualifiedExpr("ObjectTypeID", ctx)
	} else {
		result = javaTypeIDLiteral(identity.nominal, ctx)
	}
	for i := 0; i < identity.rank; i++ {
		result = stdjavaCall(ctx, "ArrayTypeID", result)
	}
	return result
}

// The parser certificate supplies exact implicit member identity; the existing
// record source proof revalidates the current file, snapshot and lexical chain.
// Java roles/signatures stay fixed while legitimate Go display names may change.
func sourceReflectionRecordComponentPlan(scope *symbol.ClassScope, ctx Ctx) ([]sourceRecordComponentPlan, bool) {
	owner := sourceReflectionJavaDeclarationScope(scope)
	if owner == nil || !sourceReflectionRecord(owner) {
		return nil, false
	}
	if !sourceReflectionRecordNamespaceCurrent(recordConstructorSourceFile(owner)) {
		return nil, false
	}
	origins, ok := owner.RecordComponentOrigins()
	if !ok {
		return nil, false
	}
	components, ok := recordConstructorComponents(owner)
	if !ok || len(origins) != len(components) {
		return nil, false
	}
	plans := make([]sourceRecordComponentPlan, len(origins))
	for i, origin := range origins {
		if origin.Node == nil || !origin.Node.Equal(components[i].node) || origin.Field != components[i].field {
			return nil, false
		}
		declaring := sourceReflectionRecordDeclarationCtx(owner)
		typeOrigin := declaredJavaTypeOrigin(definitionDeclaredJavaType(origin.Field), declaring)
		identity, known := sourceReflectionRecordErasure(typeOrigin, declaring, ctx, map[*symbol.TypeParamDeclaration]bool{})
		if !known {
			return nil, false
		}
		emission := sourceReflectionRecordEmissionCtx(scope, ctx)
		typ := sourceReflectionRecordIdentityExpr(identity, emission)
		if origin.Accessor != nil && (origin.Accessor.Name == "_" || !token.IsIdentifier(origin.Accessor.Name)) {
			return nil, false
		}
		if origin.Accessor == nil {
			method := owner.FindMethodByName(origin.Field.OriginalName, nil)
			if !sourceReflectionDeclaredMethod(owner, method) || method.IsStatic || method.IsPrivate || method.Constructor || len(method.Parameters) != 0 || len(method.TypeParameters) != 0 || !recordConstructorTypeEqual(definitionDeclaredJavaType(origin.Field), definitionDeclaredJavaType(method), declaring) {
				return nil, false
			}
		}
		plans[i] = sourceRecordComponentPlan{origin: origin, typ: typ, physical: sourceReflectionFieldStorageType(scope, origin.Field, emission)}
	}
	return plans, true
}

// Follow exactly the nominal declarations used by Java erasure, including a
// type parameter's first bound. A current record namespace alone cannot prove
// a different source file's package or a donated target declaration current.
// Type arguments are not part of the erased Class identity.
func sourceReflectionRecordErasure(origin inferredJavaTypeOrigin, ctx Ctx, registries Ctx, seen map[*symbol.TypeParamDeclaration]bool) (sourceRecordErasedIdentity, bool) {
	unknown := sourceRecordErasedIdentity{}
	if origin.unresolved {
		return unknown, false
	}
	if origin.parameter != nil {
		if seen[origin.parameter] {
			return unknown, false
		}
		seen[origin.parameter] = true
		defer delete(seen, origin.parameter)
		for _, parameter := range visibleTypeParameterDeclarations(ctx) {
			if parameter.Declaration == origin.parameter {
				identity := sourceRecordErasedIdentity{nominal: "java.lang.Object"}
				if len(parameter.Bounds) > 0 {
					var known bool
					identity, known = sourceReflectionRecordErasure(declaredJavaTypeOrigin(parameter.Bounds[0], ctx), ctx, registries, seen)
					if !known {
						return unknown, false
					}
				}
				identity.rank += origin.arrayRank
				return identity, true
			}
		}
		return unknown, false
	}
	identity := sourceRecordErasedIdentity{rank: origin.arrayRank}
	if origin.nominalScope != nil {
		file := findFileScopeForClassScope(origin.nominalScope, ctx)
		if !sourceReflectionRecordNamespaceCurrent(file) || !recordConstructorLexicalChainOwned(origin.nominalScope, file) {
			return unknown, false
		}
		identity.nominal = javaClassBinaryName(origin.nominalScope)
		// Preserve the explicit named-target conflict boundary. This registry
		// can refuse a conflicting donated ID; it cannot supply Java identity.
		for _, local := range registries.localClasses {
			if local != nil && local.scope == origin.nominalScope && local.dynamicTypeID != "" && local.dynamicTypeID != identity.nominal {
				return unknown, false
			}
		}
	} else if origin.externalName != "" {
		identity.nominal = origin.externalName
	} else {
		base, _ := javaArrayTypeParts(strings.TrimSpace(origin.javaType))
		if _, primitive := javaPrimitiveType(base); !primitive {
			return unknown, false
		}
		identity.primitive = base
	}
	return identity, true
}

// Imports and package names are derived mutable state. Re-read the actual
// validated source before using that state to resolve record component erasure.
// This proof does not impose an unrelated nested-class census/profile.
func sourceReflectionRecordNamespaceCurrent(file *symbol.FileScope) bool {
	if file == nil || file.ParsedRoot() == nil || file.ParsedRoot().HasError() || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil {
		return false
	}
	imports := map[string]string{}
	pkg := ""
	for _, node := range nodeutil.SemanticNamedChildrenOf(file.ParsedRoot()) {
		switch node.Type() {
		case "package_declaration":
			if node.NamedChild(0) == nil {
				return false
			}
			pkg = node.NamedChild(0).Content(file.Source)
		case "import_declaration":
			declaration, known := nodeutil.ReadJavaImportSyntax(node, file.Source)
			if !known {
				return false
			}
			if declaration.Kind == nodeutil.JavaImportSingleType {
				if old, exists := imports[declaration.Name]; exists && old != declaration.Scope {
					return false
				}
				imports[declaration.Name] = declaration.Scope
			}
		}
	}
	if pkg != file.Package || len(imports) != len(file.Imports) {
		return false
	}
	for name, scope := range imports {
		if file.Imports[name] != scope {
			return false
		}
	}
	return true
}

func sourceReflectionRecordPlanOrRefuse(scope *symbol.ClassScope, ctx Ctx) []sourceRecordComponentPlan {
	plans, known := sourceReflectionRecordComponentPlan(scope, ctx)
	if known {
		return plans
	}
	var source []byte
	if ctx.currentFile != nil {
		source = ctx.currentFile.Source
	}
	diagnostic := reportUnsupported("record reflection member origin", scope.Class.DeclarationNode, source, ctx)
	// Even a tolerant caller must not receive a successful empty profile after
	// a failed source proof. Strict translation already panics in reportUnsupported.
	panic(strictModeError{diagnostic: diagnostic})
}
func appendSourceReflectionRecordMetadata(descriptor *ast.CompositeLit, scope *symbol.ClassScope, ctx Ctx) {
	plans := sourceReflectionRecordPlanOrRefuse(scope, ctx)
	components := &ast.CompositeLit{Type: &ast.ArrayType{Elt: stdjavaQualifiedExpr("RecordComponentDescriptor", ctx)}}
	for _, plan := range plans {
		components.Elts = append(components.Elts, &ast.CompositeLit{Elts: []ast.Expr{metadataKey("Name", metadataString(plan.origin.Field.OriginalName)), metadataKey("Type", plan.typ)}})
	}
	descriptor.Elts = append(descriptor.Elts, metadataKey("Record", ast.NewIdent("true")), metadataKey("RecordComponentsKnown", ast.NewIdent("true")), metadataKey("RecordComponents", components))
}
func sourceReflectionRecordAccessorBridgeName(scope *symbol.ClassScope, method *symbol.Definition) string {
	return collisionSafeExecutionIdentifier(method.Name+"Java2goRecordAccessorExecution", scope)
}
func sourceReflectionRecordAccessorDescriptors(scope *symbol.ClassScope, ctx Ctx) []ast.Expr {
	if !sourceReflectionRecord(scope) {
		return nil
	}
	var result []ast.Expr
	for _, plan := range sourceReflectionRecordPlanOrRefuse(scope, ctx) {
		method := plan.origin.Accessor
		if method == nil {
			continue
		}
		result = append(result, &ast.CompositeLit{Elts: []ast.Expr{
			metadataKey("Name", metadataString(method.OriginalName)), metadataKey("GoName", metadataGoName(sourceReflectionRecordAccessorBridgeName(scope, method))),
			metadataKey("ParameterTypes", sourceReflectionTypeIDs(nil, ctx)), metadataKey("Return", plan.typ), metadataKey("Modifiers", reflectionInteger(1)), metadataKey("HasModifiers", ast.NewIdent("true")),
		}})
	}
	return result
}

// Hidden bridges preserve the existing Method.invoke Execution ABI and call
// the actual resolved implicit getter exactly once. Explicit source accessors
// retain the ordinary method path; no field read substitutes for an override.
func sourceReflectionRecordAccessorBridges(scope *symbol.ClassScope, ctx Ctx) []ast.Decl {
	if !sourceReflectionRecord(scope) {
		return nil
	}
	var result []ast.Decl
	declarationCtx := sourceReflectionRecordEmissionCtx(scope, ctx)
	for _, plan := range sourceReflectionRecordPlanOrRefuse(scope, ctx) {
		method := plan.origin.Accessor
		if method == nil {
			continue
		}
		receiverType := classSubobjectDeclarationPointerType(scope, declarationCtx)
		used := sourceReflectionCallbackUsedNames(scope, declarationCtx, receiverType, plan.physical)
		executionName := affineUniqueLocalName("execution", used)
		receiverName := affineUniqueLocalName("receiver", used)
		bridgeCtx := declarationCtx.Clone()
		bridgeCtx.executionContextName = executionName
		result = append(result, &ast.FuncDecl{
			Name: ast.NewIdent(sourceReflectionRecordAccessorBridgeName(scope, method)),
			Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent(receiverName)}, Type: receiverType}}},
			Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{executionParameterField(executionName, bridgeCtx)}}, Results: &ast.FieldList{List: []*ast.Field{{Type: plan.physical}}}},
			Body: &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(receiverName), Sel: ast.NewIdent(method.Name)}}}}}},
		})
	}
	return result
}

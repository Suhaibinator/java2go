package transpiler

import (
	"go/ast"
	"reflect"
	"strings"
	"unicode"

	"github.com/NickyBoy89/java2go/symbol"
)

// Go cannot put method type parameters in a method set. A Java generic
// instance method therefore exposes its erased descriptor for virtual calls;
// its existing typed helper remains the exact implementation of the body.
// Limit this entrypoint to bare method parameters: nested invariant Go
// instantiations need a separate representation plan.
func genericMethodHasErasedEntry(def *symbol.Definition) bool {
	return genericMethodHasErasedEntryInContext(def, Ctx{})
}

func genericMethodHasErasedEntryInContext(def *symbol.Definition, sourceCtx Ctx) bool {
	if def == nil || !def.RequiresHelper || def.IsPrivate || executionParameterIsVariadic(def, len(def.Parameters)-1) {
		return false
	}
	for _, tp := range def.TypeParameters {
		if len(tp.Bounds) > 1 || (len(tp.Bounds) == 1 && strings.Contains(tp.Bounds[0].Original, "<")) {
			return false
		}
	}
	types := []string{def.OriginalType}
	for _, p := range def.Parameters {
		types = append(types, p.OriginalType)
	}
	for _, typ := range types {
		for _, tp := range def.TypeParameters {
			if strings.TrimSpace(typ) != tp.Name && strings.TrimSpace(typ) != tp.EmittedName() && javaTypeContainsParameter(typ, tp.Name) {
				if ctx, found := genericMethodOwnerContext(def); found {
					ctx = genericMethodDeclaringSyntaxContext(ctx, sourceCtx)
					if canonicalGenericMethodSignature(def, ctx) {
						return true
					}
				}
				return false
			}
		}
	}
	return true
}

// The selected definition owns lexical meaning. Only an independently
// admitted invocation may supply syntax routes to its fresh declaring context.
func genericMethodDeclaringSyntaxContext(declaring, sourceCtx Ctx) Ctx {
	invocation := sourceCtx.syntaxInvocation
	if declaring.currentClass == nil || declaring.currentFile == nil || !sourceGenericDemandOwnedCompilationUnits(invocation) {
		return declaring
	}
	proof, known := sourceGenericDemandOwnershipContext()
	if !known || proof.sourceOwnership.files[declaring.currentClass] != declaring.currentFile {
		return declaring
	}
	owners := allSourceClassScopes()
	inventory := invocation.demandSyntax
	if inventory == nil {
		inventory = buildSourceGenericDemandSyntax(invocation, owners, proof)
	}
	if inventory == nil || !inventory.matches(invocation, owners, proof) {
		return declaring
	}
	invocation.demandSyntax = inventory
	declaring.syntaxInvocation = invocation
	return declaring
}

func javaTypeContainsParameter(typ, name string) bool {
	for _, part := range strings.FieldsFunc(typ, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '$'
	}) {
		if part == name {
			return true
		}
	}
	return false
}

func genericMethodErasedJavaType(def *symbol.Definition, typ string) string {
	declaring, ownerKnown := genericMethodOwnerContext(def)
	parameters := def.TypeParameters
	if ownerKnown {
		declaring.localScope = def
		declaring.syntheticTypeParameters = nil
		declaredType := definitionDeclaredJavaType(def)
		declaredType.Original = typ
		typ = qualifyDeclaredReferenceType(declaredType, declaring)
		parameters = qualifyTypeParameterBounds(parameters, declaring)
	}
	bindings := map[string]string{}
	slots, _, physicalKnown := declaredGenericMethodPhysicalSlots(def)
	for index, parameter := range parameters {
		erasure := rawTypeParameterErasure(parameter, parameters)
		if physicalKnown {
			erasure = slots[index].javaType
		} else if ownerKnown && parameter.Declaration != nil && len(parameter.Bounds) == 0 {
			erasure = "java.lang.Object"
		}
		if !ownerKnown {
			bindings[parameter.Name] = erasure
		}
		bindings[parameter.EmittedName()] = erasure
	}
	return substituteJavaTypeParameters(typ, bindings)
}

func eraseGenericMethodSignature(def *symbol.Definition, params, results *ast.FieldList, ctx Ctx) {
	if !genericMethodHasErasedEntry(def) {
		return
	}
	if plan := planGenericMethodEntryABI(ctx.currentClass, def, ctx); plan != nil {
		plan.apply(params, results)
		return
	}
	for i, p := range def.Parameters {
		params.List[i].Type = javaTypeStringToGoTypeExpr(genericMethodErasedJavaType(def, p.OriginalType), ctx.currentClass.TypeParameterNames(), ctx)
	}
	if results != nil && len(results.List) > 0 {
		results.List[0].Type = javaTypeStringToGoTypeExpr(genericMethodErasedJavaType(def, def.OriginalType), ctx.currentClass.TypeParameterNames(), ctx)
	}
}

func genericMethodErasedEntryDecls(ctx Ctx, def *symbol.Definition, params, results *ast.FieldList, receiverBaseType ast.Expr) []ast.Decl {
	if !genericMethodHasErasedEntry(def) {
		return nil
	}
	params, results = cloneFieldList(params), cloneFieldList(results)
	eraseGenericMethodSignature(def, params, results, ctx)
	typeArgs := typeParamExprs(ctx.currentClass.GoTypeParameterNames())
	if canonicalGenericClass(ctx.currentClass, ctx) {
		// A canonical receiver has no Go owner binders. The typed helper still
		// declares them, so instantiate its owner slots with their physical
		// erasures; method-owned parameters retain their separate slots below.
		for index, parameter := range ctx.currentClass.TypeParameters {
			typeArgs[index] = javaTypeStringToGoTypeExpr(rawTypeParameterErasure(parameter, ctx.currentClass.TypeParameters), nil, ctx)
		}
	}
	_, ownerKnown := genericMethodOwnerContext(def)
	physical, _, physicalKnown := genericMethodPhysicalTypeArguments(def, ctx)
	for index, tp := range def.TypeParameters {
		var argument ast.Expr
		if physicalKnown {
			argument = physical[index]
		} else if ownerKnown && tp.Declaration != nil && len(tp.Bounds) == 0 {
			argument = &ast.Ident{Name: "any"}
		} else {
			argument = javaTypeStringToGoTypeExpr(rawTypeParameterErasure(tp, def.TypeParameters), ctx.currentClass.TypeParameterNames(), ctx)
		}
		typeArgs = append(typeArgs, argument)
	}
	fields := append([]*ast.Field{executionParameterField(ctx.executionContextName, ctx), {Type: receiverBaseType}, {Type: applyTypeArguments(ast.NewIdent("New"+def.HelperName), typeArgs)}}, params.List...)
	if results != nil {
		fields = append(fields, results.List...)
	}
	recv := synchronizedUniqueLocalName(ShortName(ctx.className), sourceInstanceReceiverUsedNames(ctx, fields...))
	ctx = withSourceInstanceReceiver(ctx, recv)
	helper := &ast.CallExpr{Fun: applyTypeArguments(&ast.Ident{Name: "New" + def.HelperName}, typeArgs), Args: []ast.Expr{&ast.Ident{Name: recv}}}
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: helper, Sel: &ast.Ident{Name: executionImplementationName(def, ctx.currentClass, ctx)}}, Args: append([]ast.Expr{&ast.Ident{Name: ctx.executionContextName}}, methodCallArgs(params)...)}
	body := &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, results)}}
	declaration := &ast.FuncDecl{Name: &ast.Ident{Name: def.Name}, Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: recv}}, Type: &ast.StarExpr{X: receiverBaseType}}}}, Type: &ast.FuncType{Params: params, Results: results}, Body: body}
	return buildExecutionAwareFuncDecls(declaration, executionImplementationName(def, ctx.currentClass, ctx), ctx.executionContextName, ctx)
}

// Result conversion belongs after the erased call. ObjectView preserves null
// and throws Java ClassCastException when an unchecked body pollutes its result.
// Use the inferred Java descriptor as well as its Go view: structural Go
// interface satisfaction alone cannot prove nominal Java assignability.
func genericMethodProjectedResult(call ast.Expr, def *symbol.Definition, typeArgs []ast.Expr, javaBindings map[string]string, consumingJavaType string, ctx Ctx) ast.Expr {
	declaredReturn := definitionDeclaredJavaType(def)
	returnBase, returnRank := javaArrayTypeParts(strings.TrimSpace(declaredReturn.Original))
	returnParameter := declaredReturn.TypeParameterBindings[returnBase]
	_, ownerKnown := genericMethodOwnerContext(def)
	for i, tp := range def.TypeParameters {
		methodReturn := returnRank == 0 && tp.Declaration != nil && returnParameter == tp.Declaration
		if !ownerKnown && def.DeclarationNode == nil && returnParameter == nil {
			// Standalone synthetic definitions retain their existing text contract.
			// Parsed source methods instead require the captured declaration above.
			methodReturn = returnRank == 0 && (returnBase == tp.Name || returnBase == tp.EmittedName())
		}
		if methodReturn && i < len(typeArgs) {
			targetJavaType := javaBindings[tp.Name]
			targetType := typeArgs[i]
			if strings.TrimSpace(targetJavaType) != "" {
				targetType = javaTypeStringToGoTypeExpr(targetJavaType, inScopeTypeParameters(ctx), ctx)
			}
			if _, primitive := javaPrimitiveType(consumingJavaType); consumingJavaType != "" && strings.TrimSpace(consumingJavaType) != "var" && !primitive {
				// javac casts an erased invocation result to its consuming
				// reference view. A broader assignment must not introduce a
				// checkcast to the narrower argument-inferred type.
				targetJavaType = consumingJavaType
				targetType = javaTypeStringToGoTypeExpr(targetJavaType, inScopeTypeParameters(ctx), ctx)
			}
			if targetJavaType == "" {
				targetJavaType = rawTypeParameterErasure(tp, def.TypeParameters)
			}
			descriptor, _ := javaTypeDescriptorExpr(targetJavaType, ctx)
			return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{targetType}, []ast.Expr{call, descriptor})
		}
	}
	return call
}

// The compiler can already have projected an invocation to its consuming
// view. Preserve that value instead of applying a source-inferred superclass
// selector to it a second time. Both the runtime callee and Go type must match;
// an ordinary source method named ObjectView has no such guarantee.
func runtimeObjectViewHasExpectedType(expr ast.Expr, expectedJavaType string, ctx Ctx) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	indexed, ok := call.Fun.(*ast.IndexExpr)
	if !ok {
		return false
	}
	callee, ok := indexed.X.(*ast.SelectorExpr)
	if !ok || callee.Sel.Name != "ObjectView" {
		return false
	}
	if !reflect.DeepEqual(callee, stdjavaQualifiedExpr("ObjectView", ctx)) {
		return false
	}
	expectedGoType := javaTypeStringToGoTypeExpr(expectedJavaType, inScopeTypeParameters(ctx), ctx)
	return reflect.DeepEqual(indexed.Index, expectedGoType)
}

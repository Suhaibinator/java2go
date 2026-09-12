package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Go cannot put method type parameters in a method set. A Java generic
// instance method therefore exposes its erased descriptor for virtual calls;
// its existing typed helper remains the exact implementation of the body.
// Limit this entrypoint to bare method parameters: nested invariant Go
// instantiations need a separate representation plan.
func genericMethodHasErasedEntry(def *symbol.Definition) bool {
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
				return false
			}
		}
	}
	return true
}

func javaTypeContainsParameter(typ, name string) bool {
	for _, part := range strings.FieldsFunc(typ, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '$')
	}) {
		if part == name {
			return true
		}
	}
	return false
}

func genericMethodErasedJavaType(def *symbol.Definition, typ string) string {
	for _, tp := range def.TypeParameters {
		if strings.TrimSpace(typ) == tp.Name || strings.TrimSpace(typ) == tp.EmittedName() {
			return rawTypeParameterErasure(tp, def.TypeParameters)
		}
	}
	return typ
}

func eraseGenericMethodSignature(def *symbol.Definition, params, results *ast.FieldList, ctx Ctx) {
	if !genericMethodHasErasedEntry(def) {
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
	for _, tp := range def.TypeParameters {
		typeArgs = append(typeArgs, javaTypeStringToGoTypeExpr(rawTypeParameterErasure(tp, def.TypeParameters), ctx.currentClass.TypeParameterNames(), ctx))
	}
	recv := ShortName(ctx.className)
	helper := &ast.CallExpr{Fun: applyTypeArguments(&ast.Ident{Name: "New" + def.HelperName}, typeArgs), Args: []ast.Expr{&ast.Ident{Name: recv}}}
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: helper, Sel: &ast.Ident{Name: executionImplementationName(def, ctx.currentClass)}}, Args: append([]ast.Expr{&ast.Ident{Name: ctx.executionContextName}}, methodCallArgs(params)...)}
	body := &ast.BlockStmt{List: []ast.Stmt{invocationClosureCallStatement(call, results)}}
	declaration := &ast.FuncDecl{Name: &ast.Ident{Name: def.Name}, Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{{Name: recv}}, Type: &ast.StarExpr{X: receiverBaseType}}}}, Type: &ast.FuncType{Params: params, Results: results}, Body: body}
	return buildExecutionAwareFuncDecls(declaration, executionImplementationName(def, ctx.currentClass), ctx.executionContextName, ctx)
}

// Result conversion belongs after the erased call. ObjectView preserves null
// and throws Java ClassCastException when an unchecked body pollutes its result.
// Use the inferred Java descriptor as well as its Go view: structural Go
// interface satisfaction alone cannot prove nominal Java assignability.
func genericMethodProjectedResult(call ast.Expr, def *symbol.Definition, typeArgs []ast.Expr, javaBindings map[string]string, ctx Ctx) ast.Expr {
	for i, tp := range def.TypeParameters {
		if (strings.TrimSpace(def.OriginalType) == tp.Name || strings.TrimSpace(def.OriginalType) == tp.EmittedName()) && i < len(typeArgs) {
			targetJavaType := javaBindings[tp.Name]
			if targetJavaType == "" {
				targetJavaType = rawTypeParameterErasure(tp, def.TypeParameters)
			}
			descriptor, _ := javaTypeDescriptorExpr(targetJavaType, ctx)
			return stdjavaGenericCall(ctx, "ObjectView", []ast.Expr{typeArgs[i]}, []ast.Expr{call, descriptor})
		}
	}
	return call
}

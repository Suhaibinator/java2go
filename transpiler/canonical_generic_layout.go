package transpiler

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// canonicalGenericClass selects a complete physical leaf layout only when a
// universal method descriptor needs views with different type arguments. The
// source class and its binders remain unchanged; all aliases name one Go type.
func canonicalGenericClass(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || len(scope.TypeParameters) == 0 || !leafObjectMemberErasureEligible(scope, ctx) {
		return false
	}
	for _, owner := range allSourceClassScopes() {
		for _, method := range owner.Methods {
			if method == nil || len(method.TypeParameters) == 0 {
				continue
			}
			types := []string{method.OriginalType}
			for _, parameter := range method.Parameters {
				types = append(types, parameter.OriginalType)
			}
			for _, typ := range types {
				base, args := parseJavaTypeString(typ)
				if len(args) == 0 || resolveClassScopeByQualifiedName(classScopeCtx(owner, ctx), base) != scope {
					continue
				}
				for _, parameter := range method.TypeParameters {
					if javaTypeContainsParameter(typ, parameter.Name) {
						return true
					}
				}
			}
		}
	}
	return false
}

func canonicalGenericName(scope *symbol.ClassScope) string {
	base := scope.Class.Name + "Java2goErased"
	used := map[string]bool{}
	for _, owner := range allSourceClassScopes() {
		used[owner.Class.Name] = true
		for _, field := range owner.Fields {
			if field != nil && field.IsStatic {
				used[field.Name] = true
			}
		}
		for _, method := range owner.Methods {
			if method != nil {
				used[method.Name] = true
				used[method.HelperName] = true
			}
		}
	}
	name := base
	for index := 1; used[name]; index++ {
		name = base + strconv.Itoa(index)
	}
	return name
}

func canonicalGenericTypeSpecs(name string, fields *ast.FieldList, parameters []symbol.TypeParam, ctx Ctx) []ast.Spec {
	if ctx.currentClass == nil || name != ctx.currentClass.Class.Name || !canonicalGenericClass(ctx.currentClass, ctx) {
		return nil
	}
	raw := canonicalGenericName(ctx.currentClass)
	return []ast.Spec{
		&ast.TypeSpec{Name: ast.NewIdent(raw), Type: &ast.StructType{Fields: fields}},
		&ast.TypeSpec{Name: ast.NewIdent(name), TypeParams: &ast.FieldList{List: makeTypeParamFieldsInContext(parameters, ctx)}, Assign: token.Pos(1), Type: ast.NewIdent(raw)},
	}
}

// These declarations were emitted in the exact class scope. Only receiver
// binders naming that class change; constructor type parameters remain genuine
// function binders, and nested classes perform their own separate lowering.
func canonicalizeGenericReceivers(declarations []ast.Decl, ctx Ctx) {
	if !canonicalGenericClass(ctx.currentClass, ctx) {
		return
	}
	name := ctx.currentClass.Class.Name
	for _, declaration := range declarations {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok || method.Recv == nil || len(method.Recv.List) != 1 {
			continue
		}
		pointer, ok := method.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		var base ast.Expr
		switch receiver := pointer.X.(type) {
		case *ast.IndexExpr:
			base = receiver.X
		case *ast.IndexListExpr:
			base = receiver.X
		default:
			continue
		}
		ident, ok := base.(*ast.Ident)
		if !ok || ident.Name != name {
			continue
		}
		pointer.X = ast.NewIdent(canonicalGenericName(ctx.currentClass))
	}
}

func genericMethodOwnerContext(def *symbol.Definition) (Ctx, bool) {
	for _, owner := range allSourceClassScopes() {
		for _, method := range owner.Methods {
			if method == def {
				return classScopeCtx(owner, Ctx{}), true
			}
		}
	}
	return Ctx{}, false
}

// Method variables may occur inside an alias only when its entire physical
// class layout is canonical. This proof must not admit ordinary invariant Go
// containers or turn all nested parameterizations into Container[any].
func canonicalGenericMethodType(typ string, def *symbol.Definition, ctx Ctx) bool {
	contains := false
	for _, parameter := range def.TypeParameters {
		if strings.TrimSpace(typ) == parameter.Name || strings.TrimSpace(typ) == parameter.EmittedName() {
			return true
		}
		contains = contains || javaTypeContainsParameter(typ, parameter.Name) || javaTypeContainsParameter(typ, parameter.EmittedName())
	}
	if !contains {
		return true
	}
	if _, rank := javaArrayTypeParts(typ); rank != 0 {
		return false
	}
	base, args := parseJavaTypeString(typ)
	scope := resolveClassScopeByQualifiedName(ctx, base)
	if len(args) == 0 || !canonicalGenericClass(scope, ctx) {
		return false
	}
	for _, arg := range args {
		if !canonicalGenericMethodType(arg, def, ctx) {
			return false
		}
	}
	return true
}

func canonicalGenericMethodSignature(def *symbol.Definition, ctx Ctx) bool {
	if def == nil || len(def.TypeParameters) == 0 {
		return false
	}
	for _, parameter := range def.TypeParameters {
		if len(parameter.Bounds) > 1 || (len(parameter.Bounds) == 1 && strings.Contains(parameter.Bounds[0].Original, "<")) {
			return false
		}
	}
	if !canonicalGenericMethodType(def.OriginalType, def, ctx) {
		return false
	}
	for _, parameter := range def.Parameters {
		if !canonicalGenericMethodType(parameter.OriginalType, def, ctx) {
			return false
		}
	}
	return true
}

func erasedAnonymousMethodJavaType(typ string, ctx Ctx) string {
	if ctx.erasedGenericMethodBody == nil || ctx.localScope == nil {
		return typ
	}
	bindings := map[string]string{}
	for _, parameter := range ctx.erasedGenericMethodBody.TypeParameters {
		// A nested source binder with the same spelling must not be erased by an
		// enclosing callback's context when that context is cloned.
		if visibleTypeParameterDeclarationForJavaType(parameter.Name, ctx) != parameter.Declaration {
			continue
		}
		erasure := rawTypeParameterErasure(parameter, ctx.erasedGenericMethodBody.TypeParameters)
		bindings[parameter.Name] = erasure
		bindings[parameter.EmittedName()] = erasure
	}
	return substituteJavaTypeParameters(typ, bindings)
}

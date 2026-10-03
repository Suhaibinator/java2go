package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

func sourceInheritsThrowable(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil {
		return false
	}
	declaring := classScopeCtx(scope, ctx)
	return javaExceptionReferenceAssignable(scope.Superclass, "java.lang.Throwable", declaring)
}

func isThrowableMessageOverride(method *symbol.Definition, owner *symbol.ClassScope, ctx Ctx) bool {
	return method != nil && !method.IsStatic && !method.IsPrivate && !method.Constructor &&
		method.OriginalName == "getMessage" && len(method.Parameters) == 0 && sourceInheritsThrowable(owner, ctx)
}

func throwableMessageRegistration(ctx Ctx) ast.Stmt {
	return throwableTextOverrideRegistration("getMessage", "RegisterJavaThrowableMessage", ctx)
}

// The registry follows resolved source declarations, so overloads and hidden
// execution companion names cannot be mistaken for Java overrides.
func throwableTextOverrideRegistration(javaName, runtimeName string, ctx Ctx) ast.Stmt {
	for scope := ctx.currentClass; scope != nil; scope = resolveSuperclassScopeInDeclaringContext(ctx, scope) {
		for _, method := range scope.Methods {
			if !isThrowableTextOverride(method, scope, javaName, ctx) {
				continue
			}
			receiverType := &ast.StarExpr{X: ast.NewIdent(ctx.className)}
			invoke := &ast.FuncLit{Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{
				executionParameterField("execution", ctx),
				{Names: []*ast.Ident{ast.NewIdent("receiver")}, Type: ast.NewIdent("any")},
			}}, Results: &ast.FieldList{List: []*ast.Field{{Type: javaStringReferenceType(ctx)}}}}, Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.ReturnStmt{Results: []ast.Expr{&ast.CallExpr{Fun: &ast.SelectorExpr{X: &ast.TypeAssertExpr{X: ast.NewIdent("receiver"), Type: receiverType}, Sel: ast.NewIdent(executionImplementationName(method, scope, ctx))}, Args: []ast.Expr{ast.NewIdent("execution")}}}},
			}}}
			return &ast.ExprStmt{X: stdjavaCall(ctx, runtimeName, &ast.CallExpr{Fun: receiverType, Args: []ast.Expr{ast.NewIdent("nil")}}, invoke)}
		}
	}
	return nil
}

func isThrowableTextOverride(method *symbol.Definition, owner *symbol.ClassScope, javaName string, ctx Ctx) bool {
	return method != nil && !method.IsStatic && !method.IsPrivate && !method.Constructor &&
		method.OriginalName == javaName && len(method.Parameters) == 0 && sourceInheritsThrowable(owner, ctx)
}

func throwableMessageReceiverType(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		return sourceInheritsThrowable(scope, ctx)
	}
	return javaExceptionReferenceAssignable(base, "java.lang.Throwable", ctx)
}

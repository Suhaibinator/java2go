package transpiler

import "go/ast"

// A zero/default context has no execution provenance. The actual conversion
// boundary selects the supported classpath profile after unit classification;
// source ownership alone does not imply an application class loader.
type sourceRuntimeLocationProfile uint8

const (
	sourceRuntimeLocationUnknown sourceRuntimeLocationProfile = iota
	sourceRuntimeLocationClasspath
)

func sourceRuntimeLocationRegistrationStmt(id ast.Expr, ctx Ctx) ast.Stmt {
	if ctx.runtimeLocationProfile != sourceRuntimeLocationClasspath {
		return nil
	}
	location := &ast.CompositeLit{Type: stdjavaQualifiedExpr("ClassRuntimeLocation", ctx), Elts: []ast.Expr{
		metadataKey("Known", ast.NewIdent("true")),
		metadataKey("ModuleName", metadataString("")),
		metadataKey("LoaderIdentity", stdjavaQualifiedExpr("AppClassLoaderIdentity", ctx)),
		metadataKey("LoaderName", metadataString("app")),
	}}
	return &ast.ExprStmt{X: stdjavaCall(ctx, "RegisterClassRuntimeLocation", id, location)}
}

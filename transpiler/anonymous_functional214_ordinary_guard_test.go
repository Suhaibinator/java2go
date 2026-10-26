package transpiler

import (
	"go/ast"
	"testing"
)

func TestCI214AnonymousFunctionalOrdinaryBinderModel(t *testing.T) {
	rows := []struct{ name, source string }{
		{"method Function binder", `class Main{static<Function>Object make(){return new Function(){public Object apply(Object value){return value;}};}}`},
		{"qualified java binder", `class Main{static<java>Object make(){return new java.util.function.Supplier<String>(){public String get(){return "value";}};}}`},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			ctx, expression, declarations, info := anonymousFunctional209Lower(t, row.source, "")
			if info.functional != nil || info.lexicalOwner == nil || !info.scope.HasAnonymousSourceOrigin() {
				t.Fatal("ordinary source modeling acquired or lost its authority boundary")
			}
			if len(info.scope.ImplementedInterfaces) != 0 || info.scope.Superclass != "" || sourceNativeFunctionalContracts(info.scope, ctx) != nil {
				t.Fatal("ordinary TYPE header acquired a canonical family edge")
			}
			for _, name := range []string{"NewFunctionSourceView", "NewSupplierSourceView", "NewGeneratedObjectInfo", "RegisterJavaType", "RegisterJavaSourceType"} {
				if anonymousFunctional209Calls(expression, name) != 0 {
					t.Fatal("ordinary allocation acquired native carrier or registration")
				}
				for _, declaration := range declarations {
					if anonymousFunctional209Calls(declaration, name) != 0 {
						t.Fatal("ordinary source spelling emitted a canonical view or identity")
					}
				}
			}
			node := info.scope.Class.DeclarationNode
			before := len(*ctx.hoistedDecls)
			if lowerAnonymousClassToStruct(node, node.ChildByFieldName("type"), objectCreationClassBody(node), ctx.currentFile.Source, ctx) == nil || len(*ctx.hoistedDecls) != before {
				t.Fatal("same actual ordinary input did not retain ordinary cached modeling")
			}
		})
	}
}

func TestCI214AnonymousFunctionalNativeNeverDowngrades(t *testing.T) {
	for _, mode := range []string{"retired native certificate", "removed native certificate", "copied native info", "donated binder context"} {
		t.Run(mode, func(t *testing.T) {
			const source = `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`
			ctx, _, _, info := anonymousFunctional209Lower(t, source, "Function<String,String>")
			certificate := info.functional
			node := certificate.allocation
			key, _ := anonymousClassSourceKey(node)
			switch mode {
			case "retired native certificate":
				certificate.phase = anonymousFunctionalFailed
			case "removed native certificate":
				info.functional = nil
			case "copied native info":
				copied := *info
				ctx.anonymousClasses[key] = &copied
			case "donated binder context":
				donor := setupParseHelper(t, `class Donor{static<Function>Object other(){return null;}}`)
				owner := donor.Ctx.currentFile.FindClassScope("Donor")
				ctx.localScope = owner.FindMethod().ByOriginalName("other")[0]
			}
			before := anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls})
			counter := *ctx.anonClassCounter
			entries := len(ctx.anonymousClasses)
			if lowerAnonymousClassToStruct(node, certificate.typ, certificate.body, ctx.currentFile.Source, ctx) != nil || anonymousClassConstructionExpr(node, certificate.typ, info, nil, ctx.currentFile.Source, ctx) != nil {
				t.Fatal("previously native or invalid info downgraded into ordinary modeling")
			}
			if anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls}) != before || *ctx.anonClassCounter != counter || len(ctx.anonymousClasses) != entries {
				t.Fatal("failed native replay changed hoists counter or map")
			}
		})
	}
}

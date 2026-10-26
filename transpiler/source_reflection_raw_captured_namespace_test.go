package transpiler

import (
	"go/ast"
	"testing"
)

// The same valid nominal/import/binder declarations as the owner's accepted
// namespace control must retain their captured meaning on both reflection APIs.
func TestSourceReflectionRawCapturedNamespace(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/T.java":  `package p; public class T {}`,
		"q/T2.java": `package q; public class T2 {}`,
		"p/Use.java": `package p; import q.T2; public class Use<T> {
   public T2 nominal; public Class<?> metadata() { return getClass(); } }`,
	})
	scope := findQualifiedSourceClass("p.Use")
	if scope == nil || len(scope.OwnTypeParameters()) != 1 || scope.OwnTypeParameters()[0].EmittedName() != "T2" {
		t.Fatal("namespace control did not allocate the distinct emitted binder T2")
	}
	stmt := sourceClassMetadataStmt(scope, classScopeCtx(scope, ctx))
	if stmt == nil {
		t.Fatal("missing demanded source metadata")
	}
	call := stmt.(*ast.ExprStmt).X.(*ast.CallExpr)
	fields := reflectCoreSlice(t, reflectCoreKey(t, reflectCoreComposite(t, call.Args[0]), "Fields"))
	for _, field := range fields {
		descriptor := reflectCoreComposite(t, field)
		if reflectCoreString(t, reflectCoreKey(t, descriptor, "Name")) != "nominal" {
			continue
		}
		if got := reflectCoreTypeID(t, reflectCoreKey(t, descriptor, "Type")); got != "q.T2" {
			t.Fatalf("raw Field.getType descriptor=%q want nominal q.T2", got)
		}
		reflectCoreClass(t, reflectCoreKey(t, descriptor, "GenericType"), "q.T2")
		return
	}
	t.Fatal("missing nominal field descriptor")
}

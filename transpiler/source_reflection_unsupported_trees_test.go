package transpiler

import (
	"go/ast"
	"testing"
)

// These valid declarations exercise the explicit unsupported-tree policy;
// they do not predict Java wildcard/array output or weaken the JVM controls.
func TestSourceReflectionUnsupportedTrees(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"probe/Pending.java": `package probe; import java.util.List;
   public class Pending { public List<?> wildcard; public String[] array;
    public Class<?> metadata() { return getClass(); } }`,
	})
	scope := findQualifiedSourceClass("probe.Pending")
	if scope == nil {
		t.Fatal("missing Java declaration")
	}
	statement := sourceClassMetadataStmt(scope, classScopeCtx(scope, ctx))
	if statement == nil {
		t.Fatal("missing demanded metadata")
	}
	call := statement.(*ast.ExprStmt).X.(*ast.CallExpr)
	fields := reflectCoreSlice(t, reflectCoreKey(t, reflectCoreComposite(t, call.Args[0]), "Fields"))
	for _, name := range []string{"wildcard", "array"} {
		t.Run(name, func(t *testing.T) {
			for _, field := range fields {
				descriptor := reflectCoreComposite(t, field)
				if reflectCoreString(t, reflectCoreKey(t, descriptor, "Name")) != name {
					continue
				}
				kind := reflectCoreKey(t, reflectCoreComposite(t, reflectCoreKey(t, descriptor, "GenericType")), "Kind")
				conversion, ok := kind.(*ast.CallExpr)
				if !ok || len(conversion.Args) != 1 {
					t.Fatal("unsupported tree silently erased")
				}
				selector, ok := conversion.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "ReflectTypeKind" {
					t.Fatal("unsupported kind ABI")
				}
				literal, ok := conversion.Args[0].(*ast.BasicLit)
				if !ok || literal.Value != "255" {
					t.Fatal("unsupported kind must reach the closed runtime rejection")
				}
				return
			}
			t.Fatal("missing declared field", name)
		})
	}
}

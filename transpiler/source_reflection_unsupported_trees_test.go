package transpiler

import (
	"go/ast"
	"testing"
)

// Preserve the original declarations while asserting their complete supported
// metadata trees. Independent controls retain the remaining rejection policy.
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
				tree := reflectCoreKey(t, descriptor, "GenericType")
				if name == "array" {
					array := reflectCoreComposite(t, tree)
					reflectCoreKind(t, array, "ReflectClassKind")
					reflectMetadataArrayRaw(t, reflectCoreKey(t, array, "Raw"), 1, "java.lang.String")
				} else {
					parameter := reflectCoreComposite(t, tree)
					reflectCoreKind(t, parameter, "ReflectParameterizedKind")
					if got := reflectCoreTypeID(t, reflectCoreKey(t, parameter, "Raw")); got != "java.util.List" {
						t.Fatalf("raw=%q", got)
					}
					arguments := reflectCoreSlice(t, reflectCoreKey(t, parameter, "Arguments"))
					if len(arguments) != 1 {
						t.Fatal("wildcard argument arity")
					}
					wildcard := reflectCoreComposite(t, arguments[0])
					reflectCoreKind(t, wildcard, "ReflectWildcardKind")
					upper, lower := reflectCoreSlice(t, reflectCoreKey(t, wildcard, "UpperBounds")), reflectCoreSlice(t, reflectCoreKey(t, wildcard, "LowerBounds"))
					if len(upper) != 1 || len(lower) != 0 {
						t.Fatal("unbounded wildcard bounds")
					}
					reflectCoreClass(t, upper[0], "java.lang.Object")
				}
				return
			}
			t.Fatal("missing declared field", name)
		})
	}
}

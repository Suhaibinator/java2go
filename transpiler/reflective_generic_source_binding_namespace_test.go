package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// These are Java declaration-name controls, not inferred Go ABI spellings.
// The nominal p.T forces the independent binder allocator to emit T as T2.
func TestReflectiveGenericSourceBindingNamespace(t *testing.T) {
	for _, test := range []struct {
		name, owner, sourceOwner, field, want string
		uncaptured, variable, unsupported     bool
	}{
		{name: "captured_imported_alias", owner: "p.Use", field: "nominal", want: "q.T2"},
		{name: "uncaptured_imported_alias", owner: "p.Use", field: "nominal", want: "q.T2", uncaptured: true},
		{name: "static_nested_owner", owner: "p.Host.Holder", sourceOwner: "p.Host", field: "variable", unsupported: true},
		{name: "captured_class_variable", owner: "p.Use", field: "variable", want: "p.Use", variable: true},
		{name: "nonstatic_outer_variable", owner: "p.Host.Inner", field: "variable", want: "p.Host", variable: true},
		{name: "captured_shadowed_outer_variable", owner: "p.Host.Shadow", sourceOwner: "p.Host", field: "variable", want: "p.Host", variable: true},
		{name: "qualified_static_nominal_control", owner: "p.Host.Holder", field: "qualified", want: "q.T"},
	} {
		t.Run(test.name, func(t *testing.T) {
			sources := map[string]string{
				"p/T.java":   `package p; public class T {}`,
				"q/T.java":   `package q; public class T {}`,
				"q/T2.java":  `package q; public class T2 {}`,
				"p/Use.java": `package p; import q.T2; public class Use<T> { public T2 nominal; public T variable; }`,
			}
			if test.owner != "p.Use" {
				sources["p/Host.java"] = `package p; public class Host<T> { public T variable; public static class Holder { public q.T qualified; } public class Inner { public T variable; } public class Shadow<T> {} }`
			}
			ctx := sourceGenericViewDemandTestContext(t, sources)
			if test.owner == "p.Use" {
				use := findQualifiedSourceClass("p.Use")
				if use == nil || len(use.OwnTypeParameters()) != 1 || use.OwnTypeParameters()[0].EmittedName() != "T2" {
					t.Fatal("control must reserve nominal T and allocate source binder T as Go T2")
				}
			}
			owner := findQualifiedSourceClass(test.owner)
			if owner == nil {
				t.Fatal("missing declaration owner", test.owner)
			}
			fieldOwner := owner
			if test.sourceOwner != "" {
				fieldOwner = findQualifiedSourceClass(test.sourceOwner)
			}
			if fieldOwner == nil {
				t.Fatal("missing captured declaration owner")
			}
			fields := fieldOwner.FindField().ByOriginalName(test.field)
			if len(fields) != 1 {
				t.Fatal("missing declared field", test.field)
			}
			field := fields[0]
			if test.name == "captured_imported_alias" && (field.TypeParameterBindings == nil || field.TypeParameterBindings["T2"] != nil) {
				t.Fatal("captured source namespace must contain T but no emitted T2 alias")
			}
			if test.name == "static_nested_owner" && (owner.IsInner || len(owner.TypeParameters) != 0) {
				t.Fatal("static nested declaration must not carry enclosing Host.T")
			}
			javaType := symbol.JavaType{Original: field.OriginalType, TypeParameterBindings: field.TypeParameterBindings}
			if test.uncaptured {
				javaType.TypeParameterBindings = nil
			}
			expression, supported := reflectiveTypeDescriptorExpr(javaType, classScopeCtx(owner, ctx))
			if test.unsupported {
				if supported {
					t.Fatal("static nested owner admitted inaccessible captured enclosing binder")
				}
				return
			}
			if !supported {
				t.Fatal("supported Java class-owned declaration rejected")
			}
			if !test.variable {
				reflectCoreClass(t, expression, test.want)
				return
			}
			descriptor := reflectCoreComposite(t, expression)
			reflectCoreKind(t, descriptor, "ReflectVariableKind")
			if got := reflectCoreString(t, reflectCoreKey(t, descriptor, "VariableName")); got != "T" {
				t.Fatalf("source variable name=%q want T", got)
			}
			if got := reflectCoreTypeID(t, reflectCoreKey(t, descriptor, "VariableDeclaration")); got != test.want {
				t.Fatalf("declaring class=%q want %q", got, test.want)
			}
		})
	}
}

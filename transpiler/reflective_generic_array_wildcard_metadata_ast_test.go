package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func reflectMetadataArrayRaw(t *testing.T, expression ast.Expr, rank int, leaf string) {
	t.Helper()
	for i := 0; i < rank; i++ {
		call, ok := expression.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			t.Fatalf("array rank %d: %T", i, expression)
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ArrayTypeID" {
			t.Fatal("canonical array identity constructor")
		}
		expression = call.Args[0]
	}
	if got := reflectCoreTypeID(t, expression); got != leaf {
		t.Fatalf("array component=%q want%q", got, leaf)
	}
}
func reflectMetadataVariable(t *testing.T, expression ast.Expr, owner, name string) {
	t.Helper()
	tree := reflectCoreComposite(t, expression)
	reflectCoreKind(t, tree, "ReflectVariableKind")
	if got := reflectCoreTypeID(t, reflectCoreKey(t, tree, "VariableDeclaration")); got != owner {
		t.Fatalf("variable owner=%q want%q", got, owner)
	}
	if got := reflectCoreString(t, reflectCoreKey(t, tree, "VariableName")); got != name {
		t.Fatalf("variable name=%q want%q", got, name)
	}
}

func TestReflectiveGenericArrayWildcardBindingNamespace(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/T.java":   `package p;public class T {}`,
		"q/T2.java":  `package q;public class T2 {}`,
		"p/Use.java": `package p;import q.T2;public class Use<T>{T2[] nominal;T[] variable;public Class<?> metadata(){return getClass();}}`,
	})
	owner := findQualifiedSourceClass("p.Use")
	if owner.OwnTypeParameters()[0].EmittedName() != "T2" {
		t.Fatal("binder alias collision control")
	}
	for _, field := range owner.Fields {
		tree, ok := reflectiveTypeDescriptorExpr(symbol.JavaType{Original: field.OriginalType, TypeParameterBindings: field.TypeParameterBindings}, classScopeCtx(owner, ctx))
		if !ok {
			t.Fatal("supported array", field.OriginalName)
		}
		descriptor := reflectCoreComposite(t, tree)
		switch field.OriginalName {
		case "nominal":
			reflectCoreKind(t, descriptor, "ReflectClassKind")
			reflectMetadataArrayRaw(t, reflectCoreKey(t, descriptor, "Raw"), 1, "q.T2")
		case "variable":
			reflectCoreKind(t, descriptor, "ReflectGenericArrayKind")
			reflectMetadataVariable(t, reflectCoreKey(t, descriptor, "Component"), "p.Use", "T")
		}
	}
}

func TestReflectiveGenericArrayWildcardRetainedRejections(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{
		"p/Host.java": `package p;public class Host<T>{T value;public class Inner<U>{}public static class Static{}public <M> M method(M value){return value;}}`,
	})
	owner := findQualifiedSourceClass("p.Host")
	declaring := classScopeCtx(owner, ctx)
	for _, original := range []string{"? extends", "? super", "? extends int", "? super void", "?[]", "? extends ?", "? super ? extends T", "void[]", "p.Host<T>.Inner<String>[]", "p.Host<T>.Inner<String>", "? extends p.Host<T>.Inner<String>"} {
		t.Run(original, func(t *testing.T) {
			if _, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{Original: original}, declaring); supported {
				t.Fatal("invalid or retained unsupported tree admitted")
			}
		})
	}
	bindings := symbol.VisibleTypeParamBindings(owner.TypeParameters)
	static := classScopeCtx(findQualifiedSourceClass("p.Host.Static"), ctx)
	for _, original := range []string{"T[]", "? super T[]"} {
		if _, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{Original: original, TypeParameterBindings: bindings}, static); supported {
			t.Fatal("inaccessible static outer variable", original)
		}
	}
	methods := owner.FindMethod().ByOriginalName("method")
	if len(methods) != 1 {
		t.Fatal("missing method")
	}
	methodCtx := declaring.Clone()
	methodCtx.localScope = methods[0]
	for _, original := range []string{"M[]", "? extends M"} {
		if _, supported := reflectiveTypeDescriptorExpr(symbol.JavaType{Original: original, TypeParameterBindings: symbol.VisibleTypeParamBindings(methods[0].TypeParameters)}, methodCtx); supported {
			t.Fatal("method variable acquired class declaration key", original)
		}
	}
}

func TestReflectiveGenericArrayWildcardRetainedUnsupportedDescriptor(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Host.java": `package p;public class Host<T>{public Inner<String> held;public class Inner<U>{}public Class<?> metadata(){return getClass();}}`})
	owner := findQualifiedSourceClass("p.Host")
	statement := sourceClassMetadataStmt(owner, classScopeCtx(owner, ctx))
	if statement == nil {
		t.Fatal("missing reflection metadata")
	}
	descriptor := reflectCoreComposite(t, statement.(*ast.ExprStmt).X.(*ast.CallExpr).Args[0])
	fields := reflectCoreSlice(t, reflectCoreKey(t, descriptor, "Fields"))
	for _, field := range fields {
		entry := reflectCoreComposite(t, field)
		if reflectCoreString(t, reflectCoreKey(t, entry, "Name")) != "held" {
			continue
		}
		tree := reflectCoreComposite(t, reflectCoreKey(t, entry, "GenericType"))
		kind, ok := reflectCoreKey(t, tree, "Kind").(*ast.CallExpr)
		if !ok || len(kind.Args) != 1 {
			t.Fatal("unsupported owner tree silently erased")
		}
		selector, ok := kind.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "ReflectTypeKind" {
			t.Fatal("unsupported kind ABI")
		}
		value, ok := kind.Args[0].(*ast.BasicLit)
		if !ok || value.Value != "255" {
			t.Fatal("unsupported owner tree lost closed runtime rejection")
		}
		return
	}
	t.Fatal("missing original unsupported field")
}

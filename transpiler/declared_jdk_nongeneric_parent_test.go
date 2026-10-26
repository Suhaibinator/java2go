package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestDeclaredJDKNonGenericParentConversions(t *testing.T) {
	for _, row := range []struct {
		name, setup, actual, formal string
		want                        bool
	}{
		{"Class-wildcard-Type", "", "java.lang.Class<?>", "java.lang.reflect.Type", true},
		{"Class-wildcard-GenericDeclaration", "", "java.lang.Class<?>", "java.lang.reflect.GenericDeclaration", true},
		{"Class-wildcard-Object", "", "java.lang.Class<?>", "java.lang.Object", true},
		{"Class-wildcard-Serializable", "", "java.lang.Class<?>", "java.io.Serializable", true},
		{"Class-concrete-Type", "", "java.lang.Class<java.lang.String>", "java.lang.reflect.Type", true},
		{"Class-upper-Type", "", "java.lang.Class<? extends java.lang.String>", "java.lang.reflect.Type", true},
		{"TypeVariable-Type", "", "java.lang.reflect.TypeVariable<?>", "java.lang.reflect.Type", true},
		{"TypeVariable-Object-upper-Type", "", "java.lang.reflect.TypeVariable<? extends Object>", "java.lang.reflect.Type", true},
		{"ArrayList-Serializable", "", "java.util.ArrayList<java.lang.String>", "java.io.Serializable", true},
		{"ArrayList-Cloneable", "", "java.util.ArrayList<java.lang.String>", "java.lang.Cloneable", true},
		{"reverse", "", "java.lang.reflect.Type", "java.lang.Class<?>", false},
		{"unrelated", "", "java.lang.Class<?>", "java.lang.Runnable", false},
		{"typed-target-invariant", "", "java.util.ArrayList<java.lang.String>", "java.util.Collection<java.lang.Integer>", false},
		{"source-Class", "class Class<T>{}", "Class<?>", "java.lang.reflect.Type", false},
		{"source-Type", "class Type{}", "java.lang.Class<?>", "Type", false},
		{"unknown-nested", "", "java.lang.Class<Missing>", "java.lang.reflect.Type", false},
		{"invalid-proper-bound", "", "java.lang.reflect.TypeVariable<java.lang.String>", "java.lang.reflect.Type", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.setup+" public class Use {static void test("+row.actual+" value){}}")
			ctx := h.Ctx.Clone()
			ctx.currentClass = h.File.Symbols.FindClassScope("Use")
			if ctx.currentClass == nil {
				t.Fatal("actual Use caller declaration missing")
			}
			ctx.localScope = ctx.currentClass.FindMethodByName("test", nil)
			if ctx.localScope == nil {
				t.Fatal("actual Use.test declaration missing")
			}
			actual := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			formal := declaredJavaTypeOrigin(symbol.JavaType{Original: row.formal}, ctx)
			cost, exact, applies, handled := declaredOriginConversion(actual, formal, ctx)
			t.Logf("actual=%+v formal=%+v conversion=%d/%v/%v/%v", actual, formal, cost, exact, applies, handled)
			if applies != row.want || row.want && !handled {
				t.Fatalf("conversion=%d/%v/%v/%v want applicable=%v", cost, exact, applies, handled, row.want)
			}
			if row.want && row.formal != "java.lang.Object" && (cost != 1 || exact) {
				t.Fatal("existing native widening cost/specificity policy changed")
			}
		})
	}
}

func TestDeclaredJDKNonGenericParentUnknownRemainsHeld(t *testing.T) {
	for _, typ := range []string{"java.lang.Class<? super String>", "java.lang.reflect.TypeVariable<? extends java.io.Serializable>", "java.lang.Class<T>", "java.lang.Class<?>[]", "java.lang.Enum<?>"} {
		t.Run(typ, func(t *testing.T) {
			h := setupParseHelper(t, "public class Use<T> {void test("+typ+" value){}}")
			ctx := h.Ctx.Clone()
			ctx.currentClass = h.File.Symbols.FindClassScope("Use")
			if ctx.currentClass == nil {
				t.Fatal("actual Use caller declaration missing")
			}
			ctx.localScope = ctx.currentClass.FindMethodByName("test", nil)
			if ctx.localScope == nil {
				t.Fatal("actual Use.test declaration missing")
			}
			actual := declaredJavaTypeOrigin(definitionDeclaredJavaType(ctx.localScope.Parameters[0]), ctx)
			formal := declaredJavaTypeOrigin(symbol.JavaType{Original: "java.lang.reflect.Type"}, ctx)
			if declaredJDKNonGenericParentOriginAssignable(actual, formal, ctx) {
				t.Fatal("unsupported compatibility/binder/array/recipe gained positive authority")
			}
		})
	}
}

func TestDeclaredJDKNonGenericParentStaticOverloads(t *testing.T) {
	for _, row := range []struct {
		name, left, right, want string
		ambiguous               bool
	}{
		{"Type-over-Object", "java.lang.reflect.Type", "java.lang.Object", "java.lang.reflect.Type", false},
		{"Type-vs-Serializable", "java.lang.reflect.Type", "java.io.Serializable", "", true},
		{"Type-vs-GenericDeclaration", "java.lang.reflect.Type", "java.lang.reflect.GenericDeclaration", "", true},
		{"GenericDeclaration-over-AnnotatedElement", "java.lang.reflect.GenericDeclaration", "java.lang.reflect.AnnotatedElement", "java.lang.reflect.GenericDeclaration", false},
	} {
		t.Run(row.name, func(t *testing.T) {
			ctx := sourceGenericViewDemandTestContext(t, map[string]string{
				"library/Pickers.java": "package library; public class Pickers {public static int pick(" + row.left + " value){return 1;}public static int pick(" + row.right + " value){return 2;}}",
				"probe/Use.java":       "package probe; import static library.Pickers.pick; public class Use {static int test(java.lang.Class<?> value){return pick(value);}}",
			})
			owner := findQualifiedSourceClass("probe.Use")
			if owner == nil {
				t.Fatal("actual source caller absent")
			}
			ctx = classScopeCtx(owner, ctx)
			ctx.localScope = owner.FindMethodByName("test", nil)
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if call == nil {
				t.Fatal("actual source invocation absent")
			}
			resolution := resolveStaticImportedMethod(call, ctx, ctx.currentFile.Source)
			if row.ambiguous {
				if resolution.source != nil || resolution.intrinsic != "" || resolution.problem != "ambiguous static method import pick" {
					t.Fatalf("equal-cost unrelated interfaces must remain ambiguous: %+v", resolution)
				}
				return
			}
			if resolution.problem != "" || resolution.source == nil || resolution.source.def.Parameters[0].OriginalType != row.want {
				t.Fatalf("wrong exact existing overload selection: %+v", resolution)
			}
		})
	}
}

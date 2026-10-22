package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestRegisteredStaticIntrinsicFormalDeclarations(t *testing.T) {
	helper := setupParseHelper(t, `class String {} class Objects {} class Supplier<X> {}
public class Use<T> { static <T> void guard(T value) {} }`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Use")
	ctx.localScope = ctx.currentClass.FindMethodByName("guard", nil)
	if ctx.localScope == nil || len(ctx.localScope.TypeParameters) != 1 {
		t.Fatal("real caller binder missing")
	}
	callerT := ctx.localScope.TypeParameters[0].Declaration
	signatures := staticIntrinsicImportSignatures[intrinsicKey{"Objects", "requireNonNull"}]
	if len(signatures) != 3 {
		t.Fatal("actual three canonical signatures missing")
	}
	for _, def := range signatures {
		declaring, ok := registeredStaticIntrinsicDeclarationContext(def)
		if !ok || declaring.localScope != def || declaring.currentClass != nil || declaring.currentFile != nil || len(declaring.syntheticTypeParameters) != 0 {
			t.Fatal("native declaration inherited caller state")
		}
		formals := methodFormalOrigins(def, nil, nil, ctx)
		if len(formals) != len(def.Parameters) || formals[0].unresolved || formals[0].parameter != def.TypeParameters[0].Declaration || formals[0].parameter == callerT {
			t.Fatalf("formal T not bound to exact native declaration: %+v", formals)
		}
		if len(formals) == 2 && def.Parameters[1].OriginalType == "java.lang.String" && (formals[1].externalName != "java.lang.String" || formals[1].nominalScope != nil || formals[1].unresolved) {
			t.Fatal("native String formal rebound to caller source String")
		}
		copy := *def
		if _, ok := registeredStaticIntrinsicDeclarationContext(&copy); ok {
			t.Fatal("copied unregistered definition gained canonical authority")
		}
	}
	collections := staticIntrinsicImportSignatures[intrinsicKey{"Collections", "emptySet"}]
	if len(collections) != 1 {
		t.Fatal("real second registered generic family missing")
	}
	def := collections[0]
	declaring, ok := registeredStaticIntrinsicDeclarationContext(def)
	if !ok || declaring.localScope != def {
		t.Fatal("shared helper rejected actual Collections declaration")
	}
	result := declaredJavaTypeOrigin(symbol.JavaType{Original: def.OriginalType, TypeParameterBindings: symbol.VisibleTypeParamBindings(def.TypeParameters)}, declaring)
	if result.externalName != "java.util.Set" || len(result.arguments) != 1 || result.arguments[0].parameter != def.TypeParameters[0].Declaration || result.arguments[0].unresolved {
		t.Fatalf("second real generic declaration lost exact nested T: %+v", result)
	}
	for _, unknown := range []*symbol.Definition{nil, ctx.localScope, {OriginalName: "requireNonNull", IsStatic: true, TypeParameters: []symbol.TypeParam{symbol.NewTypeParam("T", nil)}}} {
		if _, ok := registeredStaticIntrinsicDeclarationContext(unknown); ok {
			t.Fatal("source or unregistered declaration borrowed intrinsic authority")
		}
	}
	if ctx.currentClass != helper.File.Symbols.FindClassScope("Use") || ctx.localScope.TypeParameters[0].Declaration != callerT {
		t.Fatal("formal interpretation changed actual caller state")
	}
}

func TestObjectsStaticImport516LexicalSourceShields(t *testing.T) {
	for _, parameter := range []string{"Class<?> value", "int value"} {
		t.Run(parameter, func(t *testing.T) {
			h := setupParseHelper(t, `import static java.util.Objects.requireNonNull; public class Use { static Class<?> requireNonNull(`+parameter+`){return null;} static Class<?> run(Class<?> value){return requireNonNull(value);} }`)
			ctx := h.Ctx.Clone()
			ctx.localScope = ctx.currentClass.FindMethodByName("run", nil)
			call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
			if call == nil {
				t.Fatal("actual source call missing")
			}
			got := resolveStaticImportedMethod(call, ctx, h.File.Source)
			if got.intrinsic != "" || got.source != nil || got.problem != "" {
				t.Fatalf("lexical source method must hide static import even for inapplicable overload: %+v", got)
			}
			sourceDef := ctx.currentClass.FindMethodByName("requireNonNull", nil)
			if _, ok := registeredStaticIntrinsicDeclarationContext(sourceDef); ok {
				t.Fatal("same-name source overload borrowed native declaration scope")
			}
		})
	}
}

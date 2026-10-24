package transpiler

import (
	"testing"
)

func TestClassBoundReturnOriginCurrentOwnerIdentity(t *testing.T) {
	for _, text := range []string{
		`class Animal {} class Bound<T extends Animal>{T once(T value){return value;} T run(T value){return once(value);}}`,
		`package probe; class Animal {} class Bound<Animal extends probe.Animal>{Animal once(Animal value){return value;} Animal run(Animal value){return once(value);}}`,
		`class Animal {} class Bound<T extends Animal>{T once(T value){return value;} <T> Object run(){return once(null);}}`,
	} {
		helper := setupParseHelper(t, text)
		ctx := helper.Ctx.Clone()
		ctx.currentClass = helper.File.Symbols.FindClassScope("Bound")
		ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
		call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
		parameter := ctx.currentClass.TypeParameters[0]
		var origin inferredJavaTypeOrigin
		got, known := inferUserMethodReturnType(call, ctx, helper.File.Source, &origin)
		if !known || got != parameter.EmittedName() || origin.parameter != parameter.Declaration {
			t.Fatalf("selected class return lost identity: got %q known=%v origin=%v expected=%v", got, known, origin.parameter, parameter.Declaration)
		}
		plain, known := inferUserMethodReturnType(call, ctx, helper.File.Source)
		if !known || plain != got {
			t.Fatal("inference requires optional origin output")
		}
	}
}
func TestClassBoundReturnOriginForeignReceiverStillRefused(t *testing.T) {
	helper := setupParseHelper(t, `class Animal{} class Bound<T extends Animal>{T once(T value){return value;}} class Other<T extends Animal>{T run(Bound<T> other,T value){return other.once(value);}}`)
	ctx := helper.Ctx.Clone()
	ctx.currentClass = helper.File.Symbols.FindClassScope("Other")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("run")[0]
	call := findNode(ctx.localScope.DeclarationNode, "method_invocation")
	if got, known := inferUserMethodReturnType(call, ctx, helper.File.Source); known {
		t.Fatalf("expanded unproved foreign receiver: %s", got)
	}
}

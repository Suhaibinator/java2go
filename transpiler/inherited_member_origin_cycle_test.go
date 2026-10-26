package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestInheritedMemberHeaderOriginUsesRegisteredCapturedSlots(t *testing.T) {
	helper := setupParseHelper(t, originalInheritedGenericFieldSource(t))
	program := helper.File.Symbols.FindClassScope("InheritedGenericFieldProgram")
	if program == nil {
		t.Fatal("original program declaration missing")
	}
	var outer, parent, child *symbol.ClassScope
	for _, scope := range program.Subclasses {
		if scope.Class.OriginalName == "Outer" {
			outer = scope
		}
	}
	if outer == nil {
		t.Fatal("original outer declaration missing")
	}
	for _, scope := range outer.Subclasses {
		switch scope.Class.OriginalName {
		case "Parent":
			parent = scope
		case "Child":
			child = scope
		}
	}
	if parent == nil || child == nil || len(outer.TypeParameters) != 1 || len(child.TypeParameters) != 2 || len(parent.TypeParameters) != 2 {
		t.Fatal("actual nested declaration layout missing")
	}
	outerParameter := outer.TypeParameters[0].Declaration
	childParameter := child.OwnTypeParameters()[0].Declaration
	if child.TypeParameters[0].Declaration != outerParameter || parent.TypeParameters[0].Declaration != outerParameter || childParameter == outerParameter {
		t.Fatal("actual hidden and written declaration identities lost")
	}
	ctx := classHeaderTypeCtx(child, helper.Ctx)
	arguments := classParameterTypeOrigins(child)
	carried := declarationCapturedClassTypeOrigins(parent, child, arguments, ctx)
	if value, ok := carried[outerParameter]; !ok || !completeJavaTypeOrigin(value) || value.parameter != outerParameter {
		t.Fatal("existing registered lexical proof does not retain the actual Outer.T")
	}
	// This is the original superclass use, with its original declaration bindings.
	// It must complete Parent's hidden slot without re-entering the same header.
	origin := declaredJavaTypeOrigin(symbol.JavaType{Original: child.Superclass, TypeParameterBindings: symbol.VisibleTypeParamBindings(child.TypeParameters)}, ctx)
	if origin.unresolved || origin.nominalScope != parent || len(origin.arguments) != 2 || origin.arguments[0].parameter != outerParameter || origin.arguments[1].parameter != childParameter {
		t.Fatalf("actual Parent<V> origin lost hidden Outer.T or written Child.V: %#v", origin)
	}
}

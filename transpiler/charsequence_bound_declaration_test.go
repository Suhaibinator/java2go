package transpiler

import "testing"

func TestCharSequenceBoundsKeepDeclarationContext(t *testing.T) {
	helper := setupParseHelper(t, `interface Textual extends java.lang.CharSequence {}
 class Scope<Parent extends Textual,Child extends Parent> {
  Child saved;
  <Parent> void parameterShadow(Parent ignored) {}
  <Textual> void nominalShadow(Textual ignored) {}
  <CharSequence extends Textual> void targetShadow(CharSequence value) {}
 }`)
	ctx := helper.Ctx
	ctx.currentClass = resolveClassScopeByQualifiedName(ctx, "Scope")
	if ctx.currentClass == nil {
		t.Fatal("missing Scope")
	}
	fieldType := definitionJavaType(ctx.currentClass.Fields[0])
	for _, method := range []string{"parameterShadow", "nominalShadow", "targetShadow"} {
		ctx.localScope = ctx.currentClass.FindMethodByName(method, nil)
		if ctx.localScope == nil {
			t.Fatalf("missing %s", method)
		}
		if !sourceCharSequenceAssignable(fieldType, "java.lang.CharSequence", ctx) {
			t.Errorf("%s lost outer dependent bound", method)
		}
		if method == "targetShadow" && sourceCharSequenceAssignable("Textual", "CharSequence", ctx) {
			t.Error("expected binder treated as nominal interface")
		}
	}
}

package transpiler

import "testing"

func TestCharSequenceCanonicalBinderAndImportGuards(t *testing.T) {
	helper := setupParseHelper(t, `class Scope<CharSequence> {<CharSequence> void call() {} }`)
	ctx := helper.Ctx
	if isBuiltinCharSequence("CharSequence", ctx) {
		t.Fatal("class type parameter treated as builtin")
	}
	if !isBuiltinCharSequence("java.lang.CharSequence", ctx) {
		t.Fatal("qualified builtin hidden by type parameter")
	}
	ctx.localScope = ctx.currentClass.FindMethodByName("call", nil)
	if isBuiltinCharSequence("CharSequence", ctx) {
		t.Fatal("method type parameter treated as builtin")
	}
	ctx.localScope = nil
	ctx.currentClass = nil
	ctx.currentFile.Imports = map[string]string{"CharSequence": "foreign"}
	if isBuiltinCharSequence("CharSequence", ctx) {
		t.Fatal("foreign explicit import treated as builtin")
	}
	if !isBuiltinCharSequence("java.lang.CharSequence", ctx) {
		t.Fatal("foreign import hid qualified builtin")
	}
}

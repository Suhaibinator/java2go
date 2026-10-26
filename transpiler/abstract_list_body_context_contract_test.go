package transpiler

import "testing"

func TestAbstractListBodyOnlyErasurePreservesCallableSignature(t *testing.T) {
	h := setupParseHelper(t, `class One<E> extends java.util.AbstractList<E>{E value;private void poison(Object input){value=(E)input;}public int size(){return 1;}public E get(int i){poison("text");return value;}}`)
	owner := h.Ctx.currentFile.TopLevelClasses[0]
	for _, method := range owner.Methods {
		if method.OriginalName != "poison" {
			continue
		}
		plan := sourceAbstractListVirtualBody(owner, method, h.Ctx)
		if plan == nil || sourceAbstractListErasedSignature(plan) || plan.erasedResult != "void" || len(plan.erasedParameters) != 1 || plan.erasedParameters[0] != "Object" {
			t.Fatal("body-only context changed helper descriptor or was omitted")
		}
		if sourceNativeVirtualHasParameterABI(owner, method, h.Ctx) {
			t.Fatal("signature-free body acquired erased parameter ABI")
		}
		if _, changed := sourceNativeVirtualResultErasure(owner, method, h.Ctx); changed {
			t.Fatal("void body acquired erased result ABI")
		}
		ctx := h.Ctx.Clone()
		ctx.currentClass = owner
		ctx.localScope = method
		ctx.sourceNativeVirtualBody = plan
		if !plan.scalar("E", ctx) {
			t.Fatal("actual helper body lost owner E erasure")
		}
		copied := *method
		if sourceAbstractListVirtualBody(owner, &copied, h.Ctx) != nil {
			t.Fatal("copied helper donated body context")
		}
		ctx.localScope = &copied
		if plan.scalar("E", ctx) {
			t.Fatal("body context transferred to copied method")
		}
		return
	}
	t.Fatal("actual helper missing")
}

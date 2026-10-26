package transpiler

import "testing"

func TestAbstractListOverrideBridgeRequiresActualDeclaringOwners(t *testing.T) {
	h := setupParseHelper(t, `class Base<E> extends java.util.AbstractList<E>{public int size(){return 1;}public E get(int i){return null;}}class Leaf extends Base<String>{public String get(int i){return "leaf";}}`)
	base, leaf := h.Ctx.currentFile.TopLevelClasses[0], h.Ctx.currentFile.TopLevelClasses[1]
	var method, ancestor = leaf.Methods[0], base.Methods[0]
	for _, candidate := range leaf.Methods {
		if candidate.OriginalName == "get" {
			method = candidate
		}
	}
	for _, candidate := range base.Methods {
		if candidate.OriginalName == "get" {
			ancestor = candidate
		}
	}
	actual, ok := sourceAbstractListSpecializedOverride(leaf, method, h.Ctx)
	if !ok || actual.family.owner != base || actual.family.method != ancestor || actual.bridge.owner != leaf || actual.bridge.method != method || actual.family.erasedResult != "java.lang.Object" || actual.bridge.sourceNativeExactExecutionName == "" {
		t.Fatal("actual native-backed override descriptor not proved")
	}
	copiedOwner := *leaf
	if _, ok := sourceAbstractListSpecializedOverride(&copiedOwner, method, h.Ctx); ok {
		t.Fatal("detached owner donated native bridge")
	}
	copiedMethod := *method
	if _, ok := sourceAbstractListSpecializedOverride(leaf, &copiedMethod, h.Ctx); ok {
		t.Fatal("copied declaration metadata donated native bridge")
	}
	originalResult := method.OriginalType
	method.OriginalType = "Object"
	if _, ok := sourceAbstractListSpecializedOverride(leaf, method, h.Ctx); ok {
		t.Fatal("altered override result donated native bridge")
	}
	method.OriginalType = originalResult
	originalParent := leaf.Superclass
	leaf.Superclass = "Base<Integer>"
	if _, ok := sourceAbstractListSpecializedOverride(leaf, method, h.Ctx); ok {
		t.Fatal("altered parent specialization donated native bridge")
	}
	leaf.Superclass = originalParent
	originalDeclaration := ancestor.DirectTypeParameter
	ancestor.DirectTypeParameter = nil
	if _, ok := sourceAbstractListSpecializedOverride(leaf, method, h.Ctx); ok {
		t.Fatal("altered ancestor binding donated native bridge")
	}
	ancestor.DirectTypeParameter = originalDeclaration
	if _, ok := sourceAbstractListSpecializedOverride(leaf, method, h.Ctx); !ok {
		t.Fatal("refused queries poisoned restored actual family")
	}
}

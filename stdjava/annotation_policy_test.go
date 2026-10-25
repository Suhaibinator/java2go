package stdjava

import "testing"

func TestAnnotationDescriptorInheritedUnavailableFactorySelectivity(t *testing.T) {
	const parent TypeID = "annotation.policy.NativeParent"
	const child TypeID = "annotation.policy.NativeChild"
	const inherited TypeID = "annotation.policy.NativeInherited"
	const direct TypeID = "annotation.policy.NativeDirect"
	const other TypeID = "annotation.policy.Other"
	RegisterJavaType(parent, ObjectTypeID)
	RegisterJavaType(child, parent)
	execution := NewExecution()
	calls := 0
	unavailable := func(actual *Execution) any {
		if actual != execution {
			t.Fatal("lost annotation execution")
		}
		calls++
		panic(NewUnsupportedOperationException("annotation proxy metadata is unavailable"))
	}
	RegisterClassDescriptor(ClassDescriptor{Type: parent, AnnotationValues: []AnnotationDescriptor{
		{Type: inherited, Inherited: true, Factory: unavailable},
		{Type: direct, Factory: unavailable},
	}})
	RegisterClassDescriptor(ClassDescriptor{Type: child})
	target := ClassLiteral(child)
	if !target.IsAnnotationPresent(ClassLiteral(inherited)) || target.IsAnnotationPresent(ClassLiteral(direct)) || target.IsAnnotationPresent(ClassLiteral(other)) {
		t.Fatal("wrong descriptor inheritance/presence policy")
	}
	if target.GetAnnotationExecution(execution, ClassLiteral(other)) != nil || target.GetAnnotationExecution(execution, ClassLiteral(direct)) != nil || calls != 0 {
		t.Fatal("unrelated lookup evaluated unavailable factory", calls)
	}
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "annotation proxy metadata is unavailable", func() { target.GetAnnotationExecution(execution, ClassLiteral(inherited)) })
	if calls != 1 {
		t.Fatal("matched factory not evaluated exactly once", calls)
	}
	accessibleModuleFailureNoCause(t, execution, "UnsupportedOperationException", "annotation proxy metadata is unavailable", func() { target.GetAnnotationsExecution(execution) })
	if calls != 2 {
		t.Fatal("whole annotation array silently omitted unavailable inherited value", calls)
	}
}

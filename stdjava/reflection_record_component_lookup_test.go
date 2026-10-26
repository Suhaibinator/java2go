package stdjava

import "testing"

func TestRecordComponentCanonicalPublicMetadata(t *testing.T) {
	const id TypeID = "tests.RecordComponentCanonicalLookup"
	RegisterJavaType(id, RecordTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: id, Record: true, RecordComponentsKnown: true, RecordComponents: []RecordComponentDescriptor{{Name: "value", Type: StringTypeID}}})
	class := ClassForName(NewExecution(), string(RecordComponentTypeID))
	if class != ClassLiteral(RecordComponentTypeID) {
		t.Fatal("canonical loading identity changed")
	}
	component := ReferenceArrayGet[*RecordComponent](ClassLiteral(id).GetRecordComponents(), 0, RecordComponentTypeID)
	execution := NewExecution()
	name := class.GetMethod("getName")
	typ := class.GetMethod("getType")
	if name.GetDeclaringClass() != class || typ.GetDeclaringClass() != class || name.GetModifiers()&1 == 0 || typ.GetModifiers()&1 == 0 {
		t.Fatal("public declaring metadata lost")
	}
	if name.GetReturnType() != ClassLiteral(StringTypeID) || typ.GetReturnType() != ClassLiteral(JavaClassTypeID) {
		t.Fatal("wrong public return descriptor")
	}
	if result := name.InvokeWithCallerExecution(execution, ObjectTypeID, component); !result.(*JavaString).Equals(JavaStringFromHostUTF8("value")) {
		t.Fatal("wrong component name")
	}
	if result := typ.InvokeWithCallerExecution(execution, ObjectTypeID, component); result != ClassLiteral(StringTypeID) {
		t.Fatal("wrong component type")
	}
	var failure any
	func() { defer func() { failure = recover() }(); class.GetMethod("getAccessor") }()
	if !CaughtAs(failure, "UnsupportedOperationException") || GetCause(failure) != nil {
		t.Fatalf("unprovided platform API falsely absent/successful: %T %v", failure, failure)
	}
}

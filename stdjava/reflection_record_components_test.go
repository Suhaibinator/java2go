package stdjava

import "testing"

func TestRecordComponentMetadataIsolationAndNominalRole(t *testing.T) {
	const id TypeID = "tests.RecordComponentMetadata"
	RegisterJavaType(id, RecordTypeID)
	components := []RecordComponentDescriptor{{Name: "count", Type: PrimitiveIntTypeID}, {Name: "label", Type: StringTypeID}}
	initialized := 0
	RegisterClassDescriptor(ClassDescriptor{Type: id, Record: true, RecordComponentsKnown: true, RecordComponents: components, Initialize: func(*Execution) { initialized++ }})
	components[0].Name = "spoof"
	components[0].Type = ObjectTypeID
	class := ClassLiteral(id)
	if !class.IsRecord() {
		t.Fatal("declared record flag lost")
	}
	first := class.GetRecordComponents()
	if ReferenceArrayLength(first) != 2 {
		t.Fatal("component header count changed")
	}
	component := ReferenceArrayGet[*RecordComponent](first, 0, RecordComponentTypeID)
	if !component.GetNameJavaString().Equals(JavaStringFromHostUTF8("count")) || component.GetType() != ClassLiteral(PrimitiveIntTypeID) {
		t.Fatal("registered metadata aliased mutable input")
	}
	if !ObjectInstanceOf(component, RecordComponentTypeID) || !ObjectInstanceOf(component, AnnotatedElementTypeID) {
		t.Fatal("platform nominal role lost")
	}
	ReferenceArraySet[any](first, 0, nil)
	second := class.GetRecordComponents()
	if first == second || ReferenceArrayGet[*RecordComponent](second, 0, RecordComponentTypeID) == nil {
		t.Fatal("returned array mutated stored metadata")
	}
	if initialized != 0 {
		t.Fatal("metadata inspection initialized declaring record")
	}
	const impostor TypeID = "tests.RecordSuperclassOnly"
	RegisterJavaType(impostor, RecordTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: impostor})
	if ClassLiteral(impostor).IsRecord() || ClassLiteral(impostor).GetRecordComponents() != nil {
		t.Fatal("nominal parent/shape invented record declaration flag")
	}
}
func TestRecordComponentEmptyAndUnavailableBoundaries(t *testing.T) {
	const empty TypeID = "tests.EmptyComponentMetadata"
	RegisterJavaType(empty, RecordTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: empty, Record: true, RecordComponentsKnown: true})
	class := ClassLiteral(empty)
	a, b := class.GetRecordComponents(), class.GetRecordComponents()
	if a == nil || b == nil || a == b || ReferenceArrayLength(a) != 0 {
		t.Fatal("empty record did not return fresh empty arrays")
	}
	const unavailable TypeID = "tests.UnavailableComponentMetadata"
	RegisterJavaType(unavailable, RecordTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: unavailable, Record: true})
	var failure any
	func() { defer func() { failure = recover() }(); ClassLiteral(unavailable).GetRecordComponents() }()
	if !CaughtAs(failure, "UnsupportedOperationException") {
		t.Fatalf("unprovided metadata falsely succeeded: %T %v", failure, failure)
	}
}

package stdjava

import "testing"

// Canonical values must prove their actual class at the allocation boundary.
func TestNativeBuilderMetadataAllocation(t *testing.T) {
	for _, value := range []*StringBuilder{NewStringBuilder(), NewStringBuilderCapacity(3), NewStringBuilderString("x"), NewStringBuilderJavaString(JavaStringLiteralUTF16([]uint16{'x'}))} {
		actual, known := ObjectDynamicType(value)
		if !known || actual != TypeID("java.lang.StringBuilder") {
			t.Fatalf("builder allocation descriptor=%q known=%v", actual, known)
		}
		if ObjectGetClass(value) != ClassLiteral("java.lang.StringBuilder") {
			t.Fatal("allocation class identity lost")
		}
	}
}

func metadataPanic(t *testing.T, name string, body func()) {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); body() }()
	if failure == nil || !CaughtAs(failure, name) {
		t.Fatalf("wanted %s, got %T (%v)", name, failure, failure)
	}
}

func TestNativeBuilderMetadataFamiliesAndReflection(t *testing.T) {
	families := []struct {
		id     TypeID
		values []*StringBuilder
	}{
		{"java.lang.StringBuilder", []*StringBuilder{NewStringBuilder(), NewStringBuilderCapacity(3), NewStringBuilderString("x"), NewStringBuilderJavaString(JavaStringLiteralUTF16([]uint16{'x'}))}},
		{"java.lang.StringBuffer", []*StringBuilder{NewStringBuffer(), NewStringBufferCapacity(3), NewStringBufferJavaString(JavaStringLiteralUTF16([]uint16{'x'}))}},
	}
	for _, family := range families {
		for _, value := range family.values {
			actual, known := ObjectDynamicType(value)
			if !known || actual != family.id {
				t.Fatalf("allocation=%q known=%v expected=%q", actual, known, family.id)
			}
			class := ObjectGetClass(value)
			if class != ClassLiteral(family.id) || class.GetModifiers() != 17 {
				t.Fatal("concrete class flags/identity lost")
			}
			parent := class.GetSuperclass()
			if parent == nil || parent.TypeID() != "java.lang.AbstractStringBuilder" || parent.GetModifiers() != 1024 || parent.GetSuperclass() != ClassLiteral(ObjectTypeID) {
				t.Fatal("actual superclass/abstract flags lost")
			}
			original := value
			if value.AppendChar('a').InsertChar(0, 'z').Reverse().Reverse().DeleteCharAt(0) != original {
				t.Fatal("fluent operation changed allocation identity")
			}
			value.SetLength(0)
			if ObjectGetClass(value) != class {
				t.Fatal("mutation changed allocation class")
			}
			if JavaReferenceEqual(value, NewStringBuilder()) || JavaReferenceEqual(value, NewStringBuffer()) {
				t.Fatal("distinct allocation identity collapsed")
			}
			for _, id := range []TypeID{SerializableTypeID, ComparableTypeID, CharSequenceTypeID, "java.lang.Appendable"} {
				iface := ClassLiteral(id)
				if iface.GetModifiers() != 1537 || iface.GetSuperclass() != nil || !iface.IsInstance(value) {
					t.Fatalf("interface metadata/membership lost: %s", id)
				}
			}
		}
	}
	metadataPanic(t, "NegativeArraySizeException", func() { NewStringBufferCapacity(-1) })
	metadataPanic(t, "NullPointerException", func() { NewStringBufferJavaString(nil) })
	if actual, known := ObjectDynamicType(&StringBuilder{}); known || actual != "" {
		t.Fatal("uninitialized host carrier fabricated allocation class")
	}
}

func TestNativeBuilderMetadataArraysAndViews(t *testing.T) {
	b, f := NewStringBuilderJavaString(JavaStringLiteralUTF16([]uint16{'A', 0xd83d, 0xde00})), NewStringBufferJavaString(JavaStringLiteralUTF16([]uint16{'B'}))
	for _, row := range []struct {
		id           TypeID
		value, wrong *StringBuilder
	}{{"java.lang.StringBuilder", b, f}, {"java.lang.StringBuffer", f, b}} {
		array := ReferenceArrayLiteral(row.id, row.value)
		if ReferenceArrayGet[*StringBuilder](array, 0, row.id) != row.value {
			t.Fatal("array read changed pointer")
		}
		metadataPanic(t, "ArrayStoreException", func() { ReferenceArraySet(array, 0, row.wrong) })
		metadataPanic(t, "ArrayStoreException", func() { ReferenceArraySet(array, 0, JavaStringLiteralUTF16([]uint16{'x'})) })
		if ReferenceArrayGet[*StringBuilder](array, 0, row.id) != row.value {
			t.Fatal("failed store modified old slot")
		}
		ReferenceArraySet(array, 0, (*StringBuilder)(nil))
		if ReferenceArrayGet[*StringBuilder](array, 0, row.id) != nil {
			t.Fatal("typednil not Java null")
		}
		metadataPanic(t, "ClassCastException", func() { ObjectView[*StringBuilder](row.wrong, row.id) })
		if !JavaTypeAssignable(ArrayTypeID(ArrayTypeID(row.id)), ArrayTypeID(ArrayTypeID(CharSequenceTypeID))) {
			t.Fatal("nested reference covariance lost")
		}
	}
	if JavaTypeAssignable(ArrayTypeID("java.lang.StringBuilder"), ArrayTypeID("java.lang.StringBuffer")) {
		t.Fatal("cross-family covariance admitted")
	}
	for _, id := range []TypeID{ObjectTypeID, SerializableTypeID, ComparableTypeID, CharSequenceTypeID, "java.lang.Appendable"} {
		array := ReferenceArrayLiteral(id, b, f)
		if ReferenceArrayGet[any](array, 0, id) != b || ReferenceArrayGet[any](array, 1, id) != f {
			t.Fatal("interface/Object view changed identity")
		}
	}
	for _, id := range []TypeID{StringTypeID, "unrelated.Interface"} {
		metadataPanic(t, "ArrayStoreException", func() { ReferenceArrayLiteral(id, b) })
	}
	if ObjectView[Appendable](f, "java.lang.Appendable") != f || ObjectView[*StringBuilder](b, "java.lang.StringBuilder") != b {
		t.Fatal("typed runtime views changed identity")
	}
	execution := NewExecution()
	view := ObjectView[any](b, CharSequenceTypeID)
	if CharSequenceLength(execution, view) != 3 || CharSequenceCharAt(execution, view, 1) != 0xd83d {
		t.Fatal("CharSequence dispatch/UTF16 lost")
	}
	if AppendableAppendExecution(execution, ObjectView[Appendable](f, "java.lang.Appendable"), rune('!')) != f || f.Length() != 2 {
		t.Fatal("Appendable view lost original object")
	}
}

func TestNativeBuilderMetadataNullAndSourcePriority(t *testing.T) {
	var absent *StringBuilder
	if _, known := ObjectDynamicType(absent); known || ObjectInstanceOf(absent, CharSequenceTypeID) {
		t.Fatal("typednil became an instance")
	}
	if ObjectView[*StringBuilder](absent, "java.lang.StringBuilder") != nil || ObjectView[any](absent, SerializableTypeID) != nil {
		t.Fatal("null view fabricated reference")
	}
	metadataPanic(t, "NullPointerException", func() { ObjectGetClass(absent) })
	// A generated most-derived ObjectInfo remains authoritative even when a host
	// representation also exposes a native carrier; no Go-shape classification.
	id := TypeID("metadata.source.Derived")
	RegisterJavaType("metadata.source.Base", ObjectTypeID, CharSequenceTypeID)
	RegisterJavaType(id, "metadata.source.Base")
	RegisterJavaSourceType(id)
	value := &metadataSourcePriority{StringBuilder: NewStringBuilder()}
	value.ObjectInfo = NewObjectInfo(id, func(requested TypeID) any { return value })
	actual, known := ObjectDynamicType(value)
	if !known || actual != id || ObjectGetClass(value) != ClassLiteral(id) || ObjectInstanceOf(value, "java.lang.StringBuilder") {
		t.Fatal("native metadata recaptured generated source object")
	}
	if ObjectView[*metadataSourcePriority](value, CharSequenceTypeID) != value {
		t.Fatal("source-derived view lost allocation")
	}
}

type metadataSourcePriority struct {
	*ObjectInfo
	*StringBuilder
}

func TestNativeBuilderMetadataReadOnlyConcurrent(t *testing.T) {
	b, f := NewStringBuilder(), NewStringBuffer()
	done := make(chan bool, 4)
	for worker := 0; worker < 4; worker++ {
		go func() {
			valid := true
			for i := 0; i < 100; i++ {
				valid = valid && ObjectGetClass(b) == ClassLiteral("java.lang.StringBuilder") && ObjectGetClass(f) == ClassLiteral("java.lang.StringBuffer") && JavaReferenceEqual(ObjectView[any](b, CharSequenceTypeID), b)
				for _, id := range []TypeID{SerializableTypeID, ComparableTypeID, CharSequenceTypeID, "java.lang.Appendable"} {
					valid = valid && ClassLiteral(id).IsInstance(b) && ClassLiteral(id).IsInstance(f) && ClassLiteral(id).GetSuperclass() == nil && ClassLiteral(id).GetModifiers() == 1537
				}
			}
			done <- valid
		}()
	}
	for worker := 0; worker < 4; worker++ {
		if !<-done {
			t.Fatal("immutable metadata read changed under concurrency")
		}
	}
}

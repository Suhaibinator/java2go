package stdjava

import "testing"

type catchDescriptorSourceProbe struct{ ThrowableBase }

func (*catchDescriptorSourceProbe) JavaDynamicTypeID() TypeID {
	return "shadow.IllegalCharsetNameException"
}

func TestRuntimeThrowableDescriptorNominalMetadata(t *testing.T) {
	for _, entry := range []struct{ name, id, parent string }{
		{"IllegalCharsetNameException", "java.nio.charset.IllegalCharsetNameException", "java.lang.IllegalArgumentException"},
		{"UnsupportedCharsetException", "java.nio.charset.UnsupportedCharsetException", "java.lang.IllegalArgumentException"},
		{"CharacterCodingException", "java.nio.charset.CharacterCodingException", "java.io.IOException"},
		{"java.nio.charset.UnmappableCharacterException", "java.nio.charset.UnmappableCharacterException", "java.nio.charset.CharacterCodingException"},
		{"java.lang.reflect.InvocationTargetException", "java.lang.reflect.InvocationTargetException", "java.lang.ReflectiveOperationException"},
	} {
		id, parent, known := RuntimeThrowableDescriptor(entry.name)
		if !known || string(id) != entry.id || string(parent) != entry.parent {
			t.Errorf("descriptor %s: %q -> %q, known=%v", entry.name, id, parent, known)
		}
	}
	for _, name := range []string{"foreign.UnsupportedCharsetException", "java.lang.UnsupportedCharsetException", "java.lang.String", "Object", "java.lang.Object", "x-missing"} {
		if _, _, known := RuntimeThrowableDescriptor(name); known {
			t.Errorf("non-Throwable/foreign metadata accepted: %s", name)
		}
	}
	// Runtime companions may register descriptors without appearing in the
	// constructor inventory or the legacy builtin descriptor table.
	ids := []TypeID{"java.control.RuntimeOnlyFailure", "java.control.one.DuplicateFailure", "java.control.two.DuplicateFailure", "java.control.SourceOnlyFailure", "java.control.CycleFailure"}
	javaTypeRegistry.Lock()
	old := map[TypeID]registeredJavaType{}
	existed := map[TypeID]bool{}
	for _, id := range ids {
		old[id], existed[id] = javaTypeRegistry.types[id]
	}
	javaTypeRegistry.Unlock()
	t.Cleanup(func() {
		javaTypeRegistry.Lock()
		defer javaTypeRegistry.Unlock()
		for _, id := range ids {
			if existed[id] {
				javaTypeRegistry.types[id] = old[id]
			} else {
				delete(javaTypeRegistry.types, id)
			}
		}
	})
	for _, id := range ids {
		RegisterJavaType(id, TypeID("java.lang.RuntimeException"))
	}
	RegisterJavaSourceType(ids[3])
	RegisterJavaType(ids[4], ids[4])
	if id, parent, known := RuntimeThrowableDescriptor("RuntimeOnlyFailure"); !known || id != ids[0] || parent != "java.lang.RuntimeException" {
		t.Fatal("init-only runtime descriptor not discoverable")
	}
	for _, name := range []string{"DuplicateFailure", "SourceOnlyFailure", "java.control.SourceOnlyFailure", "CycleFailure"} {
		if _, _, known := RuntimeThrowableDescriptor(name); known {
			t.Errorf("ambiguous/source/cyclic entry accepted: %s", name)
		}
	}
}

func TestCanonicalCharsetCatchHierarchyAndSiblingNegatives(t *testing.T) {
	failures := []struct {
		value any
		child string
	}{
		{charsetEncodingReferenceFailure(t, "IllegalCharsetNameException", func() any { return CharsetForNameReference(NewJavaStringUTF16([]uint16{'!'})) }), "java.nio.charset.IllegalCharsetNameException"},
		{charsetEncodingReferenceFailure(t, "UnsupportedCharsetException", func() any { return CharsetForNameReference(NewJavaStringUTF16(charsetEncodingASCIIUnits("UTF_8"))) }), "java.nio.charset.UnsupportedCharsetException"},
		{charsetEncodingReferenceFailure(t, "IllegalArgumentException", func() any { return CharsetForNameReference(nil) }), "java.lang.IllegalArgumentException"},
	}
	for _, failure := range failures {
		for _, expected := range []TypeID{TypeID(failure.child), "java.lang.IllegalArgumentException", "java.lang.RuntimeException", "java.lang.Exception", ThrowableTypeID} {
			if !CaughtAsType(failure.value, expected) {
				t.Errorf("%s not caught as %s", failure.child, expected)
			}
		}
		for _, sibling := range []TypeID{"java.nio.charset.IllegalCharsetNameException", "java.nio.charset.UnsupportedCharsetException", "java.lang.NullPointerException", "java.io.IOException", "java.io.UnsupportedEncodingException", "java.lang.Error", "shadow.IllegalCharsetNameException"} {
			if string(sibling) != failure.child && CaughtAsType(failure.value, sibling) {
				t.Errorf("%s caught as unrelated %s", failure.child, sibling)
			}
		}
	}
	RegisterJavaType("shadow.IllegalCharsetNameException", "java.lang.RuntimeException")
	RegisterJavaSourceType("shadow.IllegalCharsetNameException")
	probe := &catchDescriptorSourceProbe{NewRuntimeException("source").ThrowableBase}
	if !CaughtAsType(probe, "shadow.IllegalCharsetNameException") || !CaughtAsType(probe, "java.lang.RuntimeException") || CaughtAsType(probe, "java.nio.charset.IllegalCharsetNameException") || CaughtAsType(probe, "java.lang.IllegalArgumentException") {
		t.Fatal("source and JDK exception identities collided")
	}
	if CaughtAsType(nil, ThrowableTypeID) || CaughtAsType("native", ThrowableTypeID) {
		t.Fatal("null/non-Throwable caught nominally")
	}
	if !CaughtAs(NewIllegalArgumentException("native"), "RuntimeException") {
		t.Fatal("native catch API changed")
	}
}

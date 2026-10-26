package stdjava

import "testing"

func TestThrowableCatchRegistrationPreservesSourceDescriptor(t *testing.T) {
	const child TypeID = "CatchRegistrationSourceChild"
	const parent TypeID = "CatchRegistrationSourceParent"
	const marker TypeID = "CatchRegistrationSourceMarker"
	RegisterJavaType(marker, ObjectTypeID)
	RegisterJavaType(parent, BuiltinThrowableTypeID("RuntimeException"))
	RegisterJavaType(child, parent, marker)
	// A simple catch-parent spelling may denote a source class. Catch wiring
	// must neither guess its qualified reference ID nor discard interfaces.
	RegisterException(string(child), "NullPointerException")
	if !JavaTypeAssignable(child, parent) || !JavaTypeAssignable(child, marker) {
		t.Fatal("catch registration replaced resolved source superclass/interfaces")
	}
	if JavaTypeAssignable(child, BuiltinThrowableTypeID("NullPointerException")) {
		t.Fatal("catch registration reparented source descriptor to builtin")
	}
	if !CaughtAs(ThrowableBase{typeName: string(child)}, "NullPointerException") {
		t.Fatal("legacy catch hierarchy was not independently registered")
	}
}

func TestBuiltinThrowableCanonicalRegistrationParents(t *testing.T) {
	for name, descriptor := range builtinThrowableDescriptors {
		want := ObjectTypeID
		if descriptor.parent != "" {
			want = BuiltinThrowableTypeID(descriptor.parent)
		}
		javaTypeRegistry.RLock()
		got, ok := javaTypeRegistry.types[descriptor.id]
		javaTypeRegistry.RUnlock()
		if !ok || got.super != want {
			t.Errorf("%s parent=%q present=%t; want %q", name, got.super, ok, want)
		}
		if !JavaTypeAssignable(descriptor.id, SerializableTypeID) {
			t.Errorf("%s lost inherited Serializable", name)
		}
	}
}

package stdjava

import "testing"

func TestBoxedReferenceViewsRetainNominalTypeAndIdentity(t *testing.T) {
	values := []struct {
		value   any
		id      TypeID
		numeric bool
	}{
		{BoxBoolean(true), BooleanTypeID, false},
		{BoxByte(1), ByteTypeID, true},
		{BoxShort(1), ShortTypeID, true},
		{BoxCharacter(1), CharacterTypeID, false},
		{BoxInteger(1), IntegerTypeID, true},
		{BoxLong(1), LongTypeID, true},
		{BoxFloat(1), FloatTypeID, true},
		{BoxDouble(1), DoubleTypeID, true},
	}
	for _, test := range values {
		t.Run(string(test.id), func(t *testing.T) {
			if id, ok := ObjectDynamicType(test.value); !ok || id != test.id {
				t.Fatalf("dynamic type = (%q, %t), want %q", id, ok, test.id)
			}
			if got := ObjectView[any](test.value, ObjectTypeID); got != test.value {
				t.Fatal("Object view changed wrapper identity")
			}
			if got := ObjectGetClass(test.value); got != ClassLiteral(test.id) {
				t.Fatal("getClass did not return canonical wrapper class")
			}
			for _, target := range []TypeID{ObjectTypeID, SerializableTypeID, ComparableTypeID, ConstableTypeID} {
				if !JavaTypeAssignable(test.id, target) {
					t.Fatalf("%s is not assignable to %s", test.id, target)
				}
			}
			if JavaTypeAssignable(test.id, NumberTypeID) != test.numeric {
				t.Fatal("Number hierarchy includes the wrong wrapper kinds")
			}
		})
	}
	integer := NewInteger(200)
	if got := ObjectView[JavaNumber](integer, NumberTypeID); got != integer {
		t.Fatal("Number view changed wrapper identity")
	}
	if got := ObjectView[*Integer](nil, IntegerTypeID); got != nil {
		t.Fatal("null cast must retain null")
	}
	if ObjectGetClass(integer) == ClassLiteral(PrimitiveIntTypeID) {
		t.Fatal("Integer.class and int.class must be distinct")
	}
	assertBoxedReferencePanic(t, "ClassCastException", func() {
		ObjectView[*Character](integer, CharacterTypeID)
	})
}

func TestBoxedReferenceArraysPreserveNullAndCheckCovariantStores(t *testing.T) {
	integers := NewReferenceArray(2, IntegerTypeID)
	value := NewInteger(300)
	ReferenceArraySet(integers, 0, value)
	if got := ReferenceArrayGet[JavaNumber](integers, 0, NumberTypeID); got != value {
		t.Fatal("covariant array read changed wrapper identity")
	}
	if got := ReferenceArrayGet[*Integer](integers, 1, IntegerTypeID); got != nil {
		t.Fatal("default wrapper array element is not null")
	}
	assertBoxedReferencePanic(t, "ArrayStoreException", func() {
		ReferenceArraySet(integers, 0, BoxLong(300))
	})
	if got := ReferenceArrayGet[*Integer](integers, 0, IntegerTypeID); got != value {
		t.Fatal("rejected store changed array contents")
	}
	var absent *Integer
	ReferenceArraySet(integers, 0, absent)
	if got := ReferenceArrayGet[any](integers, 0, ObjectTypeID); got != nil {
		t.Fatal("typed null did not become Java null in Object view")
	}
}

func TestPrimitiveBackendValuesCannotMasqueradeAsWrapperReferences(t *testing.T) {
	for _, primitive := range []any{true, int8(1), int16(1), int32(1), int64(1), float32(1), float64(1)} {
		if id, ok := ObjectDynamicType(primitive); ok {
			t.Fatalf("unboxed %T unexpectedly has reference type %s", primitive, id)
		}
		assertBoxedReferencePanic(t, "ClassCastException", func() {
			ObjectView[any](primitive, ObjectTypeID)
		})
		assertBoxedReferencePanic(t, "ArrayStoreException", func() {
			ReferenceArraySet(NewReferenceArray(1, ObjectTypeID), 0, primitive)
		})
	}
}

func TestNullBoxedStringConversionAndClassAccess(t *testing.T) {
	var integer *Integer
	var number JavaNumber = integer
	for _, value := range []any{integer, number} {
		if StringValueOf(value) != "null" || StringValueOfExecution(NewExecution(), value) != "null" {
			t.Fatal("typed null wrapper was not rendered as Java null")
		}
		assertBoxedReferencePanic(t, "NullPointerException", func() { ObjectGetClass(value) })
	}
}

type boxedObjectExecutionProbe struct {
	want *Execution
	seen int
}

func (probe *boxedObjectExecutionProbe) EqualsJava2goExecution2(execution *Execution, other any) bool {
	probe.seen++
	return execution == probe.want && other == probe
}

func (probe *boxedObjectExecutionProbe) HashCodeJava2goExecution1(execution *Execution) int32 {
	probe.seen++
	if execution != probe.want {
		return -1
	}
	return 73
}

func TestBoxedErasedObjectMethodsAndPatterns(t *testing.T) {
	execution := NewExecution()
	first, second := NewInteger(200), NewInteger(200)
	if !ObjectEqualsExecution(execution, first, second) || JavaReferenceEqual(first, second) {
		t.Fatal("erased equals did not preserve the distinction from identity")
	}
	if ObjectEqualsExecution(execution, first, BoxLong(200)) || ObjectEqualsExecution(execution, first, nil) {
		t.Fatal("Integer equals accepted a different class or null")
	}
	if got := ObjectHashCodeExecution(execution, first); got != 200 {
		t.Fatalf("erased Integer hash = %d, want 200", got)
	}
	if ObjectEqualsExecution(execution, "200", first) || !ObjectEqualsExecution(execution, "200", "200") {
		t.Fatal("String.equals(Object) did not check the argument type")
	}
	if got := ObjectHashCodeExecution(execution, "abc"); got != 96354 {
		t.Fatalf("String hash = %d, want 96354", got)
	}
	if view, ok := ObjectPattern[JavaNumber](first, NumberTypeID); !ok || view != first {
		t.Fatal("Number pattern did not preserve wrapper identity")
	}
	var absent *Integer
	if _, ok := ObjectPattern[*Integer](absent, IntegerTypeID); ok {
		t.Fatal("typed null matched a wrapper pattern")
	}
	if !ObjectInstanceOf(&referenceIdentityProbe{}, ObjectTypeID) {
		t.Fatal("opaque runtime reference did not match Object")
	}
	probe := &boxedObjectExecutionProbe{want: execution}
	if !ObjectEqualsExecution(execution, probe, probe) || ObjectHashCodeExecution(execution, probe) != 73 || probe.seen != 2 {
		t.Fatal("Object methods lost the execution token or collision-renamed companion")
	}
	assertBoxedReferencePanic(t, "NullPointerException", func() { ObjectEqualsExecution(execution, absent, first) })
	assertBoxedReferencePanic(t, "NullPointerException", func() { ObjectHashCodeExecution(execution, absent) })
}

func assertBoxedReferencePanic(t *testing.T, expected string, call func()) {
	t.Helper()
	defer func() {
		if recovered := recover(); !CaughtAs(recovered, expected) {
			t.Fatalf("panic = %T (%v), want %s", recovered, recovered, expected)
		}
	}()
	call()
}

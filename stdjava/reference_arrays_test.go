package stdjava

import "testing"

const (
	testBaseType    TypeID = "tests.Base"
	testChildType   TypeID = "tests.Child"
	testSiblingType TypeID = "tests.Sibling"
)

// These deliberately use distinct Go view types, matching generated embedded
// class views. Using one Go struct for Base/Child/Sibling would let a direct
// type assertion conceal failures in ObjectInfo view resolution.
type referenceArrayBaseView struct {
	info  *ObjectInfo
	value int
}

func (value *referenceArrayBaseView) JavaObjectInfo() *ObjectInfo { return value.info }

type referenceArrayChildView struct {
	*referenceArrayBaseView
}

type referenceArraySiblingView struct {
	*referenceArrayBaseView
}

func newReferenceArrayBase(value int) *referenceArrayBaseView {
	base := &referenceArrayBaseView{value: value}
	base.info = NewObjectInfo(testBaseType, func(requested TypeID) any {
		if requested == testBaseType {
			return base
		}
		return nil
	})
	return base
}

func newReferenceArrayChild(value int) *referenceArrayChildView {
	child := &referenceArrayChildView{referenceArrayBaseView: &referenceArrayBaseView{value: value}}
	child.info = NewObjectInfo(testChildType, func(requested TypeID) any {
		switch requested {
		case testChildType:
			return child
		case testBaseType:
			return child.referenceArrayBaseView
		default:
			return nil
		}
	})
	return child
}

func newReferenceArraySibling(value int) *referenceArraySiblingView {
	sibling := &referenceArraySiblingView{referenceArrayBaseView: &referenceArrayBaseView{value: value}}
	sibling.info = NewObjectInfo(testSiblingType, func(requested TypeID) any {
		switch requested {
		case testSiblingType:
			return sibling
		case testBaseType:
			return sibling.referenceArrayBaseView
		default:
			return nil
		}
	})
	return sibling
}

func registerReferenceArrayTestTypes() {
	RegisterJavaType(testBaseType, ObjectTypeID)
	RegisterJavaType(testChildType, testBaseType)
	RegisterJavaType(testSiblingType, testBaseType)
}

func TestJavaTypeAssignableIncludesReferenceArrayCovariance(t *testing.T) {
	registerReferenceArrayTestTypes()
	if !JavaTypeAssignable(testChildType, testBaseType) {
		t.Fatal("Child must be assignable to Base")
	}
	if JavaTypeAssignable(testBaseType, testChildType) {
		t.Fatal("Base must not be assignable to Child")
	}
	if !JavaTypeAssignable(ArrayTypeID(testChildType), ArrayTypeID(testBaseType)) {
		t.Fatal("Child[] must be assignable to Base[]")
	}
	if !JavaTypeAssignable(ArrayTypeID(testChildType), ObjectTypeID) {
		t.Fatal("every array must be assignable to Object")
	}
	intArray := ArrayTypeID(PrimitiveTypeID("int"))
	longArray := ArrayTypeID(PrimitiveTypeID("long"))
	if JavaTypeAssignable(intArray, longArray) {
		t.Fatal("primitive arrays must not be covariant")
	}
	if !JavaTypeAssignable(ArrayTypeID(intArray), ArrayTypeID(ObjectTypeID)) {
		t.Fatal("int[][] must be assignable to Object[] because int[] is a reference")
	}
	for _, stringSupertype := range []TypeID{ObjectTypeID, SerializableTypeID, ComparableTypeID, CharSequenceTypeID, ConstableTypeID, ConstantDescTypeID} {
		if !JavaTypeAssignable(StringTypeID, stringSupertype) {
			t.Fatalf("String must be assignable to %s", stringSupertype)
		}
	}
}

func TestReferenceArrayCheckedStoreUsesGeneratedObjectViews(t *testing.T) {
	registerReferenceArrayTestTypes()
	array := NewReferenceArray(2, testChildType)
	child := newReferenceArrayChild(4)
	if returned := ReferenceArraySet(array, int32(0), child); returned != child {
		t.Fatal("successful runtime store must return its input")
	}
	if got := ReferenceArrayGet[*referenceArrayBaseView](array, 0, testBaseType); got != child.referenceArrayBaseView {
		t.Fatal("Base read did not resolve the stored Child's superclass view")
	}
	if got := ReferenceArrayGet[*referenceArrayChildView](array, 0, testChildType); got != child {
		t.Fatal("Child read lost the most-derived view")
	}

	base := newReferenceArrayBase(9)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		ReferenceArraySet(array, 0, base)
	}()
	if !CaughtAs(recovered, "ArrayStoreException") {
		t.Fatalf("rejected store panic = %T (%v), want ArrayStoreException", recovered, recovered)
	}
	if got := ReferenceArrayGet[*referenceArrayChildView](array, 0, testChildType); got != child {
		t.Fatal("rejected store mutated the array")
	}

	recovered = nil
	func() {
		defer func() { recovered = recover() }()
		_ = ObjectView[*referenceArraySiblingView](child, testSiblingType)
	}()
	if !CaughtAs(recovered, "ClassCastException") {
		t.Fatalf("unrelated requested view panic = %T (%v), want ClassCastException", recovered, recovered)
	}

	var nullChild *referenceArrayChildView
	ReferenceArraySet(array, 1, nullChild)
	if got := ReferenceArrayGet[*referenceArrayChildView](array, 1, testChildType); got != nil {
		t.Fatalf("stored Java null = %#v, want nil", got)
	}
}

func TestReferenceArrayAssignPreservesJavaOrderAndStaticResultView(t *testing.T) {
	registerReferenceArrayTestTypes()
	array := NewReferenceArray(1, testChildType)
	child := newReferenceArrayChild(4)
	rhsCalls := 0
	result := ReferenceArrayAssign[*referenceArrayBaseView](array, 0, func() *referenceArrayChildView {
		rhsCalls++
		return child
	}, testBaseType)
	if rhsCalls != 1 || result != child.referenceArrayBaseView {
		t.Fatalf("assignment result = %#v after %d RHS calls, want Base view after one call", result, rhsCalls)
	}

	assertRHSEvaluatedBeforeStoreCheck := func(name string, target *ReferenceArray, index int32, wantException string) {
		t.Helper()
		calls := 0
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = ReferenceArrayAssign[any](target, index, func() any {
				calls++
				return child
			}, ObjectTypeID)
		}()
		if !CaughtAs(recovered, wantException) {
			t.Fatalf("%s panic = %T (%v), want %s", name, recovered, recovered, wantException)
		}
		if calls != 1 {
			t.Fatalf("%s evaluated RHS %d times, want one", name, calls)
		}
	}
	assertRHSEvaluatedBeforeStoreCheck("null target", nil, 0, "NullPointerException")
	assertRHSEvaluatedBeforeStoreCheck("out-of-bounds target", array, 2, "ArrayIndexOutOfBoundsException")

	rhsPanic := NewIllegalArgumentException("rhs")
	for _, testCase := range []struct {
		name   string
		target *ReferenceArray
		index  int32
	}{
		{name: "null target", target: nil, index: 0},
		{name: "out-of-bounds target", target: array, index: 2},
	} {
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			_ = ReferenceArrayAssign[any](testCase.target, testCase.index, func() any {
				panic(rhsPanic)
			}, ObjectTypeID)
		}()
		if recovered != rhsPanic {
			t.Fatalf("%s panic = %T (%v), want RHS panic", testCase.name, recovered, recovered)
		}
	}

	base := newReferenceArrayBase(9)
	storeCalls := 0
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_ = ReferenceArrayAssign[*referenceArrayBaseView](array, 0, func() *referenceArrayBaseView {
			storeCalls++
			return base
		}, testBaseType)
	}()
	if !CaughtAs(recovered, "ArrayStoreException") || storeCalls != 1 {
		t.Fatalf("covariant rejection = %T (%v) after %d RHS calls, want ArrayStoreException after one", recovered, recovered, storeCalls)
	}
	if got := ReferenceArrayGet[*referenceArrayChildView](array, 0, testChildType); got != child {
		t.Fatal("covariant rejection mutated the previously stored Child")
	}
}

func TestPrimitiveArraysRetainRuntimeTypeThroughNestedAndErasedViews(t *testing.T) {
	intType := PrimitiveTypeID("int")
	charType := PrimitiveTypeID("char")
	ints := NewPrimitiveArray[int32](2, intType)
	PrimitiveArrayAssign(ints, 0, func() int32 { return 7 })
	if PrimitiveArrayGet(ints, 0) != 7 || PrimitiveArrayLength(ints) != 2 {
		t.Fatal("primitive array lost its stored element or length")
	}

	nested := NewReferenceArray(1, ArrayTypeID(intType))
	ReferenceArraySet(nested, 0, ints)
	if got := ReferenceArrayGet[*PrimitiveArray[int32]](nested, 0, ArrayTypeID(intType)); got != ints {
		t.Fatal("int[][] did not retain its primitive leaf identity")
	}
	objects := NewReferenceArray(1, ObjectTypeID)
	ReferenceArraySet(objects, 0, ints)
	if got := ReferenceArrayGet[any](objects, 0, ObjectTypeID); got != ints {
		t.Fatal("Object[] did not retain its primitive-array value")
	}

	// char and int intentionally share int32 as their Go element shape. The
	// explicit Java descriptor must still reject char[] from int[][].
	chars := NewPrimitiveArray[int32](1, charType)
	var recovered any
	func() {
		defer func() { recovered = recover() }()
		ReferenceArraySet(nested, 0, chars)
	}()
	if !CaughtAs(recovered, "ArrayStoreException") {
		t.Fatalf("char[] into int[][] panic = %T (%v), want ArrayStoreException", recovered, recovered)
	}
}

func TestReferenceArrayCanonicalizesTypedAndStringNullAfterErasure(t *testing.T) {
	registerReferenceArrayTestTypes()
	array := NewReferenceArray(2, ObjectTypeID)
	var typedNull *referenceArrayChildView
	ReferenceArraySet(array, 0, typedNull)
	ReferenceArraySet(array, 1, NullString())
	for index := int32(0); index < 2; index++ {
		if got := ReferenceArrayGet[any](array, index, ObjectTypeID); got != nil {
			t.Fatalf("erased Java null at %d = %#v (%T), want nil interface", index, got, got)
		}
	}
}

func TestArrayAliasAndAllocationIdentity(t *testing.T) {
	registerReferenceArrayTestTypes()
	first := NewReferenceArray(0, testChildType)
	alias := first
	second := NewReferenceArray(0, testChildType)
	if first != alias || first == second {
		t.Fatal("reference-array pointer identity is not allocation based")
	}
	if ReferenceArrayLength(first) != 0 || ReferenceArrayLength(second) != 0 {
		t.Fatal("empty reference arrays must expose length zero")
	}

	firstPrimitive := NewPrimitiveArray[int32](0, PrimitiveTypeID("int"))
	secondPrimitive := NewPrimitiveArray[int32](0, PrimitiveTypeID("int"))
	if firstPrimitive == secondPrimitive || PrimitiveArrayLength(firstPrimitive) != 0 {
		t.Fatal("empty primitive arrays must retain distinct pointer identities")
	}
}

func TestReferenceArrayLiteralChecksEveryElement(t *testing.T) {
	registerReferenceArrayTestTypes()
	child := newReferenceArrayChild(1)
	array := ReferenceArrayLiteral(testChildType, child, nil)
	if ReferenceArrayLength(array) != 2 {
		t.Fatalf("literal length = %d, want 2", ReferenceArrayLength(array))
	}
	if got := ReferenceArrayGet[*referenceArrayChildView](array, 0, testChildType); got != child {
		t.Fatal("literal lost its child element")
	}

	defer func() {
		if recovered := recover(); !CaughtAs(recovered, "ArrayStoreException") {
			t.Fatalf("incompatible literal panic = %T (%v), want ArrayStoreException", recovered, recovered)
		}
	}()
	ReferenceArrayLiteral(testChildType, newReferenceArrayBase(2))
}

func TestReferenceStringArrayPreservesNullByStaticView(t *testing.T) {
	array := NewReferenceArray(2, StringTypeID)
	if first := ReferenceArrayGet[string](array, 0, StringTypeID); !StringIsNull(first) {
		t.Fatalf("default String element = %q, want Java null", first)
	}
	if erased := ReferenceArrayGet[any](array, 0, ObjectTypeID); erased != nil {
		t.Fatalf("default String through Object = %#v, want nil", erased)
	}
	ReferenceArraySet(array, 0, "ready")
	ReferenceArraySet[any](array, 1, nil)
	if got := ReferenceArrayGet[string](array, 0, StringTypeID); got != "ready" {
		t.Fatalf("stored String = %q, want ready", got)
	}
	if got := ReferenceArrayGet[string](array, 1, StringTypeID); !StringIsNull(got) {
		t.Fatalf("stored null String = %q, want sentinel", got)
	}
}

func TestArrayConstructorsRejectMalformedDescriptors(t *testing.T) {
	tests := []struct {
		name string
		call func()
	}{
		{name: "reference empty", call: func() { NewReferenceArray(1, "") }},
		{name: "reference primitive", call: func() { NewReferenceArray(1, PrimitiveTypeID("int")) }},
		{name: "reference malformed nested", call: func() { NewReferenceArray(1, ArrayTypeID("")) }},
		{name: "primitive class", call: func() { NewPrimitiveArray[int32](1, ObjectTypeID) }},
		{name: "primitive unknown", call: func() { NewPrimitiveArray[int32](1, PrimitiveTypeID("word")) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recovered := recover(); !CaughtAs(recovered, "IllegalArgumentException") {
					t.Fatalf("panic = %T (%v), want IllegalArgumentException", recovered, recovered)
				}
			}()
			test.call()
		})
	}
}

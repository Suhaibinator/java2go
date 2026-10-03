package stdjava

import (
	"slices"
	"testing"
	"unicode/utf16"
)

func TestJavaArrayCanonicalPrimitiveText(t *testing.T) {
	e := NewExecution()
	for _, test := range []struct {
		value any
		want  string
	}{
		{PrimitiveArrayLiteral(PrimitiveTypeID("boolean"), true, false), "[true, false]"},
		{PrimitiveArrayLiteral(PrimitiveTypeID("byte"), int8(-128), int8(127)), "[-128, 127]"},
		{PrimitiveArrayLiteral(PrimitiveTypeID("short"), int16(-32768)), "[-32768]"},
		{PrimitiveArrayLiteral(PrimitiveTypeID("int"), int32(65)), "[65]"},
		{PrimitiveArrayLiteral(PrimitiveTypeID("long"), int64(-9223372036854775808)), "[-9223372036854775808]"},
		{PrimitiveArrayLiteral(PrimitiveTypeID("float"), float32(1.5)), "[1.5]"},
		{PrimitiveArrayLiteral(PrimitiveTypeID("double"), float64(1.5)), "[1.5]"},
	} {
		got := JavaArrayToStringExecution(e, test.value)
		if !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune(test.want))) {
			t.Fatalf("array text: %v, want %q", got.UTF16Copy(), test.want)
		}
		if got == JavaArrayToStringExecution(e, test.value) {
			t.Fatal("nonempty array text was reused")
		}
	}
	chars := PrimitiveArrayLiteral(PrimitiveTypeID("char"), rune(0xD800), rune(0), rune(0xDFFF))
	want := []uint16{'[', 0xD800, ',', ' ', 0, ',', ' ', 0xDFFF, ']'}
	if got := JavaArrayToStringExecution(e, chars); !slices.Equal(got.UTF16Copy(), want) {
		t.Fatalf("char array lost UTF16: %v", got.UTF16Copy())
	}
	for _, value := range []any{nil, (*PrimitiveArray[int32])(nil), (*ReferenceArray)(nil)} {
		if got := JavaArrayToStringExecution(e, value); got != JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'}) {
			t.Fatal("null array result must be a literal")
		}
	}
	for _, value := range []any{nil, (*ReferenceArray)(nil)} {
		if got := JavaArrayDeepToStringExecution(e, value); got != JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'}) {
			t.Fatal("null deep array result must be a literal")
		}
		if got := JavaStringValueOfExecution(e, value); got != JavaStringLiteralUTF16([]uint16{'n', 'u', 'l', 'l'}) {
			t.Fatal("typed null array entered its Object adapter")
		}
	}
	empty := NewReferenceArray(0, ObjectTypeID)
	if JavaArrayToStringExecution(e, empty) != JavaStringLiteralUTF16([]uint16{'[', ']'}) {
		t.Fatal("shallow empty result must be literal")
	}
	first, second := JavaArrayDeepToStringExecution(e, empty), JavaArrayDeepToStringExecution(e, empty)
	if first == second || first == JavaStringLiteralUTF16([]uint16{'[', ']'}) || !first.Equals(second) {
		t.Fatal("deep empty result must be fresh")
	}
}

type arrayReferenceTextProbe struct {
	execution *Execution
	value     *JavaString
	calls     int
	mutate    *ReferenceArray
	abrupt    any
}

func (*arrayReferenceTextProbe) JavaDynamicTypeID() TypeID { return "array.reference.Text" }
func (p *arrayReferenceTextProbe) DeclaredText(e *Execution) *JavaString {
	if e != p.execution {
		panic("array callback lost caller execution")
	}
	p.calls++
	if p.abrupt != nil {
		panic(p.abrupt)
	}
	if p.mutate != nil {
		ReferenceArraySet(p.mutate, 1, JavaStringLiteralUTF16([]uint16{'n', 'e', 'w'}))
	}
	return p.value
}
func (*arrayReferenceTextProbe) String() string { panic("host array callback") }
func (*arrayReferenceTextProbe) StringJava2goExecution(*Execution) *JavaString {
	panic("ordinary structural callback")
}

func TestJavaArrayCanonicalReferenceText(t *testing.T) {
	RegisterJavaType("array.reference.Text", ObjectTypeID)
	RegisterJavaSourceType("array.reference.Text")
	RegisterJavaSourceToString("array.reference.Text", "DeclaredText")
	e := NewExecution()
	raw := &arrayReferenceTextProbe{execution: e, value: NewJavaStringUTF16([]uint16{0xD800, 0, 0xDFFF})}
	array := ReferenceArrayLiteral(ObjectTypeID, raw, nil)
	want := []uint16{'[', 0xD800, 0, 0xDFFF, ',', ' ', 'n', 'u', 'l', 'l', ']'}
	for _, convert := range []func(*Execution, any) *JavaString{JavaArrayToStringExecution, JavaArrayDeepToStringExecution} {
		if got := convert(e, array); !slices.Equal(got.UTF16Copy(), want) {
			t.Fatalf("reference array lost UTF16: %v", got.UTF16Copy())
		}
	}
	if raw.calls != 2 {
		t.Fatal("element callback was repeated or skipped")
	}
	raw.value = nil
	if got := JavaArrayToStringExecution(e, array); !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune("[null, null]"))) {
		t.Fatal("null callback text was not appended as null")
	}
	mutating := &arrayReferenceTextProbe{execution: e, value: JavaStringLiteralUTF16([]uint16{'k'})}
	live := ReferenceArrayLiteral(ObjectTypeID, mutating, JavaStringLiteralUTF16([]uint16{'o', 'l', 'd'}))
	mutating.mutate = live
	if got := JavaArrayToStringExecution(e, live); !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune("[k, new]"))) {
		t.Fatal("array callback did not observe later live slot")
	}
	cycle := NewReferenceArray(2, ObjectTypeID)
	ReferenceArraySet(cycle, 0, cycle)
	ReferenceArraySet(cycle, 1, array)
	raw.abrupt = NewIllegalStateException("marker")
	skipped := &arrayReferenceTextProbe{execution: e}
	ReferenceArraySet(array, 1, skipped)
	func() {
		defer func() {
			if got := recover(); got != raw.abrupt {
				t.Fatalf("array callback exception changed: %v", got)
			}
		}()
		JavaArrayDeepToStringExecution(e, cycle)
	}()
	if skipped.calls != 0 {
		t.Fatal("throwing callback did not skip later array elements")
	}
	ReferenceArraySet[any](array, 1, nil)
	raw.abrupt = nil
	if got := JavaArrayDeepToStringExecution(e, cycle); !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune("[[...], [null, null]]"))) {
		t.Fatal("deep cycle or post-throw cleanup changed")
	}
	nested := ReferenceArrayLiteral(ObjectTypeID, PrimitiveArrayLiteral(PrimitiveTypeID("int"), int32(1), int32(2)), cycle, cycle)
	if got := JavaArrayDeepToStringExecution(e, nested); !slices.Equal(got.UTF16Copy(), utf16.Encode([]rune("[[1, 2], [[...], [null, null]], [[...], [null, null]]]"))) {
		t.Fatal("deep recursion retained a completed sibling")
	}
}

func TestJavaArrayCanonicalObjectDefaultText(t *testing.T) {
	e := NewExecution()
	for _, array := range []any{NewPrimitiveArray[int32](0, PrimitiveTypeID("int")), NewReferenceArray(0, ObjectTypeID)} {
		first := JavaStringValueOfExecution(e, array)
		second := JavaStringValueOfExecution(e, array)
		if first == second || !first.Equals(ObjectDefaultJavaStringExecution(e, array)) {
			t.Fatal("array Object text lost default identity or freshness")
		}
		outer := ReferenceArrayLiteral(ObjectTypeID, array)
		want := append([]uint16{'['}, first.UTF16Copy()...)
		want = append(want, ']')
		if got := JavaArrayToStringExecution(e, outer); !slices.Equal(got.UTF16Copy(), want) {
			t.Fatal("shallow array did not render nested array identity")
		}
	}
}

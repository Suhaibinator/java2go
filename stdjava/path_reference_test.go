package stdjava

import (
	"slices"
	"testing"
)

// Unix provider expectations are pinned to actual-contract-outcomes.json
// b5653ae03db58277a6d789aa1d628230c113e5bfc7d76da08eed9fb894b430ff.
func pathReferencePanic(t *testing.T, invoke func()) (failure any) {
	t.Helper()
	defer func() {
		failure = recover()
		if failure == nil {
			t.Fatal("expected abrupt completion")
		}
	}()
	invoke()
	return nil
}

func pathReferenceString(text string) *JavaString {
	units := make([]uint16, len(text))
	for index := range text {
		units[index] = uint16(text[index])
	}
	return NewJavaStringUTF16(units)
}

func pathReferenceArray(values ...*JavaString) *ReferenceArray {
	array := NewReferenceArray(len(values), StringTypeID)
	for index, value := range values {
		ReferenceArraySet(array, index, value)
	}
	return array
}

func TestJavaPathReferenceStringArrayAndExpandedVarargs(t *testing.T) {
	first := pathReferenceString("root//")
	parts := []*JavaString{pathReferenceString(""), pathReferenceString("part/"), pathReferenceString("leaf17")}
	array := pathReferenceArray(parts...)
	var expanded *JavaPath = PathsGetReference(first, parts...)
	var packed *JavaPath = PathsGetArrayReference(first, array)
	want := pathReferenceString("root/part/leaf17").UTF16Copy()
	for name, path := range map[string]*JavaPath{"expanded": expanded, "array": packed} {
		t.Run(name, func(t *testing.T) {
			if got := PathToStringReference(path); got == nil || !slices.Equal(got.UTF16Copy(), want) || path.GetNameCount() != 3 {
				t.Fatal("String varargs/array overload changed observed path or name count")
			}
		})
	}
	for index, value := range parts {
		if ReferenceArrayGet[*JavaString](array, index, StringTypeID) != value {
			t.Fatal("Paths.get changed an input String-array reference")
		}
	}
}

func TestJavaPathReferenceEmptyRootAbsoluteAndUTF16(t *testing.T) {
	for _, test := range []struct {
		name  string
		first *JavaString
		more  []*JavaString
		units []uint16
		names int32
	}{
		{"empty", pathReferenceString(""), []*JavaString{pathReferenceString(""), pathReferenceString("")}, nil, 1},
		{"root", pathReferenceString("/"), []*JavaString{pathReferenceString(""), pathReferenceString("")}, []uint16{'/'}, 0},
		{"absolute-later", pathReferenceString(""), []*JavaString{pathReferenceString("/"), pathReferenceString("leaf17")}, pathReferenceString("/leaf17").UTF16Copy(), 1},
		{"pair", NewJavaStringUTF16([]uint16{'a', 0xd800, 0xdc00, 'b'}), nil, []uint16{'a', 0xd800, 0xdc00, 'b'}, 1},
		{"bmp", NewJavaStringUTF16([]uint16{0x03b1}), []*JavaString{pathReferenceString("leaf17")}, []uint16{0x03b1, '/', 'l', 'e', 'a', 'f', '1', '7'}, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := PathsGetReference(test.first, test.more...)
			if text := PathToStringReference(path); text == nil || !slices.Equal(text.UTF16Copy(), test.units) || path.GetNameCount() != test.names {
				t.Fatal("canonical path changed observed UTF16/name count")
			}
		})
	}
}

func TestJavaPathReferenceNullOrderAndMessages(t *testing.T) {
	segment := pathReferenceString("Cannot invoke \"String.isEmpty()\" because \"segment\" is null")
	for _, test := range []struct {
		name    string
		first   *JavaString
		more    *ReferenceArray
		message *JavaString
	}{
		{"null-first", nil, pathReferenceArray(pathReferenceString("ok")), nil},
		{"null-first-and-array", nil, nil, nil},
		{"null-array", pathReferenceString("ok"), nil, pathReferenceString("Cannot read the array length because \"more\" is null")},
		{"null-segment", pathReferenceString("ok"), pathReferenceArray(pathReferenceString("part"), nil), segment},
		{"NUL-before-null", NewJavaStringUTF16([]uint16{'a', 0, 'b'}), pathReferenceArray(nil), segment},
		{"surrogate-before-null", NewJavaStringUTF16([]uint16{'a', 0xd800, 'b'}), pathReferenceArray(nil), segment},
	} {
		t.Run(test.name, func(t *testing.T) {
			failure := pathReferencePanic(t, func() { PathsGetArrayReference(test.first, test.more) })
			if exception, ok := failure.(Throwable); !ok || exception.ThrowableTypeName() != "NullPointerException" {
				t.Fatalf("null path argument panic = %T, want NullPointerException", failure)
			}
			got := JavaThrowableMessageDefault(failure)
			if test.message == nil {
				if got != nil {
					t.Fatal("null first String message must remain null")
				}
			} else if got == nil || !slices.Equal(got.UTF16Copy(), test.message.UTF16Copy()) {
				t.Fatal("null array/segment message differs from installed JDK observation")
			}
		})
	}
}

func TestJavaPathReferenceInvalidInputReasonIndex(t *testing.T) {
	for _, test := range []struct {
		name   string
		first  []uint16
		more   []*JavaString
		input  []uint16
		reason string
	}{
		{"NUL-first", []uint16{'a', 0, 'b'}, nil, []uint16{'a', 0, 'b'}, "Nul character not allowed"},
		{"NUL-segment", []uint16{'r', 'o', 'o', 't'}, []*JavaString{NewJavaStringUTF16([]uint16{'a', 0, 'b'})}, []uint16{'r', 'o', 'o', 't', '/', 'a', 0, 'b'}, "Nul character not allowed"},
		{"high", []uint16{'a', 0xd800, 'b'}, nil, []uint16{'a', 0xd800, 'b'}, "Malformed input or input contains unmappable characters"},
		{"low", []uint16{'a', 0xdc00, 'b'}, nil, []uint16{'a', 0xdc00, 'b'}, "Malformed input or input contains unmappable characters"},
	} {
		t.Run(test.name, func(t *testing.T) {
			failure := pathReferencePanic(t, func() { PathsGetReference(NewJavaStringUTF16(test.first), test.more...) })
			if exception, ok := failure.(Throwable); !ok || exception.ThrowableTypeName() != "InvalidPathException" {
				t.Fatalf("invalid path panic = %T, want InvalidPathException", failure)
			}
			detail, ok := failure.(interface {
				GetInput() *JavaString
				GetReason() *JavaString
				GetIndex() int32
			})
			if !ok || detail.GetInput() == nil || !slices.Equal(detail.GetInput().UTF16Copy(), test.input) || detail.GetReason() == nil || !slices.Equal(detail.GetReason().UTF16Copy(), pathReferenceString(test.reason).UTF16Copy()) || detail.GetIndex() != -1 {
				t.Fatal("InvalidPathException changed observed raw UTF16 input/reason/index")
			}
			message := append(pathReferenceString(test.reason+": ").UTF16Copy(), test.input...)
			if got := JavaThrowableMessageDefault(failure); got == nil || !slices.Equal(got.UTF16Copy(), message) {
				t.Fatal("InvalidPathException rendered a lossy native message")
			}
		})
	}
}

func TestJavaPathReferenceArgumentEvaluationAndNativeAPI(t *testing.T) {
	trace := []string{}
	evaluate := func(label, value string) *JavaString {
		trace = append(trace, label)
		return pathReferenceString(value)
	}
	path := PathsGetReference(evaluate("first", "root"), evaluate("one", "a"), evaluate("two", "leaf17"))
	if !slices.Equal(trace, []string{"first", "one", "two"}) || !slices.Equal(PathToStringReference(path).UTF16Copy(), pathReferenceString("root/a/leaf17").UTF16Copy()) {
		t.Fatal("path helper changed the observed argument evaluation sequence")
	}
	if got := PathsGet("native", "entry").ToString(); got != "native/entry" {
		t.Fatal("native Go PathsGet API changed")
	}
}

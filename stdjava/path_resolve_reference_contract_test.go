package stdjava

import (
	"slices"
	"strconv"
	"testing"
)

// These cases are derived from actual JDK21 capture213d68ea, independently
// audited in review4a7efd04. The original Paths/IOException RED tests are frozen.
func TestJavaPathResolveReferenceAuditedCases(t *testing.T) {
	for _, seed := range []int{17, 41, 97} {
		t.Run(strconv.Itoa(seed), func(t *testing.T) {
			leaf := "leaf" + strconv.Itoa(seed)
			for _, test := range []struct {
				name, base, text string
				other            *JavaString
				pathOverload     bool
				names            int32
				absolute         bool
				sameOther        bool
			}{
				{"relative", "root//part/", "root/part/" + leaf, pathReferenceString(leaf), false, 3, false, false},
				{"empty", "root//part/", "root/part", pathReferenceString(""), false, 2, false, false},
				{"absolute", "root//part/", "/" + leaf, pathReferenceString("/" + leaf), false, 1, true, false},
				{"slashes", "root//part/", "root/part/a/" + leaf, pathReferenceString("a//" + leaf + "/"), false, 4, false, false},
				{"dots", "root//part/", "root/part/./../" + leaf, pathReferenceString("./../" + leaf), false, 5, false, false},
				{"root-base", "/", "/" + leaf, pathReferenceString(leaf), false, 1, true, false},
				{"empty-base", "", leaf, pathReferenceString(leaf), false, 1, false, false},
				{"pair", "root//part/", "", NewJavaStringUTF16([]uint16{'a', 0xd800, 0xdc00, 'b'}), false, 3, false, false},
				{"bmp", "root//part/", "", NewJavaStringUTF16(append([]uint16{0x03b1, '/'}, pathReferenceString(leaf).UTF16Copy()...)), false, 4, false, false},
				{"path-relative", "root//part/", "root/part/" + leaf, pathReferenceString(leaf), true, 3, false, false},
				{"path-empty", "root//part/", "root/part", pathReferenceString(""), true, 2, false, false},
				{"path-absolute", "root//part/", "/" + leaf, pathReferenceString("/" + leaf), true, 1, true, true},
			} {
				t.Run(test.name, func(t *testing.T) {
					base := PathsGetReference(pathReferenceString(test.base))
					var result, other *JavaPath
					if test.pathOverload {
						other = PathsGetReference(test.other)
						result = PathResolvePathReference(base, other)
					} else {
						result = PathResolveStringReference(base, test.other)
					}
					want := pathReferenceString(test.text).UTF16Copy()
					if test.name == "pair" || test.name == "bmp" {
						want = append(pathReferenceString("root/part/").UTF16Copy(), test.other.UTF16Copy()...)
					}
					if result == base || PathIsAbsoluteReference(result) != test.absolute || result.GetNameCount() != test.names || !slices.Equal(PathToStringReference(result).UTF16Copy(), want) {
						t.Fatal("resolve String/Path changed audited text, count, absolute flag, or base identity")
					}
					if test.pathOverload && (result == other) != test.sameOther {
						t.Fatal("resolve(Path) changed audited other identity")
					}
				})
			}
			for _, test := range []struct {
				name, class, reason string
				other               *JavaString
				pathOverload        bool
			}{
				{"null-string", "NullPointerException", "", nil, false},
				{"null-path", "NullPointerException", "", nil, true},
				{"NUL", "InvalidPathException", "Nul character not allowed", NewJavaStringUTF16([]uint16{'a', 0, 'b'}), false},
				{"high", "InvalidPathException", "Malformed input or input contains unmappable characters", NewJavaStringUTF16([]uint16{'a', 0xd800, 'b'}), false},
				{"low", "InvalidPathException", "Malformed input or input contains unmappable characters", NewJavaStringUTF16([]uint16{'a', 0xdc00, 'b'}), false},
			} {
				t.Run(test.name, func(t *testing.T) {
					base := PathsGetReference(pathReferenceString("root//part/"))
					failure := pathReferencePanic(t, func() {
						if test.pathOverload {
							PathResolvePathReference(base, nil)
						} else {
							PathResolveStringReference(base, test.other)
						}
					})
					if throwable, ok := failure.(Throwable); !ok || throwable.ThrowableTypeName() != test.class {
						t.Fatalf("resolve exception = %T, want %s", failure, test.class)
					}
					if test.other == nil {
						if JavaThrowableMessageDefault(failure) != nil {
							t.Fatal("null resolve overload must preserve null message")
						}
					} else {
						detail, ok := failure.(interface {
							GetInput() *JavaString
							GetReason() *JavaString
							GetIndex() int32
						})
						if !ok || !slices.Equal(detail.GetInput().UTF16Copy(), test.other.UTF16Copy()) || detail.GetIndex() != -1 || !slices.Equal(detail.GetReason().UTF16Copy(), pathReferenceString(test.reason).UTF16Copy()) {
							t.Fatal("invalid resolve input must retain only the other String, reason, and index")
						}
						wantMessage := append(pathReferenceString(test.reason+": ").UTF16Copy(), test.other.UTF16Copy()...)
						if !slices.Equal(JavaThrowableMessageDefault(failure).UTF16Copy(), wantMessage) || ObjectGetClass(failure).GetName() != "java.nio.file.InvalidPathException" {
							t.Fatal("invalid resolve message/class changed audited raw UTF16 or qualification")
						}
						requested := BuiltinThrowableTypeID("InvalidPathException")
						if requested != "java.nio.file.InvalidPathException" || !ObjectInstanceOf(failure, requested) || !ObjectInstanceOf(failure, BuiltinThrowableTypeID("IllegalArgumentException")) {
							t.Fatal("invalid resolve exception lost its canonical builtin descriptor or ancestry")
						}
						view := ObjectView[InvalidPathException](failure, requested)
						if !JavaReferenceEqual(view, failure) || !slices.Equal(view.GetInput().UTF16Copy(), test.other.UTF16Copy()) {
							t.Fatal("nominal InvalidPathException cast changed allocation identity or raw input")
						}
					}
					cause := NewJavaExceptionMessage(pathReferenceString("cause"))
					if !JavaReferenceEqual(ThrowableInitCauseExecution(NewExecution(), failure, cause), failure) || GetCause(failure) != cause {
						t.Fatal("resolve exception initCause did not preserve receiver/cause identity")
					}
				})
			}
		})
	}
}

func TestJavaPathResolveReferenceEvaluationAndEquality(t *testing.T) {
	trace := []string{}
	evaluate := func(label, text string) *JavaString {
		trace = append(trace, label)
		return pathReferenceString(text)
	}
	path := PathResolveStringReference(PathsGetReference(evaluate("base", "root")), evaluate("other", "leaf17"))
	if !slices.Equal(trace, []string{"base", "other"}) || !slices.Equal(PathToStringReference(path).UTF16Copy(), pathReferenceString("root/leaf17").UTF16Copy()) {
		t.Fatal("resolve changed the audited two argument evaluations")
	}
	equal := PathsGetArrayReference(pathReferenceString("root"), pathReferenceArray(pathReferenceString("leaf17")))
	if !path.Equals(equal) || !equal.Equals(path) || path.Equals((*JavaPath)(nil)) || path.Equals(pathReferenceString("root/leaf17")) || path.HashCode() != equal.HashCode() {
		t.Fatal("Path equality borrowed identity/text from a foreign type or disagreed with lexical hash")
	}
}

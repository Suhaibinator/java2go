package stdjava

import "testing"

func TestJavaStringContainsExecutionCanonicalUTF16(t *testing.T) {
	execution := NewExecution()
	text := NewJavaStringUTF16([]uint16{'a', 0xd800, 0, 0xdfff, 'b'})
	for _, units := range [][]uint16{nil, {0xd800}, {0}, {0xdfff}, {0xd800, 0, 0xdfff}} {
		if !JavaStringContainsExecution(execution, text, NewJavaStringUTF16(units)) {
			t.Fatalf("missing UTF16 target %x", units)
		}
	}
	if JavaStringContainsExecution(execution, text, NewJavaStringUTF16([]uint16{0xfffd})) {
		t.Fatal("isolated surrogate matched replacement character")
	}
	id := TypeID("contains.reference.Source")
	RegisterJavaType(id, ObjectTypeID)
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredReference")
	value := &sourceReferenceTextGuard{id: id, value: NewJavaStringUTF16([]uint16{0xd800, 0})}
	if !JavaStringContainsExecution(execution, text, value) || value.calls != 1 || value.entered != execution {
		t.Fatal("contains lost source toString execution or invocation count")
	}
	value.value = nil
	defer func() {
		if !CaughtAs(recover(), "NullPointerException") {
			t.Error("null toString result must throw NPE")
		}
	}()
	JavaStringContainsExecution(execution, text, value)
}

func TestJavaStringContainsExecutionRejectsNull(t *testing.T) {
	for _, test := range []struct {
		name   string
		text   *JavaString
		target any
	}{
		{"receiver", nil, JavaStringLiteralUTF16(nil)},
		{"target", JavaStringLiteralUTF16(nil), nil},
		{"typed target", JavaStringLiteralUTF16(nil), (*StringBuilder)(nil)},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if !CaughtAs(recover(), "NullPointerException") {
					t.Error("expected NPE")
				}
			}()
			JavaStringContainsExecution(NewExecution(), test.text, test.target)
		})
	}
}

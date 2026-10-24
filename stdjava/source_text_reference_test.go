package stdjava

import "testing"

type sourceReferenceTextGuard struct {
	id      TypeID
	value   *JavaString
	calls   int
	entered *Execution
}

func (s *sourceReferenceTextGuard) JavaDynamicTypeID() TypeID { return s.id }
func (s *sourceReferenceTextGuard) DeclaredReference(execution *Execution) *JavaString {
	s.calls++
	s.entered = execution
	// Registration during a callback must not run under a registry read lock.
	RegisterJavaType(s.id, ObjectTypeID)
	RegisterJavaSourceToString(s.id, "DeclaredReference")
	return s.value
}
func (s *sourceReferenceTextGuard) NativeResult(*Execution) string { return "native" }
func (s *sourceReferenceTextGuard) StringJava2goExecution(*Execution) *JavaString {
	s.calls++
	return JavaStringLiteralUTF16([]uint16{'i'})
}

func TestRegisteredSourceJavaStringReferenceBoundary(t *testing.T) {
	id := TypeID("source.reference.guard.Present")
	RegisterJavaType(id, ObjectTypeID)
	RegisterJavaSourceType(id)
	RegisterJavaSourceToString(id, "DeclaredReference")
	value := &sourceReferenceTextGuard{id: id, value: NewJavaStringUTF16([]uint16{0xD800, 0, 0xDFFF})}
	execution := NewExecution()
	got, found := callRegisteredSourceJavaString(execution, value)
	if !found || got != value.value || value.calls != 1 || value.entered != execution {
		t.Fatalf("registered result lost reference or execution: found=%t same=%t calls=%d", found, got == value.value, value.calls)
	}
	value.value = nil
	got, found = callRegisteredSourceJavaString(execution, value)
	if !found || got != nil || value.calls != 2 {
		t.Fatalf("registered null was treated as absent: found=%t value=%v calls=%d", found, got, value.calls)
	}
}

func TestRegisteredSourceJavaStringRejectsAbsentAndNativeResult(t *testing.T) {
	id := TypeID("source.reference.guard.Absent")
	RegisterJavaType(id, ObjectTypeID)
	RegisterJavaSourceType(id)
	value := &sourceReferenceTextGuard{id: id}
	execution := NewExecution()
	if got, found := callRegisteredSourceJavaString(execution, value); found || got != nil || value.calls != 0 {
		t.Fatal("unregistered ordinary source method used as Java toString")
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("source without migrated default adapter fell through to structural method")
			}
		}()
		JavaStringValueOfExecution(execution, value)
	}()
	if value.calls != 0 {
		t.Fatal("ordinary source String method was invoked")
	}
	RegisterJavaSourceToString(id, "NativeResult")
	defer func() {
		if recover() == nil {
			t.Error("native string result was silently converted to Java reference")
		}
	}()
	callRegisteredSourceJavaString(execution, value)
}

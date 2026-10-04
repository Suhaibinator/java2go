package stdjava

import (
	"slices"
	"testing"
	"unicode/utf16"
)

type sourceObjectReferenceTextGuard struct {
	id      TypeID
	entered *Execution
	hashes  int
}

func (s *sourceObjectReferenceTextGuard) JavaDynamicTypeID() TypeID { return s.id }
func (s *sourceObjectReferenceTextGuard) HashCodeJava2goExecution(execution *Execution) int32 {
	s.entered = execution
	s.hashes++
	return -1
}
func (*sourceObjectReferenceTextGuard) StringJava2goExecution(*Execution) *JavaString {
	panic("ordinary structural method used as Object default")
}
func (*sourceObjectReferenceTextGuard) String() string {
	panic("host presentation used as Object default")
}

func TestRegisteredSourceObjectJavaStringDefaultExecution(t *testing.T) {
	id := TypeID("source.object.reference.\u00c9ntry")
	RegisterJavaType(id, ObjectTypeID)
	RegisterJavaSourceType(id)
	RegisterJavaSourceObjectToString(id)
	value := &sourceObjectReferenceTextGuard{id: id}
	execution := NewExecution()
	first := JavaStringValueOfExecution(execution, value)
	second := JavaStringValueOfExecution(execution, value)
	want := utf16.Encode([]rune(string(id) + "@ffffffff"))
	if !slices.Equal(first.UTF16Copy(), want) || !slices.Equal(second.UTF16Copy(), want) || first == second {
		t.Fatal("Object default lost nominal text or fresh allocation")
	}
	if value.entered != execution || value.hashes != 2 {
		t.Fatal("Object default lost the caller execution or virtual hashCode count")
	}
}

func TestRegisteredSourceObjectDefaultKeepsInheritedOverride(t *testing.T) {
	parent := TypeID("source.object.reference.Parent")
	child := TypeID("source.object.reference.Child")
	RegisterJavaType(parent, ObjectTypeID)
	RegisterJavaSourceType(parent)
	RegisterJavaSourceToString(parent, "DeclaredReference")
	RegisterJavaType(child, parent)
	RegisterJavaSourceType(child)
	RegisterJavaSourceObjectToString(child)
	execution := NewExecution()
	value := &sourceReferenceTextGuard{id: child, value: NewJavaStringUTF16([]uint16{0xD800})}
	if got := JavaStringValueOfExecution(execution, value); got != value.value || value.entered != execution || value.calls != 1 {
		t.Fatal("child default metadata shadowed its parent's exact Java String override")
	}
	value.value = nil
	if got := JavaStringValueOfExecution(execution, value); got != nil || value.calls != 2 {
		t.Fatal("child default metadata replaced its parent's null override")
	}
}

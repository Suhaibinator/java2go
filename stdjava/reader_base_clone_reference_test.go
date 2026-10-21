package stdjava

import "testing"

func TestReaderBaseClonePreservesShallowLockAndIndependentState(t *testing.T) {
	source := &struct{ value int }{value: 7}
	original := NewReaderBase(source)
	copy := CloneReaderBaseReference(original)
	if copy == original || !JavaReferenceEqual(copy.Lock, source) {
		t.Fatal("native clone changed Java lock reference or reused mutable carrier")
	}
	sentinel := &struct{ value int }{value: 9}
	original.Lock = sentinel
	if !JavaReferenceEqual(copy.Lock, source) {
		t.Fatal("original native-state write changed clone")
	}
	other := CloneReaderBaseReference(original)
	other.Lock = nil
	if !JavaReferenceEqual(original.Lock, sentinel) || other.Lock != nil {
		t.Fatal("clone native-state write changed original")
	}
	original.Lock = nil
	nullable := CloneReaderBaseReference(original)
	if nullable == original || nullable.Lock != nil {
		t.Fatal("nullable lock was rebound or passed through constructor validation")
	}
	nativeDefault := NewReaderBase()
	defaultCopy := CloneReaderBaseReference(nativeDefault)
	if defaultCopy == nativeDefault || !JavaReferenceEqual(defaultCopy.Lock, nativeDefault) {
		t.Fatal("default self-valued lock was rebound to clone")
	}
}

func TestReaderBaseCloneRejectsAbsentCarrier(t *testing.T) {
	defer func() {
		if _, ok := recover().(NullPointerException); !ok {
			t.Fatal("absent native carrier must not fabricate state")
		}
	}()
	CloneReaderBaseReference(nil)
}

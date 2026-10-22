package stdjava

import "testing"

func TestReentrantLock180PublicAllocationIdentity(t *testing.T) {
	lock := NewReentrantLock()
	var erased any = lock
	view := ObjectView[any](lock, ObjectTypeID)
	if !JavaReferenceEqual(lock, erased) || !JavaReferenceEqual(lock, view) || JavaReferenceEqual(lock, NewReentrantLock()) {
		t.Fatal("Object aliases must preserve the same distinct Java lock allocation")
	}
	execution := NewExecution()
	if ObjectHashCodeExecution(execution, lock) != ObjectHashCodeExecution(execution, view) || monitorRecord(lock) != monitorRecord(view) {
		t.Fatal("Object hash/monitor changed across views of the same lock")
	}
	if monitorRecord(lock) == monitorRecord(&lock.anchor) {
		t.Fatal("public allocation monitor aliases the private explicit lock")
	}
}

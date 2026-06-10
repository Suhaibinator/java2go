package stdjava

import "testing"

func TestListAddGetSize(t *testing.T) {
	l := NewList[string]()
	l.Add("a")
	l.Add("b")
	l.Add("c")
	if l.Size() != 3 {
		t.Fatalf("Size = %d, want 3", l.Size())
	}
	if l.Get(1) != "b" {
		t.Fatalf("Get(1) = %q, want b", l.Get(1))
	}
}

func TestListSetRemoveContains(t *testing.T) {
	l := NewListFrom(1, 2, 3)
	if old := l.Set(0, 9); old != 1 {
		t.Fatalf("Set returned %d, want 1", old)
	}
	if l.Get(0) != 9 {
		t.Fatalf("after Set Get(0) = %d, want 9", l.Get(0))
	}
	if !l.Contains(2) {
		t.Fatalf("Contains(2) = false, want true")
	}
	if l.IndexOf(3) != 2 {
		t.Fatalf("IndexOf(3) = %d, want 2", l.IndexOf(3))
	}
	if removed := l.RemoveAt(1); removed != 2 {
		t.Fatalf("RemoveAt(1) = %d, want 2", removed)
	}
	if l.Size() != 2 {
		t.Fatalf("Size after remove = %d, want 2", l.Size())
	}
}

func TestListIsEmptyClearAddAll(t *testing.T) {
	l := NewList[int32]()
	if !l.IsEmpty() {
		t.Fatalf("new list IsEmpty = false")
	}
	l.AddAll(NewListFrom[int32](1, 2))
	if l.Size() != 2 {
		t.Fatalf("Size after AddAll = %d, want 2", l.Size())
	}
	l.Clear()
	if !l.IsEmpty() {
		t.Fatalf("IsEmpty after Clear = false")
	}
}

func TestListSliceForRange(t *testing.T) {
	l := NewListFrom("x", "y")
	got := ""
	for _, e := range l.Slice() {
		got += e
	}
	if got != "xy" {
		t.Fatalf("ranged over Slice got %q, want xy", got)
	}
}

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

func TestListString(t *testing.T) {
	if got := NewListFrom(1, 2, 3).String(); got != "[1, 2, 3]" {
		t.Fatalf("List.String() = %q, want [1, 2, 3]", got)
	}
	if got := NewList[string]().String(); got != "[]" {
		t.Fatalf("empty List.String() = %q, want []", got)
	}
}

func TestListStringRendersNullStringWithoutCollapsingStringValues(t *testing.T) {
	list := NewListFrom(NullString(), "null", "")
	if list.Size() != 3 || !StringIsNull(list.Get(0)) || StringIsNull(list.Get(1)) || StringIsNull(list.Get(2)) {
		t.Fatalf("List lost null/literal-null/empty distinction: %#v", list.Slice())
	}
	if got := list.String(); got != "[null, null, ]" {
		t.Fatalf("List.String(nullable Strings) = %q, want [null, null, ]", got)
	}
}

func TestListToArrayReturnsDistinctIdentityPreservingCopies(t *testing.T) {
	empty := NewList[int32]()
	first := empty.ToArray()
	second := empty.ToArray()
	if len(first) != 0 || cap(first) != 1 || len(second) != 0 || cap(second) != 1 {
		t.Fatalf("empty ToArray shapes = (%d,%d) and (%d,%d), want len 0 cap 1", len(first), cap(first), len(second), cap(second))
	}
	if monitorFor(first) == monitorFor(second) {
		t.Fatal("separate empty List.toArray calls must return distinct Java array objects")
	}

	values := NewListFrom[int32](1, 2, 3)
	copy := values.ToArray()
	copy[0] = 9
	if values.Get(0) != 1 {
		t.Fatalf("ToArray result aliases List backing storage: list[0] = %d, want 1", values.Get(0))
	}
}

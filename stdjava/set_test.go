package stdjava

import "testing"

func TestSetAddContainsRemove(t *testing.T) {
	s := NewSet[string]()
	if !s.Add("a") {
		t.Fatalf("first Add(a) = false, want true")
	}
	if s.Add("a") {
		t.Fatalf("duplicate Add(a) = true, want false")
	}
	s.Add("b")
	if !s.Contains("a") || !s.Contains("b") {
		t.Fatalf("Contains failed")
	}
	if s.Size() != 2 {
		t.Fatalf("Size = %d, want 2", s.Size())
	}
	if !s.Remove("a") {
		t.Fatalf("Remove(a) = false, want true")
	}
	if s.Contains("a") {
		t.Fatalf("a present after Remove")
	}
}

func TestSetSliceOrder(t *testing.T) {
	s := NewSet[int32]()
	s.Add(3)
	s.Add(1)
	s.Add(2)
	got := s.Slice()
	want := []int32{3, 1, 2}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("Slice[%d] = %d, want %d (insertion order)", i, got[i], v)
		}
	}
}

func TestSetString(t *testing.T) {
	s := NewSet[int32]()
	s.Add(1)
	s.Add(2)
	if got := s.String(); got != "[1, 2]" {
		t.Fatalf("Set.String() = %q, want [1, 2]", got)
	}
}

func TestSetStringRendersNullStringWithoutCollapsingStringValues(t *testing.T) {
	set := NewSet[string]()
	set.Add(NullString())
	set.Add("null")
	set.Add("")

	if set.Size() != 3 || !set.Contains(NullString()) || !set.Contains("null") || !set.Contains("") {
		t.Fatalf("Set lost distinct null/literal-null/empty elements: %#v", set.Slice())
	}
	if got := set.String(); got != "[null, null, ]" {
		t.Fatalf("Set.String(nullable Strings) = %q, want [null, null, ]", got)
	}
}

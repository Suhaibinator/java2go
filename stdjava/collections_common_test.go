package stdjava

import "testing"

func TestSortReverseMinMax(t *testing.T) {
	l := NewListFrom(3, 1, 2)
	SortOrdered(l)
	if l.Get(0) != 1 || l.Get(2) != 3 {
		t.Fatalf("SortOrdered failed: %v", l.Slice())
	}
	ReverseList(l)
	if l.Get(0) != 3 || l.Get(2) != 1 {
		t.Fatalf("ReverseList failed: %v", l.Slice())
	}
	if MaxOrdered(l) != 3 {
		t.Fatalf("MaxOrdered = %d, want 3", MaxOrdered(l))
	}
	if MinOrdered(l) != 1 {
		t.Fatalf("MinOrdered = %d, want 1", MinOrdered(l))
	}
}

func TestArraysHelpers(t *testing.T) {
	l := AsList("a", "b")
	if l.Size() != 2 || l.Get(0) != "a" {
		t.Fatalf("AsList failed: %v", l.Slice())
	}
	arr := []int32{3, 1, 2}
	SortSlice(arr)
	if arr[0] != 1 || arr[2] != 3 {
		t.Fatalf("SortSlice failed: %v", arr)
	}
	if got := SliceToString([]int32{1, 2, 3}); got != "[1, 2, 3]" {
		t.Fatalf("SliceToString = %q, want [1, 2, 3]", got)
	}
}

func TestSingletonAndEmptyList(t *testing.T) {
	if SingletonList("x").Size() != 1 {
		t.Fatalf("SingletonList size wrong")
	}
	if !EmptyList[int]().IsEmpty() {
		t.Fatalf("EmptyList not empty")
	}
}

func TestObjectsEqual(t *testing.T) {
	if !ObjectsEqual("a", "a") || ObjectsEqual("a", "b") {
		t.Fatalf("ObjectsEqual string failed")
	}
	if !ObjectsEqual([]int{1, 2}, []int{1, 2}) {
		t.Fatalf("ObjectsEqual deep failed")
	}
}

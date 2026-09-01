package stdjava

import "testing"

func TestStreamFilterMapCollect(t *testing.T) {
	s := NewStream[int32](1, 2, 3, 4)
	evens := s.Filter(func(n int32) bool { return n%2 == 0 })
	if evens.Count() != 2 {
		t.Fatalf("Filter count = %d, want 2", evens.Count())
	}
	doubled := StreamMap(evens, func(n int32) int32 { return n * 2 })
	got := doubled.ToList().Slice()
	if len(got) != 2 || got[0] != 4 || got[1] != 8 {
		t.Fatalf("StreamMap result = %v, want [4 8]", got)
	}
}

func TestStreamMapChangesType(t *testing.T) {
	s := NewStream[int32](1, 2)
	strs := StreamMap(s, func(n int32) string {
		if n == 1 {
			return "a"
		}
		return "b"
	})
	got := strs.ToSlice()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("type-changing map = %v, want [a b]", got)
	}
}

func TestStreamReduce(t *testing.T) {
	s := NewStream[int32](1, 2, 3, 4)
	sum := StreamReduce(s, 0, func(a, b int32) int32 { return a + b })
	if sum != 10 {
		t.Fatalf("StreamReduce sum = %d, want 10", sum)
	}
}

func TestStreamMatchAndForEach(t *testing.T) {
	s := NewStream[int32](2, 4, 6)
	if !s.AllMatch(func(n int32) bool { return n%2 == 0 }) {
		t.Fatalf("AllMatch even = false, want true")
	}
	if !s.AnyMatch(func(n int32) bool { return n == 4 }) {
		t.Fatalf("AnyMatch 4 = false, want true")
	}
	if s.NoneMatch(func(n int32) bool { return n == 4 }) {
		t.Fatalf("NoneMatch 4 = true, want false")
	}
	total := int32(0)
	s.ForEach(func(n int32) { total += n })
	if total != 12 {
		t.Fatalf("ForEach sum = %d, want 12", total)
	}
}

func TestStreamSortedAndLimit(t *testing.T) {
	s := NewStream[int32](3, 1, 2)
	sorted := StreamSorted(s).ToSlice()
	if sorted[0] != 1 || sorted[2] != 3 {
		t.Fatalf("StreamSorted = %v, want [1 2 3]", sorted)
	}
	limited := NewStream[int32](1, 2, 3, 4).Limit(2).ToSlice()
	if len(limited) != 2 || limited[1] != 2 {
		t.Fatalf("Limit(2) = %v, want [1 2]", limited)
	}
}

func TestStreamOfSlice(t *testing.T) {
	s := StreamOfSlice([]string{"x", "y"})
	if s.Count() != 2 {
		t.Fatalf("StreamOfSlice count = %d, want 2", s.Count())
	}
}

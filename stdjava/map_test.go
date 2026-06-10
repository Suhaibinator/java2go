package stdjava

import "testing"

func TestMapPutGetContains(t *testing.T) {
	m := NewMap[string, int32]()
	m.Put("one", 1)
	m.Put("two", 2)
	if m.Get("one") != 1 {
		t.Fatalf("Get(one) = %d, want 1", m.Get("one"))
	}
	if !m.ContainsKey("two") {
		t.Fatalf("ContainsKey(two) = false, want true")
	}
	if m.ContainsKey("three") {
		t.Fatalf("ContainsKey(three) = true, want false")
	}
	if m.Size() != 2 {
		t.Fatalf("Size = %d, want 2", m.Size())
	}
}

func TestMapGetOrDefaultAndValueOps(t *testing.T) {
	m := NewMap[string, int32]()
	m.Put("a", 5)
	if m.GetOrDefault("a", -1) != 5 {
		t.Fatalf("GetOrDefault present = %d, want 5", m.GetOrDefault("a", -1))
	}
	if m.GetOrDefault("z", -1) != -1 {
		t.Fatalf("GetOrDefault absent = %d, want -1", m.GetOrDefault("z", -1))
	}
	if !m.ContainsValue(5) {
		t.Fatalf("ContainsValue(5) = false, want true")
	}
	if old := m.Remove("a"); old != 5 {
		t.Fatalf("Remove returned %d, want 5", old)
	}
	if m.ContainsKey("a") {
		t.Fatalf("key present after Remove")
	}
}

func TestMapInsertionOrderViews(t *testing.T) {
	m := NewMap[string, int32]()
	m.Put("a", 1)
	m.Put("b", 2)
	m.Put("c", 3)
	keys := m.KeySet()
	want := []string{"a", "b", "c"}
	for i, k := range want {
		if keys[i] != k {
			t.Fatalf("KeySet[%d] = %q, want %q", i, keys[i], k)
		}
	}
	vals := m.Values()
	for i, v := range []int32{1, 2, 3} {
		if vals[i] != v {
			t.Fatalf("Values[%d] = %d, want %d", i, vals[i], v)
		}
	}
	entries := m.EntrySet()
	if len(entries) != 3 || entries[0].Key != "a" || entries[0].Value != 1 {
		t.Fatalf("EntrySet unexpected: %+v", entries)
	}
}

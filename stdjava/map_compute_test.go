package stdjava

import "testing"

func TestMapComputeIfAbsentStructuralMutation(t *testing.T) {
	for _, sorted := range []bool{false, true} {
		for _, change := range []string{"clear-empty", "insert-remove", "remove-existing"} {
			m := NewMap[string, string]()
			if sorted {
				m = NewTreeMap[string, string]()
			}
			if change == "remove-existing" {
				m.Put("existing", "value")
			}
			func() {
				defer func() {
					failure := recover()
					if failure == nil || !CaughtAs(failure, "ConcurrentModificationException") {
						t.Errorf("sorted=%v change=%s failure=%v", sorted, change, failure)
					}
				}()
				m.ComputeIfAbsent("target", func(string) string {
					switch change {
					case "clear-empty":
						m.Clear()
					case "insert-remove":
						m.Put("temporary", "value")
						m.Remove("temporary")
					case "remove-existing":
						m.Remove("existing")
					}
					return "unused"
				})
			}()
			if m.ContainsKey("target") {
				t.Errorf("sorted=%v change=%s inserted rejected result", sorted, change)
			}
		}
	}
}

func TestMapComputeIfAbsentNullValueAndKey(t *testing.T) {
	m := NewMap[string, string]()
	calls := 0
	m.Put(NullString(), NullString())
	got := m.ComputeIfAbsent(NullString(), func(key string) string {
		calls++
		if !javaReferenceIsNull(key) {
			t.Fatal("null key lost")
		}
		return "made"
	})
	if got != "made" || calls != 1 || m.Get(NullString()) != "made" {
		t.Fatalf("value=%q calls=%d", got, calls)
	}
	tree := NewTreeMap[string, string]()
	if value := tree.ComputeIfAbsent(NullString(), func(string) string { return NullString() }); !javaReferenceIsNull(value) || tree.Size() != 0 {
		t.Fatal("empty TreeMap computed-null contract")
	}
	calls = 0
	func() {
		defer func() {
			failure := recover()
			if failure == nil || !CaughtAs(failure, "NullPointerException") {
				t.Fatalf("invalid TreeMap key failure=%v", failure)
			}
		}()
		tree.ComputeIfAbsent(NullString(), func(string) string { calls++; return "invalid" })
	}()
	if calls != 1 {
		t.Fatalf("TreeMap validated empty-tree key before callback: %d", calls)
	}
}

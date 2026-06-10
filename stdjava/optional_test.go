package stdjava

import "testing"

func TestOptionalPresentAndEmpty(t *testing.T) {
	o := OptionalOf(42)
	if !o.IsPresent() || o.IsEmpty() {
		t.Fatalf("OptionalOf should be present")
	}
	if o.Get() != 42 {
		t.Fatalf("Get = %d, want 42", o.Get())
	}
	if o.OrElse(0) != 42 {
		t.Fatalf("OrElse present = %d, want 42", o.OrElse(0))
	}

	e := OptionalEmpty[int]()
	if e.IsPresent() || !e.IsEmpty() {
		t.Fatalf("OptionalEmpty should be empty")
	}
	if e.OrElse(7) != 7 {
		t.Fatalf("OrElse empty = %d, want 7", e.OrElse(7))
	}
}

func TestOptionalGetPanicsWhenEmpty(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("Get on empty Optional did not panic")
		}
	}()
	OptionalEmpty[int]().Get()
}

func TestOptionalIfPresentAndMap(t *testing.T) {
	called := false
	OptionalOf("x").IfPresent(func(string) { called = true })
	if !called {
		t.Fatalf("IfPresent did not invoke action")
	}
	OptionalEmpty[string]().IfPresent(func(string) { t.Fatalf("IfPresent invoked on empty") })

	mapped := OptionalMap(OptionalOf(3), func(n int) int { return n * 2 })
	if mapped.Get() != 6 {
		t.Fatalf("OptionalMap = %d, want 6", mapped.Get())
	}
	if !OptionalMap(OptionalEmpty[int](), func(n int) int { return n }).IsEmpty() {
		t.Fatalf("OptionalMap over empty should be empty")
	}
}

func TestOfNullable(t *testing.T) {
	if !OptionalOfNullable("v", true).IsPresent() {
		t.Fatalf("OfNullable(present) should be present")
	}
	if !OptionalOfNullable("v", false).IsEmpty() {
		t.Fatalf("OfNullable(absent) should be empty")
	}
}

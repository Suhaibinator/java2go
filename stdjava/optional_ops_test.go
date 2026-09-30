package stdjava

import "testing"

func TestOptionalFilter(t *testing.T) {
	present := OptionalOf("java")
	if !present.Filter(func(s string) bool { return len(s) == 4 }).IsPresent() {
		t.Fatal("Filter dropped a matching value")
	}
	if present.Filter(func(s string) bool { return len(s) == 9 }).IsPresent() {
		t.Fatal("Filter kept a non-matching value")
	}
	// An empty Optional is returned as-is, and the predicate is never invoked.
	invoked := false
	empty := OptionalEmpty[string]()
	if empty.Filter(func(s string) bool { invoked = true; return true }).IsPresent() {
		t.Fatal("Filter on an empty Optional reported a value")
	}
	if invoked {
		t.Fatal("Filter invoked the predicate on an empty Optional")
	}
}

func TestOptionalOrElseGetOnlyCallsSupplierWhenEmpty(t *testing.T) {
	calls := 0
	supplier := func() string { calls++; return "fallback" }

	if got := OptionalOf("value").OrElseGet(supplier); got != "value" {
		t.Fatalf("OrElseGet = %q, want %q", got, "value")
	}
	if calls != 0 {
		t.Fatalf("OrElseGet invoked the supplier %d times on a present Optional, want 0", calls)
	}
	if got := OptionalEmpty[string]().OrElseGet(supplier); got != "fallback" {
		t.Fatalf("OrElseGet = %q, want %q", got, "fallback")
	}
	if calls != 1 {
		t.Fatalf("OrElseGet invoked the supplier %d times on an empty Optional, want 1", calls)
	}
}

func TestOptionalOrElseThrow(t *testing.T) {
	if got := OptionalOf("value").OrElseThrow(nil); got != "value" {
		t.Fatalf("OrElseThrow = %q, want %q", got, "value")
	}

	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("OrElseThrow(nil) on an empty Optional did not panic")
			}
			if !CaughtAs(recovered, "NoSuchElementException") {
				t.Fatalf("panicked with %v, want NoSuchElementException", recovered)
			}
		}()
		OptionalEmpty[string]().OrElseThrow(nil)
	}()

	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("OrElseThrow(supplier) on an empty Optional did not panic")
			}
			if !CaughtAs(recovered, "IllegalStateException") {
				t.Fatalf("panicked with %v, want IllegalStateException", recovered)
			}
		}()
		OptionalEmpty[string]().OrElseThrow(func() any {
			return NewIllegalStateException("nothing here")
		})
	}()
}

func TestOptionalFlatMapDoesNotDoubleWrap(t *testing.T) {
	got := OptionalFlatMap(OptionalOf("java"), func(s string) Optional[string] {
		return OptionalOf(s + "2go")
	})
	if got.Get() != "java2go" {
		t.Fatalf("OptionalFlatMap = %q, want %q", got.Get(), "java2go")
	}
	if OptionalFlatMap(OptionalEmpty[string](), func(s string) Optional[string] {
		return OptionalOf(s)
	}).IsPresent() {
		t.Fatal("OptionalFlatMap on an empty Optional reported a value")
	}
	// A mapper returning empty must stay empty rather than becoming present.
	if OptionalFlatMap(OptionalOf("java"), func(s string) Optional[string] {
		return OptionalEmpty[string]()
	}).IsPresent() {
		t.Fatal("OptionalFlatMap wrapped an empty mapper result")
	}
}

func TestOptionalIfPresentOrElse(t *testing.T) {
	seen := ""
	fellBack := false
	OptionalOf("value").IfPresentOrElse(func(s string) { seen = s }, func() { fellBack = true })
	if seen != "value" || fellBack {
		t.Fatalf("IfPresentOrElse on a present Optional: seen=%q fellBack=%v", seen, fellBack)
	}

	seen, fellBack = "", false
	OptionalEmpty[string]().IfPresentOrElse(func(s string) { seen = s }, func() { fellBack = true })
	if seen != "" || !fellBack {
		t.Fatalf("IfPresentOrElse on an empty Optional: seen=%q fellBack=%v", seen, fellBack)
	}
}

func TestOptionalStream(t *testing.T) {
	if got := OptionalStream(OptionalOf("value")).Count(); got != 1 {
		t.Fatalf("OptionalStream on a present Optional yielded %d elements, want 1", got)
	}
	if got := OptionalStream(OptionalEmpty[string]()).Count(); got != 0 {
		t.Fatalf("OptionalStream on an empty Optional yielded %d elements, want 0", got)
	}
}

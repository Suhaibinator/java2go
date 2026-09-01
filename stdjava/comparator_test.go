package stdjava

import (
	"testing"
)

type comparatorPerson struct {
	name string
	age  int32
}

// comparatorVersion models a generated Comparable implementation: the CompareTo
// parameter is the concrete type, not `any`, which is why the natural-ordering
// bridge has to be reflective.
type comparatorVersion struct {
	major int32
	minor int32
}

func (v *comparatorVersion) CompareTo(other *comparatorVersion) int32 {
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}
	if v.minor == other.minor {
		return 0
	}
	if v.minor < other.minor {
		return -1
	}
	return 1
}

func TestNaturalOrderAndReverseOrder(t *testing.T) {
	natural := NaturalOrder[int32]()
	if got := natural(1, 2); got >= 0 {
		t.Fatalf("NaturalOrder(1, 2) = %d, want negative", got)
	}
	if got := natural(2, 2); got != 0 {
		t.Fatalf("NaturalOrder(2, 2) = %d, want 0", got)
	}
	reverse := ReverseOrder[int32]()
	if got := reverse(1, 2); got <= 0 {
		t.Fatalf("ReverseOrder(1, 2) = %d, want positive", got)
	}
}

func TestComparatorComparingAndReversed(t *testing.T) {
	byAge := ComparatorComparing(func(p comparatorPerson) int32 { return p.age })
	young := comparatorPerson{name: "young", age: 20}
	old := comparatorPerson{name: "old", age: 40}

	if got := byAge(young, old); got >= 0 {
		t.Fatalf("byAge(young, old) = %d, want negative", got)
	}
	if got := byAge.Reversed()(young, old); got <= 0 {
		t.Fatalf("byAge.Reversed()(young, old) = %d, want positive", got)
	}
	if got := ComparatorComparingReversed(func(p comparatorPerson) int32 { return p.age })(young, old); got <= 0 {
		t.Fatalf("ComparatorComparingReversed(young, old) = %d, want positive", got)
	}
}

func TestComparatorThenComparing(t *testing.T) {
	byAge := ComparatorComparing(func(p comparatorPerson) int32 { return p.age })
	byName := ComparatorComparing(func(p comparatorPerson) string { return p.name })

	a := comparatorPerson{name: "alice", age: 30}
	b := comparatorPerson{name: "bob", age: 30}

	if got := byAge(a, b); got != 0 {
		t.Fatalf("byAge on equal ages = %d, want 0", got)
	}
	if got := byAge.ThenComparing(byName)(a, b); got >= 0 {
		t.Fatalf("thenComparing(byName)(alice, bob) = %d, want negative", got)
	}
	if got := ComparatorThenComparingKey(byAge, func(p comparatorPerson) string { return p.name })(a, b); got >= 0 {
		t.Fatalf("ComparatorThenComparingKey(alice, bob) = %d, want negative", got)
	}
}

// TestSortWithIsStable pins Java's guarantee that Collections.sort and List.sort
// are stable: equal elements keep their relative input order.
func TestSortWithIsStable(t *testing.T) {
	list := NewListFrom(
		comparatorPerson{name: "first", age: 30},
		comparatorPerson{name: "second", age: 10},
		comparatorPerson{name: "third", age: 30},
		comparatorPerson{name: "fourth", age: 10},
	)
	SortWith(list, ComparatorComparing(func(p comparatorPerson) int32 { return p.age }))

	want := []string{"second", "fourth", "first", "third"}
	for i, name := range want {
		if got := list.Get(int32(i)).name; got != name {
			t.Fatalf("SortWith result[%d] = %q, want %q (sort is not stable)", i, got, name)
		}
	}
}

func TestSortSliceWithIsStable(t *testing.T) {
	elements := []comparatorPerson{
		{name: "first", age: 2},
		{name: "second", age: 1},
		{name: "third", age: 2},
	}
	SortSliceWith(elements, ComparatorComparing(func(p comparatorPerson) int32 { return p.age }))

	want := []string{"second", "first", "third"}
	for i, name := range want {
		if elements[i].name != name {
			t.Fatalf("SortSliceWith result[%d] = %q, want %q", i, elements[i].name, name)
		}
	}
}

// TestSortWithNilComparatorUsesNaturalOrder pins Java's rule that a null
// comparator means natural ordering rather than a NullPointerException.
func TestSortWithNilComparatorUsesNaturalOrder(t *testing.T) {
	list := NewListFrom[int32](3, 1, 2)
	SortWith(list, nil)
	for i, want := range []int32{1, 2, 3} {
		if got := list.Get(int32(i)); got != want {
			t.Fatalf("SortWith(nil) result[%d] = %d, want %d", i, got, want)
		}
	}
}

// TestMinMaxWithKeepFirstOnTies matches Collections.min/max, which replace the
// incumbent only on a strict improvement.
func TestMinMaxWithKeepFirstOnTies(t *testing.T) {
	list := NewListFrom(
		comparatorPerson{name: "firstMax", age: 30},
		comparatorPerson{name: "firstMin", age: 10},
		comparatorPerson{name: "laterMax", age: 30},
		comparatorPerson{name: "laterMin", age: 10},
	)
	byAge := ComparatorComparing(func(p comparatorPerson) int32 { return p.age })

	if got := MaxWith(list, byAge).name; got != "firstMax" {
		t.Fatalf("MaxWith = %q, want %q", got, "firstMax")
	}
	if got := MinWith(list, byAge).name; got != "firstMin" {
		t.Fatalf("MinWith = %q, want %q", got, "firstMin")
	}
}

func TestMaxWithOnEmptyThrowsNoSuchElement(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("MaxWith on an empty list did not panic")
		}
		if !CaughtAs(recovered, "NoSuchElementException") {
			t.Fatalf("MaxWith panicked with %v, want NoSuchElementException", recovered)
		}
	}()
	MaxWith(NewList[int32](), NaturalOrder[int32]())
}

// TestJavaCompareValuesBridgesCompareTo covers the reflective bridge that lets a
// generated Comparable type sort through Arrays.sort and the Collections
// utilities.
func TestJavaCompareValuesBridgesCompareTo(t *testing.T) {
	older := &comparatorVersion{major: 1, minor: 2}
	newer := &comparatorVersion{major: 1, minor: 9}

	if got := javaCompareValues(older, newer); got >= 0 {
		t.Fatalf("javaCompareValues(1.2, 1.9) = %d, want negative", got)
	}
	if got := javaCompareValues(newer, older); got <= 0 {
		t.Fatalf("javaCompareValues(1.9, 1.2) = %d, want positive", got)
	}
	if got := javaCompareValues(older, older); got != 0 {
		t.Fatalf("javaCompareValues(1.2, 1.2) = %d, want 0", got)
	}
	if !javaComparableLess(older, newer) {
		t.Fatal("javaComparableLess did not use the CompareTo bridge")
	}
}

func TestSortSliceStableNaturalUsesCompareTo(t *testing.T) {
	elements := []any{
		&comparatorVersion{major: 2, minor: 0},
		&comparatorVersion{major: 1, minor: 5},
		&comparatorVersion{major: 1, minor: 0},
	}
	SortSliceStableNatural(elements)

	want := []comparatorVersion{{1, 0}, {1, 5}, {2, 0}}
	for i, expected := range want {
		got := elements[i].(*comparatorVersion)
		if got.major != expected.major || got.minor != expected.minor {
			t.Fatalf("result[%d] = %d.%d, want %d.%d", i, got.major, got.minor, expected.major, expected.minor)
		}
	}
}

func TestJavaCompareValuesRejectsIncomparable(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("comparing incomparable values did not panic")
		}
		if !CaughtAs(recovered, "ClassCastException") {
			t.Fatalf("panicked with %v, want ClassCastException", recovered)
		}
	}()
	javaCompareValues(struct{ x int }{1}, struct{ x int }{2})
}

// TestJavaCompareValuesMixedTypesThrows matches Java, where sorting a
// heterogeneous Object[] throws ClassCastException rather than silently
// producing an arbitrary order.
func TestJavaCompareValuesMixedTypesThrows(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil {
			t.Fatal("comparing an int32 with a string did not panic")
		}
	}()
	javaCompareValues(int32(1), "two")
}

package stdjava

import (
	"math"
	"sync"
	"testing"
)

func expectBoxedCollectionException(t *testing.T, name string, action func()) {
	t.Helper()
	defer func() {
		if value := recover(); value == nil || !CaughtAs(value, name) {
			t.Errorf("exception = %v, want %s", value, name)
		}
	}()
	action()
}

func TestBoxedCollectionEqualityRetainsFirstObject(t *testing.T) {
	pairs := [][2]any{
		{NewBoolean(true), NewBoolean(true)},
		{NewByte(7), NewByte(7)},
		{NewShort(500), NewShort(500)},
		{NewCharacter('x'), NewCharacter('x')},
		{NewInteger(500), NewInteger(500)},
		{NewLong(500), NewLong(500)},
		{NewFloat(1.25), NewFloat(1.25)},
		{NewDouble(1.25), NewDouble(1.25)},
	}
	for _, pair := range pairs {
		first, equal := pair[0], pair[1]
		if JavaReferenceEqual(first, equal) || !ObjectsEqual(first, equal) {
			t.Fatalf("%T constructors must be distinct equal objects", first)
		}
		m := NewMap[any, *Integer]()
		m.Put(first, BoxInteger(1))
		if old := m.Put(equal, BoxInteger(2)); old.IntValue() != 1 {
			t.Fatalf("%T replacement returned %v", first, old)
		}
		if m.Size() != 1 || m.KeySet()[0] != first || m.EntrySet()[0].Key != first || m.Get(equal).IntValue() != 2 {
			t.Fatalf("%T map failed value equality or key identity", first)
		}
		if !m.ContainsValue(NewInteger(2)) || m.ContainsValue(NewLong(2)) || m.Get("absent") != nil {
			t.Fatalf("%T map query did not preserve wrapper kind or missing null", first)
		}
		if removed := m.Remove(equal); removed.IntValue() != 2 || !m.IsEmpty() {
			t.Fatalf("%T map remove failed", first)
		}
		set := NewSet[any]()
		if !set.Add(first) || set.Add(equal) || !set.Contains(equal) || set.Slice()[0] != first || !set.Remove(equal) {
			t.Fatalf("%T set failed value equality or identity", first)
		}
		distinct := StreamDistinct(NewStream(first, equal)).ToSlice()
		if len(distinct) != 1 || distinct[0] != first {
			t.Fatalf("%T stream distinct lost first object", first)
		}
		list := NewListFrom(first, equal)
		if !list.Contains(equal) || list.IndexOf(equal) != 0 || !list.RemoveObject(equal) || list.Get(0) != equal {
			t.Fatalf("%T list failed object membership or removal", first)
		}
	}
}

func TestBoxedCollectionFloatingAndNominalKeys(t *testing.T) {
	nanA := NewDouble(math.Float64frombits(0x7ff8000000000001))
	nanB := NewDouble(math.Float64frombits(0x7ff8000000001234))
	floatNaNA := NewFloat(math.Float32frombits(0x7fc00001))
	floatNaNB := NewFloat(math.Float32frombits(0x7fc00123))
	set := NewSet[any]()
	for _, value := range []any{nanA, nanB, floatNaNA, floatNaNB, NewDouble(0), NewDouble(math.Copysign(0, -1)), NewFloat(0), NewFloat(float32(math.Copysign(0, -1))), NewInteger(65), NewCharacter('A'), NewLong(65)} {
		set.Add(value)
	}
	if set.Size() != 9 || !ObjectsEqual(nanA, nanB) || !ObjectsEqual(floatNaNA, floatNaNB) {
		t.Fatalf("wrapper key normalization lost NaN, signed zero, or wrapper kind: %v", set)
	}
	if ObjectsEqual(NewDouble(0), NewDouble(math.Copysign(0, -1))) || ObjectsEqual(NewCharacter('A'), NewInteger(65)) {
		t.Fatal("different wrapper values compare equal")
	}
}

func TestBoxedCollectionsCanonicalNull(t *testing.T) {
	var integer *Integer
	var long *Long
	m := NewMap[any, *Integer]()
	m.Put(integer, BoxInteger(3))
	m.Put(nil, BoxInteger(4))
	if m.Size() != 1 || !m.ContainsKey(long) || m.Get(nil).IntValue() != 4 || m.KeySet()[0] != any(integer) {
		t.Fatal("map did not preserve one null key and its original representation")
	}
	values := StreamDistinct(NewStream[any](integer, nil, long, NullString(), "null")).ToSlice()
	if len(values) != 2 || values[0] != any(integer) || values[1] != "null" {
		t.Fatalf("distinct null values = %v", values)
	}
	if !ObjectsEqual(integer, long) || !ObjectsEqual(integer, NullString()) || ObjectsEqual(integer, "null") {
		t.Fatal("ObjectsEqual did not recognize Java null across typed views")
	}
}

func TestBoxedConcurrentMapUsesValueKeysAndRejectsNull(t *testing.T) {
	m := NewConcurrentHashMap[*Integer, *Long]()
	first := NewInteger(400)
	m.Put(first, BoxLong(1))
	var workers sync.WaitGroup
	for index := 0; index < 32; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			m.Put(NewInteger(400), BoxLong(2))
			m.Get(NewInteger(400))
		}()
	}
	workers.Wait()
	if m.Size() != 1 || m.KeySet()[0] != first || !m.ContainsValue(NewLong(2)) || m.ContainsKey(NewLong(400)) {
		t.Fatal("concurrent map lost wrapper key semantics")
	}
	for _, action := range []func(){
		func() { m.Put(nil, BoxLong(1)) },
		func() { m.Put(first, nil) },
		func() { m.Get((*Integer)(nil)) },
		func() { m.ContainsKey(nil) },
		func() { m.ContainsValue((*Long)(nil)) },
		func() { m.Remove(nil) },
	} {
		expectBoxedCollectionException(t, "NullPointerException", action)
	}
	if value := m.Remove(NewInteger(400)); value.LongValue() != 2 || m.Size() != 0 {
		t.Fatal("concurrent map remove did not use value equality")
	}
}

func TestBoxedOptionalNullSemantics(t *testing.T) {
	var value *Integer
	expectBoxedCollectionException(t, "NullPointerException", func() { OptionalOf(value) })
	expectBoxedCollectionException(t, "NullPointerException", func() { OptionalOf[any](value) })
	expectBoxedCollectionException(t, "NullPointerException", func() { OptionalOf(NullString()) })
	if OptionalOfNullable(value).IsPresent() || OptionalOfNullable[any](value).IsPresent() || OptionalOfNullable(NullString()).IsPresent() {
		t.Fatal("ofNullable retained Java null")
	}
	if OptionalMap(OptionalOf(BoxInteger(1)), func(*Integer) *Integer { return nil }).IsPresent() {
		t.Fatal("map retained a typed null result")
	}
	if !OptionalOfNullable("").IsPresent() || !OptionalOf(BoxInteger(0)).IsPresent() {
		t.Fatal("non-null zero value became empty")
	}
}

type boxedCollectionComparable struct {
	execution *Execution
	value     int32
}

func (*boxedCollectionComparable) JavaDynamicTypeID() TypeID { return "test.BoxedCollectionComparable" }
func (value *boxedCollectionComparable) CompareTo(other *boxedCollectionComparable) int32 {
	panic("execution companion was bypassed")
}
func (value *boxedCollectionComparable) CompareToJava2goExecution1(execution *Execution, other *boxedCollectionComparable) int32 {
	value.execution = execution
	return value.value - other.value
}

func TestBoxedNaturalOrderingAndErasedComparable(t *testing.T) {
	items := NewListFrom(NewInteger(3), NewInteger(1), NewInteger(2))
	SortOrdered(items)
	if items.Get(0).IntValue() != 1 || NaturalOrder[*Integer]()(NewInteger(1), NewInteger(2)) >= 0 || ReverseOrder[*Integer]()(NewInteger(1), NewInteger(2)) <= 0 {
		t.Fatal("natural wrapper ordering failed")
	}
	if ComparableCompareTo(NewInteger(1), NewInteger(2)) >= 0 || ComparableCompareTo("a", "b") >= 0 {
		t.Fatal("erased Comparable wrapper or string dispatch failed")
	}
	expectBoxedCollectionException(t, "NullPointerException", func() { ComparableCompareTo((*Integer)(nil), NewInteger(2)) })
	expectBoxedCollectionException(t, "NullPointerException", func() { ComparableCompareTo(NewInteger(1), nil) })
	expectBoxedCollectionException(t, "ClassCastException", func() { ComparableCompareTo(NewInteger(1), NewLong(2)) })
	expectBoxedCollectionException(t, "ClassCastException", func() { ComparableCompareTo(&comparatorVersion{}, &comparatorVersion{}) })
	RegisterJavaType("test.BoxedCollectionComparable", ObjectTypeID, ComparableTypeID)
	execution := NewExecution()
	left, right := &boxedCollectionComparable{value: 1}, &boxedCollectionComparable{value: 2}
	if ComparableCompareToExecution(execution, left, right) >= 0 || left.execution != execution {
		t.Fatal("Comparable dispatch did not forward the execution token")
	}
}

func TestBoxedNaturalOperationsPreserveExecution(t *testing.T) {
	RegisterJavaType("test.BoxedCollectionComparable", ObjectTypeID, ComparableTypeID)
	identity := func(value *boxedCollectionComparable) *boxedCollectionComparable { return value }
	tests := []struct {
		name string
		run  func(*boxedCollectionComparable, *boxedCollectionComparable, *Execution)
	}{
		{"natural", func(a, b *boxedCollectionComparable, execution *Execution) {
			NaturalOrder[*boxedCollectionComparable](execution)(a, b)
		}},
		{"reverse", func(a, b *boxedCollectionComparable, execution *Execution) {
			ReverseOrder[*boxedCollectionComparable](execution)(a, b)
		}},
		{"comparing", func(a, b *boxedCollectionComparable, execution *Execution) {
			ComparatorComparing(identity, execution)(a, b)
		}},
		{"thenComparing", func(a, b *boxedCollectionComparable, execution *Execution) {
			tied := Comparator[*boxedCollectionComparable](func(a, b *boxedCollectionComparable) int32 { return 0 })
			ComparatorThenComparingKey(tied, identity, execution)(a, b)
		}},
		{"sort", func(a, b *boxedCollectionComparable, execution *Execution) { SortOrdered(NewListFrom(b, a), execution) }},
		{"max", func(a, b *boxedCollectionComparable, execution *Execution) { MaxOrdered(NewListFrom(b, a), execution) }},
		{"min", func(a, b *boxedCollectionComparable, execution *Execution) { MinOrdered(NewListFrom(b, a), execution) }},
		{"sortNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			SortWith(NewListFrom(b, a), nil, execution)
		}},
		{"maxNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			MaxWith(NewListFrom(b, a), nil, execution)
		}},
		{"minNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			MinWith(NewListFrom(b, a), nil, execution)
		}},
		{"array", func(a, b *boxedCollectionComparable, execution *Execution) {
			SortArray(ReferenceArrayLiteral("test.BoxedCollectionComparable", b, a), execution)
		}},
		{"arrayNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			SortArrayWith[*boxedCollectionComparable](ReferenceArrayLiteral("test.BoxedCollectionComparable", b, a), nil, execution)
		}},
		{"streamSort", func(a, b *boxedCollectionComparable, execution *Execution) { StreamSorted(NewStream(b, a), execution) }},
		{"streamMin", func(a, b *boxedCollectionComparable, execution *Execution) { StreamMin(NewStream(b, a), execution) }},
		{"streamMax", func(a, b *boxedCollectionComparable, execution *Execution) { StreamMax(NewStream(b, a), execution) }},
		{"streamSortNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			StreamSortedWith(NewStream(b, a), nil, execution)
		}},
		{"streamMinNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			StreamMinWith(NewStream(b, a), nil, execution)
		}},
		{"streamMaxNullComparator", func(a, b *boxedCollectionComparable, execution *Execution) {
			StreamMaxWith(NewStream(b, a), nil, execution)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			execution := NewExecution()
			a, b := &boxedCollectionComparable{value: 1}, &boxedCollectionComparable{value: 2}
			test.run(a, b, execution)
			if (a.execution == nil && b.execution == nil) || (a.execution != nil && a.execution != execution) || (b.execution != nil && b.execution != execution) {
				t.Fatal("natural comparison did not retain the caller's execution")
			}
		})
	}
}

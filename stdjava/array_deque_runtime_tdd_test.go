package stdjava

import "testing"

// These are the original five contract cases with canonical Java values.
// The physical class is erased; consumers perform logical E checkcasts.
func TestArrayDequeRuntimeContract224(t *testing.T) {
	t.Run("ordering_queue_stack_and_growth", func(t *testing.T) {
		a, b, c, z, p := JavaStringFromHostUTF8("a"), JavaStringFromHostUTF8("b"), JavaStringFromHostUTF8("c"), JavaStringFromHostUTF8("z"), JavaStringFromHostUTF8("p")
		d := NewArrayDequeWithCapacity[*JavaString](-17)
		if !d.IsEmpty() {
			t.Fatal("negative capacity must construct an empty deque")
		}
		d.AddLast(b)
		d.AddFirst(a)
		d.OfferLast(c)
		d.OfferFirst(z)
		if d.String() != "[z, a, b, c]" || d.GetFirst() != z || d.GetLast() != c {
			t.Fatal(d.String())
		}
		if d.RemoveFirst() != z || d.RemoveLast() != c {
			t.Fatal("end order")
		}
		d.Push(p)
		if d.Pop() != p || d.Remove() != a || d.Poll() != b || !d.IsEmpty() {
			t.Fatal("queue/stack aliases")
		}
		grow := NewArrayDequeWithCapacity[*Integer](1)
		for i := int32(0); i < 96; i++ {
			grow.AddLast(BoxInteger(i))
			if i%3 == 0 {
				grow.RemoveFirst()
			}
		}
		first := ObjectView[*Integer](grow.GetFirst(), IntegerTypeID)
		last := ObjectView[*Integer](grow.GetLast(), IntegerTypeID)
		if grow.Size() != 64 || first.IntValue() != 32 || last.IntValue() != 95 {
			t.Fatal("growth lost order")
		}
	})
	t.Run("empty_null_and_nominal_guards", func(t *testing.T) {
		d := NewArrayDeque[any]()
		if d.PollFirst() != nil || d.PollLast() != nil || d.Peek() != nil || d.PeekLast() != nil {
			t.Fatal("empty nullable operations")
		}
		arrayDeque224Panic(t, "java.util.NoSuchElementException", func() { d.Remove() })
		arrayDeque224Panic(t, "java.util.NoSuchElementException", func() { d.GetLast() })
		arrayDeque224Panic(t, "java.lang.NullPointerException", func() { d.AddFirst(nil) })
		arrayDeque224Panic(t, "java.lang.NullPointerException", func() { d.OfferLast(nil) })
		arrayDeque224Panic(t, "java.lang.NullPointerException", func() { NewArrayDequeFromCollection[any](nil) })
		if d.Contains(nil) || d.RemoveFirstOccurrence(nil) || !d.IsEmpty() {
			t.Fatal("null query mutated empty deque")
		}
		for _, id := range []TypeID{"java.util.ArrayDeque", "java.util.Deque", "java.util.Collection", "java.lang.Iterable"} {
			if !ObjectInstanceOf(d, id) {
				t.Fatalf("native carrier lost %s", id)
			}
		}
		if ObjectInstanceOf(d, "java.util.List") {
			t.Fatal("deque became List")
		}
		if ObjectInstanceOf(&arrayDeque224ShapeOnly{}, "java.util.Deque") {
			t.Fatal("shape-only Deque admission")
		}
	})
	t.Run("collection_copy_occurrences_and_partial_failure", func(t *testing.T) {
		e := NewExecution()
		values := NewArrayList[any]()
		for _, v := range []string{"a", "b", "a", "c"} {
			values.Add(JavaStringFromHostUTF8(v))
		}
		d := NewArrayDequeFromCollection[any](values, e)
		if !d.RemoveFirstOccurrence(JavaStringFromHostUTF8("a")) || !d.RemoveLastOccurrence(JavaStringFromHostUTF8("a")) || d.String() != "[b, c]" {
			t.Fatal(d.String())
		}
		copied := NewArrayDequeFromCollection[any](d, e)
		d.AddLast(JavaStringFromHostUTF8("original-only"))
		if copied.String() != "[b, c]" {
			t.Fatal("copy shares storage")
		}
		clone := d.Clone()
		d.Clear()
		if clone.String() != "[b, c, original-only]" || !d.IsEmpty() {
			t.Fatal("clone/clear isolation")
		}
		partial := NewArrayList[any]()
		partial.Add(JavaStringFromHostUTF8("kept"))
		partial.Add(nil)
		partial.Add(JavaStringFromHostUTF8("unreached"))
		arrayDeque224Panic(t, "java.lang.NullPointerException", func() { d.AddAll(partial, e) })
		if d.String() != "[kept]" {
			t.Fatal("addAll lost JDK partial mutation")
		}
	})
	t.Run("iterators_and_alias_identity", func(t *testing.T) {
		e := NewExecution()
		a, b, c := JavaStringFromHostUTF8("a"), JavaStringFromHostUTF8("b"), JavaStringFromHostUTF8("c")
		d := NewArrayDeque[any]()
		d.AddLast(b)
		d.AddLast(c)
		it := d.IteratorJava2goExecution(e)
		arrayDeque224Panic(t, "java.lang.IllegalStateException", func() { IteratorRemoveExecution(e, it) })
		if IteratorNextExecution(e, it) != b {
			t.Fatal("iterator order")
		}
		IteratorRemoveExecution(e, it)
		if d.String() != "[c]" {
			t.Fatal("iterator remove lost backing mutation")
		}
		arrayDeque224Panic(t, "java.lang.IllegalStateException", func() { IteratorRemoveExecution(e, it) })
		d.AddFirst(a)
		desc := d.DescendingIteratorJava2goExecution(e)
		var observed []any
		for IteratorHasNextExecution(e, desc) {
			observed = append(observed, IteratorNextExecution(e, desc))
		}
		if len(observed) != 2 || observed[0] != c || observed[1] != a {
			t.Fatal("descending order")
		}
		collection := ObjectView[JavaIterable](d, "java.util.Collection")
		CollectionAddExecution(e, collection, JavaStringFromHostUTF8("tail"))
		if !JavaReferenceEqual(d, collection) || d.String() != "[a, c, tail]" {
			t.Fatal("collection alias copied or lost receiver")
		}
	})
	t.Run("raw_store_checks_on_read_after_removal", func(t *testing.T) {
		e := NewExecution()
		d := NewArrayDeque[*JavaString]()
		CollectionAddExecution(e, d, BoxInteger(7))
		if d.Size() != 1 {
			t.Fatal("raw write rejected a legal Object slot")
		}
		arrayDeque224Panic(t, "java.lang.ClassCastException", func() { ObjectView[*JavaString](d.RemoveFirst(), StringTypeID) })
		if !d.IsEmpty() {
			t.Fatal("typed read checkcast undid removal")
		}
	})
}

func arrayDeque224Panic(t *testing.T, id TypeID, action func()) {
	t.Helper()
	defer func() {
		value := recover()
		if value == nil || !ObjectInstanceOf(value, id) {
			t.Fatalf("expected %s, got %T %v", id, value, value)
		}
	}()
	action()
}

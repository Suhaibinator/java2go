package stdjava

import "testing"

func sublistPanic(t *testing.T, want TypeID, fn func()) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil || !ObjectInstanceOf(got, want) {
			t.Fatalf("panic = %T %v, want %s", got, got, want)
		}
	}()
	fn()
}
func TestCampaignSubListLiveView(t *testing.T) {
	execution := NewExecution()
	base := NewListFrom("a", "b", "c", "d", "e")
	view := base.SubList(1, 4)
	nested := view.SubList(1, 2)
	sibling := view.SubList(0, 1)
	base.Set(2, "C")
	if view.String() != "[b, C, d]" || nested.String() != "[C]" {
		t.Fatal("nonstructural parent writes are not live")
	}
	nested.Add("x")
	if base.String() != "[a, b, C, x, d, e]" || view.String() != "[b, C, x, d]" || nested.Size() != 2 {
		t.Fatal("nested insertion failed")
	}
	sublistPanic(t, "java.util.ConcurrentModificationException", func() { sibling.Size() })
	cursor := nested.IteratorJava2goExecution(execution)
	cursor.NextJava2goExecution(execution)
	cursor.(interface{ IteratorRemoveJava2goExecution(*Execution) }).IteratorRemoveJava2goExecution(execution)
	if base.String() != "[a, b, x, d, e]" || view.Size() != 3 || nested.Size() != 1 {
		t.Fatal("nested iterator removal failed")
	}
	nested.Clear()
	if base.String() != "[a, b, d, e]" || view.String() != "[b, d]" || nested.Size() != 0 {
		t.Fatal("nested clear failed")
	}
	empty := view.SubList(1, 1)
	empty.Add("z")
	CollectionAddExecution(execution, empty, int32(7))
	if CollectionListGetExecution(execution, base, 3) != int32(7) {
		t.Fatal("raw insertion did not update original backing")
	}
	view.Clear()
	if base.String() != "[a, e]" {
		t.Fatal(base.String())
	}
	sublistPanic(t, "java.util.ConcurrentModificationException", func() { empty.Size() })
}
func TestCampaignSubListInvalidationAndOrdering(t *testing.T) {
	execution := NewExecution()
	base := NewListFrom("a", "b", "c")
	stale := base.SubList(0, 1)
	cursor := stale.IteratorJava2goExecution(execution)
	base.Add("d")
	sublistPanic(t, "java.util.ConcurrentModificationException", func() { stale.Size() })
	sublistPanic(t, "java.lang.IndexOutOfBoundsException", func() { stale.Get(-1) })
	sublistPanic(t, "java.lang.IndexOutOfBoundsException", func() { stale.SubList(-1, 9) })
	sublistPanic(t, "java.util.ConcurrentModificationException", func() { stale.Get(0) })
	sublistPanic(t, "java.util.ConcurrentModificationException", func() { cursor.NextJava2goExecution(execution) })
	sublistPanic(t, "java.lang.IndexOutOfBoundsException", func() { base.SubList(-1, 0) })
	sublistPanic(t, "java.lang.IndexOutOfBoundsException", func() { base.SubList(0, 99) })
	sublistPanic(t, "java.lang.IllegalArgumentException", func() { base.SubList(2, 1) })
	zero := base.SubList(1, 1)
	rootCursor := base.IteratorJava2goExecution(execution)
	zero.Clear()
	sublistPanic(t, "java.util.ConcurrentModificationException", func() { rootCursor.NextJava2goExecution(execution) })
}
func TestCampaignSubListFixedSortAndRetention(t *testing.T) {
	fixed := AsList("p", "q", "r")
	view := fixed.SubList(1, 3)
	view.Set(0, "Q")
	if fixed.Get(1) != "Q" {
		t.Fatal("fixed view lost alias")
	}
	sublistPanic(t, "java.lang.UnsupportedOperationException", func() { view.Add("s") })
	sublistPanic(t, "java.lang.UnsupportedOperationException", func() { view.Clear() })
	base := NewListFrom("f", "d", "c", "e", "g")
	sorted := base.SubList(1, 4)
	SortOrdered(sorted)
	ReverseList(sorted)
	if base.String() != "[f, e, d, c, g]" {
		t.Fatal(base.String())
	}
	pointers := NewListFrom(new(int), new(int), new(int), new(int))
	backing := pointers.elements
	pointers.SubList(1, 3).Clear()
	if backing[2] != nil || backing[3] != nil {
		t.Fatal("removed references retained beyond live length")
	}
}

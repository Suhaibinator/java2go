package stdjava

import (
	"reflect"
	"testing"
)

type joinTextProbe struct {
	execution *Execution
	calls     int
	text      string
	effect    func()
}

func (*joinTextProbe) JavaDynamicTypeID() TypeID { return "join.test.Text" }
func (p *joinTextProbe) StringJava2goExecution(execution *Execution) string {
	if execution != p.execution {
		panic("changed execution")
	}
	p.calls++
	if p.effect != nil {
		p.effect()
	}
	return p.text
}
func joinExpectPanic(t *testing.T, name string, callback func()) {
	t.Helper()
	defer func() {
		value := recover()
		if value == nil {
			t.Errorf("expected %s, returned normally", name)
			return
		}
		throwable, ok := value.(Throwable)
		if !ok || throwable.ThrowableTypeName() != name {
			t.Errorf("expected %s, got %T %v", name, value, value)
		}
	}()
	callback()
}
func TestStringJoinLiveListAndStructuralRevision(t *testing.T) {
	RegisterJavaType("join.test.Text", ObjectTypeID, CharSequenceTypeID)
	execution := NewExecution()
	list := NewList[any]()
	first := &joinTextProbe{execution: execution, text: "a", effect: func() { list.Set(1, "changed") }}
	list.Add(first)
	list.Add("old")
	if got := StringJoinIterableExecution(execution, "|", list); got != "a|changed" {
		t.Fatal(got)
	}
	if first.calls != 1 {
		t.Fatalf("conversion count %d", first.calls)
	}
	first.effect = func() { list.Add("extra"); list.RemoveAt(list.Size() - 1) }
	joinExpectPanic(t, "ConcurrentModificationException", func() { StringJoinIterableExecution(execution, "|", list) })
	if list.Size() != 2 || first.calls != 2 {
		t.Fatalf("size/calls %d/%d", list.Size(), first.calls)
	}
	// Like ArrayList.Itr.hasNext, reaching the live end does not perform a
	// revision check. Mutation from the final conversion therefore can complete.
	singleton := NewList[any]()
	last := &joinTextProbe{execution: execution, text: "last", effect: func() { singleton.Add("x"); singleton.RemoveAt(1) }}
	singleton.Add(last)
	if got := StringJoinIterableExecution(execution, ",", singleton); got != "last" {
		t.Fatal(got)
	}
}
func TestStringJoinValidationAndDelayedNullText(t *testing.T) {
	RegisterJavaType("join.test.Text", ObjectTypeID, CharSequenceTypeID)
	execution := NewExecution()
	delimiter := &joinTextProbe{execution: execution, text: "|"}
	joinExpectPanic(t, "NullPointerException", func() { StringJoinIterableExecution(execution, delimiter, nil) })
	if delimiter.calls != 0 {
		t.Fatal("Iterable converted delimiter before null input validation")
	}
	joinExpectPanic(t, "NullPointerException", func() { StringJoinArrayExecution(execution, delimiter, nil) })
	if delimiter.calls != 1 {
		t.Fatal("array did not convert delimiter before null array check")
	}
	a := &joinTextProbe{execution: execution, text: NullString()}
	b := &joinTextProbe{execution: execution, text: "b"}
	joinExpectPanic(t, "NullPointerException", func() { StringJoinIterableExecution(execution, delimiter, NewListFrom[any](a, b)) })
	if a.calls != 1 || b.calls != 1 {
		t.Fatal("join validated null text before all conversions")
	}
	delimiter.text = NullString()
	if got := StringJoinIterableExecution(execution, delimiter, NewListFrom("one")); got != "one" {
		t.Fatal(got)
	}
	if got := StringJoinIterableExecution(execution, delimiter, NewList[string]()); got != "" {
		t.Fatal(got)
	}
	joinExpectPanic(t, "NullPointerException", func() { StringJoinIterableExecution(execution, delimiter, NewListFrom("a", "b")) })
}
func TestStringJoinConversionAbruptIdentityAndArrayReads(t *testing.T) {
	RegisterJavaType("join.test.Text", ObjectTypeID, CharSequenceTypeID)
	execution := NewExecution()
	failure := &struct{ marker bool }{true}
	calls := 0
	bad := &joinTextProbe{execution: execution, text: "bad", effect: func() { panic(failure) }}
	later := &joinTextProbe{execution: execution, text: "later", effect: func() { calls++ }}
	func() {
		defer func() {
			if got := recover(); got != failure {
				t.Errorf("changed failure identity: %v", got)
			}
		}()
		StringJoinValuesExecution(execution, ",", bad, later)
	}()
	if calls != 0 || bad.calls != 1 {
		t.Fatal("conversion continued after abrupt completion")
	}
	array := NewReferenceArray(2, CharSequenceTypeID)
	first := &joinTextProbe{execution: execution, text: "first", effect: func() { ReferenceArraySet(array, 1, "replacement") }}
	ReferenceArraySet(array, 0, first)
	ReferenceArraySet(array, 1, "old")
	if got := StringJoinArrayExecution(execution, "/", array); got != "first/replacement" {
		t.Fatal(got)
	}
}

type joinRawIterator struct {
	events *[]string
	value  any
	used   bool
}

func (i *joinRawIterator) HasNextJava2goExecution(*Execution) bool {
	*i.events = append(*i.events, "hasNext")
	return !i.used
}
func (i *joinRawIterator) NextJava2goExecution(*Execution) any {
	*i.events = append(*i.events, "next")
	i.used = true
	return i.value
}

type joinRawIterable struct{ iterator *joinRawIterator }

func (i *joinRawIterable) IteratorJava2goExecution(*Execution) JavaIterator {
	*i.iterator.events = append(*i.iterator.events, "iterator")
	return i.iterator
}
func TestStringJoinConsumerCastAfterNext(t *testing.T) {
	events := []string{}
	iterator := &joinRawIterator{events: &events, value: BoxInteger(1)}
	joinExpectPanic(t, "ClassCastException", func() { StringJoinIterableExecution(NewExecution(), ",", &joinRawIterable{iterator}) })
	if !reflect.DeepEqual(events, []string{"iterator", "hasNext", "next"}) {
		t.Fatal(events)
	}
}
func TestCollectionIteratorLiveMapAndExhaustion(t *testing.T) {
	execution := NewExecution()
	values := NewMap[string, string]()
	values.Put("first", "a")
	values.Put("second", "old")
	iterator := IterableIteratorExecution(execution, MapValuesView(values).(JavaIterable))
	if got := IteratorNextExecution(execution, iterator); got != "a" {
		t.Fatal(got)
	}
	values.Put("second", "changed")
	if got := IteratorNextExecution(execution, iterator); got != "changed" {
		t.Fatal(got)
	}
	joinExpectPanic(t, "NoSuchElementException", func() { IteratorNextExecution(execution, iterator) })
	iterator = IterableIteratorExecution(execution, MapValuesView(values).(JavaIterable))
	values.Put("extra", "x")
	values.Remove("extra")
	joinExpectPanic(t, "ConcurrentModificationException", func() { IteratorNextExecution(execution, iterator) })
}

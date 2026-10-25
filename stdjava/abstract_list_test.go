package stdjava

import "testing"

type abstractListControl struct {
	info        *ObjectInfo
	base        *List[any]
	data        []any
	seen        []*Execution
	gets, sizes int
	structural  bool
	getHook     func(*Execution, int32) any
}

func (s *abstractListControl) JavaObjectInfo() *ObjectInfo { return s.info }
func (s *abstractListControl) AbstractListStateJava2go() *AbstractListState {
	return s.base.AbstractListStateJava2go()
}
func (s *abstractListControl) IteratorJava2goExecution(e *Execution) JavaIterator {
	return AbstractListIteratorExecution(e, s)
}
func (s *abstractListControl) CollectionSizeJava2goExecution(e *Execution) int32 {
	s.seen = append(s.seen, e)
	s.sizes++
	return int32(len(s.data))
}
func (s *abstractListControl) CollectionListGetJava2goExecution(e *Execution, index int32) any {
	s.seen = append(s.seen, e)
	s.gets++
	if s.getHook != nil {
		return s.getHook(e, index)
	}
	if index < 0 || int(index) >= len(s.data) {
		panic(NewJavaIndexOutOfBoundsExceptionMessage(uriRuntimeText("sentinel bounds")))
	}
	return s.data[index]
}
func (s *abstractListControl) CollectionListSetJava2goExecution(e *Execution, index int32, v any) any {
	old := s.CollectionListGetJava2goExecution(e, index)
	s.data[index] = v
	return old
}
func (s *abstractListControl) CollectionListAddAtJava2goExecution(e *Execution, index int32, v any) {
	s.seen = append(s.seen, e)
	if index < 0 || int(index) > len(s.data) {
		panic(NewIndexOutOfBoundsException("add bounds"))
	}
	s.data = append(s.data, nil)
	copy(s.data[index+1:], s.data[index:])
	s.data[index] = v
	if s.structural {
		s.AbstractListStateJava2go().ModCount++
	}
}
func (s *abstractListControl) CollectionListRemoveAtJava2goExecution(e *Execution, index int32) any {
	old := s.CollectionListGetJava2goExecution(e, index)
	copy(s.data[index:], s.data[index+1:])
	s.data[len(s.data)-1] = nil
	s.data = s.data[:len(s.data)-1]
	if s.structural {
		s.AbstractListStateJava2go().ModCount++
	}
	return old
}
func (s *abstractListControl) CollectionAddJava2goExecution(e *Execution, v any) bool {
	return AbstractListAddExecution(e, s, v)
}
func newAbstractListControl(id TypeID, structural bool, values ...any) *abstractListControl {
	RegisterJavaType(id, "java.util.AbstractList")
	RegisterJavaSourceType(id)
	s := &abstractListControl{base: NewAbstractListProjection[any](), data: values, structural: structural}
	s.info = NewObjectInfo(id, func(want TypeID) any {
		switch want {
		case id, ObjectTypeID, "java.util.Collection", "java.lang.Iterable":
			return s
		case "java.util.AbstractList", "java.util.List":
			return s.base
		}
		return nil
	})
	BindAbstractListSourceReference(s.base, s.info)
	return s
}
func abstractListFailure(t *testing.T, kind string, work func()) any {
	t.Helper()
	var f any
	func() { defer func() { f = recover() }(); work() }()
	if !CaughtAs(f, kind) {
		t.Fatalf("failure=%T %v want %s", f, f, kind)
	}
	return f
}
func TestAbstractListVirtualGetSizeExecutionAndAliases(t *testing.T) {
	a, b := new(int), new(int)
	s := newAbstractListControl("abstractList.control.Aliases", true, a, b)
	e := NewExecution()
	projection := s.base
	if !JavaReferenceEqual(s, projection) || ObjectView[*List[any]](s, "java.util.List") != projection || ObjectView[JavaIterable](projection, "java.util.Collection") != s || monitorIdentityFor(s) != monitorIdentityFor(projection) {
		t.Fatal("alias identity split")
	}
	cursor := IterableIteratorExecution(e, projection)
	if IteratorNextExecution(e, cursor) != a || s.gets != 1 || s.sizes != 0 {
		t.Fatal("next must invoke get once and not size")
	}
	if CollectionListSetExecution(e, projection, 1, a) != b || CollectionListGetExecution(e, projection, 1) != a {
		t.Fatal("virtual set/get bypassed owner")
	}
	if !CollectionAddExecution(e, projection, b) || len(s.data) != 3 {
		t.Fatal("default add omitted indexed source override")
	}
	abstractListFailure(t, "ConcurrentModificationException", func() { IteratorNextExecution(e, cursor) })
	for _, actual := range s.seen {
		if actual != e {
			t.Fatal("callback lost caller execution")
		}
	}
	before := len(s.seen)
	abstractListFailure(t, "IllegalArgumentException", func() { CollectionListGetExecution(nil, projection, 0) })
	if len(s.seen) != before {
		t.Fatal("nil execution entered source callback")
	}
	abstractListFailure(t, "NullPointerException", func() { CollectionListGetExecution(e, nil, 0) })
	if ObjectInstanceOf(s, "java.util.RandomAccess") || ObjectInstanceOf(struct{ Get func(int32) any }{}, "java.util.RandomAccess") {
		t.Fatal("marker acquired through Go shape")
	}
}
func TestAbstractListIteratorWrapperGenerationAndCleanup(t *testing.T) {
	a, b, c := new(int), new(int), new(int)
	s := newAbstractListControl("abstractList.control.Wrapper", false, a, b)
	e := NewExecution()
	it := IterableIteratorExecution(e, s)
	abstractListFailure(t, "IllegalStateException", func() { IteratorRemoveExecution(e, it) })
	if IteratorNextExecution(e, it) != a {
		t.Fatal("first")
	}
	s.data = append(s.data, c)
	if IteratorNextExecution(e, it) != b {
		t.Fatal("delegate edit falsely invalidated wrapper generation")
	}
	IteratorRemoveExecution(e, it)
	abstractListFailure(t, "IllegalStateException", func() { IteratorRemoveExecution(e, it) })
	if IteratorNextExecution(e, it) != c {
		t.Fatal("remove cursor not adjusted")
	}
	f := abstractListFailure(t, "NoSuchElementException", func() { IteratorNextExecution(e, it) })
	if !CaughtAs(GetCause(f), "IndexOutOfBoundsException") {
		t.Fatal("exhaustion lost source bounds cause")
	}
	if len(s.data) != 2 || s.data[0] != a || s.data[1] != c || s.AbstractListStateJava2go().ModCount != 0 {
		t.Fatal("wrapper generation borrowed delegate changes")
	}
}

func (s *abstractListControl) CollectionListIteratorJava2goExecution(e *Execution, index int32) JavaListIterator {
	return AbstractListListIteratorExecution(e, s, index)
}

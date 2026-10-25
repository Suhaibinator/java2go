package stdjava

import "testing"

type abstractListEqualityEffect struct {
	calls int
	want  *Execution
}

func (v *abstractListEqualityEffect) EqualsJava2goExecution(e *Execution, other any) bool {
	if e != v.want {
		panic("lost equality execution")
	}
	v.calls++
	return false
}
func TestAbstractListEqualityInvokesElementEvenOnIdentity(t *testing.T) {
	e := NewExecution()
	v := &abstractListEqualityEffect{want: e}
	left := newAbstractListControl("abstractList.control.Equals", false, v)
	right := NewArrayList[any]()
	right.Add(v)
	if AbstractListEqualsExecution(e, left, right) || v.calls != 1 {
		t.Fatal("AbstractList.equals must invoke element.equals even on shared element identity")
	}
	if !ObjectEqualsExecution(e, right, left) || v.calls != 1 {
		t.Fatal("ArrayList's Objects.equals identity shortcut changed")
	}
}
func TestAbstractListBoundsCauseAndCallbackRecovery(t *testing.T) {
	e := NewExecution()
	a := new(int)
	s := newAbstractListControl("abstractList.control.Abrupt", true, a)
	cursor := IterableIteratorExecution(e, s)
	sentinel := NewJavaIllegalArgumentExceptionMessage(uriRuntimeText("sentinel"))
	s.getHook = func(actual *Execution, index int32) any {
		if actual != e {
			t.Fatal("execution changed")
		}
		panic(sentinel)
	}
	if f := abstractListFailure(t, "IllegalArgumentException", func() { IteratorNextExecution(e, cursor) }); !JavaReferenceEqual(f, sentinel) {
		t.Fatal("callback exception replaced")
	}
	s.getHook = nil
	if IteratorNextExecution(e, cursor) != a {
		t.Fatal("abrupt get consumed cursor position")
	}
	abstractListFailure(t, "NoSuchElementException", func() { IteratorNextExecution(e, cursor) })
	s.data = append(s.data, a)
	if IteratorNextExecution(e, cursor) != a {
		t.Fatal("exhaustion consumed cursor position")
	}
	s.getHook = func(*Execution, int32) any {
		s.AbstractListStateJava2go().ModCount++
		panic(NewIndexOutOfBoundsException("late"))
	}
	abstractListFailure(t, "ConcurrentModificationException", func() { IteratorNextExecution(e, cursor) })
	before := s.gets
	abstractListFailure(t, "UnsupportedOperationException", func() { AbstractListSetExecution(e, s, -1, nil) })
	if s.gets != before {
		t.Fatal("default set inspected index before throwing UOE")
	}
}
func TestAbstractListSubViewIndependentGenerationAndMarker(t *testing.T) {
	e := NewExecution()
	a, b, c := new(int), new(int), new(int)
	s := newAbstractListControl("abstractList.control.View", true, a, b, c)
	view := AbstractListSubListTypedExecution[any](e, s, 1, 3)
	if ObjectInstanceOf(view, "java.util.RandomAccess") || !ObjectInstanceOf(view, "java.util.AbstractList") || JavaReferenceEqual(view, s) {
		t.Fatal("non-marker view or identity misclassified")
	}
	cursor := IterableIteratorExecution(e, view)
	s.AbstractListStateJava2go().ModCount++
	if !IteratorHasNextExecution(e, cursor) {
		t.Fatal("sublist cursor hasNext unexpectedly checked generation")
	}
	abstractListFailure(t, "IndexOutOfBoundsException", func() { CollectionListGetExecution(e, view, -1) })
	abstractListFailure(t, "ConcurrentModificationException", func() { CollectionSizeExecution(e, view) })
	abstractListFailure(t, "ConcurrentModificationException", func() { IteratorNextExecution(e, cursor) })
	marker := newAbstractListControl("abstractList.control.Marker", false, a, b, c)
	RegisterJavaType(marker.info.DynamicType(), "java.util.AbstractList", "java.util.RandomAccess")
	marked := AbstractListSubListTypedExecution[any](e, marker, 1, 3)
	if !ObjectInstanceOf(marked, "java.util.RandomAccess") {
		t.Fatal("declared RandomAccess marker not retained by sublist")
	}
	marker.data = append([]any{c}, marker.data...)
	if CollectionListGetExecution(e, marked, 0) != a || CollectionSizeExecution(e, marked) != 2 {
		t.Fatal("view borrowed delegate generation or forgot fixed offset/size")
	}
}
func TestAbstractListBindingRejectsFalseProvidersBeforeMutation(t *testing.T) {
	for _, name := range []string{"native-array", "foreign-parent", "unregistered", "wrong-list", "wrong-source"} {
		t.Run(name, func(t *testing.T) {
			base := NewAbstractListProjection[any]()
			if name == "native-array" {
				base = NewArrayList[any]()
			}
			id := TypeID("abstractList.control.Binding." + name)
			parent := TypeID("java.util.AbstractList")
			if name == "foreign-parent" {
				parent = ObjectTypeID
			}
			RegisterJavaType(id, parent)
			if name != "unregistered" {
				RegisterJavaSourceType(id)
			}
			source := &abstractListControl{base: base}
			other := NewAbstractListProjection[any]()
			source.info = NewObjectInfo(id, func(want TypeID) any {
				switch want {
				case id, ObjectTypeID, CollectionTypeID, IterableTypeID:
					if name == "wrong-source" {
						return &arrayListSourceIdentityOnly{info: source.info}
					}
					return source
				case "java.util.List":
					if name == "wrong-list" {
						return other
					}
					return base
				case "java.util.AbstractList":
					return base
				}
				return nil
			})
			abstractListFailure(t, "UnsupportedOperationException", func() { BindAbstractListSourceReference(base, source.info) })
			if base.sourceInfo != nil || base.abstractState != nil && base.abstractState.owner != nil {
				t.Fatal("failed provider mutated binding")
			}
		})
	}
	var nilBase *List[any]
	abstractListFailure(t, "NullPointerException", func() { BindAbstractListSourceReference(nilBase, nil) })
	unbound := NewAbstractListProjection[any]()
	abstractListFailure(t, "UnsupportedOperationException", func() { CollectionListGetExecution(NewExecution(), unbound, 0) })
}

func TestAbstractListArraySizeEstimateAndPrefixEffects(t *testing.T) {
	e := NewExecution()
	a, b, c := uriRuntimeText("a"), uriRuntimeText("b"), uriRuntimeText("c")
	for _, targetLength := range []int32{0, 2, 4} {
		source := newAbstractListControl("abstractList.control.ArrayShrink", false, a, b, c)
		target := NewReferenceArray(targetLength, StringTypeID)
		source.getHook = func(_ *Execution, index int32) any {
			if index != 0 {
				t.Fatal("shrunken iterator fetched past first")
			}
			source.data = source.data[:1]
			return a
		}
		result := AbstractListToArrayIntoExecution(e, source, target)
		if ReferenceArrayLength(result) != max(targetLength, 1) || ReferenceArrayGet[any](result, 0, ObjectTypeID) != a {
			t.Fatal("size hint did not shrink to actual iterator or preserve target")
		}
		if targetLength > 0 && result != target {
			t.Fatal("oversized estimate failed to copy back into fitting input")
		}
		if targetLength > 1 && ReferenceArrayGet[any](result, 1, ObjectTypeID) != nil {
			t.Fatal("input not null-terminated")
		}
	}
	source := newAbstractListControl("abstractList.control.ArrayGrow", false, a)
	source.getHook = func(_ *Execution, index int32) any {
		if index == 0 {
			source.data = append(source.data, b, c)
		}
		return source.data[index]
	}
	result := AbstractListToArrayIntoExecution(e, source, NewReferenceArray(0, StringTypeID))
	if ReferenceArrayLength(result) != 3 || ReferenceArrayGet[any](result, 2, ObjectTypeID) != c {
		t.Fatal("iterator growth lost elements or retained spare capacity")
	}
	source = newAbstractListControl("abstractList.control.ArrayNull", false, a)
	abstractListFailure(t, "NullPointerException", func() { AbstractListToArrayIntoExecution(e, source, nil) })
	if source.sizes != 1 || source.gets != 0 {
		t.Fatal("null array check preceded size or entered iterator")
	}
}

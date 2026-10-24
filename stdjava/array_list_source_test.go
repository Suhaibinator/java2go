package stdjava

import "testing"

type arrayListSourceControl struct {
	info *ObjectInfo
	base *List[any]
	add  func(*Execution, any) bool
}

type arrayListSourceIdentityOnly struct{ info *ObjectInfo }

func (source *arrayListSourceIdentityOnly) JavaObjectInfo() *ObjectInfo { return source.info }

type arrayListSourceWithoutAdd struct {
	info *ObjectInfo
	base *List[any]
}

func (source *arrayListSourceWithoutAdd) JavaObjectInfo() *ObjectInfo { return source.info }
func (source *arrayListSourceWithoutAdd) IteratorJava2goExecution(e *Execution) JavaIterator {
	return source.base.IteratorJava2goExecution(e)
}

type arrayListNativePublicShape struct{}

func (arrayListNativePublicShape) JavaDynamicTypeID() TypeID { return "java.util.ArrayList" }
func (arrayListNativePublicShape) Size() int32               { return 0 }
func (arrayListNativePublicShape) Add(any) bool              { return true }

func (source *arrayListSourceControl) JavaObjectInfo() *ObjectInfo { return source.info }
func (source *arrayListSourceControl) IteratorJava2goExecution(e *Execution) JavaIterator {
	return source.base.IteratorJava2goExecution(e)
}
func (source *arrayListSourceControl) CollectionAddJava2goExecution(e *Execution, value any) bool {
	return source.add(e, value)
}

func newArrayListSourceControl(id TypeID, base *List[any]) *arrayListSourceControl {
	RegisterJavaType(id, "java.util.ArrayList")
	RegisterJavaSourceType(id)
	source := &arrayListSourceControl{base: base}
	source.info = NewObjectInfo(id, func(want TypeID) any {
		switch want {
		case id, ObjectTypeID, "java.util.Collection", "java.lang.Iterable":
			return source
		case "java.util.ArrayList", "java.util.List":
			return base
		}
		return nil
	})
	source.add = func(e *Execution, value any) bool { return ArrayListBaseAddExecution(e, base, value) }
	return source
}

func arrayListSourcePanic(t *testing.T, name string, action func()) any {
	t.Helper()
	var failure any
	func() {
		defer func() { failure = recover() }()
		action()
	}()
	if failure == nil || !CaughtAs(failure, name) {
		t.Fatalf("panic = %T, want %s", failure, name)
	}
	return failure
}

func TestArrayListSourceVirtualAddUsesCurrentExecutionAndNativeSuper(t *testing.T) {
	base := NewArrayList[any]()
	source := newArrayListSourceControl("arrayList.source.CurrentExecution", base)
	first, second := NewExecution(), NewExecution()
	var seen []*Execution
	source.add = func(e *Execution, value any) bool {
		seen = append(seen, e)
		if e != first && e != second {
			t.Fatal("virtual add received a constructing or fresh execution")
		}
		return ArrayListBaseAddExecution(e, base, value)
	}
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	marker := new(int)
	if !CollectionAddExecution(first, source, marker) || !CollectionAddExecution(second, base, nil) {
		t.Fatal("add result")
	}
	if len(seen) != 2 || seen[0] != first || seen[1] != second || base.Size() != 2 || base.Get(0) != marker || base.Get(1) != nil {
		t.Fatal("virtual add repeated, recursed or copied storage")
	}
	ArrayListBaseAddExecution(second, base, marker)
	if len(seen) != 2 || ArrayListBaseSizeExecution(second, base) != 3 || ArrayListBaseGetExecution(second, base, 2) != marker {
		t.Fatal("nonvirtual native service consulted virtual add")
	}
	arrayListSourcePanic(t, "IllegalArgumentException", func() { CollectionAddExecution(nil, base, marker) })
	if len(seen) != 2 || base.Size() != 3 {
		t.Fatal("nil execution reached source body or mutated storage")
	}
	// The pre-existing direct native Add entry keeps its own storage contract.
	base.Add(marker)
	if len(seen) != 2 || base.Size() != 4 {
		t.Fatal("ordinary native Add was reinterpreted")
	}
}

func TestArrayListSourceAliasesShareIdentityMonitorAndNativeIterator(t *testing.T) {
	base := NewArrayList[any]()
	source := newArrayListSourceControl("arrayList.source.Identity", base)
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	e := NewExecution()
	marker := new(int)
	CollectionAddExecution(e, source, marker)
	if ObjectView[*List[any]](source, "java.util.ArrayList") != base || ObjectView[*List[any]](source, "java.util.List") != base || ObjectView[JavaIterable](base, "java.util.Collection") != source || ObjectView[JavaIterable](base, "java.lang.Iterable") != source {
		t.Fatal("provider changed a native projection or whole source alias")
	}
	if !JavaReferenceEqual(base, source) || monitorIdentityFor(base) != monitorIdentityFor(source) || SystemIdentityHashCode(base) != SystemIdentityHashCode(source) || ObjectGetClass(base) != ClassLiteral(source.info.DynamicType()) {
		t.Fatal("source and native aliases split object identity")
	}
	if base.JavaDynamicTypeID() != "java.util.ArrayList" || base.JavaObjectInfo() != source.info || base.allocationKind != listAllocationArray {
		t.Fatal("binding changed native allocation kind or dynamic-type fallback")
	}
	cursor := IterableIteratorExecution(e, source)
	if IteratorNextExecution(e, cursor) != marker {
		t.Fatal("source iterator did not read real native storage")
	}
	CollectionAddExecution(e, base, marker)
	arrayListSourcePanic(t, "ConcurrentModificationException", func() { IteratorNextExecution(e, cursor) })
	view := base.SubList(0, 1)
	if view.JavaObjectInfo() != nil || JavaReferenceEqual(view, source) || monitorIdentityFor(view) == monitorIdentityFor(source) {
		t.Fatal("subList inherited the source allocation identity")
	}
}

func TestArrayListSourceBindingRefusesInvalidStorageBeforeMutation(t *testing.T) {
	root := NewArrayList[any]()
	root.Add(new(int))
	rows := []struct {
		name string
		base *List[any]
	}{
		{"legacy", NewList[any]()},
		{"linked", NewLinkedList[any]()},
		{"fixed", &List[any]{allocationKind: listAllocationArray, fixed: true}},
		{"array", &List[any]{allocationKind: listAllocationArray, array: NewReferenceArray(0, ObjectTypeID)}},
		{"view", root.SubList(0, 1)},
		{"forgedViewKind", &List[any]{allocationKind: listAllocationArray, viewRoot: root, modCount: root.modCount}},
		{"parentView", &List[any]{allocationKind: listAllocationArray, viewParent: root}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			source := newArrayListSourceControl(TypeID("arrayList.source.BadStorage."+row.name), row.base)
			beforeSize, beforeMod := row.base.Size(), row.base.modCount
			arrayListSourcePanic(t, "UnsupportedOperationException", func() {
				BindArrayListSourceReference(row.base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
			})
			if row.base.sourceInfo != nil || row.base.sourceMethods.Add != nil || row.base.Size() != beforeSize || row.base.modCount != beforeMod {
				t.Fatal("refusal partially bound or mutated storage")
			}
		})
	}
	arrayListSourcePanic(t, "NullPointerException", func() { BindArrayListSourceReference[any](nil, nil, ArrayListSourceMethods{}) })
}

func TestArrayListSourceBindingChecksProviderInBothDirectionsAndRefusesRebind(t *testing.T) {
	for _, broken := range []string{"nilInfo", "nilAdd", "unregistered", "wrongAncestry", "arrayProjection", "listProjection", "mostDerived", "collection", "iterable", "differentInfo", "nativeProjectionAsRoot", "noIterator", "noAdd"} {
		t.Run(broken, func(t *testing.T) {
			base := NewArrayList[any]()
			source := newArrayListSourceControl(TypeID("arrayList.source.BadProvider."+broken), base)
			valid := source.info
			methods := ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution}
			switch broken {
			case "nilInfo":
				source.info = nil
			case "nilAdd":
				methods.Add = nil
			case "unregistered":
				const id TypeID = "arrayList.native.Unregistered"
				RegisterJavaType(id, "java.util.ArrayList")
				source.info = NewObjectInfo(id, valid.view)
			case "wrongAncestry":
				const id TypeID = "arrayList.source.WrongAncestry"
				RegisterJavaType(id, ObjectTypeID)
				RegisterJavaSourceType(id)
				source.info = NewObjectInfo(id, valid.view)
			default:
				badRequest := map[string]TypeID{"arrayProjection": "java.util.ArrayList", "listProjection": "java.util.List", "mostDerived": valid.DynamicType(), "collection": "java.util.Collection", "iterable": "java.lang.Iterable", "differentInfo": valid.DynamicType(), "nativeProjectionAsRoot": valid.DynamicType(), "noIterator": valid.DynamicType(), "noAdd": valid.DynamicType()}[broken]
				source.info = NewObjectInfo(valid.DynamicType(), func(want TypeID) any {
					if want == badRequest {
						if broken == "differentInfo" {
							return &arrayListSourceControl{info: valid, base: base}
						}
						if broken == "nativeProjectionAsRoot" {
							return base
						}
						if broken == "noIterator" {
							return &arrayListSourceIdentityOnly{info: source.info}
						}
						if broken == "noAdd" {
							return &arrayListSourceWithoutAdd{info: source.info, base: base}
						}
						return nil
					}
					return valid.resolveView(want)
				})
			}
			arrayListSourcePanic(t, "UnsupportedOperationException", func() { BindArrayListSourceReference(base, source.info, methods) })
			if base.sourceInfo != nil || base.sourceMethods.Add != nil || base.Size() != 0 || base.modCount != 0 || base.erasedStorage {
				t.Fatal("provider refusal partially installed identity, dispatch or storage")
			}
		})
	}
	base := NewArrayList[any]()
	source := newArrayListSourceControl("arrayList.source.Once", base)
	calls := 0
	source.add = func(e *Execution, value any) bool { calls++; return ArrayListBaseAddExecution(e, base, value) }
	methods := ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution}
	BindArrayListSourceReference(base, source.info, methods)
	for _, info := range []*ObjectInfo{source.info, newArrayListSourceControl("arrayList.source.OtherOwner", base).info} {
		arrayListSourcePanic(t, "UnsupportedOperationException", func() {
			BindArrayListSourceReference(base, info, ArrayListSourceMethods{Add: func(*Execution, any) bool { t.Fatal("rebound closure installed"); return false }})
		})
	}
	CollectionAddExecution(NewExecution(), base, nil)
	if base.sourceInfo != source.info || calls != 1 || base.Size() != 1 {
		t.Fatal("duplicate binding replaced original owner or hook")
	}
}

func TestArrayListSourcePlainDefaultAndVirtualExceptionIdentity(t *testing.T) {
	base := NewArrayList[any]()
	source := newArrayListSourceControl("arrayList.source.PlainDefault", base)
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	e := NewExecution()
	marker := new(int)
	CollectionAddExecution(e, base, marker)
	if base.Get(0) != marker || base.Size() != 1 {
		t.Fatal("plain inherited add lost real native default")
	}
	failure := NewIllegalStateException("source marker")
	source.add = func(got *Execution, value any) bool {
		if got != e || value != marker {
			t.Fatal("virtual exception body lost current execution or marker")
		}
		panic(failure)
	}
	if arrayListSourcePanic(t, "IllegalStateException", func() { CollectionAddExecution(e, base, marker) }) != failure || base.Size() != 1 {
		t.Fatal("virtual exception changed identity or mutated native storage")
	}
}

func TestArrayListNativeBaseServicesRequireRealStorageAndExecution(t *testing.T) {
	e := NewExecution()
	for _, base := range []any{NewList[any](), NewLinkedList[any](), AsList[any](nil), arrayListNativePublicShape{}} {
		arrayListSourcePanic(t, "UnsupportedOperationException", func() { ArrayListBaseAddExecution(e, base, nil) })
		arrayListSourcePanic(t, "UnsupportedOperationException", func() { ArrayListBaseGetExecution(e, base, 0) })
		arrayListSourcePanic(t, "UnsupportedOperationException", func() { ArrayListBaseSizeExecution(e, base) })
	}
	arrayListSourcePanic(t, "NullPointerException", func() { ArrayListBaseSizeExecution(e, nil) })
	base := NewArrayList[any]()
	arrayListSourcePanic(t, "IllegalArgumentException", func() { ArrayListBaseAddExecution(nil, base, nil) })
	arrayListSourcePanic(t, "IllegalArgumentException", func() { ArrayListBaseGetExecution(nil, base, 0) })
	arrayListSourcePanic(t, "IllegalArgumentException", func() { ArrayListBaseSizeExecution(nil, base) })
	if base.Size() != 0 || base.erasedStorage || base.modCount != 0 {
		t.Fatal("base-service refusal changed storage")
	}
}

func TestArrayListUnboundNativeFallbackAndViewsRemainUnchanged(t *testing.T) {
	var null *List[any]
	if null.JavaObjectInfo() != nil {
		t.Fatal("nil native list acquired metadata")
	}
	for _, row := range []struct {
		name string
		base *List[any]
		want TypeID
	}{
		{"array", NewArrayList[any](), "java.util.ArrayList"},
		{"linked", NewLinkedList[any](), "java.util.LinkedList"},
		{"legacy", NewList[any](), ""},
	} {
		t.Run(row.name, func(t *testing.T) {
			before := monitorIdentityFor(row.base)
			marker := new(int)
			CollectionAddExecution(nil, row.base, marker)
			actual, _ := ObjectDynamicType(row.base)
			if row.base.JavaObjectInfo() != nil || actual != row.want || row.base.Size() != 1 || row.base.Get(0) != marker || monitorIdentityFor(row.base) != before || !JavaReferenceEqual(row.base, row.base) {
				t.Fatal("unbound native metadata, storage or identity fallback changed")
			}
			if row.want != "" && (ObjectGetClass(row.base) != ClassLiteral(row.want) || reflectionReceiver(row.base, ClassLiteral(row.want)).Interface() != row.base) {
				t.Fatal("unbound native reflection fallback changed")
			}
			view := row.base.SubList(0, 1)
			CollectionAddExecution(nil, view, marker)
			if view.JavaObjectInfo() != nil || view.Size() != 2 || row.base.Size() != 2 || JavaReferenceEqual(view, row.base) {
				t.Fatal("native subList mutation or identity fallback changed")
			}
		})
	}
	arrayListSourcePanic(t, "UnsupportedOperationException", func() { CollectionAddExecution(nil, AsList[any](new(int)), nil) })
}

package stdjava

import "testing"

type threadLocalReferenceSupplierSource struct{ info *ObjectInfo }

func (source *threadLocalReferenceSupplierSource) JavaObjectInfo() *ObjectInfo {
	if source == nil {
		return nil
	}
	return source.info
}

func threadLocalReferenceObservedPanic(action func()) (failure any) {
	defer func() { failure = recover() }()
	action()
	return nil
}

// Every call in this baseline RED uses the existing public runtime API. The
// new erased getter is deliberately absent: typed Get must publish a raw Java
// Supplier result before its own consumer projection can fail.
func TestThreadLocalReferenceExistingAPIBoundary(t *testing.T) {
	t.Run("covariant_native_supplier", func(t *testing.T) {
		e := NewExecution()
		marker := JavaStringFromHostUTF8("covariant")
		calls := 0
		local := ThreadLocalWithInitial[any](NewSupplierFuncAdapter(func(got *Execution) *JavaString {
			if got != e {
				t.Fatal("supplier replaced the current execution")
			}
			calls++
			return marker
		}))
		if calls != 0 {
			t.Fatal("withInitial invoked supplier eagerly")
		}
		var first any
		failure := threadLocalReferenceObservedPanic(func() { first = local.Get(e) })
		if failure != nil {
			t.Fatalf("covariant Supplier was rejected before erased caching: %T %v; calls=%d", failure, failure, calls)
		}
		if first != marker || local.Get(e) != marker || calls != 1 {
			t.Fatal("covariant supplier lost cached reference identity or repeated initialization")
		}
	})
	t.Run("covariant_owned_source_supplier", func(t *testing.T) {
		const id TypeID = "threadLocal.reference.ExistingSourceSupplier"
		RegisterJavaType(id, ObjectTypeID, SupplierTypeID)
		RegisterJavaSourceType(id)
		e := NewExecution()
		marker := JavaStringFromHostUTF8("source")
		calls := 0
		source := &threadLocalReferenceSupplierSource{}
		var view *SupplierSourceView[*JavaString]
		source.info = NewObjectInfo(id, func(want TypeID) any {
			switch want {
			case id, ObjectTypeID:
				return source
			case SupplierTypeID:
				return view
			}
			return nil
		})
		view = NewSupplierSourceView(source.info, func(got *Execution) *JavaString {
			if got != e {
				t.Fatal("source supplier lost current execution")
			}
			calls++
			return marker
		})
		local := ThreadLocalWithInitial[any](source)
		var first any
		failure := threadLocalReferenceObservedPanic(func() { first = local.Get(e) })
		if failure != nil {
			t.Fatalf("owned source Supplier was rejected before erased caching: %T %v; calls=%d", failure, failure, calls)
		}
		if first != marker || local.Get(e) != marker || calls != 1 || !JavaReferenceEqual(source, view) {
			t.Fatal("source supplier lost lazy cache, identity or owned view")
		}
	})
	t.Run("cast_after_cache_retry_and_null", func(t *testing.T) {
		e := NewExecution()
		marker := BoxInteger(1000)
		abrupt := NewIllegalArgumentException("initializer marker")
		calls := 0
		fail, nulls := false, false
		local := ThreadLocalWithInitial[*JavaString](NewSupplierFuncAdapter(func(got *Execution) *Integer {
			if got != e || ThreadCurrentThread(got) != ThreadCurrentThread(e) {
				t.Fatal("initializer changed logical execution/thread")
			}
			calls++
			if fail {
				panic(abrupt)
			}
			if nulls {
				return nil
			}
			return marker
		}))
		for attempt := 0; attempt < 2; attempt++ {
			failure := threadLocalReferenceObservedPanic(func() { local.Get(e) })
			if !CaughtAs(failure, "ClassCastException") {
				t.Fatalf("typed consumer must fail after raw cache publication: %T %v; calls=%d", failure, failure, calls)
			}
			if _, present := e.threadLocals[local.key]; !present || calls != 1 {
				t.Fatal("typed consumer failure lost cache or repeated initialization")
			}
		}
		local.Remove(e)
		fail = true
		failure := threadLocalReferenceObservedPanic(func() { local.Get(e) })
		if !sameThrowableIdentity(failure, abrupt) || calls != 2 {
			t.Fatal("initializer exception lost identity or was translated into a cast")
		}
		if _, present := e.threadLocals[local.key]; present {
			t.Fatal("failed initialization published a final cache value")
		}
		fail = false
		failure = threadLocalReferenceObservedPanic(func() { local.Get(e) })
		if !CaughtAs(failure, "ClassCastException") || calls != 3 {
			t.Fatal("failed initialization did not retry before typed consumer check")
		}
		failure = threadLocalReferenceObservedPanic(func() { local.Get(e) })
		if !CaughtAs(failure, "ClassCastException") || calls != 3 {
			t.Fatal("retry's cached value was discarded by its consumer failure")
		}
		local.Remove(e)
		nulls = true
		first, second := local.Get(e), local.Get(e)
		if first != nil || second != nil || calls != 4 {
			t.Fatal("null result was not cached across typed consumers")
		}
		failure = threadLocalReferenceObservedPanic(func() { ThreadLocalWithInitial[any](nil) })
		if !CaughtAs(failure, "NullPointerException") || calls != 4 {
			t.Fatal("null supplier validation was not eager or invoked initializer")
		}
	})
	t.Run("execution_isolation_and_worker_reuse", func(t *testing.T) {
		first, second := NewExecution(), NewExecution()
		threads := map[*Execution]*Thread{first: ThreadCurrentThread(first), second: ThreadCurrentThread(second)}
		calls := map[*Execution]int{}
		local := ThreadLocalWithInitial[*JavaString](NewSupplierFuncAdapter(func(got *Execution) *Integer {
			want, known := threads[got]
			if !known || ThreadCurrentThread(got) != want {
				t.Fatal("initializer replaced execution or logical thread")
			}
			calls[got]++
			return BoxInteger(int32(calls[got]))
		}))
		for _, e := range []*Execution{first, second, first, second} {
			failure := threadLocalReferenceObservedPanic(func() { local.Get(e) })
			if !CaughtAs(failure, "ClassCastException") {
				t.Fatalf("logical execution should reach cached-result consumer cast: %T %v", failure, failure)
			}
		}
		if calls[first] != 1 || calls[second] != 1 {
			t.Fatal("executions shared a value or reused workers lost their cache")
		}
		local.Remove(first)
		for _, e := range []*Execution{second, first} {
			failure := threadLocalReferenceObservedPanic(func() { local.Get(e) })
			if !CaughtAs(failure, "ClassCastException") {
				t.Fatal("remove altered another execution or skipped consumer projection")
			}
		}
		if calls[first] != 2 || calls[second] != 1 {
			t.Fatal("remove was not isolated to its execution/key")
		}
	})
	t.Run("legacy_native_go_controls", func(t *testing.T) {
		e := NewExecution()
		integer := NewThreadLocal[int32]()
		if integer.Get(e) != 0 {
			t.Fatal("legacy native primitive default changed")
		}
		integer.Set(e, 7)
		if integer.Get(e) != 7 {
			t.Fatal("legacy native typed Set/Get changed")
		}
		integer.Remove(e)
		if integer.Get(e) != 0 || !StringIsNull(NewThreadLocal[string]().Get(e)) {
			t.Fatal("legacy remove/default/null sentinel changed")
		}
		calls := 0
		plain := ThreadLocalWithInitial[string](func(got *Execution) string {
			if got != e {
				t.Fatal("legacy plain initializer lost execution")
			}
			calls++
			return "legacy"
		})
		first, second := plain.Get(e), plain.Get(e)
		if first != "legacy" || second != "legacy" || calls != 1 {
			t.Fatal("legacy plain callback API/cache changed")
		}
	})
}

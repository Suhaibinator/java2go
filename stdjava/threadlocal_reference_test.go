package stdjava

import "testing"

type threadLocalReferenceSourceGetShape struct {
	info   *ObjectInfo
	calls  int
	result *JavaString
}

func (source *threadLocalReferenceSourceGetShape) JavaObjectInfo() *ObjectInfo { return source.info }
func (source *threadLocalReferenceSourceGetShape) Get() *JavaString {
	source.calls++
	return source.result
}

type threadLocalReferencePlainGoSupplier struct{ calls int }

func (source *threadLocalReferencePlainGoSupplier) Get() string {
	source.calls++
	return "legacy"
}

func TestThreadLocalReferenceRawCellSurvivesTypedConsumer(t *testing.T) {
	e := NewExecution()
	marker := BoxInteger(1000)
	calls := 0
	local := ThreadLocalWithInitial[*JavaString](NewSupplierFuncAdapter(func(got *Execution) *Integer {
		if got != e {
			t.Fatal("raw initializer replaced current execution")
		}
		calls++
		return marker
	}))
	failure := threadLocalReferenceObservedPanic(func() { local.Get(e) })
	if !CaughtAs(failure, "ClassCastException") || local.GetObject(e) != marker || local.GetObject(e) != marker || calls != 1 {
		t.Fatal("typed Get failure preceded cache or discarded raw reference identity")
	}
	replacement := JavaStringFromHostUTF8("replacement")
	local.SetObject(e, replacement)
	if local.Get(e) != replacement || local.GetObject(e) != replacement || calls != 1 {
		t.Fatal("erased Set/Get did not share typed API's key/storage")
	}
	local.Remove(e)
	if local.GetObject(e) != marker || calls != 2 {
		t.Fatal("remove did not reinitialize the same raw cell")
	}
}

func TestThreadLocalReferenceJavaFactoryValidationIsLazy(t *testing.T) {
	e := NewExecution()
	thread := ThreadCurrentThread(e)
	selects, calls := 0, 0
	marker := JavaStringFromHostUTF8("reference")
	choose := func() any {
		selects++
		return NewSupplierFuncAdapter(func(got *Execution) *JavaString {
			if got != e || ThreadCurrentThread(got) != thread {
				t.Fatal("reference initializer replaced the current execution/thread")
			}
			calls++
			return marker
		})
	}
	local := ThreadLocalWithInitialReference(choose())
	if selects != 1 || calls != 0 {
		t.Fatal("reference factory reevaluated or invoked Supplier")
	}
	for _, action := range []func(){
		func() { local.GetObject(nil) },
		func() { local.SetObject(nil, nil) },
	} {
		failure := threadLocalReferenceObservedPanic(action)
		if !CaughtAs(failure, "IllegalArgumentException") || calls != 0 || len(e.threadLocals) != 0 {
			t.Fatal("nil execution entered callback or published a value")
		}
	}
	var absent *ThreadLocal[any]
	for _, action := range []func(){
		func() { absent.GetObject(e) },
		func() { absent.SetObject(e, marker) },
		func() { ThreadLocalWithInitialReference(nil) },
		func() { var typed *SupplierFuncAdapter[any]; ThreadLocalWithInitialReference(typed) },
	} {
		if !CaughtAs(threadLocalReferenceObservedPanic(action), "NullPointerException") || calls != 0 {
			t.Fatal("null receiver/supplier did not fail before callback")
		}
	}
	first, second := local.GetObject(e), local.GetObject(e)
	if first != marker || second != marker || calls != 1 || selects != 1 {
		t.Fatal("reference factory lost lazy cached identity or reevaluated Supplier expression")
	}
}

func TestThreadLocalReferenceSupplierAuthorityAndLegacyFallback(t *testing.T) {
	for _, broken := range []string{"missing", "foreign", "unregistered"} {
		t.Run(broken, func(t *testing.T) {
			id := TypeID("threadLocal.reference.InvalidSource." + broken)
			RegisterJavaType(id, ObjectTypeID, SupplierTypeID)
			if broken != "unregistered" {
				RegisterJavaSourceType(id)
			}
			source := &threadLocalReferenceSourceGetShape{result: JavaStringFromHostUTF8("must not run")}
			var view JavaSupplier
			source.info = NewObjectInfo(id, func(want TypeID) any {
				switch want {
				case id, ObjectTypeID:
					return source
				case SupplierTypeID:
					return view
				}
				return nil
			})
			if broken == "foreign" {
				foreignID := TypeID("threadLocal.reference.ForeignSource." + broken)
				RegisterJavaType(foreignID, ObjectTypeID, SupplierTypeID)
				RegisterJavaSourceType(foreignID)
				view = NewSupplierSourceView(NewObjectInfo(foreignID, nil), func(*Execution) *JavaString {
					t.Fatal("foreign Supplier body acquired authority")
					return nil
				})
			}
			e := NewExecution()
			reference := ThreadLocalWithInitialReference(source)
			legacy := ThreadLocalWithInitial[*JavaString](source)
			for _, action := range []func(){func() { reference.GetObject(e) }, func() { legacy.Get(e) }} {
				if !CaughtAs(threadLocalReferenceObservedPanic(action), "UnsupportedOperationException") {
					t.Fatal("canonical source Supplier did not require its owned sealed view")
				}
			}
			if source.calls != 0 || len(e.threadLocals) != 0 {
				t.Fatal("invalid source Supplier fell back to public Get or published cache")
			}
		})
	}
	e := NewExecution()
	plain := &threadLocalReferencePlainGoSupplier{}
	legacy := ThreadLocalWithInitial[string](plain)
	first, second := legacy.Get(e), legacy.Get(e)
	if first != "legacy" || second != "legacy" || plain.calls != 1 {
		t.Fatal("native Go Supplier[T] fallback changed")
	}
	reference := ThreadLocalWithInitialReference(plain)
	if !CaughtAs(threadLocalReferenceObservedPanic(func() { reference.GetObject(e) }), "ClassCastException") || plain.calls != 1 {
		t.Fatal("Java reference path accepted a public Go Get-shaped object")
	}
	goSourceID := TypeID("threadLocal.reference.LegacyNonSupplierGoValue")
	RegisterJavaType(goSourceID, ObjectTypeID)
	goSource := &threadLocalReferenceSourceGetShape{result: JavaStringFromHostUTF8("go value")}
	goSource.info = NewObjectInfo(goSourceID, nil)
	goLocal := ThreadLocalWithInitial[*JavaString](goSource)
	if goLocal.Get(e) != goSource.result || goSource.calls != 1 {
		t.Fatal("non-Supplier native Go metadata blocked legacy callback API")
	}
}

func TestThreadLocalReferenceRawSetNullAndAliases(t *testing.T) {
	e := NewExecution()
	marker := BoxInteger(1001)
	calls := 0
	local := ThreadLocalWithInitialReference(NewSupplierFuncAdapter(func(got *Execution) *Integer {
		if got != e {
			t.Fatal("remove/retry replaced the consuming execution")
		}
		calls++
		return marker
	}))
	rawAlias := local
	var objectAlias any = local
	if objectAlias.(*ThreadLocal[any]) != rawAlias || !JavaReferenceEqual(objectAlias, rawAlias) || monitorIdentityFor(objectAlias) != monitorIdentityFor(local) {
		t.Fatal("raw aliases changed ThreadLocal object/key identity")
	}
	rawAlias.SetObject(e, nil)
	if local.GetObject(e) != nil || rawAlias.GetObject(e) != nil || calls != 0 {
		t.Fatal("SetObject(null) was treated as absence or invoked Supplier")
	}
	if _, present := e.threadLocals[local.key]; !present {
		t.Fatal("null cell was not present")
	}
	rawAlias.Remove(e)
	if local.GetObject(e) != marker || rawAlias.GetObject(e) != marker || calls != 1 {
		t.Fatal("alias remove did not reinitialize one shared raw cache")
	}
	failure := threadLocalReferenceObservedPanic(func() { ObjectView[*JavaString](local.GetObject(e), StringTypeID) })
	if !CaughtAs(failure, "ClassCastException") || local.GetObject(e) != marker || calls != 1 {
		t.Fatal("actual typed consumer cast invalidated raw cached reference")
	}
}

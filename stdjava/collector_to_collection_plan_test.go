package stdjava

import "testing"

func TestCollectorToCollectionSupplierIsLazyAndUsesOperationExecution(t *testing.T) {
	creator, operation := NewExecution(), NewExecution()
	calls := 0
	var products []*List[any]
	factory := NewSupplierFuncAdapter(func(e *Execution) any {
		if e != operation || e == creator {
			t.Fatal("supplier used creator or fresh Execution")
		}
		calls++
		product := NewArrayList[any]()
		products = append(products, product)
		return product
	})
	plan := CollectorToCollectionPlan[any](factory)
	if calls != 0 || !plan.identityFinish {
		t.Fatal("recipe ran supplier or lost identity finish")
	}
	first, last := NewObject(), NewObject()
	for _, input := range []Stream[any]{collectorToCollectionInput[any](first, nil, last), collectorToCollectionInput[any]()} {
		result := StreamCollectPlanExecution(operation, input, plan)
		if result != products[len(products)-1] {
			t.Fatal("supplier product was copied or replaced")
		}
	}
	if calls != 2 || products[0] == products[1] || products[0].Size() != 3 || products[0].GetObject(0) != first || products[0].GetObject(1) != nil || products[0].GetObject(2) != last || products[1].Size() != 0 {
		t.Fatal("terminal supplier count, encounter order, null or reuse changed")
	}
}

func TestCollectorToCollectionNativeErasurePreservesActualProduct(t *testing.T) {
	e := NewExecution()
	product := NewArrayList[*Integer]()
	first := BoxInteger(7)
	product.Add(first)
	item := JavaStringFromHostUTF8("raw")
	plan := CollectorToCollectionPlan[*JavaString](NewSupplierFuncAdapter(func(*Execution) any { return product }))
	result := StreamCollectPlanExecution(e, collectorToCollectionInput(item), plan)
	if result != product || product.Size() != 2 || product.GetObject(0) != first || product.GetObject(1) != item {
		t.Fatal("T narrowed actual Collection storage, identity or prefilled state")
	}
	collectorRuntimePanic(t, "ClassCastException", func() { product.Get(1) })
}

func TestCollectorToCollectionNullSupplierAndProductTiming(t *testing.T) {
	e := NewExecution()
	plan := CollectorToCollectionPlan[any](nil) // Construction itself is allowed.
	input := collectorToCollectionInput[any]()
	collectorRuntimePanic(t, "NullPointerException", func() { StreamCollectPlanExecution(e, input, plan) })
	collectorRuntimePanic(t, "IllegalStateException", func() { StreamCollectPlanExecution(e, input, plan) })
	gets := 0
	nullPlan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { gets++; return (*List[any])(nil) }))
	if result := StreamCollectPlanExecution(e, collectorToCollectionInput[any](), nullPlan); !nilJavaReference(result) || gets != 1 {
		t.Fatal("empty identity finish narrowed or dereferenced null supplier product")
	}
	nonempty := collectorToCollectionInput[any](NewObject())
	collectorRuntimePanic(t, "NullPointerException", func() { StreamCollectPlanExecution(e, nonempty, nullPlan) })
	if gets != 2 {
		t.Fatal("null product supplier count changed")
	}
	collectorRuntimePanic(t, "IllegalStateException", func() { StreamCollectPlanExecution(e, nonempty, nullPlan) })
}

func TestCollectorToCollectionHeapPollutedProductCastIsAtOperation(t *testing.T) {
	e := NewExecution()
	product := NewObject()
	plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return product }))
	if StreamCollectPlanExecution(e, collectorToCollectionInput[any](), plan) != product {
		t.Fatal("empty erased identity result acquired an early Collection/C cast")
	}
	collectorRuntimePanic(t, "ClassCastException", func() { StreamCollectPlanExecution(e, collectorToCollectionInput[any](nil), plan) })
	if plan.finish(e, product) != product {
		t.Fatal("identity finish copied or narrowed heap-polluted state")
	}
}

func TestCollectorToCollectionBoundSourceRetainsNativeAliasesAndCurrentExecution(t *testing.T) {
	e := NewExecution()
	base := NewArrayList[any]()
	source := newArrayListSourceControl("collector.toCollection.Source", base)
	adds, gets := 0, 0
	source.add = func(got *Execution, item any) bool {
		if got != e {
			t.Fatal("source body lost current terminal Execution")
		}
		adds++
		return ArrayListBaseAddExecution(got, base, item)
	}
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(got *Execution) any {
		if got != e {
			t.Fatal("source supplier lost current terminal Execution")
		}
		gets++
		return source
	}))
	first, last := NewObject(), NewObject()
	result := StreamCollectPlanExecution(e, collectorToCollectionInput[any](first, nil, last), plan)
	if result != source || gets != 1 || adds != 3 || base.Size() != 3 || base.GetObject(0) != first || base.GetObject(1) != nil || base.GetObject(2) != last {
		t.Fatal("source add dispatch or native storage identity changed")
	}
	if ObjectView[*List[any]](result, "java.util.ArrayList") != base || ObjectView[*List[any]](result, "java.util.List") != base || !JavaReferenceEqual(result, base) || source.info != base.JavaObjectInfo() || monitorIdentityFor(source) != monitorIdentityFor(base) || SystemIdentityHashCode(source) != SystemIdentityHashCode(base) {
		t.Fatal("collector replaced whole source/provider/native parent identity")
	}
}

type collectorToCollectionStringSource struct {
	base  *List[*JavaString]
	info  *ObjectInfo
	want  *Execution
	calls int
}

func (s *collectorToCollectionStringSource) JavaObjectInfo() *ObjectInfo { return s.info }
func (s *collectorToCollectionStringSource) IteratorJava2goExecution(e *Execution) JavaIterator {
	return s.base.IteratorJava2goExecution(e)
}
func (s *collectorToCollectionStringSource) CollectionAddJava2goExecution(e *Execution, value any) bool {
	if e != s.want {
		panic("source String add lost operation Execution")
	}
	item := ObjectView[*JavaString](value, StringTypeID)
	s.calls++
	return ArrayListBaseAddExecution(e, s.base, item)
}

func TestCollectorToCollectionSourceStringBridgePrecedesBody(t *testing.T) {
	e := NewExecution()
	const id TypeID = "collector.toCollection.StringSource"
	RegisterJavaType(id, "java.util.ArrayList")
	RegisterJavaSourceType(id)
	source := &collectorToCollectionStringSource{base: NewArrayList[*JavaString](), want: e}
	source.info = NewObjectInfo(id, func(requested TypeID) any {
		switch requested {
		case id, ObjectTypeID, nativeCollectionTypeID, IterableTypeID:
			return source
		case "java.util.ArrayList", nativeListTypeID:
			return source.base
		}
		return nil
	})
	BindArrayListSourceReference(source.base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return source }))
	collectorRuntimePanic(t, "ClassCastException", func() { StreamCollectPlanExecution(e, collectorToCollectionInput[any](BoxInteger(3)), plan) })
	if source.calls != 0 || source.base.Size() != 0 {
		t.Fatal("String bridge cast occurred after source body/store")
	}
	item := JavaStringFromHostUTF8("typed")
	if StreamCollectPlanExecution(e, collectorToCollectionInput[any](item, nil), plan) != source || source.calls != 2 || source.base.Get(0) != item || source.base.Get(1) != nil {
		t.Fatal("valid String/null add or original result identity changed")
	}
}

type collectorToCollectionPublicLookalike struct{ adds int }

func (*collectorToCollectionPublicLookalike) JavaDynamicTypeID() TypeID {
	return nativeCollectionTypeID
}
func (*collectorToCollectionPublicLookalike) IteratorJava2goExecution(e *Execution) JavaIterator {
	return NewArrayList[any]().IteratorJava2goExecution(e)
}
func (s *collectorToCollectionPublicLookalike) CollectionAddJava2goExecution(*Execution, any) bool {
	s.adds++
	return true
}

func TestCollectorToCollectionRejectsUnboundSourcesAndPublicLookalikes(t *testing.T) {
	e := NewExecution()
	source := collectorToListUnbridgedSource()
	lookalike := &collectorToCollectionPublicLookalike{}
	for _, row := range []struct {
		product any
		failure string
	}{{source, "UnsupportedOperationException"}, {lookalike, "ClassCastException"}} {
		plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return row.product }))
		collectorRuntimePanic(t, row.failure, func() { StreamCollectPlanExecution(e, collectorToCollectionInput[any](NewObject()), plan) })
	}
	if source.Size() != 0 || lookalike.adds != 0 {
		t.Fatal("unbound source or public hook shape mutated before refusal")
	}
}

func TestCollectorToCollectionAbruptSupplierAndSourceAddPreserveIdentity(t *testing.T) {
	e := NewExecution()
	marker := NewIllegalArgumentException("toCollection")
	gets := 0
	failurePlan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { gets++; panic(marker) }))
	input := collectorToCollectionInput[any](NewObject())
	if got := collectorRuntimePanic(t, "IllegalArgumentException", func() { StreamCollectPlanExecution(e, input, failurePlan) }); !sameThrowableIdentity(got, marker) || gets != 1 {
		t.Fatal("supplier failure identity/count changed")
	}
	collectorRuntimePanic(t, "IllegalStateException", func() { StreamCollectPlanExecution(e, input, failurePlan) })
	source := collectorToListActualSource(e)
	source.failure = marker
	addPlan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return source }))
	input = collectorToCollectionInput[any](NewObject(), NewObject())
	if got := collectorRuntimePanic(t, "IllegalArgumentException", func() { StreamCollectPlanExecution(e, input, addPlan) }); !sameThrowableIdentity(got, marker) || source.adds != 1 || len(source.values) != 0 {
		t.Fatal("add failure identity or traversal stopping changed")
	}
	collectorRuntimePanic(t, "IllegalStateException", func() { StreamCollectPlanExecution(e, input, addPlan) })
}

func TestCollectorToCollectionCombinerRefusesUnprovedAddAll(t *testing.T) {
	e := NewExecution()
	product := NewArrayList[any]()
	product.Add(NewObject())
	plan := CollectorToCollectionPlan[any](nil)
	before := product.modCount
	for _, pair := range [][2]any{{product, product}, {product, NewArrayList[any]()}, {nil, product}, {product, nil}} {
		collectorRuntimePanic(t, "UnsupportedOperationException", func() { plan.combine(e, pair[0], pair[1]) })
	}
	if product.Size() != 1 || product.modCount != before {
		t.Fatal("unproved combiner approximated addAll through virtual adds")
	}
}

type collectorToCollectionNilInfoSource struct{ *List[any] }

func (*collectorToCollectionNilInfoSource) JavaDynamicTypeID() TypeID {
	return "collector.toCollection.NilInfoSource"
}

type collectorToCollectionPromotedNativeHost struct{ *List[any] }

func TestCollectorToCollectionPromotedNativeMarkerCannotCertifyWholeHost(t *testing.T) {
	e := NewExecution()
	const id TypeID = "collector.toCollection.NilInfoSource"
	RegisterJavaType(id, "java.util.ArrayList")
	RegisterJavaSourceType(id)
	source := &collectorToCollectionNilInfoSource{NewArrayList[any]()}
	host := &collectorToCollectionPromotedNativeHost{NewArrayList[any]()}
	for _, row := range []struct {
		value   any
		failure string
	}{{source, "UnsupportedOperationException"}, {host, "ClassCastException"}} {
		plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return row.value }))
		collectorRuntimePanic(t, row.failure, func() { StreamCollectPlanExecution(e, collectorToCollectionInput[any](NewObject()), plan) })
	}
	if source.Size() != 0 || host.Size() != 0 {
		t.Fatal("promoted private membership/reference marker certified or mutated an outer host")
	}
}

func TestCollectorToCollectionBoundPromotedHostCannotDonateSourceReference(t *testing.T) {
	e := NewExecution()
	base := NewArrayList[any]()
	source := newArrayListSourceControl("collector.toCollection.BoundHost", base)
	adds := 0
	source.add = func(got *Execution, item any) bool { adds++; return ArrayListBaseAddExecution(got, base, item) }
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	host := &collectorToCollectionPromotedNativeHost{base}
	plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return host }))
	collectorRuntimePanic(t, "UnsupportedOperationException", func() { StreamCollectPlanExecution(e, collectorToCollectionInput[any](NewObject()), plan) })
	if adds != 0 || base.Size() != 0 {
		t.Fatal("bound native embedding donated another receiver's source body or storage")
	}
}

type collectorToCollectionSourceChild struct{ *arrayListSourceControl }

func TestCollectorToCollectionSourceAncestorAndNativeAliasesKeepProductIdentity(t *testing.T) {
	e := NewExecution()
	const rootID TypeID = "collector.toCollection.AliasRoot"
	const childID TypeID = "collector.toCollection.AliasChild"
	base := NewArrayList[any]()
	root := newArrayListSourceControl(rootID, base)
	child := &collectorToCollectionSourceChild{root}
	RegisterJavaType(childID, rootID)
	RegisterJavaSourceType(childID)
	root.info = NewObjectInfo(childID, func(requested TypeID) any {
		switch requested {
		case childID, ObjectTypeID, nativeCollectionTypeID, IterableTypeID:
			return child
		case rootID:
			return root
		case "java.util.ArrayList", nativeListTypeID:
			return base
		}
		return nil
	})
	adds := 0
	root.add = func(got *Execution, item any) bool {
		if got != e {
			t.Fatal("alias accumulation lost current E")
		}
		adds++
		return ArrayListBaseAddExecution(got, base, item)
	}
	BindArrayListSourceReference(base, root.info, ArrayListSourceMethods{Add: root.CollectionAddJava2goExecution})
	for _, product := range []any{child, root, base} {
		plan := CollectorToCollectionPlan[any](NewSupplierFuncAdapter(func(*Execution) any { return product }))
		if StreamCollectPlanExecution(e, collectorToCollectionInput[any](NewObject()), plan) != product {
			t.Fatal("whole source, superclass or exact native alias was replaced at finish")
		}
	}
	if adds != 3 || base.Size() != 3 || root.info != base.JavaObjectInfo() || !JavaReferenceEqual(child, root) || !JavaReferenceEqual(root, base) {
		t.Fatal("owned aliases lost most-derived add, shared storage or allocation identity")
	}
}

// CollectorPlan terminals require an initialized, consumable pipeline source.
// NewStream intentionally remains the legacy eager API and is not adopted here.
func collectorToCollectionInput[T any](elements ...T) Stream[T] {
	return streamFromSlice(elements)
}

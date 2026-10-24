package stdjava

import "testing"

func TestArrayListCollectionStatePromotedHookRequiresPrivateBinding(t *testing.T) {
	e := NewExecution()
	source := collectorToListUnbridgedSource()
	if _, promoted := any(source).(interface {
		CollectionAddJava2goExecution(*Execution, any) bool
	}); !promoted {
		t.Fatal("control must retain the promoted native hook")
	}
	plan := CollectorToCollectionPlan[any](nil)
	arrayListSourcePanic(t, "UnsupportedOperationException", func() { plan.accumulate(e, source, NewObject()) })
	if source.Size() != 0 || source.sourceInfo != nil || source.sourceMethods.Add != nil || source.erasedStorage || source.modCount != 0 {
		t.Fatal("unbound source acquired authority or mutated before refusal")
	}
	// It has an actual ArrayList marker and nominal source ancestry; neither
	// may bypass missing binding via the unchanged native marker branch.
	if !source.arrayListNativeStorage() || !ObjectInstanceOf(source, "java.util.ArrayList") {
		t.Fatal("control lost its nominal/physical marker premise")
	}
}

func TestArrayListCollectionStateRejectsProviderMismatchAndForeignBinding(t *testing.T) {
	for _, broken := range []string{"arrayProjection", "listProjection", "foreignBinding", "mostDerived", "collection", "iterable", "wrongKind", "unboundProjection"} {
		t.Run(broken, func(t *testing.T) {
			base := NewArrayList[any]()
			source := newArrayListSourceControl(TypeID("arrayList.source.StateMismatch."+broken), base)
			routes := map[TypeID]any{
				source.info.DynamicType(): source,
				"java.util.ArrayList":     base,
				"java.util.List":          base,
				"java.util.Collection":    source,
				"java.lang.Iterable":      source,
			}
			source.info.view = func(want TypeID) any { return routes[want] }
			calls := 0
			source.add = func(e *Execution, value any) bool { calls++; return ArrayListBaseAddExecution(e, base, value) }
			BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
			foreignBase := NewArrayList[any]()
			foreign := newArrayListSourceControl(TypeID("arrayList.source.StateForeign."+broken), foreignBase)
			BindArrayListSourceReference(foreignBase, foreign.info, ArrayListSourceMethods{Add: foreign.CollectionAddJava2goExecution})
			switch broken {
			case "arrayProjection":
				routes["java.util.ArrayList"] = foreignBase
			case "listProjection":
				routes["java.util.List"] = foreignBase
			case "foreignBinding":
				routes["java.util.ArrayList"], routes["java.util.List"] = foreignBase, foreignBase
			case "mostDerived":
				routes[source.info.DynamicType()] = foreign
			case "collection":
				routes["java.util.Collection"] = foreign
			case "iterable":
				routes["java.lang.Iterable"] = foreign
			case "wrongKind":
				base.allocationKind = listAllocationLinked
			case "unboundProjection":
				unbound := NewArrayList[any]()
				routes["java.util.ArrayList"], routes["java.util.List"] = unbound, unbound
			}
			plan := CollectorToCollectionPlan[any](nil)
			arrayListSourcePanic(t, "UnsupportedOperationException", func() { plan.accumulate(NewExecution(), source, NewObject()) })
			if calls != 0 || base.Size() != 0 || base.modCount != 0 || base.erasedStorage || foreignBase.Size() != 0 || foreignBase.modCount != 0 {
				t.Fatal("provider refusal invoked a body or mutated native state")
			}
		})
	}
}

func TestArrayListCollectionStateNativeAndSourceAliasesUseOneVirtualReceiver(t *testing.T) {
	base := NewArrayList[any]()
	source := newArrayListSourceControl("arrayList.source.StateAliases", base)
	e := NewExecution()
	marker := NewObject()
	calls := 0
	source.add = func(got *Execution, value any) bool {
		if got != e || value != marker {
			t.Fatal("collector alias lost current execution or reference marker")
		}
		calls++
		return ArrayListBaseAddExecution(got, base, value)
	}
	BindArrayListSourceReference(base, source.info, ArrayListSourceMethods{Add: source.CollectionAddJava2goExecution})
	plan := CollectorToCollectionPlan[any](nil)
	for _, alias := range []any{source, base, ObjectView[JavaIterable](base, "java.util.Collection"), ObjectView[JavaIterable](base, "java.lang.Iterable")} {
		if got := collectorToCollectionStateView(alias); got != source || !JavaReferenceEqual(got, alias) {
			t.Fatal("source state certification copied or substituted receiver identity")
		}
		plan.accumulate(e, alias, marker)
	}
	if calls != 4 || base.Size() != 4 {
		t.Fatal("alias accumulation bypassed virtual body or real native storage")
	}
	for index := int32(0); index < base.Size(); index++ {
		if base.Get(index) != marker {
			t.Fatal("alias accumulation changed reference identity")
		}
	}
}

package stdjava

import (
	"reflect"
	"testing"
)

type sortNullValue struct {
	*ObjectInfo
	key   int32
	label string
}

func sortNullNew(key int32, label string) *sortNullValue {
	v := &sortNullValue{key: key, label: label}
	v.ObjectInfo = NewObjectInfo("campaign.SortNullValue", func(id TypeID) any {
		if id == "campaign.SortNullValue" || id == ObjectTypeID {
			return v
		}
		return nil
	})
	return v
}
func sortNullPanic(f func()) (p any) { defer func() { p = recover() }(); f(); return }
func sortNullLess(a, b *sortNullValue) int32 {
	if a == nil {
		if b == nil {
			return 0
		}
		return -1
	}
	if b == nil {
		return 1
	}
	return a.key - b.key
}
func sortNullArray(v ...any) *ReferenceArray {
	RegisterJavaType("campaign.SortNullValue", ObjectTypeID)
	return ReferenceArrayLiteral("campaign.SortNullValue", v...)
}
func sortNullRequire(t *testing.T, f func()) bool {
	t.Helper()
	if p := sortNullPanic(f); p != nil {
		t.Errorf("unexpected exception: %T %v", p, p)
		return false
	}
	return true
}
func TestCampaignSortArrayNullSourcePointers(t *testing.T) {
	a, b, c := sortNullNew(3, "a"), sortNullNew(1, "b"), sortNullNew(1, "c")
	for _, input := range [][]any{{a, nil, b, nil}, {nil, nil}, {a, c, nil, b}} {
		array := sortNullArray(input...)
		alias := array
		descriptor := array.JavaArrayTypeID()
		calls := 0
		if !sortNullRequire(t, func() {
			SortArrayWith(array, Comparator[*sortNullValue](func(x, y *sortNullValue) int32 { calls++; return sortNullLess(x, y) }))
		}) {
			continue
		}
		if array != alias || array.JavaArrayTypeID() != descriptor || calls == 0 {
			t.Fatal("array identity/descriptor/callback changed")
		}
		if len(input) == 4 && input[0] == a && input[1] == nil {
			if !reflect.DeepEqual(array.elements, []any{nil, nil, b, a}) || calls != 6 {
				t.Fatalf("mixed output/callbacks=%v/%d", array.elements, calls)
			}
		}
		if len(input) == 4 && input[1] == c {
			if !reflect.DeepEqual(array.elements, []any{nil, c, b, a}) {
				t.Fatal("stable equal reference identity")
			}
		}
	}
	var absent *sortNullValue
	array := NewReferenceArray(2, "campaign.SortNullValue")
	ReferenceArraySet(array, 0, absent)
	ReferenceArraySet(array, 1, a)
	if array.elements[0] != nil {
		t.Fatal("store did not canonicalize typed null")
	}
	if !sortNullRequire(t, func() { SortArrayWith(array, Comparator[*sortNullValue](sortNullLess)) }) {
		return
	}
	if array.elements[0] != nil || array.elements[1] != a {
		t.Fatal("typed-null alias lost")
	}
}
func TestCampaignSortArrayNullReferenceRepresentations(t *testing.T) {
	t.Run("canonicalString", func(t *testing.T) {
		a, b := JavaStringLiteralUTF16([]uint16{'b'}), JavaStringLiteralUTF16([]uint16{'a'})
		v := ReferenceArrayLiteral(StringTypeID, a, nil, b)
		calls := 0
		if !sortNullRequire(t, func() {
			SortArrayWith(v, Comparator[*JavaString](func(x, y *JavaString) int32 {
				calls++
				if x == nil {
					if y == nil {
						return 0
					}
					return -1
				}
				if y == nil {
					return 1
				}
				return x.CompareTo(y)
			}))
		}) {
			return
		}
		if !reflect.DeepEqual(v.elements, []any{nil, b, a}) || calls == 0 {
			t.Fatal("String null/identity")
		}
	})
	t.Run("ObjectInterface", func(t *testing.T) {
		a := NewObject()
		v := ReferenceArrayLiteral(ObjectTypeID, a, nil)
		calls := 0
		if !sortNullRequire(t, func() {
			SortArrayWith(v, Comparator[any](func(x, y any) int32 {
				calls++
				if x == nil {
					return -1
				}
				if y == nil {
					return 1
				}
				return 0
			}))
		}) {
			return
		}
		if !reflect.DeepEqual(v.elements, []any{nil, a}) || calls != 1 {
			t.Fatal("erased null")
		}
	})
	t.Run("boxedReference", func(t *testing.T) {
		a, b := NewInteger(9), NewInteger(2)
		v := ReferenceArrayLiteral(IntegerTypeID, a, nil, b)
		if !sortNullRequire(t, func() {
			SortArrayWith(v, Comparator[*Integer](func(x, y *Integer) int32 {
				if x == nil {
					return -1
				}
				if y == nil {
					return 1
				}
				return x.IntValue() - y.IntValue()
			}))
		}) {
			return
		}
		if !reflect.DeepEqual(v.elements, []any{nil, b, a}) {
			t.Fatal("boxed references")
		}
	})
	t.Run("nestedReferenceArray", func(t *testing.T) {
		a, b := NewReferenceArray(3, StringTypeID), NewReferenceArray(1, StringTypeID)
		v := ReferenceArrayLiteral(ArrayTypeID(StringTypeID), a, nil, b)
		if !sortNullRequire(t, func() {
			SortArrayWith(v, Comparator[*ReferenceArray](func(x, y *ReferenceArray) int32 {
				if x == nil {
					return -1
				}
				if y == nil {
					return 1
				}
				return ReferenceArrayLength(x) - ReferenceArrayLength(y)
			}))
		}) {
			return
		}
		if !reflect.DeepEqual(v.elements, []any{nil, b, a}) {
			t.Fatal("nested reference descriptors")
		}
	})
	t.Run("nestedPrimitiveArrayReference", func(t *testing.T) {
		a, b := NewPrimitiveArray[int32](3, PrimitiveIntTypeID), NewPrimitiveArray[int32](1, PrimitiveIntTypeID)
		v := ReferenceArrayLiteral(ArrayTypeID(PrimitiveIntTypeID), a, nil, b)
		if !sortNullRequire(t, func() {
			SortArrayWith(v, Comparator[*PrimitiveArray[int32]](func(x, y *PrimitiveArray[int32]) int32 {
				if x == nil {
					return -1
				}
				if y == nil {
					return 1
				}
				return int32(len(x.Elements) - len(y.Elements))
			}))
		}) {
			return
		}
		if !reflect.DeepEqual(v.elements, []any{nil, b, a}) {
			t.Fatal("primitive array is a reference element")
		}
	})
}
func sortNullRefusal[T any](t *testing.T) {
	t.Helper()
	calls := 0
	v := ReferenceArrayLiteral(ObjectTypeID, nil, nil)
	p := sortNullPanic(func() { SortArrayWith(v, Comparator[T](func(a, b T) int32 { calls++; return 0 })) })
	if !CaughtAs(p, "ClassCastException") || calls != 0 || v.elements[0] != nil || v.elements[1] != nil {
		t.Fatalf("refusal=%v callbacks=%d", p, calls)
	}
}
func TestCampaignSortArrayNullRefusals(t *testing.T) {
	t.Run("int", sortNullRefusal[int32])
	t.Run("bool", sortNullRefusal[bool])
	t.Run("float", sortNullRefusal[float64])
	t.Run("struct", sortNullRefusal[struct{}])
	t.Run("string", sortNullRefusal[string])
	t.Run("slice", sortNullRefusal[[]any])
	t.Run("map", sortNullRefusal[map[string]any])
	t.Run("function", sortNullRefusal[func()])
	t.Run("channel", sortNullRefusal[chan int])
	calls := 0
	c := Comparator[*sortNullValue](func(a, b *sortNullValue) int32 { calls++; return 0 })
	for _, v := range []any{NewPrimitiveArray[int32](2, PrimitiveIntTypeID), []any{nil, nil}} {
		if p := sortNullPanic(func() { SortArrayWith(v, c) }); !CaughtAs(p, "IllegalArgumentException") {
			t.Fatal(p)
		}
	}
	for _, v := range []any{nil, (*ReferenceArray)(nil)} {
		if p := sortNullPanic(func() { SortArrayWith(v, c) }); !CaughtAs(p, "NullPointerException") {
			t.Fatal(p)
		}
	}
	// An opaque Object reference remains foreign to this source comparator.
	foreign := NewObject()
	array := ReferenceArrayLiteral(ObjectTypeID, foreign, foreign)
	if p := sortNullPanic(func() { SortArrayWith(array, c) }); !CaughtAs(p, "ClassCastException") || calls != 0 {
		t.Fatal("foreign value admitted", p, calls)
	}
	registerReferenceArrayTestTypes()
	child := newReferenceArrayChild(2)
	v := ReferenceArrayLiteral(testChildType, child, child)
	views := 0
	child.info.view = func(TypeID) any { views++; return child.referenceArrayBaseView }
	if p := sortNullPanic(func() {
		SortArrayWith(v, Comparator[*referenceArrayBaseView](func(a, b *referenceArrayBaseView) int32 { calls++; return 0 }))
	}); !CaughtAs(p, "ClassCastException") || views != 0 || calls != 0 {
		t.Fatal("non-null view adaptation widened", p, views, calls)
	}
}
func TestCampaignSortArrayNullNoEagerValidation(t *testing.T) {
	calls := 0
	c := Comparator[int32](func(a, b int32) int32 { calls++; return 0 })
	for _, array := range []*ReferenceArray{ReferenceArrayLiteral(ObjectTypeID), ReferenceArrayLiteral(ObjectTypeID, nil), ReferenceArrayLiteral(ObjectTypeID, NewObject())} {
		if !sortNullRequire(t, func() { SortArrayWith(array, c) }) {
			return
		}
	}
	if calls != 0 {
		t.Fatal("eager comparison")
	}
	if p := sortNullPanic(func() { SortArrayWith[*sortNullValue](sortNullArray(nil, nil), nil) }); !CaughtAs(p, "NullPointerException") {
		t.Fatal("natural null ordering changed", p)
	}
}
func TestCampaignSortArrayNullExecutionOrderAndAbrupt(t *testing.T) {
	a, b := sortNullNew(3, "a"), sortNullNew(1, "b")
	v := sortNullArray(a, nil, b, nil)
	alias := v
	e := NewExecution()
	local := NewThreadLocal[string]()
	local.Set(e, "one")
	lock := NewObject()
	calls := ""
	c := Comparator[*sortNullValue](func(x, y *sortNullValue) int32 {
		g := MonitorEnterExecution(e, lock)
		defer MonitorExitExecution(g)
		if !ThreadHoldsLockExecution(e, lock) || local.Get(e) != "one" {
			t.Fatal("execution state lost")
		}
		label := func(z *sortNullValue) string {
			if z == nil {
				return "n"
			}
			return z.label
		}
		calls += label(x) + label(y) + ";"
		return sortNullLess(x, y)
	})
	if !sortNullRequire(t, func() { SortArrayWith(v, c, e) }) {
		return
	}
	if calls != "na;bn;ba;bn;nb;nn;" || !reflect.DeepEqual(alias.elements, []any{nil, nil, b, a}) || ThreadHoldsLockExecution(e, lock) {
		t.Fatal("comparison order/backing/monitor", calls)
	}
	v = sortNullArray(a, nil, b, nil)
	alias = v
	marker := NewIllegalStateException("sort marker")
	count := 0
	p := sortNullPanic(func() {
		SortArrayWith(v, Comparator[*sortNullValue](func(x, y *sortNullValue) int32 {
			count++
			if count == 2 {
				panic(marker)
			}
			return sortNullLess(x, y)
		}), e)
	})
	if p != marker || count != 2 || !reflect.DeepEqual(alias.elements, []any{a, nil, b, nil}) {
		t.Fatal("abrupt identity/partial backing", p, count, alias.elements)
	}
}
func TestCampaignSortArrayNullFutureMutation(t *testing.T) {
	a, b, c := sortNullNew(3, "a"), sortNullNew(2, "b"), sortNullNew(1, "c")
	v := sortNullArray(a, nil, b)
	calls := 0
	sawC := false
	if !sortNullRequire(t, func() {
		SortArrayWith(v, Comparator[*sortNullValue](func(x, y *sortNullValue) int32 {
			calls++
			if calls == 1 {
				ReferenceArraySet(v, 2, c)
			}
			if x == c || y == c {
				sawC = true
			}
			return sortNullLess(x, y)
		}))
	}) {
		return
	}
	if !sawC || !reflect.DeepEqual(v.elements, []any{nil, c, a}) {
		t.Fatal("future slot snapshot/alias", calls, sawC, v.elements)
	}
}

package stdjava

import (
	"sync"
	"testing"
)

const updaterOwner TypeID = "updater.fixture.Owner"

type updaterTarget struct {
	i, l, r, ordinary, shadow VolatileFieldCell
	info                      *ObjectInfo
}

func (v *updaterTarget) JavaDynamicTypeID() TypeID {
	if v.info != nil {
		return v.info.DynamicType()
	}
	return updaterOwner
}
func (v *updaterTarget) JavaObjectInfo() *ObjectInfo { return v.info }
func (v *updaterTarget) Java2goVolatileCellExecution(e *Execution) *VolatileFieldCell {
	requireExecution(e)
	return &v.i
}
func registerUpdaterFixture() {
	RegisterJavaType(updaterOwner, ObjectTypeID)
	RegisterClassDescriptor(ClassDescriptor{Type: updaterOwner, HasModifiers: true, Modifiers: 1, Fields: []FieldDescriptor{
		{Name: "i", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*updaterTarget).i }, Get: func(_ *Execution, v any) any { return VolatileLoad[int32](&v.(*updaterTarget).i) }, Set: func(_ *Execution, v, x any) { VolatileStore(&v.(*updaterTarget).i, x.(int32)) }},
		{Name: "l", Type: PrimitiveLongTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*updaterTarget).l }},
		{Name: "r", Type: ObjectTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*updaterTarget).r }},
		{Name: "plain", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 1},
		{Name: "private", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 66, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*updaterTarget).i }},
		{Name: "static", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 73},
		{Name: "final", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 81},
		{Name: "unlowered", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 65},
	}})
}
func updaterFailure(t *testing.T, want TypeID, call func()) any {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); call() }()
	actual, known := ObjectDynamicType(failure)
	if !known || actual != want {
		t.Fatalf("panic type %v (%T), want %v", actual, failure, want)
	}
	return failure
}
func TestAtomicFieldUpdaterNumericAndReflectionCoherence(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	e := NewExecution()
	i := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "i", updaterOwner)
	j := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "i", updaterOwner)
	if i.Get(target) != 0 || !i.CompareAndSet(target, 0, 7) || j.Get(target) != 7 {
		t.Fatal("zero or independent updater storage")
	}
	if i.GetAndSet(target, 4) != 7 || i.GetAndAdd(target, -3) != 4 || i.AddAndGet(target, 2) != 3 {
		t.Fatal("exchange/add return values")
	}
	i.Set(target, 2147483647)
	if i.IncrementAndGet(target) != -2147483648 {
		t.Fatal("integer overflow")
	}
	i.LazySetExecution(e, target, 42)
	field := ClassLiteral(updaterOwner).GetDeclaredField("i")
	if value := field.GetExecution(e, target); value.(*Integer).IntValue() != 42 {
		t.Fatal("reflection misses updater value")
	}
	field.SetExecution(e, target, BoxInteger(18))
	if i.Get(target) != 18 {
		t.Fatal("updater misses reflection value")
	}
	if cell := ReflectGeneratedVolatileFieldCellExecution(e, target, updaterOwner, "Java2goVolatileCellExecution"); cell != &target.i {
		t.Fatal("accessor lost actual cell")
	}
	l := NewAtomicLongFieldUpdater(ClassLiteral(updaterOwner), "l", updaterOwner)
	l.Set(target, 9223372036854775807)
	if l.GetAndIncrement(target) != 9223372036854775807 || l.Get(target) != -9223372036854775808 {
		t.Fatal("long overflow")
	}
	if l.GetAndSet(target, 9) != -9223372036854775808 || !l.WeakCompareAndSet(target, 9, 6) || l.DecrementAndGet(target) != 5 {
		t.Fatal("long operations")
	}
}
func TestAtomicFieldUpdaterReferenceIdentityAndTypeChecks(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
	a, b := BoxInteger(1000), BoxInteger(1000)
	u.Set(target, a)
	if u.CompareAndSet(target, b, nil) || !JavaReferenceEqual(u.Get(target), a) {
		t.Fatal("CAS used value equality")
	}
	if !u.CompareAndSet(target, a, b) || !JavaReferenceEqual(u.GetAndSet(target, nil), b) {
		t.Fatal("CAS/exchange identity")
	}
	var typedNil *Integer
	if !u.CompareAndSet(target, typedNil, a) {
		t.Fatal("typed nil expected not Java null")
	}
	updaterFailure(t, BuiltinThrowableTypeID("ClassCastException"), func() { u.CompareAndSet(target, a, int32(1)) })
	if !JavaReferenceEqual(u.Get(target), a) {
		t.Fatal("failed value check mutated storage")
	}
	wrongExpected := NewReferenceArray(0, ObjectTypeID)
	if u.CompareAndSet(target, wrongExpected, nil) {
		t.Fatal("unrelated expected value matched")
	}
}
func TestAtomicFieldUpdaterFactoryAndReceiverRejections(t *testing.T) {
	registerUpdaterFixture()
	owner := ClassLiteral(updaterOwner)
	for _, name := range []string{"plain", "static", "final"} {
		updaterFailure(t, BuiltinThrowableTypeID("IllegalArgumentException"), func() { NewAtomicIntegerFieldUpdater(owner, name, updaterOwner) })
	}
	updaterFailure(t, BuiltinThrowableTypeID("IllegalArgumentException"), func() { NewAtomicLongFieldUpdater(owner, "i", updaterOwner) })
	updaterFailure(t, BuiltinThrowableTypeID("ClassCastException"), func() { NewAtomicReferenceFieldUpdater(owner, ClassLiteral(StringTypeID), "r", updaterOwner) })
	updaterFailure(t, BuiltinThrowableTypeID("UnsupportedOperationException"), func() { NewAtomicIntegerFieldUpdater(owner, "unlowered", updaterOwner) })
	missing := updaterFailure(t, BuiltinThrowableTypeID("RuntimeException"), func() { NewAtomicIntegerFieldUpdater(owner, "missing", updaterOwner) })
	if cause, ok := ObjectDynamicType(GetCause(missing)); !ok || cause != BuiltinThrowableTypeID("NoSuchFieldException") {
		t.Fatal("missing field cause")
	}
	denied := updaterFailure(t, BuiltinThrowableTypeID("RuntimeException"), func() { NewAtomicIntegerFieldUpdater(owner, "private", "other.Caller") })
	if cause, ok := ObjectDynamicType(GetCause(denied)); !ok || cause != BuiltinThrowableTypeID("IllegalAccessException") {
		t.Fatal("access cause")
	}
	u := NewAtomicIntegerFieldUpdater(owner, "private", updaterOwner)
	updaterFailure(t, BuiltinThrowableTypeID("ClassCastException"), func() { u.Get(nil) })
	updaterFailure(t, BuiltinThrowableTypeID("ClassCastException"), func() { u.Set(BoxInteger(3), 2) })
	sub := TypeID("updater.fixture.Sub")
	RegisterJavaType(sub, updaterOwner)
	RegisterClassDescriptor(ClassDescriptor{Type: sub})
	updaterFailure(t, BuiltinThrowableTypeID("RuntimeException"), func() { NewAtomicIntegerFieldUpdater(ClassLiteral(sub), "i", sub) })
}
func TestAtomicFieldUpdaterCallbacksRetryAbruptAndExecution(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	e := NewExecution()
	u := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "i", updaterOwner)
	calls := 0
	previous := u.GetAndUpdateExecution(e, target, func(actual *Execution, value int32) int32 {
		if actual != e {
			t.Fatal("execution lost")
		}
		calls++
		if calls == 1 {
			u.SetExecution(e, target, 10)
		}
		return value + 1
	})
	if previous != 10 || u.Get(target) != 11 || calls != 2 {
		t.Fatalf("callback retry: previous=%d current=%d calls=%d", previous, u.Get(target), calls)
	}
	marker := NewIllegalStateException("marker")
	got := updaterFailure(t, BuiltinThrowableTypeID("IllegalStateException"), func() { u.UpdateAndGetExecution(e, target, func(*Execution, int32) int32 { panic(marker) }) })
	if !JavaReferenceEqual(got, marker) || u.Get(target) != 11 {
		t.Fatal("callback abrupt completion changed identity/storage")
	}
	if u.AccumulateAndGetExecution(e, target, 3, func(actual *Execution, a, b int32) int32 {
		if actual != e {
			t.Fatal("execution lost")
		}
		return a * b
	}) != 33 {
		t.Fatal("accumulate result")
	}
}
func TestAtomicFieldUpdaterConcurrentCASAndSharedStorage(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	a := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "i", updaterOwner)
	b := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "i", updaterOwner)
	var workers sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		workers.Add(1)
		go func(n int) {
			defer workers.Done()
			for i := 0; i < 200; i++ {
				if n&1 == 0 {
					a.GetAndAdd(target, 1)
				} else {
					b.GetAndAdd(target, 1)
				}
			}
		}(worker)
	}
	workers.Wait()
	if VolatileLoad[int32](&target.i) != 800 {
		t.Fatal("lost atomic updates")
	}
}
func TestAtomicFieldUpdaterDeclaringOwnerShadowAndProtectedReceiver(t *testing.T) {
	registerUpdaterFixture()
	sub := TypeID("foreign.Child")
	RegisterJavaType(sub, updaterOwner)
	RegisterClassDescriptor(ClassDescriptor{Type: sub, HasModifiers: true, Modifiers: 1, Fields: []FieldDescriptor{{Name: "i", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 65, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*updaterTarget).shadow }}}})
	target := &updaterTarget{}
	target.info = NewObjectInfo(sub, func(id TypeID) any {
		if id == sub || id == updaterOwner || id == ObjectTypeID {
			return target
		}
		return nil
	})
	base := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "i", updaterOwner)
	child := NewAtomicIntegerFieldUpdater(ClassLiteral(sub), "i", sub)
	base.Set(target, 9)
	child.Set(target, 3)
	if base.Get(target) != 9 || child.Get(target) != 3 {
		t.Fatal("shadow field replaced declaring storage")
	}
	d := classDescriptor(updaterOwner)
	d.Fields = append(d.Fields, FieldDescriptor{Name: "protected", Type: PrimitiveIntTypeID, HasModifiers: true, Modifiers: 68, VolatileCell: func(_ *Execution, v any) *VolatileFieldCell { return &v.(*updaterTarget).i }})
	RegisterClassDescriptor(d)
	protected := NewAtomicIntegerFieldUpdater(ClassLiteral(updaterOwner), "protected", sub)
	protected.Set(target, 11)
	if base.Get(target) != 11 {
		t.Fatal("protected updater uses different storage")
	}
	denied := updaterFailure(t, BuiltinThrowableTypeID("RuntimeException"), func() { protected.Get(&updaterTarget{}) })
	if cause, ok := ObjectDynamicType(GetCause(denied)); !ok || cause != BuiltinThrowableTypeID("IllegalAccessException") {
		t.Fatal("protected receiver cause")
	}
	updaterFailure(t, BuiltinThrowableTypeID("NullPointerException"), func() { protected.Get(nil) })
}
func TestAtomicFieldUpdaterReferenceViewsAndCanonicalStrings(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
	dynamic := TypeID("updater.fixture.Dynamic")
	RegisterJavaType(dynamic, updaterOwner)
	base, derived := &updaterTarget{}, &updaterTarget{}
	info := NewObjectInfo(dynamic, func(id TypeID) any {
		switch id {
		case dynamic:
			return derived
		case updaterOwner, ObjectTypeID:
			return base
		}
		return nil
	})
	base.info = info
	derived.info = info
	u.Set(target, base)
	if !u.CompareAndSet(target, derived, nil) {
		t.Fatal("same allocation superclass view mismatched")
	}
	a, b := NewJavaStringUTF16([]uint16{'s'}), NewJavaStringUTF16([]uint16{'s'})
	u.Set(target, a)
	if u.CompareAndSet(target, b, nil) || !u.CompareAndSet(target, a, b) {
		t.Fatal("canonical string identity lost")
	}
	u.Set(target, nil)
	if VolatileLoad[*JavaString](&target.r) != nil {
		t.Fatal("typed null volatile load")
	}
	var zero VolatileFieldCell
	if !StringIsNull(VolatileLoad[string](&zero)) {
		t.Fatal("native String Java null zero")
	}
	u.Set(target, derived)
	if got := VolatileLoad[*updaterTarget](&target.r, updaterOwner); got != base {
		t.Fatal("requested reference view lost")
	}
}
func TestAtomicFieldUpdaterReferenceCallbackRetryAndWrongUpdate(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	e := NewExecution()
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
	a, b, c := BoxInteger(1000), BoxInteger(1001), BoxInteger(1002)
	u.Set(target, a)
	calls := 0
	previous := u.GetAndUpdateExecution(e, target, func(actual *Execution, value any) any {
		if actual != e {
			t.Fatal("execution lost")
		}
		calls++
		if calls == 1 {
			u.SetExecution(e, target, b)
		}
		return c
	})
	if calls != 2 || !JavaReferenceEqual(previous, b) || !JavaReferenceEqual(u.Get(target), c) {
		t.Fatal("reference callback retry/return")
	}
	updaterFailure(t, BuiltinThrowableTypeID("ClassCastException"), func() { u.UpdateAndGetExecution(e, target, func(*Execution, any) any { return int32(1) }) })
	if !JavaReferenceEqual(u.Get(target), c) {
		t.Fatal("wrong callback result changed storage")
	}
}

type updaterIdentityProbe struct {
	info    *ObjectInfo
	touched *VolatileFieldCell
}

func (*updaterIdentityProbe) JavaDynamicTypeID() TypeID { return ObjectTypeID }
func (p *updaterIdentityProbe) JavaObjectInfo() *ObjectInfo {
	VolatileStore(p.touched, int32(1))
	return p.info
}
func TestAtomicFieldUpdaterIdentityProvidersRunOutsideLock(t *testing.T) {
	registerUpdaterFixture()
	target := &updaterTarget{}
	var touched VolatileFieldCell
	u := NewAtomicReferenceFieldUpdater(ClassLiteral(updaterOwner), ClassLiteral(ObjectTypeID), "r", updaterOwner)
	a, b := &updaterIdentityProbe{touched: &touched}, &updaterIdentityProbe{touched: &touched}
	info := NewObjectInfo(ObjectTypeID, func(TypeID) any { return a })
	a.info = info
	b.info = info
	u.Set(target, a)
	if !u.CompareAndSet(target, b, nil) || VolatileLoad[int32](&touched) != 1 {
		t.Fatal("identity provider reentrancy failed")
	}
}

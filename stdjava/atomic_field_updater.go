package stdjava

import (
	"reflect"
	"strings"
	"sync"
)

// VolatileFieldCell is the actual storage of one generated volatile field.
// It must never be copied after first use. No lock surrounds a raw Go field.
// A single synchronization order covers cells of every primitive/reference
// type. Callbacks and Java object view resolution run outside this lock.
type volatileFieldStamp struct{ used byte }
type VolatileFieldCell struct {
	value any
	stamp *volatileFieldStamp
}

var volatileFieldOrder sync.Mutex

func volatileFieldRead(cell *VolatileFieldCell) any {
	if cell == nil {
		panic(NewNullPointerException("volatile field cell is null"))
	}
	volatileFieldOrder.Lock()
	value := cell.value
	volatileFieldOrder.Unlock()
	return value
}
func volatileFieldWrite(cell *VolatileFieldCell, value any) {
	if cell == nil {
		panic(NewNullPointerException("volatile field cell is null"))
	}
	volatileFieldOrder.Lock()
	cell.value = value
	cell.stamp = &volatileFieldStamp{}
	volatileFieldOrder.Unlock()
}

// requestedType resolves a stored reference into its statically selected Java
// view when generated inheritance/generics give the allocation several Go views.
func VolatileLoad[T any](cell *VolatileFieldCell, requestedType ...TypeID) T {
	value := volatileFieldRead(cell)
	var zero T
	if nilJavaReference(value) {
		if reflect.TypeOf((*T)(nil)).Elem().Kind() == reflect.String {
			if sentinel, ok := any(NullString()).(T); ok {
				return sentinel
			}
		}
		return zero
	}
	if len(requestedType) == 1 && !isPrimitiveTypeID(requestedType[0]) {
		return ObjectView[T](value, requestedType[0])
	}
	if result, ok := value.(T); ok {
		return result
	}
	if len(requestedType) == 1 {
		return ObjectView[T](value, requestedType[0])
	}
	panic(NewUnsupportedOperationException("volatile load requires a compatible Java view"))
}
func VolatileStore[T any](cell *VolatileFieldCell, value T) { volatileFieldWrite(cell, value) }

// Generated field accessors return the address of the cell declared by this
// class, rather than a subclass shadow or a separately allocated updater cell.
func ReflectGeneratedVolatileFieldCellExecution(execution *Execution, receiver any, declaringType TypeID, method string) *VolatileFieldCell {
	requireExecution(execution)
	target := reflectionReceiver(receiver, ClassLiteral(declaringType)).MethodByName(method)
	if !target.IsValid() || target.Type().NumIn() != 1 || target.Type().In(0) != reflect.TypeOf((*Execution)(nil)) || target.Type().NumOut() != 1 || target.Type().Out(0) != reflect.TypeOf((*VolatileFieldCell)(nil)) {
		panic(NewUnsupportedOperationException("generated volatile field cell accessor is unavailable"))
	}
	value := target.Call([]reflect.Value{reflect.ValueOf(execution)})[0]
	if value.IsNil() {
		panic(NewUnsupportedOperationException("generated volatile field cell is unavailable"))
	}
	return value.Interface().(*VolatileFieldCell)
}

type atomicFieldUpdater struct {
	owner, receiverType TypeID
	field               FieldDescriptor
}

func atomicUpdaterPackage(id TypeID) string {
	name := string(id)
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[:i]
	}
	return ""
}
func atomicUpdaterAccess(owner *Class, field *Field, caller TypeID) TypeID {
	id := owner.TypeID()
	mods := field.GetModifiers()
	samePackage := atomicUpdaterPackage(id) == atomicUpdaterPackage(caller)
	allowedClass := caller == id || samePackage || owner.GetModifiers()&1 != 0
	allowedMember := mods&1 != 0 || caller == id
	if mods&2 == 0 && samePackage {
		allowedMember = true
	}
	if mods&4 != 0 && JavaTypeAssignable(caller, id) {
		allowedMember = true
	}
	if caller == "" || !allowedClass || !allowedMember {
		panic(NewRuntimeException(reflectionException("IllegalAccessException", "updater caller cannot access field")))
	}
	if mods&4 != 0 && !samePackage && JavaTypeAssignable(caller, id) {
		return caller
	}
	return id
}
func newAtomicFieldUpdater(owner *Class, valueType TypeID, name string, caller TypeID, reference bool) atomicFieldUpdater {
	// The JDK wraps declared lookup/access exceptions, but type/volatile
	// validation failures following lookup retain their own exception class.
	var field *Field
	var receiverType TypeID
	func() {
		defer func() {
			if failure := recover(); failure != nil {
				panic(NewRuntimeException(failure))
			}
		}()
		field = owner.GetDeclaredField(name)
	}()
	receiverType = atomicUpdaterAccess(owner, field, caller)
	if field.descriptor.Type != valueType {
		if reference {
			panic(NewClassCastException("updater value class differs from declared field type"))
		}
		panic(NewIllegalArgumentException("updater requires exact primitive field type"))
	}
	if reference && isPrimitiveTypeID(valueType) {
		panic(NewIllegalArgumentException("updater requires reference field"))
	}
	mods := field.GetModifiers()
	if mods&64 == 0 {
		panic(NewIllegalArgumentException("updater requires volatile field"))
	}
	if mods&(8|16) != 0 {
		panic(NewIllegalArgumentException("updater requires writable instance field"))
	}
	if field.descriptor.VolatileCell == nil {
		panic(NewUnsupportedOperationException("volatile storage metadata is unavailable"))
	}
	return atomicFieldUpdater{owner: owner.TypeID(), receiverType: receiverType, field: field.descriptor}
}
func (u *atomicFieldUpdater) cell(execution *Execution, receiver any) *VolatileFieldCell {
	requireExecution(execution)
	id, known := ObjectDynamicType(receiver)
	if !known || !JavaTypeAssignable(id, u.receiverType) {
		if u.receiverType != u.owner {
			if nilJavaReference(receiver) {
				panic(NewNullPointerException("protected updater receiver is null"))
			}
			panic(NewRuntimeException(reflectionException("IllegalAccessException", "protected updater receiver has wrong class")))
		}
		panic(NewClassCastException("updater receiver has wrong class"))
	}
	view := reflectionReceiver(receiver, ClassLiteral(u.owner)).Interface()
	cell := u.field.VolatileCell(execution, view)
	if cell == nil {
		panic(NewUnsupportedOperationException("volatile storage metadata returned no cell"))
	}
	return cell
}
func atomicUpdaterEqual(left, right any, id TypeID) bool {
	if left == nil {
		switch id {
		case PrimitiveIntTypeID:
			left = int32(0)
		case PrimitiveLongTypeID:
			left = int64(0)
		}
	}
	switch id {
	case PrimitiveIntTypeID:
		l, lok := left.(int32)
		r, rok := right.(int32)
		return lok && rok && l == r
	case PrimitiveLongTypeID:
		l, lok := left.(int64)
		r, rok := right.(int64)
		return lok && rok && l == r
	default:
		return JavaReferenceEqual(left, right)
	}
}
func (u *atomicFieldUpdater) cas(execution *Execution, receiver, expect, update any) bool {
	cell := u.cell(execution, receiver)
	for {
		volatileFieldOrder.Lock()
		previous, stamp := cell.value, cell.stamp
		volatileFieldOrder.Unlock()
		if !atomicUpdaterEqual(previous, expect, u.field.Type) {
			return false
		}
		// Java identity providers are arbitrary generated methods. They run
		// outside the lock; a non-zero-sized stamp detects intervening writes
		// without a wrapping counter or retaining old values.
		volatileFieldOrder.Lock()
		if cell.stamp != stamp {
			volatileFieldOrder.Unlock()
			continue
		}
		cell.value = update
		cell.stamp = &volatileFieldStamp{}
		volatileFieldOrder.Unlock()
		return true
	}
}
func (u *atomicFieldUpdater) swap(execution *Execution, receiver, update any) any {
	cell := u.cell(execution, receiver)
	volatileFieldOrder.Lock()
	previous := cell.value
	cell.value = update
	cell.stamp = &volatileFieldStamp{}
	volatileFieldOrder.Unlock()
	return previous
}

const AtomicReferenceFieldUpdaterTypeID TypeID = "java.util.concurrent.atomic.AtomicReferenceFieldUpdater"

type AtomicReferenceFieldUpdater struct{ core atomicFieldUpdater }

func (*AtomicReferenceFieldUpdater) JavaDynamicTypeID() TypeID {
	return AtomicReferenceFieldUpdaterTypeID
}
func NewAtomicReferenceFieldUpdater(owner, valueClass *Class, name string, caller TypeID) *AtomicReferenceFieldUpdater {
	// A null value class differs from every declared class in the JDK.
	valueType := TypeID("")
	if valueClass != nil {
		valueType = valueClass.TypeID()
	}
	return &AtomicReferenceFieldUpdater{core: newAtomicFieldUpdater(owner, valueType, name, caller, true)}
}
func (u *AtomicReferenceFieldUpdater) checkedValue(value any) any {
	if nilJavaReference(value) {
		return nil
	}
	if !ObjectInstanceOf(value, u.core.field.Type) {
		panic(NewClassCastException("updater value has wrong class"))
	}
	// Native Go strings have value-backed identity. They cannot implement a
	// reference updater's allocation CAS; canonical *JavaString values can.
	if _, native := value.(string); native {
		panic(NewUnsupportedOperationException("reference updater requires canonical Java String identity"))
	}
	return reflectionFieldValue(value, u.core.field.Type)
}
func (u *AtomicReferenceFieldUpdater) Get(receiver any) any {
	return u.GetExecution(NewExecution(), receiver)
}
func (u *AtomicReferenceFieldUpdater) GetExecution(e *Execution, receiver any) any {
	value := volatileFieldRead(u.core.cell(e, receiver))
	if _, native := value.(string); native && !nilJavaReference(value) {
		panic(NewUnsupportedOperationException("reference updater requires canonical Java String identity"))
	}
	return reflectionBox(value, u.core.field.Type)
}
func (u *AtomicReferenceFieldUpdater) Set(receiver, value any) {
	u.SetExecution(NewExecution(), receiver, value)
}
func (u *AtomicReferenceFieldUpdater) SetExecution(e *Execution, receiver, value any) {
	cell := u.core.cell(e, receiver)
	update := u.checkedValue(value)
	volatileFieldWrite(cell, update)
}
func (u *AtomicReferenceFieldUpdater) LazySet(receiver, value any) { u.Set(receiver, value) }
func (u *AtomicReferenceFieldUpdater) LazySetExecution(e *Execution, receiver, value any) {
	u.SetExecution(e, receiver, value)
}
func (u *AtomicReferenceFieldUpdater) CompareAndSet(receiver, expect, update any) bool {
	return u.CompareAndSetExecution(NewExecution(), receiver, expect, update)
}
func (u *AtomicReferenceFieldUpdater) CompareAndSetExecution(e *Execution, receiver, expect, update any) bool {
	u.core.cell(e, receiver)
	value := u.checkedValue(update)
	// Only the new value is type checked by JDK reference CAS. An unrelated
	// expected value simply fails identity comparison.
	if _, native := expect.(string); native && !nilJavaReference(expect) {
		panic(NewUnsupportedOperationException("reference updater requires canonical Java String identity"))
	}
	return u.core.cas(e, receiver, expect, value)
}
func (u *AtomicReferenceFieldUpdater) WeakCompareAndSet(receiver, expect, update any) bool {
	return u.CompareAndSet(receiver, expect, update)
}
func (u *AtomicReferenceFieldUpdater) WeakCompareAndSetExecution(e *Execution, receiver, expect, update any) bool {
	return u.CompareAndSetExecution(e, receiver, expect, update)
}
func (u *AtomicReferenceFieldUpdater) GetAndSet(receiver, update any) any {
	return u.GetAndSetExecution(NewExecution(), receiver, update)
}
func (u *AtomicReferenceFieldUpdater) GetAndSetExecution(e *Execution, receiver, update any) any {
	cell := u.core.cell(e, receiver)
	value := u.checkedValue(update)
	for {
		volatileFieldOrder.Lock()
		previous, stamp := cell.value, cell.stamp
		volatileFieldOrder.Unlock()
		if _, native := previous.(string); native && !nilJavaReference(previous) {
			panic(NewUnsupportedOperationException("reference updater requires canonical Java String identity"))
		}
		// Exchange does not compare Java identity: opaque references may have
		// conservative, non-reflexive identity. Resolve the returned view before
		// committing, outside the lock, then retry if that resolution changed this cell.
		result := reflectionBox(previous, u.core.field.Type)
		volatileFieldOrder.Lock()
		if cell.stamp != stamp {
			volatileFieldOrder.Unlock()
			continue
		}
		cell.value = value
		cell.stamp = &volatileFieldStamp{}
		volatileFieldOrder.Unlock()
		return result
	}
}
func (u *AtomicReferenceFieldUpdater) updateExecution(e *Execution, receiver any, update func(*Execution, any) any, returnNew bool) any {
	cell := u.core.cell(e, receiver)
	if update == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	for {
		volatileFieldOrder.Lock()
		previous, stamp := cell.value, cell.stamp
		volatileFieldOrder.Unlock()
		if _, native := previous.(string); native && !nilJavaReference(previous) {
			panic(NewUnsupportedOperationException("reference updater requires canonical Java String identity"))
		}
		// Internal RMW uses the write stamp rather than external Java identity.
		// View resolution, the callback and update validation can invoke user code;
		// keep them outside the lock and retry if any of them changed this cell.
		result := reflectionBox(previous, u.core.field.Type)
		next := update(e, result)
		value := u.checkedValue(next)
		volatileFieldOrder.Lock()
		if cell.stamp != stamp {
			volatileFieldOrder.Unlock()
			continue
		}
		cell.value = value
		cell.stamp = &volatileFieldStamp{}
		volatileFieldOrder.Unlock()
		if returnNew {
			return next
		}
		return result
	}
}
func (u *AtomicReferenceFieldUpdater) GetAndUpdateExecution(e *Execution, receiver any, update func(*Execution, any) any) any {
	return u.updateExecution(e, receiver, update, false)
}
func (u *AtomicReferenceFieldUpdater) UpdateAndGetExecution(e *Execution, receiver any, update func(*Execution, any) any) any {
	return u.updateExecution(e, receiver, update, true)
}
func (u *AtomicReferenceFieldUpdater) GetAndAccumulateExecution(e *Execution, receiver, value any, accumulate func(*Execution, any, any) any) any {
	u.core.cell(e, receiver)
	if accumulate == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	return u.GetAndUpdateExecution(e, receiver, func(e *Execution, previous any) any { return accumulate(e, previous, value) })
}
func (u *AtomicReferenceFieldUpdater) AccumulateAndGetExecution(e *Execution, receiver, value any, accumulate func(*Execution, any, any) any) any {
	u.core.cell(e, receiver)
	if accumulate == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	return u.UpdateAndGetExecution(e, receiver, func(e *Execution, previous any) any { return accumulate(e, previous, value) })
}
func init() {
	RegisterJavaType(AtomicIntegerFieldUpdaterTypeID, ObjectTypeID)
	RegisterJavaType(AtomicLongFieldUpdaterTypeID, ObjectTypeID)
	RegisterJavaType(AtomicReferenceFieldUpdaterTypeID, ObjectTypeID)
}

const AtomicIntegerFieldUpdaterTypeID TypeID = "java.util.concurrent.atomic.AtomicIntegerFieldUpdater"

type AtomicIntegerFieldUpdater struct{ core atomicFieldUpdater }

func (*AtomicIntegerFieldUpdater) JavaDynamicTypeID() TypeID { return AtomicIntegerFieldUpdaterTypeID }
func NewAtomicIntegerFieldUpdater(owner *Class, name string, caller TypeID) *AtomicIntegerFieldUpdater {
	return &AtomicIntegerFieldUpdater{core: newAtomicFieldUpdater(owner, PrimitiveIntTypeID, name, caller, false)}
}
func (u *AtomicIntegerFieldUpdater) Get(receiver any) int32 {
	return u.GetExecution(NewExecution(), receiver)
}
func (u *AtomicIntegerFieldUpdater) GetExecution(e *Execution, receiver any) int32 {
	return VolatileLoad[int32](u.core.cell(e, receiver))
}
func (u *AtomicIntegerFieldUpdater) Set(receiver any, value int32) {
	u.SetExecution(NewExecution(), receiver, value)
}
func (u *AtomicIntegerFieldUpdater) SetExecution(e *Execution, receiver any, value int32) {
	VolatileStore(u.core.cell(e, receiver), value)
}
func (u *AtomicIntegerFieldUpdater) LazySet(receiver any, value int32) { u.Set(receiver, value) }
func (u *AtomicIntegerFieldUpdater) LazySetExecution(e *Execution, receiver any, value int32) {
	u.SetExecution(e, receiver, value)
}
func (u *AtomicIntegerFieldUpdater) CompareAndSet(receiver any, expect, update int32) bool {
	return u.CompareAndSetExecution(NewExecution(), receiver, expect, update)
}
func (u *AtomicIntegerFieldUpdater) CompareAndSetExecution(e *Execution, receiver any, expect, update int32) bool {
	return u.core.cas(e, receiver, expect, update)
}
func (u *AtomicIntegerFieldUpdater) WeakCompareAndSet(receiver any, expect, update int32) bool {
	return u.CompareAndSet(receiver, expect, update)
}
func (u *AtomicIntegerFieldUpdater) WeakCompareAndSetExecution(e *Execution, receiver any, expect, update int32) bool {
	return u.CompareAndSetExecution(e, receiver, expect, update)
}
func (u *AtomicIntegerFieldUpdater) GetAndSet(receiver any, update int32) int32 {
	return u.GetAndSetExecution(NewExecution(), receiver, update)
}
func (u *AtomicIntegerFieldUpdater) GetAndSetExecution(e *Execution, receiver any, update int32) int32 {
	previous := u.core.swap(e, receiver, update)
	if previous == nil {
		return 0
	}
	result, ok := previous.(int32)
	if !ok {
		panic(NewUnsupportedOperationException("volatile primitive storage has wrong representation"))
	}
	return result
}
func (u *AtomicIntegerFieldUpdater) GetAndAdd(receiver any, delta int32) int32 {
	return u.GetAndAddExecution(NewExecution(), receiver, delta)
}
func (u *AtomicIntegerFieldUpdater) GetAndAddExecution(e *Execution, receiver any, delta int32) int32 {
	return u.GetAndUpdateExecution(e, receiver, func(_ *Execution, previous int32) int32 { return previous + delta })
}
func (u *AtomicIntegerFieldUpdater) AddAndGet(receiver any, delta int32) int32 {
	return u.AddAndGetExecution(NewExecution(), receiver, delta)
}
func (u *AtomicIntegerFieldUpdater) AddAndGetExecution(e *Execution, receiver any, delta int32) int32 {
	return u.GetAndAddExecution(e, receiver, delta) + delta
}
func (u *AtomicIntegerFieldUpdater) GetAndIncrement(receiver any) int32 {
	return u.GetAndAdd(receiver, 1)
}
func (u *AtomicIntegerFieldUpdater) GetAndIncrementExecution(e *Execution, receiver any) int32 {
	return u.GetAndAddExecution(e, receiver, 1)
}
func (u *AtomicIntegerFieldUpdater) GetAndDecrement(receiver any) int32 {
	return u.GetAndAdd(receiver, -1)
}
func (u *AtomicIntegerFieldUpdater) GetAndDecrementExecution(e *Execution, receiver any) int32 {
	return u.GetAndAddExecution(e, receiver, -1)
}
func (u *AtomicIntegerFieldUpdater) IncrementAndGet(receiver any) int32 {
	return u.AddAndGet(receiver, 1)
}
func (u *AtomicIntegerFieldUpdater) IncrementAndGetExecution(e *Execution, receiver any) int32 {
	return u.AddAndGetExecution(e, receiver, 1)
}
func (u *AtomicIntegerFieldUpdater) DecrementAndGet(receiver any) int32 {
	return u.AddAndGet(receiver, -1)
}
func (u *AtomicIntegerFieldUpdater) DecrementAndGetExecution(e *Execution, receiver any) int32 {
	return u.AddAndGetExecution(e, receiver, -1)
}
func (u *AtomicIntegerFieldUpdater) GetAndUpdate(receiver any, update func(int32) int32) int32 {
	e := NewExecution()
	if update == nil {
		return u.GetAndUpdateExecution(e, receiver, nil)
	}
	return u.GetAndUpdateExecution(e, receiver, func(_ *Execution, value int32) int32 { return update(value) })
}
func (u *AtomicIntegerFieldUpdater) GetAndUpdateExecution(e *Execution, receiver any, update func(*Execution, int32) int32) int32 {
	u.core.cell(e, receiver)
	if update == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	for {
		previous := u.GetExecution(e, receiver)
		next := update(e, previous)
		if u.CompareAndSetExecution(e, receiver, previous, next) {
			return previous
		}
	}
}
func (u *AtomicIntegerFieldUpdater) UpdateAndGet(receiver any, update func(int32) int32) int32 {
	e := NewExecution()
	if update == nil {
		return u.UpdateAndGetExecution(e, receiver, nil)
	}
	return u.UpdateAndGetExecution(e, receiver, func(_ *Execution, value int32) int32 { return update(value) })
}
func (u *AtomicIntegerFieldUpdater) UpdateAndGetExecution(e *Execution, receiver any, update func(*Execution, int32) int32) int32 {
	u.core.cell(e, receiver)
	if update == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	for {
		previous := u.GetExecution(e, receiver)
		next := update(e, previous)
		if u.CompareAndSetExecution(e, receiver, previous, next) {
			return next
		}
	}
}
func (u *AtomicIntegerFieldUpdater) GetAndAccumulate(receiver any, value int32, accumulate func(int32, int32) int32) int32 {
	e := NewExecution()
	if accumulate == nil {
		return u.GetAndAccumulateExecution(e, receiver, value, nil)
	}
	return u.GetAndAccumulateExecution(e, receiver, value, func(_ *Execution, a, b int32) int32 { return accumulate(a, b) })
}
func (u *AtomicIntegerFieldUpdater) GetAndAccumulateExecution(e *Execution, receiver any, value int32, accumulate func(*Execution, int32, int32) int32) int32 {
	u.core.cell(e, receiver)
	if accumulate == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	return u.GetAndUpdateExecution(e, receiver, func(e *Execution, previous int32) int32 { return accumulate(e, previous, value) })
}
func (u *AtomicIntegerFieldUpdater) AccumulateAndGet(receiver any, value int32, accumulate func(int32, int32) int32) int32 {
	e := NewExecution()
	if accumulate == nil {
		return u.AccumulateAndGetExecution(e, receiver, value, nil)
	}
	return u.AccumulateAndGetExecution(e, receiver, value, func(_ *Execution, a, b int32) int32 { return accumulate(a, b) })
}
func (u *AtomicIntegerFieldUpdater) AccumulateAndGetExecution(e *Execution, receiver any, value int32, accumulate func(*Execution, int32, int32) int32) int32 {
	u.core.cell(e, receiver)
	if accumulate == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	return u.UpdateAndGetExecution(e, receiver, func(e *Execution, previous int32) int32 { return accumulate(e, previous, value) })
}

const AtomicLongFieldUpdaterTypeID TypeID = "java.util.concurrent.atomic.AtomicLongFieldUpdater"

type AtomicLongFieldUpdater struct{ core atomicFieldUpdater }

func (*AtomicLongFieldUpdater) JavaDynamicTypeID() TypeID { return AtomicLongFieldUpdaterTypeID }
func NewAtomicLongFieldUpdater(owner *Class, name string, caller TypeID) *AtomicLongFieldUpdater {
	return &AtomicLongFieldUpdater{core: newAtomicFieldUpdater(owner, PrimitiveLongTypeID, name, caller, false)}
}
func (u *AtomicLongFieldUpdater) Get(receiver any) int64 {
	return u.GetExecution(NewExecution(), receiver)
}
func (u *AtomicLongFieldUpdater) GetExecution(e *Execution, receiver any) int64 {
	return VolatileLoad[int64](u.core.cell(e, receiver))
}
func (u *AtomicLongFieldUpdater) Set(receiver any, value int64) {
	u.SetExecution(NewExecution(), receiver, value)
}
func (u *AtomicLongFieldUpdater) SetExecution(e *Execution, receiver any, value int64) {
	VolatileStore(u.core.cell(e, receiver), value)
}
func (u *AtomicLongFieldUpdater) LazySet(receiver any, value int64) { u.Set(receiver, value) }
func (u *AtomicLongFieldUpdater) LazySetExecution(e *Execution, receiver any, value int64) {
	u.SetExecution(e, receiver, value)
}
func (u *AtomicLongFieldUpdater) CompareAndSet(receiver any, expect, update int64) bool {
	return u.CompareAndSetExecution(NewExecution(), receiver, expect, update)
}
func (u *AtomicLongFieldUpdater) CompareAndSetExecution(e *Execution, receiver any, expect, update int64) bool {
	return u.core.cas(e, receiver, expect, update)
}
func (u *AtomicLongFieldUpdater) WeakCompareAndSet(receiver any, expect, update int64) bool {
	return u.CompareAndSet(receiver, expect, update)
}
func (u *AtomicLongFieldUpdater) WeakCompareAndSetExecution(e *Execution, receiver any, expect, update int64) bool {
	return u.CompareAndSetExecution(e, receiver, expect, update)
}
func (u *AtomicLongFieldUpdater) GetAndSet(receiver any, update int64) int64 {
	return u.GetAndSetExecution(NewExecution(), receiver, update)
}
func (u *AtomicLongFieldUpdater) GetAndSetExecution(e *Execution, receiver any, update int64) int64 {
	previous := u.core.swap(e, receiver, update)
	if previous == nil {
		return 0
	}
	result, ok := previous.(int64)
	if !ok {
		panic(NewUnsupportedOperationException("volatile primitive storage has wrong representation"))
	}
	return result
}
func (u *AtomicLongFieldUpdater) GetAndAdd(receiver any, delta int64) int64 {
	return u.GetAndAddExecution(NewExecution(), receiver, delta)
}
func (u *AtomicLongFieldUpdater) GetAndAddExecution(e *Execution, receiver any, delta int64) int64 {
	return u.GetAndUpdateExecution(e, receiver, func(_ *Execution, previous int64) int64 { return previous + delta })
}
func (u *AtomicLongFieldUpdater) AddAndGet(receiver any, delta int64) int64 {
	return u.AddAndGetExecution(NewExecution(), receiver, delta)
}
func (u *AtomicLongFieldUpdater) AddAndGetExecution(e *Execution, receiver any, delta int64) int64 {
	return u.GetAndAddExecution(e, receiver, delta) + delta
}
func (u *AtomicLongFieldUpdater) GetAndIncrement(receiver any) int64 { return u.GetAndAdd(receiver, 1) }
func (u *AtomicLongFieldUpdater) GetAndIncrementExecution(e *Execution, receiver any) int64 {
	return u.GetAndAddExecution(e, receiver, 1)
}
func (u *AtomicLongFieldUpdater) GetAndDecrement(receiver any) int64 {
	return u.GetAndAdd(receiver, -1)
}
func (u *AtomicLongFieldUpdater) GetAndDecrementExecution(e *Execution, receiver any) int64 {
	return u.GetAndAddExecution(e, receiver, -1)
}
func (u *AtomicLongFieldUpdater) IncrementAndGet(receiver any) int64 { return u.AddAndGet(receiver, 1) }
func (u *AtomicLongFieldUpdater) IncrementAndGetExecution(e *Execution, receiver any) int64 {
	return u.AddAndGetExecution(e, receiver, 1)
}
func (u *AtomicLongFieldUpdater) DecrementAndGet(receiver any) int64 {
	return u.AddAndGet(receiver, -1)
}
func (u *AtomicLongFieldUpdater) DecrementAndGetExecution(e *Execution, receiver any) int64 {
	return u.AddAndGetExecution(e, receiver, -1)
}
func (u *AtomicLongFieldUpdater) GetAndUpdate(receiver any, update func(int64) int64) int64 {
	e := NewExecution()
	if update == nil {
		return u.GetAndUpdateExecution(e, receiver, nil)
	}
	return u.GetAndUpdateExecution(e, receiver, func(_ *Execution, value int64) int64 { return update(value) })
}
func (u *AtomicLongFieldUpdater) GetAndUpdateExecution(e *Execution, receiver any, update func(*Execution, int64) int64) int64 {
	u.core.cell(e, receiver)
	if update == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	for {
		previous := u.GetExecution(e, receiver)
		next := update(e, previous)
		if u.CompareAndSetExecution(e, receiver, previous, next) {
			return previous
		}
	}
}
func (u *AtomicLongFieldUpdater) UpdateAndGet(receiver any, update func(int64) int64) int64 {
	e := NewExecution()
	if update == nil {
		return u.UpdateAndGetExecution(e, receiver, nil)
	}
	return u.UpdateAndGetExecution(e, receiver, func(_ *Execution, value int64) int64 { return update(value) })
}
func (u *AtomicLongFieldUpdater) UpdateAndGetExecution(e *Execution, receiver any, update func(*Execution, int64) int64) int64 {
	u.core.cell(e, receiver)
	if update == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	for {
		previous := u.GetExecution(e, receiver)
		next := update(e, previous)
		if u.CompareAndSetExecution(e, receiver, previous, next) {
			return next
		}
	}
}
func (u *AtomicLongFieldUpdater) GetAndAccumulate(receiver any, value int64, accumulate func(int64, int64) int64) int64 {
	e := NewExecution()
	if accumulate == nil {
		return u.GetAndAccumulateExecution(e, receiver, value, nil)
	}
	return u.GetAndAccumulateExecution(e, receiver, value, func(_ *Execution, a, b int64) int64 { return accumulate(a, b) })
}
func (u *AtomicLongFieldUpdater) GetAndAccumulateExecution(e *Execution, receiver any, value int64, accumulate func(*Execution, int64, int64) int64) int64 {
	u.core.cell(e, receiver)
	if accumulate == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	return u.GetAndUpdateExecution(e, receiver, func(e *Execution, previous int64) int64 { return accumulate(e, previous, value) })
}
func (u *AtomicLongFieldUpdater) AccumulateAndGet(receiver any, value int64, accumulate func(int64, int64) int64) int64 {
	e := NewExecution()
	if accumulate == nil {
		return u.AccumulateAndGetExecution(e, receiver, value, nil)
	}
	return u.AccumulateAndGetExecution(e, receiver, value, func(_ *Execution, a, b int64) int64 { return accumulate(a, b) })
}
func (u *AtomicLongFieldUpdater) AccumulateAndGetExecution(e *Execution, receiver any, value int64, accumulate func(*Execution, int64, int64) int64) int64 {
	u.core.cell(e, receiver)
	if accumulate == nil {
		panic(NewNullPointerException("updater function is null"))
	}
	return u.UpdateAndGetExecution(e, receiver, func(e *Execution, previous int64) int64 { return accumulate(e, previous, value) })
}

func (u *AtomicReferenceFieldUpdater) GetAndUpdate(receiver any, update func(any) any) any {
	e := NewExecution()
	if update == nil {
		return u.GetAndUpdateExecution(e, receiver, nil)
	}
	return u.GetAndUpdateExecution(e, receiver, func(_ *Execution, value any) any { return update(value) })
}
func (u *AtomicReferenceFieldUpdater) UpdateAndGet(receiver any, update func(any) any) any {
	e := NewExecution()
	if update == nil {
		return u.UpdateAndGetExecution(e, receiver, nil)
	}
	return u.UpdateAndGetExecution(e, receiver, func(_ *Execution, value any) any { return update(value) })
}
func (u *AtomicReferenceFieldUpdater) GetAndAccumulate(receiver, value any, accumulate func(any, any) any) any {
	e := NewExecution()
	if accumulate == nil {
		return u.GetAndAccumulateExecution(e, receiver, value, nil)
	}
	return u.GetAndAccumulateExecution(e, receiver, value, func(_ *Execution, a, b any) any { return accumulate(a, b) })
}
func (u *AtomicReferenceFieldUpdater) AccumulateAndGet(receiver, value any, accumulate func(any, any) any) any {
	e := NewExecution()
	if accumulate == nil {
		return u.AccumulateAndGetExecution(e, receiver, value, nil)
	}
	return u.AccumulateAndGetExecution(e, receiver, value, func(_ *Execution, a, b any) any { return accumulate(a, b) })
}

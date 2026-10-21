package stdjava

import (
	"os"
	"strings"
	"sync"
	"testing"
	"unicode/utf16"
)

type emptySetTDDRoles interface {
	JavaIterable
	CollectionToArrayJava2goExecution(*Execution) *ReferenceArray
	CollectionToArrayIntoJava2goExecution(*Execution, *ReferenceArray) *ReferenceArray
	CollectionAddAllJava2goExecution(*Execution, JavaIterable) bool
	CollectionRetainAllJava2goExecution(*Execution, JavaIterable) bool
	CollectionRemoveAllJava2goExecution(*Execution, JavaIterable) bool
	CollectionForEachJava2goExecution(*Execution, any)
	CollectionRemoveIfJava2goExecution(*Execution, any) bool
	EqualsJava2goExecution(*Execution, any) bool
	HashCodeJava2goExecution(*Execution) int32
}
type emptySetTDDIteratorRoles interface {
	JavaIterator
	JavaIteratorRemovable
	IteratorForEachRemainingJava2goExecution(*Execution, any)
}

func emptySetTDDRolesOf(t *testing.T, value any) emptySetTDDRoles {
	t.Helper()
	roles, ok := value.(emptySetTDDRoles)
	if !ok {
		t.Fatalf("empty singleton lacks contracts: %T", value)
	}
	return roles
}
func emptySetTDDIteratorOf(t *testing.T, value any) emptySetTDDIteratorRoles {
	t.Helper()
	roles, ok := value.(emptySetTDDIteratorRoles)
	if !ok {
		t.Fatalf("shared iterator lacks contracts: %T", value)
	}
	return roles
}
func emptySetTDDPanic(t *testing.T, name string, call func()) any {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); call() }()
	if failure == nil || !CaughtAs(failure, name) {
		t.Fatalf("want %s, got %T %v", name, failure, failure)
	}
	return failure
}

type emptySetTDDBomb struct{}

func (*emptySetTDDBomb) Equals(any) bool { panic("unexpected equality callback") }
func (*emptySetTDDBomb) HashCode() int32 { panic("unexpected hash callback") }

type emptySetTDDProbe struct {
	*ObjectInfo
	want                      *Execution
	trace                     string
	size                      int32
	empty                     bool
	elements                  []any
	sizeFailure, emptyFailure any
	lock                      any
	local                     *ThreadLocal[string]
	thread                    *Thread
}

func emptySetTDDProbeOf(e *Execution, size int32, empty bool, elements ...any) *emptySetTDDProbe {
	const id TypeID = "empty.set.tdd.SourceSet"
	RegisterJavaType(id, AbstractSetTypeID)
	p := &emptySetTDDProbe{want: e, size: size, empty: empty, elements: elements}
	p.ObjectInfo = NewObjectInfo(id, func(TypeID) any { return p })
	return p
}
func (p *emptySetTDDProbe) observe(e *Execution, call string) {
	if p.want != nil && p.want != e {
		panic("substituted caller execution")
	}
	p.trace += call
	if p.lock != nil && (!ThreadHoldsLockExecution(e, p.lock) || ThreadCurrentThread(e) != p.thread || p.local.Get(e) != "kept") {
		panic("lost thread/local/monitor context")
	}
}
func (p *emptySetTDDProbe) CollectionSizeJava2goExecution(e *Execution) int32 {
	p.observe(e, "S")
	if p.sizeFailure != nil {
		panic(p.sizeFailure)
	}
	return p.size
}
func (p *emptySetTDDProbe) CollectionIsEmptyJava2goExecution(e *Execution) bool {
	p.observe(e, "E")
	if p.emptyFailure != nil {
		panic(p.emptyFailure)
	}
	return p.empty
}
func (p *emptySetTDDProbe) CollectionContainsJava2goExecution(e *Execution, _ any) bool {
	p.observe(e, "C")
	panic("unexpected contains callback")
}
func (p *emptySetTDDProbe) IteratorJava2goExecution(e *Execution) JavaIterator {
	p.observe(e, "I")
	return &emptySetTDDCursor{probe: p}
}

type emptySetTDDCursor struct {
	probe *emptySetTDDProbe
	index int
}

func (c *emptySetTDDCursor) HasNextJava2goExecution(e *Execution) bool {
	c.probe.observe(e, "H")
	return c.index < len(c.probe.elements)
}
func (c *emptySetTDDCursor) NextJava2goExecution(e *Execution) any {
	c.probe.observe(e, "N")
	value := c.probe.elements[c.index]
	c.index++
	return value
}
func (c *emptySetTDDCursor) IteratorRemoveJava2goExecution(e *Execution) {
	c.probe.observe(e, "R")
	panic("unexpected iterator remove")
}

func TestCollectionsEmptySetNativeIdentityAndMutation(t *testing.T) {
	e := NewExecution()
	set := EmptySet()
	other := EmptySet()
	if !JavaReferenceEqual(set, other) {
		t.Fatal("generic factory changed allocation")
	}
	info, ok := set.(JavaObjectInfoCarrier)
	if !ok || info.JavaObjectInfo() == nil {
		t.Fatal("singleton missing stable ObjectInfo")
	}
	id, known := ObjectDynamicType(set)
	if !known || id != "java.util.Collections$EmptySet" {
		t.Fatalf("descriptor %s/%t", id, known)
	}
	for _, id := range []TypeID{ObjectTypeID, AbstractCollectionTypeID, AbstractSetTypeID, SetTypeID, CollectionTypeID, IterableTypeID, SerializableTypeID} {
		if !ObjectInstanceOf(set, id) || !JavaReferenceEqual(ObjectView[any](set, id), set) {
			t.Fatalf("nominal alias %s", id)
		}
	}
	for _, id := range []TypeID{MapTypeID, nativeListTypeID, IteratorTypeID, "java.util.HashSet", "java.util.ArrayList"} {
		if ObjectInstanceOf(set, id) {
			t.Fatalf("unrelated nominal %s", id)
		}
	}
	guard := MonitorEnterExecution(e, ObjectView[any](set, SerializableTypeID))
	if !ThreadHoldsLockExecution(e, other) {
		t.Fatal("singleton aliases lost monitor")
	}
	MonitorExitExecution(guard)
	for _, value := range []any{nil, &emptySetTDDBomb{}, NewInteger(1)} {
		emptySetTDDPanic(t, "UnsupportedOperationException", func() { CollectionAddExecution(e, set, value) })
	}
	if CollectionRemoveExecution(e, set, &emptySetTDDBomb{}) || CollectionRemoveExecution(e, set, nil) || CollectionContainsExecution(e, set, &emptySetTDDBomb{}) || CollectionContainsExecution(e, set, nil) {
		t.Fatal("empty read/removal")
	}
	CollectionClearExecution(e, set)
	if CollectionSizeExecution(e, set) != 0 || !CollectionIsEmptyExecution(e, set) || !JavaReferenceEqual(set, EmptySet()) {
		t.Fatal("empty singleton mutated")
	}
}
func TestCollectionsEmptySetNativeIteratorAndExceptions(t *testing.T) {
	e := NewExecution()
	cursor := IterableIteratorExecution(e, EmptySet())
	other := IterableIteratorExecution(e, EmptySet())
	if !JavaReferenceEqual(cursor, other) || IteratorHasNextExecution(e, cursor) {
		t.Fatal("empty shared cursor")
	}
	id, known := ObjectDynamicType(cursor)
	if !known || id != "java.util.Collections$EmptyIterator" || !ObjectInstanceOf(cursor, IteratorTypeID) {
		t.Fatal("iterator nominal descriptor")
	}
	for _, id := range []TypeID{SetTypeID, CollectionTypeID, IterableTypeID, SerializableTypeID} {
		if ObjectInstanceOf(cursor, id) {
			t.Fatalf("iterator unrelated %s", id)
		}
	}
	first := emptySetTDDPanic(t, "NoSuchElementException", func() { IteratorNextExecution(e, cursor) })
	second := emptySetTDDPanic(t, "NoSuchElementException", func() { IteratorNextExecution(e, cursor) })
	remove := emptySetTDDPanic(t, "IllegalStateException", func() { IteratorRemoveExecution(e, cursor) })
	add := emptySetTDDPanic(t, "UnsupportedOperationException", func() { CollectionAddExecution(e, EmptySet(), nil) })
	if JavaReferenceEqual(first, second) {
		t.Fatal("exceptions reused")
	}
	for _, failure := range []any{first, second, remove, add} {
		if JavaThrowableMessageExecution(e, failure) != nil {
			t.Fatalf("noarg detail message is nonnull: %T", failure)
		}
		_ = ThrowableInitCauseExecution(e, failure, nil)
		emptySetTDDPanic(t, "IllegalStateException", func() { _ = ThrowableInitCauseExecution(e, failure, nil) })
	}
	if !JavaReferenceEqual(cursor, IterableIteratorExecution(e, EmptySet())) || IteratorHasNextExecution(e, cursor) {
		t.Fatal("failed next/remove altered shared cursor")
	}
}
func TestCollectionsEmptySetNativeVirtualContainsAllAndEquality(t *testing.T) {
	e := NewExecution()
	set := EmptySet()
	roles := emptySetTDDRolesOf(t, set)
	p := emptySetTDDProbeOf(e, 0, true)
	p.lock = set
	p.thread = ThreadCurrentThread(e)
	p.local = NewThreadLocal[string]()
	p.local.Set(e, "kept")
	guard := MonitorEnterExecution(e, set)
	if !CollectionContainsAllExecution(e, set, p) || p.trace != "E" {
		t.Fatalf("containsAll callbacks %s", p.trace)
	}
	p.trace = ""
	p.empty = false
	if CollectionContainsAllExecution(e, set, p) || p.trace != "E" {
		t.Fatal("virtual isEmpty result")
	}
	MonitorExitExecution(guard)
	p.lock = nil
	p.local.Remove(e)
	p.empty = true
	p.trace = ""
	if !roles.EqualsJava2goExecution(e, p) || p.trace != "SE" {
		t.Fatalf("equals callbacks %s", p.trace)
	}
	p.empty = false
	p.trace = ""
	if roles.EqualsJava2goExecution(e, p) || p.trace != "SE" {
		t.Fatal("equals ignored isEmpty")
	}
	p.empty = true
	p.size = 1
	p.trace = ""
	if roles.EqualsJava2goExecution(e, p) || p.trace != "S" {
		t.Fatal("unequal size invoked containsAll")
	}
	p.size = 0
	for _, failure := range []any{NewClassCastException("size"), NewNullPointerException("size")} {
		p.trace = ""
		p.sizeFailure = failure
		name := "ClassCastException"
		if CaughtAs(failure, "NullPointerException") {
			name = "NullPointerException"
		}
		observed := emptySetTDDPanic(t, name, func() { roles.EqualsJava2goExecution(e, p) })
		if !JavaReferenceEqual(observed, failure) || p.trace != "S" {
			t.Fatal("size exception boundary")
		}
	}
	p.sizeFailure = nil
	for _, failure := range []any{NewClassCastException("empty"), NewNullPointerException("empty")} {
		p.trace = ""
		p.emptyFailure = failure
		if roles.EqualsJava2goExecution(e, p) || p.trace != "SE" {
			t.Fatal("containsAll exception not caught")
		}
	}
	p.emptyFailure = NewIllegalArgumentException("empty")
	p.trace = ""
	observed := emptySetTDDPanic(t, "IllegalArgumentException", func() { roles.EqualsJava2goExecution(e, p) })
	if !JavaReferenceEqual(observed, p.emptyFailure) || p.trace != "SE" {
		t.Fatal("other exception swallowed")
	}
	p.emptyFailure = nil
	p.want = nil
	p.trace = ""
	if !ObjectsEqual(set, p) || p.trace != "SE" {
		t.Fatal("legacy Object dispatch failed")
	}
	native := NewSet[any]()
	if !ObjectsEqual(set, native) || !ObjectsEqual(native, set) || ObjectsHashCode(set) != 0 {
		t.Fatal("legacy native Set/erased carrier compatibility")
	}
	native.Add(NewInteger(2))
	if ObjectsEqual(set, native) || ObjectsEqual(native, set) || roles.EqualsJava2goExecution(e, NewList[any]()) || roles.EqualsJava2goExecution(e, nil) {
		t.Fatal("Set equality nominal/size frontier")
	}
}
func TestCollectionsEmptySetNativeArrayContracts(t *testing.T) {
	e := NewExecution()
	roles := emptySetTDDRolesOf(t, EmptySet())
	first := roles.CollectionToArrayJava2goExecution(e)
	second := roles.CollectionToArrayJava2goExecution(e)
	if JavaReferenceEqual(first, second) || ReferenceArrayLength(first) != 0 || first.JavaArrayTypeID() != ArrayTypeID(ObjectTypeID) {
		t.Fatal("fresh Object[] identity/descriptor")
	}
	zero := NewReferenceArray(0, StringTypeID)
	if roles.CollectionToArrayIntoJava2goExecution(e, zero) != zero || zero.ComponentType() != StringTypeID {
		t.Fatal("zero typed array identity")
	}
	firstText, lastText := JavaStringFromHostUTF8("first"), JavaStringFromHostUTF8("tail")
	array := ReferenceArrayLiteral(StringTypeID, firstText, lastText)
	if roles.CollectionToArrayIntoJava2goExecution(e, array) != array || ReferenceArrayGet[any](array, 0, ObjectTypeID) != nil || ReferenceArrayGet[any](array, 1, ObjectTypeID) != lastText || array.ComponentType() != StringTypeID {
		t.Fatal("typed array slot0/remaining identity")
	}
	kept := NewPrimitiveArray[int32](1, PrimitiveIntTypeID)
	rank := ReferenceArrayLiteral(ArrayTypeID(PrimitiveIntTypeID), NewPrimitiveArray[int32](1, PrimitiveIntTypeID), kept)
	if roles.CollectionToArrayIntoJava2goExecution(e, rank) != rank || ReferenceArrayGet[any](rank, 0, ObjectTypeID) != nil || ReferenceArrayGet[any](rank, 1, ObjectTypeID) != kept || rank.ComponentType() != ArrayTypeID(PrimitiveIntTypeID) {
		t.Fatal("primitive-array reference leaf")
	}
	strings := ReferenceArrayLiteral(ArrayTypeID(StringTypeID), NewReferenceArray(1, StringTypeID), zero)
	if roles.CollectionToArrayIntoJava2goExecution(e, strings) != strings || ReferenceArrayGet[any](strings, 0, ObjectTypeID) != nil || ReferenceArrayGet[any](strings, 1, ObjectTypeID) != zero || strings.ComponentType() != ArrayTypeID(StringTypeID) {
		t.Fatal("nested string array descriptor")
	}
}
func TestCollectionsEmptySetNativeBulkAndFunctionalEffects(t *testing.T) {
	e := NewExecution()
	roles := emptySetTDDRolesOf(t, EmptySet())
	p := emptySetTDDProbeOf(e, 91, false)
	if roles.CollectionAddAllJava2goExecution(e, p) || p.trace != "IH" {
		t.Fatalf("empty addAll trace %s", p.trace)
	}
	p = emptySetTDDProbeOf(e, 0, true, &emptySetTDDBomb{})
	emptySetTDDPanic(t, "UnsupportedOperationException", func() { roles.CollectionAddAllJava2goExecution(e, p) })
	if p.trace != "IHN" {
		t.Fatalf("addAll must consume before UOE: %s", p.trace)
	}
	p = emptySetTDDProbeOf(e, -1, false, &emptySetTDDBomb{})
	if roles.CollectionRetainAllJava2goExecution(e, p) || p.trace != "" {
		t.Fatal("retainAll queried source")
	}
	p = emptySetTDDProbeOf(e, 1, false, &emptySetTDDBomb{})
	if roles.CollectionRemoveAllJava2goExecution(e, p) || p.trace != "S" {
		t.Fatalf("removeAll regular trace %s", p.trace)
	}
	p = emptySetTDDProbeOf(e, -1, false, &emptySetTDDBomb{})
	if roles.CollectionRemoveAllJava2goExecution(e, p) || p.trace != "SIHNH" {
		t.Fatalf("removeAll negative size branch %s", p.trace)
	}
	calls := 0
	consumer := NewConsumerFuncAdapter[any](func(*Execution, any) { calls++; panic("unexpected consumer") })
	roles.CollectionForEachJava2goExecution(e, consumer)
	cursor := emptySetTDDIteratorOf(t, IterableIteratorExecution(e, EmptySet()))
	cursor.IteratorForEachRemainingJava2goExecution(e, consumer)
	predicate := &emptySetTDDBomb{}
	if roles.CollectionRemoveIfJava2goExecution(e, predicate) || calls != 0 {
		t.Fatal("empty functional callback invoked")
	}
	for _, call := range []func(){func() { roles.CollectionRetainAllJava2goExecution(e, nil) }, func() { roles.CollectionRemoveAllJava2goExecution(e, nil) }, func() { roles.CollectionForEachJava2goExecution(e, nil) }, func() { roles.CollectionRemoveIfJava2goExecution(e, nil) }, func() { cursor.IteratorForEachRemainingJava2goExecution(e, nil) }} {
		failure := emptySetTDDPanic(t, "NullPointerException", call)
		if JavaThrowableMessageExecution(e, failure) != nil {
			t.Fatal("Objects.requireNonNull detail message")
		}
	}
	var typedNull *ConsumerFuncAdapter[any]
	emptySetTDDPanic(t, "NullPointerException", func() { roles.CollectionForEachJava2goExecution(e, typedNull) })
	emptySetTDDPanic(t, "NullPointerException", func() { cursor.IteratorForEachRemainingJava2goExecution(e, typedNull) })
}
func TestCollectionsEmptySetNativeDiagnosticsJDK21(t *testing.T) {
	// The primary owner freezes this whole JVM output before runtime production.
	// No expected platform diagnostic is inferred from an existing generic helper.
	oracle, err := os.ReadFile("testdata/collections_empty_set/native.expected.stdout")
	if err != nil {
		t.Fatal(err)
	}
	expected := make(map[string]string)
	for _, line := range strings.Split(string(oracle), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if ok {
			expected[key] = value
		}
	}
	e := NewExecution()
	set := EmptySet()
	roles := emptySetTDDRolesOf(t, set)
	cursor := emptySetTDDIteratorOf(t, IterableIteratorExecution(e, set))
	checks := map[string]func(){"containsAllNull": func() { CollectionContainsAllExecution(e, set, nil) }, "toArrayNull": func() { roles.CollectionToArrayIntoJava2goExecution(e, nil) }, "retainAllNull": func() { roles.CollectionRetainAllJava2goExecution(e, nil) }, "removeAllNull": func() { roles.CollectionRemoveAllJava2goExecution(e, nil) }, "forEachNull": func() { roles.CollectionForEachJava2goExecution(e, nil) }, "removeIfNull": func() { roles.CollectionRemoveIfJava2goExecution(e, nil) }, "iteratorForEachNull": func() { cursor.IteratorForEachRemainingJava2goExecution(e, nil) }}
	for key, call := range checks {
		failure := emptySetTDDPanic(t, "NullPointerException", call)
		message := JavaThrowableMessageExecution(e, failure)
		text := "null"
		if message != nil {
			text = string(utf16.Decode(message.UTF16Copy()))
		}
		got := "java.lang.NullPointerException:" + text
		if expected[key] == "" || got != expected[key] {
			t.Fatalf("%s JVM=%q Go=%q", key, expected[key], got)
		}
	}
}
func TestCollectionsEmptySetNativeBoundedSharedRace(t *testing.T) {
	set := EmptySet()
	e := NewExecution()
	cursor := IterableIteratorExecution(e, set)
	if allocations := testing.AllocsPerRun(100, func() {
		if !JavaReferenceEqual(set, EmptySet()) || !JavaReferenceEqual(cursor, IterableIteratorExecution(e, set)) {
			panic("singleton replaced")
		}
	}); allocations != 0 {
		t.Fatalf("factory/iterator allocations=%v", allocations)
	}
	var workers sync.WaitGroup
	var failureMu sync.Mutex
	bad := false
	for n := 0; n < 8; n++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			caller := NewExecution()
			for i := 0; i < 100; i++ {
				actual := EmptySet()
				it := IterableIteratorExecution(caller, actual)
				guard := MonitorEnterExecution(caller, actual)
				ok := JavaReferenceEqual(actual, set) && JavaReferenceEqual(it, cursor) && ThreadHoldsLockExecution(caller, set) && !IteratorHasNextExecution(caller, it) && CollectionSizeExecution(caller, actual) == 0
				MonitorExitExecution(guard)
				if !ok {
					failureMu.Lock()
					bad = true
					failureMu.Unlock()
				}
			}
		}()
	}
	workers.Wait()
	if bad {
		t.Fatal("shared race identity/owner/size")
	}
}

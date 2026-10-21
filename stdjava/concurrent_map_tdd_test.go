package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func cmPanic(t *testing.T, class string, f func()) {
	t.Helper()
	defer func() {
		v := recover()
		if v == nil || !CaughtAs(v, class) {
			t.Fatalf("panic %v, want %s", v, class)
		}
	}()
	f()
}
func TestConcurrentMapErasedAtomicAndNull(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[string, string]()
	if m.PutIfAbsentObject("k", int32(7), e) != nil {
		t.Fatal("absent prior")
	}
	if m.PutIfAbsentObject("k", "ignored", e) != int32(7) || m.GetObject("k", e) != int32(7) {
		t.Fatal("only absent")
	}
	cmPanic(t, "ClassCastException", func() { m.Put("k", "fixed", e) })
	if m.Get("k", e) != "fixed" {
		t.Fatal("write undone by result cast")
	}
	m.PutObject(int32(1), int32(8), e)
	if m.Size() != 2 || m.GetObject(int32(1), e) != int32(8) {
		t.Fatal("raw key")
	}
	cmPanic(t, "NullPointerException", func() { m.PutIfAbsentObject("k", nil, e) })
	cmPanic(t, "NullPointerException", func() { m.GetObject(nil, e) })
	var null *ConcurrentHashMap[any, any]
	cmPanic(t, "NullPointerException", func() { null.PutObject("k", "v", e) })
	if MapReferenceView(m) == nil || ConcurrentMapReferenceView(m) == nil || m.JavaDynamicTypeID() != ConcurrentHashMapTypeID {
		t.Fatal("nominal ancestry")
	}
}

func TestConcurrentMapAtomicRace(t *testing.T) {
	m := NewConcurrentHashMap[any, any]()
	start := make(chan struct{})
	out := make(chan any, 32)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(v int) { defer wg.Done(); <-start; out <- m.PutIfAbsentObject("shared", v, NewExecution()) }(i)
	}
	close(start)
	wg.Wait()
	close(out)
	missing := 0
	winner := m.GetObject("shared", NewExecution())
	for prior := range out {
		if prior == nil {
			missing++
		} else if prior != winner {
			t.Fatal("inconsistent previous value")
		}
	}
	if missing != 1 || m.Size() != 1 {
		t.Fatalf("missing=%d size=%d", missing, m.Size())
	}
}

type cmCallbackKey struct {
	id      int
	e       *Execution
	m       *ConcurrentHashMap[any, any]
	gate    chan struct{}
	entered chan struct{}
	once    atomic.Bool
	calls   atomic.Int32
}

func (k *cmCallbackKey) HashCodeJava2goExecution(e *Execution) int32 {
	if e != k.e {
		panic("hash execution")
	}
	if k.m.GetObject("guard", e) != "guard" {
		panic("reentrant read")
	}
	return 11
}
func (k *cmCallbackKey) EqualsJava2goExecution(e *Execution, other any) bool {
	if e != k.e {
		panic("equals execution")
	}
	k.calls.Add(1)
	if k.entered != nil && k.once.CompareAndSwap(false, true) {
		close(k.entered)
		<-k.gate
	}
	v, ok := other.(*cmCallbackKey)
	return ok && v.id == k.id
}
func TestConcurrentMapCallbackRetryAndExecution(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	m.PutObject("guard", "guard", e)
	stored := &cmCallbackKey{id: 1, e: e, m: m}
	m.PutObject(stored, "first", e)
	key := &cmCallbackKey{id: 1, e: e, m: m, gate: make(chan struct{}), entered: make(chan struct{})}
	done := make(chan any, 1)
	go func() { done <- m.PutIfAbsentObject(key, "ignored", e) }()
	select {
	case <-key.entered:
	case <-time.After(3 * time.Second):
		t.Fatal("callback held map mutex")
	}
	m.PutObject("other", "change-version", e)
	close(key.gate)
	select {
	case prior := <-done:
		if prior != "first" {
			t.Fatal(prior)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("retry deadlock")
	}
	if key.calls.Load() < 2 {
		t.Fatal("did not validate snapshot version")
	}
}

func TestConcurrentMapLiveViewsAndEntryValues(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	keys := MapKeySetExecution(e, m)
	values := MapValuesExecution(e, m)
	entries := MapEntrySetExecution(e, m)
	m.PutObject("one", "before", e)
	if CollectionSizeExecution(e, keys) != 1 || !CollectionContainsExecution(e, values, "before") {
		t.Fatal("views stale")
	}
	cursor := IterableIteratorExecution(e, entries)
	m.PutObject("one", "at-next", e)
	entry := ObjectView[JavaMapEntry](IteratorNextExecution(e, cursor), JavaMapEntryTypeID)
	if MapEntryGetValueExecution(e, entry) != "at-next" {
		t.Fatal("premature value snapshot")
	}
	m.PutObject("one", "after-next", e)
	if MapEntryGetValueExecution(e, entry) != "at-next" || MapEntrySetValueExecution(e, entry, "through") != "at-next" || m.GetObject("one", e) != "through" {
		t.Fatal("entry semantics")
	}
	m.RemoveObject("one", e)
	if MapEntrySetValueExecution(e, entry, "recreated") != "through" || m.GetObject("one", e) != "recreated" {
		t.Fatal("entry reinsert")
	}
	cmPanic(t, "NullPointerException", func() { MapEntrySetValueExecution(e, entry, nil) })
	IteratorRemoveExecution(e, cursor)
	if m.Size() != 0 {
		t.Fatal("iterator removal")
	}
	cmPanic(t, "IllegalStateException", func() { IteratorRemoveExecution(e, cursor) })
	m.PutObject("two", "remove", e)
	if !CollectionRemoveExecution(e, values, "remove") || m.Size() != 0 {
		t.Fatal("value removal")
	}
	m.PutObject("three", "clear", e)
	CollectionClearExecution(e, keys)
	if CollectionSizeExecution(e, entries) != 0 {
		t.Fatal("clear")
	}
}

type cmSource struct {
	trace   *string
	entries []JavaMapEntry
	e       *Execution
}

func (*cmSource) JavaDynamicTypeID() TypeID { return "campaign.runtime.ConcurrentMapSource" }
func (s *cmSource) MapSizeJava2goExecution(e *Execution) int32 {
	if e != s.e {
		panic("size execution")
	}
	*s.trace += "S"
	return int32(len(s.entries))
}
func (s *cmSource) MapEntrySetJava2goExecution(e *Execution) JavaIterable {
	if e != s.e {
		panic("entrySet execution")
	}
	*s.trace += "E"
	return NewIterableExecution(func(caller *Execution) JavaIterator {
		if caller != s.e {
			panic("iterator execution")
		}
		*s.trace += "I"
		return &cmSourceIterator{source: s}
	})
}

type cmSourceIterator struct {
	source *cmSource
	index  int
}

func (i *cmSourceIterator) HasNextJava2goExecution(e *Execution) bool {
	if e != i.source.e {
		panic("hasNext execution")
	}
	return i.index < len(i.source.entries)
}
func (i *cmSourceIterator) NextJava2goExecution(e *Execution) any {
	if e != i.source.e {
		panic("next execution")
	}
	v := i.source.entries[i.index]
	i.index++
	return v
}

type cmSourceEntry struct {
	key, value any
	trace      *string
	e          *Execution
	fail       bool
}

func (*cmSourceEntry) nativeJavaInterfaces() []TypeID { return []TypeID{JavaMapEntryTypeID} }
func (s *cmSourceEntry) GetKeyJava2goExecution(e *Execution) any {
	if e != s.e {
		panic("key execution")
	}
	*s.trace += "K"
	return s.key
}
func (s *cmSourceEntry) GetValueJava2goExecution(e *Execution) any {
	if e != s.e {
		panic("value execution")
	}
	*s.trace += "V"
	if s.fail {
		panic(NewIllegalStateException("entry"))
	}
	return s.value
}
func (s *cmSourceEntry) SetValueJava2goExecution(e *Execution, value any) any {
	panic(NewUnsupportedOperationException(""))
}
func TestConcurrentMapPutAllOrderingPartialEffects(t *testing.T) {
	RegisterJavaType("campaign.runtime.ConcurrentMapSource", ObjectTypeID, MapTypeID)
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	trace := ""
	s := &cmSource{trace: &trace, e: e}
	m.PutAll(s, e)
	if trace != "SEI" {
		t.Fatal(trace)
	}
	s.entries = []JavaMapEntry{&cmSourceEntry{key: "prefix", value: "yes", trace: &trace, e: e}, &cmSourceEntry{key: nil, value: "evaluated", trace: &trace, e: e}}
	trace = ""
	cmPanic(t, "NullPointerException", func() { m.PutAll(s, e) })
	if trace != "SEIKVKV" || m.GetObject("prefix", e) != "yes" || m.Size() != 1 {
		t.Fatalf("%s/%d", trace, m.Size())
	}
	s.entries = []JavaMapEntry{&cmSourceEntry{key: "second", value: "yes", trace: &trace, e: e}, &cmSourceEntry{key: "fail", trace: &trace, e: e, fail: true}}
	trace = ""
	cmPanic(t, "IllegalStateException", func() { MapPutAllExecution(e, m, s) })
	if trace != "SEIKVKV" || m.Size() != 2 {
		t.Fatalf("%s/%d", trace, m.Size())
	}
}

func TestConcurrentMapPublicationAndWeakIteration(t *testing.T) {
	type payload struct{ a, b int }
	m := NewConcurrentHashMap[any, any]()
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.PutIfAbsentObject("published", &payload{a: 17, b: 29}, NewExecution())
		close(done)
	}()
	e := NewExecution()
	var read any
	deadline := time.After(3 * time.Second)
	for read == nil {
		read = m.GetObject("published", e)
		select {
		case <-deadline:
			t.Fatal("publication timeout")
		default:
		}
	}
	if p := read.(*payload); p.a != 17 || p.b != 29 {
		t.Fatal("unpublished fields")
	}
	<-done
	wg.Wait()
	entries := MapEntrySetExecution(e, m)
	cursor := IterableIteratorExecution(e, entries)
	m.PutObject("new", "new", e)
	m.Clear()
	if !IteratorHasNextExecution(e, cursor) {
		t.Fatal("lost initial membership")
	}
	_ = IteratorNextExecution(e, cursor)
	if IteratorHasNextExecution(e, cursor) {
		t.Fatal("singleton cursor")
	}
	cmPanic(t, "NoSuchElementException", func() { IteratorNextExecution(e, cursor) })
}

type cmBlockValue struct {
	entered, release chan struct{}
	once             atomic.Bool
}

func (v *cmBlockValue) EqualsJava2goExecution(e *Execution, other any) bool {
	if v.once.CompareAndSwap(false, true) {
		close(v.entered)
		<-v.release
	}
	return other == "old"
}
func TestConcurrentMapViewRemovalReplacementSchedules(t *testing.T) {
	for _, entries := range []bool{false, true} {
		e := NewExecution()
		m := NewConcurrentHashMap[any, any]()
		m.PutObject("k", "old", e)
		query := &cmBlockValue{entered: make(chan struct{}), release: make(chan struct{})}
		done := make(chan bool, 1)
		view := MapValuesExecution(e, m)
		var argument any = query
		if entries {
			view = MapEntrySetExecution(e, m)
			argument = &cmSourceEntry{key: "k", value: query, trace: new(string), e: e}
		}
		go func() { done <- CollectionRemoveExecution(e, view, argument) }()
		select {
		case <-query.entered:
		case <-time.After(3 * time.Second):
			t.Fatal("value callback held mutex")
		}
		m.PutObject("k", "new", e)
		close(query.release)
		select {
		case removed := <-done:
			if entries {
				if removed || m.GetObject("k", e) != "new" {
					t.Fatal("conditional remove lost replacement")
				}
			} else {
				if !removed || m.Size() != 0 {
					t.Fatal("values iterator remove must remove latest mapping")
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatal("remove deadlock")
		}
	}
}
func TestConcurrentMapEntryAdditionAndGetterShortCircuit(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	view := MapEntrySetExecution(e, m)
	trace := ""
	entry := &cmSourceEntry{key: "k", value: "one", trace: &trace, e: e}
	if !CollectionAddExecution(e, view, entry) || trace != "KV" {
		t.Fatal("entry add")
	}
	trace = ""
	entry.value = "two"
	if CollectionAddExecution(e, view, entry) || m.GetObject("k", e) != "two" {
		t.Fatal("entry replacement add")
	}
	trace = ""
	entry.key = nil
	entry.fail = true
	if CollectionContainsExecution(e, view, entry) || trace != "K" {
		t.Fatal("null key must short circuit value getter")
	}
	trace = ""
	entry.key = "missing"
	if CollectionContainsExecution(e, view, entry) || trace != "K" {
		t.Fatal("missing key must short circuit value getter")
	}
	cmPanic(t, "UnsupportedOperationException", func() { CollectionAddExecution(e, MapKeySetExecution(e, m), "x") })
	cmPanic(t, "UnsupportedOperationException", func() { CollectionAddExecution(e, MapValuesExecution(e, m), "x") })
}

func TestConcurrentMapErasedNativeMapComputeAndDefaults(t *testing.T) {
	e := NewExecution()
	m := NewMap[string, string]()
	mapping := NewFunctionFuncAdapter(func(caller *Execution, key any) any {
		if caller != e {
			panic("mapping execution")
		}
		return int32(3)
	})
	result := MapComputeIfAbsentExecution(e, m, "raw", mapping, func(caller *Execution, key any) any { return CallFunctionExecution(caller, mapping, key) })
	if result != int32(3) || m.GetObject("raw", e) != int32(3) {
		t.Fatal("early compute result cast")
	}
	cmPanic(t, "ClassCastException", func() { m.Get("raw", e) })
	cmPanic(t, "NullPointerException", func() { MapComputeIfAbsentExecution(e, m, "raw", nil, func(*Execution, any) any { return "wrong" }) })
	m.PutObject("nil", nil, e)
	if MapGetOrDefaultExecution(e, m, "nil", "fallback") != nil {
		t.Fatal("present null")
	}
	if MapPutIfAbsentExecution(e, m, "nil", "filled") != nil || m.GetObject("nil", e) != "filled" {
		t.Fatal("Map null default policy")
	}
	cmPanic(t, "UnsupportedOperationException", func() {
		MapComputeIfAbsentExecution(e, NewConcurrentHashMap[any, any](), "k", mapping, func(*Execution, any) any { return "wrong" })
	})
}

type cmRoleSource struct {
	cmSource
	data    *Map[any, any]
	calls   string
	mapping any
}

func (*cmRoleSource) JavaDynamicTypeID() TypeID { return "campaign.runtime.ConcurrentMapRoles" }
func (s *cmRoleSource) MapGetJava2goExecution(e *Execution, key any) any {
	if e != s.e {
		panic("get execution")
	}
	s.calls += "G"
	return s.data.GetObject(key, e)
}
func (s *cmRoleSource) MapContainsKeyJava2goExecution(e *Execution, key any) bool {
	if e != s.e {
		panic("contains execution")
	}
	s.calls += "C"
	return s.data.ContainsKey(key, e)
}
func (s *cmRoleSource) MapPutJava2goExecution(e *Execution, key, value any) any {
	if e != s.e {
		panic("put execution")
	}
	s.calls += "W"
	return s.data.PutObject(key, value, e)
}
func (s *cmRoleSource) MapPutIfAbsentJava2goExecution(e *Execution, key, value any) any {
	if e != s.e {
		panic("putIfAbsent execution")
	}
	s.calls += "P"
	return s.data.PutIfAbsentObject(key, value, e)
}
func (s *cmRoleSource) MapPutAllJava2goExecution(e *Execution, input any) {
	if e != s.e {
		panic("putAll execution")
	}
	s.calls += "A"
}
func (s *cmRoleSource) MapGetOrDefaultJava2goExecution(e *Execution, key, value any) any {
	if e != s.e {
		panic("default execution")
	}
	s.calls += "D"
	return value
}
func (s *cmRoleSource) MapComputeIfAbsentJava2goExecution(e *Execution, key, mapping any, apply func(*Execution, any) any) any {
	if e != s.e {
		panic("compute execution")
	}
	s.calls += "F"
	s.mapping = mapping
	return "override"
}

type cmShapeOnly struct{}

func (*cmShapeOnly) MapEntrySetJava2goExecution(e *Execution) JavaIterable { return nil }
func TestConcurrentMapNominalOverridesAndInterfaceDefaults(t *testing.T) {
	RegisterJavaType("campaign.runtime.ConcurrentMapRoles", ObjectTypeID, ConcurrentMapTypeID)
	e := NewExecution()
	trace := ""
	s := &cmRoleSource{cmSource: cmSource{trace: &trace, e: e}, data: NewMap[any, any]()}
	ConcurrentMapReferenceView(s)
	MapPutAllExecution(e, s, nil)
	if MapPutIfAbsentExecution(e, s, nil, nil) != nil {
		t.Fatal("source null policy")
	}
	if MapGetOrDefaultExecution(e, s, "absent", "override-default") != "override-default" {
		t.Fatal("default override")
	}
	mapping := new(int)
	if MapComputeIfAbsentExecution(e, s, "x", mapping, nil) != "override" || s.mapping != mapping || s.calls != "APDF" {
		t.Fatal("source method routing")
	}
	cmPanic(t, "ClassCastException", func() { MapReferenceView(&cmShapeOnly{}) })
	s.calls = ""
	s.data.PutObject("null", nil, e)
	if MapDefaultGetOrDefaultExecution(e, s, "null", "fallback") != nil || s.calls != "GC" {
		t.Fatal("present null default")
	}
	s.calls = ""
	if MapDefaultPutIfAbsentExecution(e, s, "null", "filled") != nil || s.calls != "GW" {
		t.Fatal("Map default uses put")
	}
	s.calls = ""
	computed := ConcurrentMapDefaultComputeIfAbsentExecution(e, s, "race", mapping, func(caller *Execution, key any) any {
		if caller != e {
			panic("apply execution")
		}
		s.data.PutIfAbsentObject(key, "winner", e)
		return "computed"
	})
	if computed != "winner" || s.calls != "GP" {
		t.Fatal("ConcurrentMap default must arbitrate winner")
	}
}
func TestConcurrentMapObjectMethodsErased(t *testing.T) {
	e := NewExecution()
	key := JavaStringLiteralUTF16([]uint16{'r', 'a', 'w'})
	value := BoxInteger(9)
	m := NewConcurrentHashMap[*JavaString, *JavaString]()
	m.PutObject(key, value, e)
	other := NewMap[any, any]()
	other.PutObject(key, value, e)
	if !m.EqualsJava2goExecution(e, other) || m.HashCodeJava2goExecution(e) != other.HashCodeJava2goExecution(e) {
		t.Fatal("erased map equality/hash")
	}
	if string(unsignedBytes(JavaStringGetBytes(m.StringJava2goExecution(e), UTF_8).Elements)) != "{raw=9}" {
		t.Fatal("erased rendering")
	}
	self := NewConcurrentHashMap[any, any]()
	self.PutObject(JavaStringLiteralUTF16([]uint16{'s', 'e', 'l', 'f'}), self, e)
	if string(unsignedBytes(JavaStringGetBytes(self.StringJava2goExecution(e), UTF_8).Elements)) != "{self=(this Map)}" {
		t.Fatal("map self rendering")
	}
}

type cmHashMutationKey struct {
	m     *ConcurrentHashMap[any, any]
	e     *Execution
	armed bool
}

func (k *cmHashMutationKey) HashCodeJava2goExecution(e *Execution) int32 {
	if e != k.e {
		panic("hash execution")
	}
	if k.armed {
		k.armed = false
		k.m.PutObject(k, BoxInteger(9), e)
	}
	return 11
}
func TestConcurrentMapHashReadsValueAfterKeyCallback(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	key := &cmHashMutationKey{m: m, e: e}
	m.PutObject(key, BoxInteger(7), e)
	key.armed = true
	if m.HashCodeJava2goExecution(e) != 11^9 {
		t.Fatal("hashCode read value before key callback")
	}
	m.PutObject(key, BoxInteger(7), e)
	key.armed = true
	if ObjectsHashCode(MapEntrySetExecution(e, m), e) != 11^9 {
		t.Fatal("entrySet hashCode read value before key callback")
	}
}
func TestConcurrentMapConsumedIteratorStorageBound(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	for i := 0; i < 128; i++ {
		m.PutObject(int32(i), BoxInteger(int32(i)), e)
	}
	cursor := IterableIteratorExecution(e, MapKeySetExecution(e, m)).(*concurrentMapIterator)
	for IteratorHasNextExecution(e, cursor) {
		IteratorNextExecution(e, cursor)
	}
	if len(cursor.records) != 0 {
		t.Fatal("exhausted iterator pins consumed membership snapshot")
	}
	IteratorRemoveExecution(e, cursor)
	m.Clear()
	if m.Size() != 0 || len(m.recordSnapshot()) != 0 || len(m.buckets) != 0 {
		t.Fatal("clear retains records")
	}
	for i := 0; i < 4096; i++ {
		m.PutObject(int32(i%32), BoxInteger(int32(i)), e)
		if m.Size() > 32 || len(m.recordSnapshot()) > 32 {
			t.Fatal("unbounded live records")
		}
	}
	m.Clear()
}

type cmFixedKey struct{ id int }

func (*cmFixedKey) HashCodeJava2goExecution(*Execution) int32 { return 11 }
func (k *cmFixedKey) EqualsJava2goExecution(e *Execution, other any) bool {
	v, ok := other.(*cmFixedKey)
	return ok && v.id == k.id
}

type cmChangingContains struct {
	m      *ConcurrentHashMap[any, any]
	second any
}

func (v *cmChangingContains) EqualsJava2goExecution(e *Execution, other any) bool {
	if other == "first" {
		v.m.PutObject(v.second, "wanted", e)
		return false
	}
	return other == "wanted"
}
func TestConcurrentMapContainsValueReadsAtTraversal(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	first := &cmFixedKey{1}
	second := &cmFixedKey{2}
	m.PutObject(first, "first", e)
	m.PutObject(second, "old", e)
	query := &cmChangingContains{m: m, second: second}
	if !m.ContainsValue(query, e) {
		t.Fatal("containsValue captured later value before first equality callback")
	}
	m.PutObject(second, "old", e)
	if !CollectionContainsExecution(e, MapValuesExecution(e, m), query) {
		t.Fatal("values.contains captured later value early")
	}
}

type cmSizeForbiddenSet struct{ backing JavaIterable }

func (*cmSizeForbiddenSet) nativeJavaInterfaces() []TypeID { return []TypeID{SetTypeID} }
func (s *cmSizeForbiddenSet) IteratorJava2goExecution(e *Execution) JavaIterator {
	return IterableIteratorExecution(e, s.backing)
}
func (s *cmSizeForbiddenSet) CollectionContainsJava2goExecution(e *Execution, value any) bool {
	return CollectionContainsExecution(e, s.backing, value)
}
func (*cmSizeForbiddenSet) CollectionSizeJava2goExecution(*Execution) int32 {
	panic(NewIllegalStateException("forbidden size"))
}
func TestConcurrentMapSetEqualsDoesNotConsultSize(t *testing.T) {
	e := NewExecution()
	m := NewConcurrentHashMap[any, any]()
	m.PutObject("k", "v", e)
	for _, view := range []JavaIterable{MapKeySetExecution(e, m), MapEntrySetExecution(e, m)} {
		if !ObjectsEqual(view, &cmSizeForbiddenSet{backing: view}, e) {
			t.Fatal("set equality")
		}
	}
}

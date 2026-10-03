package stdjava

import (
	"math"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"weak"
)

// This file provides Go runtime equivalents for the java.util.concurrent and
// java.lang.Thread APIs that the transpiler maps onto. They are deliberately
// thin shims over the Go standard library: the goal is behavioural fidelity for
// the common cases (atomic counters, a concurrent map, a fixed worker pool, and
// thread join), not a complete reimplementation of the JDK.

// AtomicInteger mirrors java.util.concurrent.atomic.AtomicInteger. Java's
// AtomicInteger is 32-bit, so it wraps atomic.Int32.
type AtomicInteger struct {
	v atomic.Int32
}

// NewAtomicInteger constructs an AtomicInteger with the given initial value,
// mirroring `new AtomicInteger(initial)`.
func NewAtomicInteger(initial int32) *AtomicInteger {
	a := &AtomicInteger{}
	a.v.Store(initial)
	return a
}

func (a *AtomicInteger) Get() int32                  { return a.v.Load() }
func (a *AtomicInteger) Set(value int32)             { a.v.Store(value) }
func (a *AtomicInteger) IncrementAndGet() int32      { return a.v.Add(1) }
func (a *AtomicInteger) DecrementAndGet() int32      { return a.v.Add(-1) }
func (a *AtomicInteger) AddAndGet(delta int32) int32 { return a.v.Add(delta) }

// GetAndIncrement returns the value before incrementing, like the Java method.
func (a *AtomicInteger) GetAndIncrement() int32      { return a.v.Add(1) - 1 }
func (a *AtomicInteger) GetAndDecrement() int32      { return a.v.Add(-1) + 1 }
func (a *AtomicInteger) GetAndAdd(delta int32) int32 { return a.v.Add(delta) - delta }

// CompareAndSet atomically sets the value to update if it currently equals
// expect, returning whether the swap happened.
func (a *AtomicInteger) CompareAndSet(expect, update int32) bool {
	return a.v.CompareAndSwap(expect, update)
}

// AtomicLong mirrors java.util.concurrent.atomic.AtomicLong (64-bit).
type AtomicLong struct {
	v atomic.Int64
}

func NewAtomicLong(initial int64) *AtomicLong {
	a := &AtomicLong{}
	a.v.Store(initial)
	return a
}

func (a *AtomicLong) Get() int64                  { return a.v.Load() }
func (a *AtomicLong) Set(value int64)             { a.v.Store(value) }
func (a *AtomicLong) IncrementAndGet() int64      { return a.v.Add(1) }
func (a *AtomicLong) DecrementAndGet() int64      { return a.v.Add(-1) }
func (a *AtomicLong) AddAndGet(delta int64) int64 { return a.v.Add(delta) }
func (a *AtomicLong) GetAndIncrement() int64      { return a.v.Add(1) - 1 }
func (a *AtomicLong) GetAndDecrement() int64      { return a.v.Add(-1) + 1 }
func (a *AtomicLong) GetAndAdd(delta int64) int64 { return a.v.Add(delta) - delta }

func (a *AtomicLong) CompareAndSet(expect, update int64) bool {
	return a.v.CompareAndSwap(expect, update)
}

// AtomicBoolean mirrors java.util.concurrent.atomic.AtomicBoolean.
type AtomicBoolean struct {
	v atomic.Bool
}

func NewAtomicBoolean(initial bool) *AtomicBoolean {
	a := &AtomicBoolean{}
	a.v.Store(initial)
	return a
}

func (a *AtomicBoolean) Get() bool      { return a.v.Load() }
func (a *AtomicBoolean) Set(value bool) { a.v.Store(value) }
func (a *AtomicBoolean) CompareAndSet(expect, update bool) bool {
	return a.v.CompareAndSwap(expect, update)
}

// ConcurrentHashMap publishes immutable collision-bucket snapshots. User
// hashCode/equals callbacks run outside the mutex, allowing reentrant reads.
// Writers validate the snapshot version before publishing; concurrent writes
// may cause a callback to be retried. Key/entry views are snapshots.
type ConcurrentHashMap[K, V any] struct {
	mu      sync.RWMutex
	buckets map[int32][]MapEntry[K, V]
	version uint64
	size    int32
}

func NewConcurrentHashMap[K, V any]() *ConcurrentHashMap[K, V] {
	return &ConcurrentHashMap[K, V]{buckets: make(map[int32][]MapEntry[K, V])}
}
func (c *ConcurrentHashMap[K, V]) snapshot(hash int32) ([]MapEntry[K, V], uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.buckets[hash], c.version
}
func concurrentEntryIndex[K, V any](entries []MapEntry[K, V], key any, execution *Execution) int {
	for i, entry := range entries {
		if ObjectsEqual(key, entry.Key, execution) {
			return i
		}
	}
	return -1
}
func (c *ConcurrentHashMap[K, V]) Put(key K, value V, execution ...*Execution) V {
	ReferenceRequireNonNull(key)
	ReferenceRequireNonNull(value)
	exec := optionalComparisonExecution(execution)
	hash := ObjectsHashCode(key, exec)
	for {
		entries, version := c.snapshot(hash)
		index := concurrentEntryIndex(entries, key, exec)
		updated := append([]MapEntry[K, V](nil), entries...)
		previous := collectionZero[V]()
		if index >= 0 {
			previous = updated[index].Value
			updated[index].Value = value
		} else {
			updated = append(updated, MapEntry[K, V]{Key: key, Value: value})
		}
		c.mu.Lock()
		if c.version != version {
			c.mu.Unlock()
			continue
		}
		if c.buckets == nil {
			c.buckets = make(map[int32][]MapEntry[K, V])
		}
		c.buckets[hash] = updated
		c.version++
		if index < 0 {
			c.size++
		}
		c.mu.Unlock()
		return previous
	}
}
func (c *ConcurrentHashMap[K, V]) Get(key any, execution ...*Execution) V {
	value, _ := c.GetOk(key, execution...)
	return value
}
func (c *ConcurrentHashMap[K, V]) GetOk(key any, execution ...*Execution) (V, bool) {
	ReferenceRequireNonNull(key)
	exec := optionalComparisonExecution(execution)
	entries, _ := c.snapshot(ObjectsHashCode(key, exec))
	if index := concurrentEntryIndex(entries, key, exec); index >= 0 {
		return entries[index].Value, true
	}
	return collectionZero[V](), false
}
func (c *ConcurrentHashMap[K, V]) ContainsKey(key any, execution ...*Execution) bool {
	_, ok := c.GetOk(key, execution...)
	return ok
}
func (c *ConcurrentHashMap[K, V]) ContainsValue(value any, execution ...*Execution) bool {
	ReferenceRequireNonNull(value)
	for _, entry := range c.EntrySet() {
		if ObjectsEqual(value, entry.Value, execution...) {
			return true
		}
	}
	return false
}
func (c *ConcurrentHashMap[K, V]) Remove(key any, execution ...*Execution) V {
	ReferenceRequireNonNull(key)
	exec := optionalComparisonExecution(execution)
	hash := ObjectsHashCode(key, exec)
	for {
		entries, version := c.snapshot(hash)
		index := concurrentEntryIndex(entries, key, exec)
		if index < 0 {
			return collectionZero[V]()
		}
		previous := entries[index].Value
		updated := make([]MapEntry[K, V], 0, len(entries)-1)
		updated = append(updated, entries[:index]...)
		updated = append(updated, entries[index+1:]...)
		c.mu.Lock()
		if c.version != version {
			c.mu.Unlock()
			continue
		}
		if len(updated) == 0 {
			delete(c.buckets, hash)
		} else {
			c.buckets[hash] = updated
		}
		c.version++
		c.size--
		c.mu.Unlock()
		return previous
	}
}
func (c *ConcurrentHashMap[K, V]) Size() int32 { c.mu.RLock(); defer c.mu.RUnlock(); return c.size }
func (c *ConcurrentHashMap[K, V]) EntrySet() []MapEntry[K, V] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entries := make([]MapEntry[K, V], 0, c.size)
	for _, bucket := range c.buckets {
		entries = append(entries, bucket...)
	}
	return entries
}
func (c *ConcurrentHashMap[K, V]) KeySet() []K {
	entries := c.EntrySet()
	keys := make([]K, len(entries))
	for i, entry := range entries {
		keys[i] = entry.Key
	}
	return keys
}

// Runnable is the Go counterpart of java.lang.Runnable: anything with a Run()
// method. Anonymous Runnable classes and Thread subclasses (whose run() override
// is generated as Run()) satisfy it, so they can be handed to a Thread directly.
const (
	RunnableTypeID TypeID = "Runnable"
	ThreadTypeID   TypeID = "java.lang.Thread"
)

type Runnable interface {
	Run()
}

// executionRunnable is implemented by generated Runnable adapters. Public
// Run remains the Go-facing entry point, while calls already inside generated
// Java code use RunJava2goExecution to preserve monitor reentrancy through the
// callback boundary.
type executionRunnable interface {
	RunJava2goExecution(*Execution)
}

// runnableFunc adapts a plain func() (a lambda or method reference) to Runnable.
type runnableFunc func()

func (f runnableFunc) Run() {
	if f != nil {
		f()
	}
}

// RunnableFuncAdapter gives a target-typed Java Runnable lambda stable object
// identity. A raw Go function is not comparable and exposes only its shared
// entry-code pointer through reflection, so it cannot faithfully participate in
// Java reference equality. The pointer-backed adapter is both identity-bearing
// and execution-aware.
type RunnableFuncAdapter struct {
	run func(*Execution)
}

func NewRunnableFuncAdapter(run func(*Execution)) *RunnableFuncAdapter {
	return &RunnableFuncAdapter{run: run}
}

func (f *RunnableFuncAdapter) Run() {
	if f != nil && f.run != nil {
		f.run(NewExecution())
	}
}

func (f *RunnableFuncAdapter) RunJava2goExecution(execution *Execution) {
	if f != nil && f.run != nil {
		f.run(execution)
	}
}

func (*RunnableFuncAdapter) JavaDynamicTypeID() TypeID {
	return RunnableTypeID
}

// PlainRunnableFuncAdapter is the identity-bearing counterpart for an external
// method reference that has only a Go func() entry point. The execution-aware
// bridge intentionally ignores the token because no generated Java body exists
// on the other side to receive it.
type PlainRunnableFuncAdapter struct {
	run func()
}

func NewPlainRunnableFuncAdapter(run func()) *PlainRunnableFuncAdapter {
	return &PlainRunnableFuncAdapter{run: run}
}

func (f *PlainRunnableFuncAdapter) Run() {
	if f != nil && f.run != nil {
		f.run()
	}
}

func (f *PlainRunnableFuncAdapter) RunJava2goExecution(_ *Execution) {
	f.Run()
}

func (*PlainRunnableFuncAdapter) JavaDynamicTypeID() TypeID {
	return RunnableTypeID
}

// RunRunnableExecution invokes a Runnable inside an existing logical Java
// execution. It accepts any because a target-typed Java Runnable lambda lowers
// to func(*Execution); asRunnable supplies the Go interface adapter without
// discarding the caller's token. Generated object callbacks normally expose the
// fixed hidden method below. If a Java member occupied that generated name, the
// transpiler appends a numeric suffix, which invokeSuffixedExecutionRunnable
// discovers before falling back to the public Go entry point.
func RunRunnableExecution(execution *Execution, value any) {
	r := asRunnable(value)
	if r == nil {
		return
	}
	// Preserve the caller's logical execution for a direct Thread.run() without
	// exporting an execution-aware method that Go would promote through embedded
	// Thread subclasses. The exact type assertion deliberately excludes those
	// subclasses so their generated/overridden Run method remains authoritative.
	if thread, ok := r.(*Thread); ok {
		thread.runJava2goExecution(execution)
		return
	}
	if generated, ok := r.(executionRunnable); ok {
		generated.RunJava2goExecution(execution)
		return
	}
	if invokeSuffixedExecutionRunnable(execution, r) {
		return
	}
	r.Run()
}

// invokeSuffixedExecutionRunnable handles the collision-safe names emitted when
// user Java source already declares RunJava2goExecution. Only the transpiler's
// exact hidden signature is eligible: one *Execution argument, no results, and
// a name consisting of RunJava2goExecution followed solely by decimal digits.
func invokeSuffixedExecutionRunnable(execution *Execution, r Runnable) bool {
	const prefix = "RunJava2goExecution"
	executionType := reflect.TypeOf((*Execution)(nil))
	value := reflect.ValueOf(r)
	typeOfValue := value.Type()
	for index := 0; index < typeOfValue.NumMethod(); index++ {
		method := typeOfValue.Method(index)
		if !decimalSuffix(method.Name, prefix) {
			continue
		}
		bound := value.Method(index)
		methodType := bound.Type()
		if methodType.NumIn() != 1 || methodType.In(0) != executionType || methodType.NumOut() != 0 {
			continue
		}
		bound.Call([]reflect.Value{reflect.ValueOf(execution)})
		return true
	}
	return false
}

func decimalSuffix(name, prefix string) bool {
	if len(name) <= len(prefix) || name[:len(prefix)] != prefix {
		return false
	}
	for _, digit := range name[len(prefix):] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}

// Thread mirrors the subset of java.lang.Thread used by transpiled code: it holds
// a Runnable and runs it in a goroutine on Start(), with Join() blocking until it
// finishes. Java's Thread is far richer (priorities, interruption, daemon status,
// fairness); those are out of scope and documented as such.
type Thread struct {
	interruptMu     sync.Mutex
	interrupted     bool
	interruptSignal chan struct{}
	name            string
	nameText        *JavaString
	run             Runnable
	done            chan struct{}
	started         atomic.Bool
}

var threadSequence atomic.Int64
var mainThread = func() *Thread {
	thread := newNamedThread("main")
	thread.started.Store(true)
	return thread
}()

// ThreadCurrentThread resolves Java thread identity from the explicit execution
// token, including callbacks and reentrant calls on the same worker.
func ThreadCurrentThread(execution *Execution) *Thread {
	if execution != nil && execution.thread != nil {
		return execution.thread
	}
	return mainThread
}

func (t *Thread) GetName() string { return t.name }

func (t *Thread) GetNameReference() *JavaString {
	ReferenceRequireNonNull(t)
	return t.nameText
}

func newNamedThread(name string) *Thread {
	return &Thread{name: name, nameText: JavaStringFromHostUTF8(name), done: make(chan struct{})}
}

// JavaDynamicTypeID lets the reified reference-array runtime recognize the
// compact stdjava-backed representation using its canonical java.lang.Thread
// descriptor, matching compiler-generated casts and reference-array components.
func (*Thread) JavaDynamicTypeID() TypeID {
	return ThreadTypeID
}

func init() {
	// Thread extends Object and implements Runnable. Register both edges so a
	// Thread stored through Object[] or Runnable[] receives the same nominal
	// assignment treatment as a generated source class.
	RegisterJavaType(RunnableTypeID, ObjectTypeID)
	RegisterJavaType(ThreadTypeID, ObjectTypeID, RunnableTypeID)
}

// NewThread builds a Thread from a Runnable. The argument is either a plain
// func() (the lambda / method-reference form) or a value implementing Runnable
// (an anonymous Runnable class). Other values produce a Thread that does
// nothing when started.
func NewThread(runnable any) *Thread {
	if name, ok := runnable.(*JavaString); ok {
		return NewThreadNamedReference(nil, name)
	}
	if name, ok := runnable.(string); ok {
		return NewThreadNamed(nil, name)
	}
	thread := newNamedThread("Thread-" + strconv.FormatInt(threadSequence.Add(1)-1, 10))
	thread.run = asRunnable(runnable)
	return thread
}

// NewThreadNamed covers Thread(Runnable, String) and Thread(String).
func NewThreadNamed(runnable any, name string) *Thread {
	StringRequireNonNull(name)
	thread := newNamedThread(name)
	thread.run = asRunnable(runnable)
	return thread
}

func NewThreadNamedReference(runnable any, name *JavaString) *Thread {
	ReferenceRequireNonNull(name)
	thread := &Thread{
		name:     string(unsignedBytes(JavaStringGetBytes(name, UTF_8).Elements)),
		nameText: name,
		done:     make(chan struct{}),
	}
	thread.run = asRunnable(runnable)
	return thread
}

// asRunnable coerces the accepted Thread argument forms into a Runnable.
func asRunnable(runnable any) Runnable {
	switch r := runnable.(type) {
	case nil:
		return nil
	case Runnable:
		return r
	case func():
		return runnableFunc(r)
	case func(*Execution):
		return NewRunnableFuncAdapter(r)
	default:
		return nil
	}
}

// NewThreadBase backs a `class X extends Thread` subclass. The generated
// constructor passes the subclass instance (which provides the run() override as
// Run()) so that Start() dispatches to it. It is embedded as the *Thread field of
// the subclass struct.
func NewThreadBase(self Runnable) *Thread {
	return NewThread(self)
}

// Run executes this Thread's target synchronously. java.lang.Thread implements
// Runnable even when it has not been subclassed, so the runtime representation
// must expose Run to remain a valid generated Runnable and Runnable[] view.
func (t *Thread) Run() {
	t.runJava2goExecution(NewExecution())
}

func (t *Thread) runJava2goExecution(execution *Execution) {
	RunRunnableExecution(execution, t.run)
}

// Start can transition a Thread from NEW to alive only once.
func (t *Thread) Start() {
	if !t.started.CompareAndSwap(false, true) {
		panic(NewIllegalThreadStateException("thread already started"))
	}
	go func() {
		defer close(t.done)
		defer reportUncaughtTaskException()
		RunRunnableExecution(&Execution{thread: t}, t.run)
	}()
}

// Join on a NEW or terminated Thread returns immediately, as in Java.
func (t *Thread) Join() {
	if t.started.Load() {
		<-t.done
	}
}
func (t *Thread) IsAlive() bool {
	if !t.started.Load() {
		return false
	}
	select {
	case <-t.done:
		return false
	default:
		return true
	}
}
func (t *Thread) JoinTimed(millis int64, nanos ...int32) {
	extra := int32(0)
	if len(nanos) > 0 {
		extra = nanos[0]
	}
	if millis < 0 || extra < 0 || extra > 999999 {
		panic(NewIllegalArgumentException("invalid join timeout"))
	}
	if extra > 0 && millis < math.MaxInt64 {
		millis++
	}
	if millis == 0 {
		t.Join()
		return
	}
	if !t.IsAlive() {
		return
	}
	timer := time.NewTimer(MILLISECONDS.duration(millis))
	defer timer.Stop()
	select {
	case <-t.done:
	case <-timer.C:
	}
}

// ThreadSleep mirrors Thread.sleep(millis), including negative-time rejection.
func ThreadSleep(millis int64) {
	if millis < 0 {
		panic(NewIllegalArgumentException("timeout value is negative"))
	}
	time.Sleep(MILLISECONDS.duration(millis))
}

// NewObject mirrors `new Object()`, which in Java is most often used purely as a
// lock token for `synchronized`. It returns a fresh, unique pointer so each
// Object() has a distinct identity suitable as a monitor key.
func NewObject() any {
	// Pointers to distinct zero-sized Go variables are permitted to compare
	// equal. Use a non-zero-sized token so every live Java Object has a distinct
	// identity, including when two objects are used as nested monitor locks.
	return new(byte)
}

// --- intrinsic object monitors (synchronized) ------------------------------

// Every Java object has an intrinsic monitor that `synchronized` acquires. Go
// has no per-object reentrant lock, so we maintain a registry that associates a
// monitor record with each object identity. The record's owner is an explicit
// Execution token: entering again with the same token increments its depth,
// while every different token waits until the depth returns to zero.
//
// Generated superclass views share the ObjectInfo identity of their Java
// allocation. Other supported runtime references retain their existing identity.
// Managed Java allocations use scalar identity keys and weak monitor values.
// A live guard, entrant, waiter, or legacy mutex handle retains the monitor and
// its anchor; an idle managed registry entry does not retain the Java object.
// Arbitrary comparable host values keep the legacy value-key fallback described
// below; they are outside the managed Java allocation lifecycle contract.
type monitor struct {
	// mu protects explicit logical ownership. The outermost explicit entry also
	// holds legacyMu across the generated body; reentrant entries by the same
	// execution only increase depth. Sharing that physical mutex with the legacy
	// API keeps old and newly generated callers mutually exclusive.
	mu    sync.Mutex
	owner *Execution
	depth int

	// legacyMu and legacyCond retain the original one-argument monitor API until
	// generated code is migrated atomically to the explicit Execution protocol.
	// New generated code must use the *Execution entry points below.
	legacyMu   sync.Mutex
	legacyCond *sync.Cond

	// anchor keeps identity-bearing reference storage alive while callers hold
	// this monitor. The registry holds only a weak pointer to the monitor, so
	// this strong edge cannot turn an idle registry entry into an object root.
	// It also prevents scalar addresses from being recycled while a live
	// monitor, including an interior legacyMu pointer, still uses that identity.
	anchor interface{}
}

// MonitorGuard represents one successful monitor entry. Every entry, including
// a reentrant one, receives its own guard and must be paired with MonitorExit.
type MonitorGuard struct {
	monitor   *monitor
	execution *Execution
	released  bool
}

// monitorIdentity is a nonretaining description of a managed Java reference. Pointer
// identities are scalar addresses, never strong Go pointers. Legacy native
// value identities (including strings and class names) retain their existing
// value comparison; these representations do not carry Java allocation identity.
// The legacy comparable-host-value fallback can retain pointer fields until
// monitor cleanup. A host value that points back to its monitor can form a rooted
// cycle; managed Java allocations never take this compatibility path.
// Java arrays are represented as Go slices, which cannot be map
// keys, so their identity is the typed address of their first backing-storage
// element together with the slice shape. An aliased Java array carries the
// same slice header and therefore resolves to the same monitor.
type monitorIdentity struct {
	comparable interface{}
	reference  reflect.Type
	data       uintptr
	length     int
	capacity   int
}

var (
	monitorsMu sync.Mutex
	monitors   = map[monitorIdentity]weak.Pointer[monitor]{}
)

func monitorIdentityFor(obj interface{}) monitorIdentity {
	if carrier, ok := obj.(JavaObjectInfoCarrier); ok {
		if identity := carrier.JavaObjectInfo(); identity != nil {
			return monitorIdentity{reference: reflect.TypeOf(identity), data: reflect.ValueOf(identity).Pointer()}
		}
	}
	// Builtin throwables are Go values whose copies share this state pointer.
	// Use that Java allocation identity without storing the pointer in the key.
	if carrier, ok := obj.(interface{ ThrowableIdentity() *throwableState }); ok {
		if identity := carrier.ThrowableIdentity(); identity != nil {
			return monitorIdentity{reference: reflect.TypeOf(identity), data: reflect.ValueOf(identity).Pointer()}
		}
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Chan:
		return monitorIdentity{reference: value.Type(), data: value.Pointer()}
	case reflect.Slice:
		return monitorIdentity{
			reference: value.Type(),
			data:      value.Pointer(),
			length:    value.Len(),
			capacity:  value.Cap(),
		}
	case reflect.Map:
		// Maps are not emitted as Java object representations, but accepting a
		// map-backed external reference is safe: reflect exposes its stable
		// runtime identity and a live monitor's anchor keeps it alive.
		return monitorIdentity{reference: value.Type(), data: value.Pointer()}
	default:
		if value.Type().Comparable() {
			return monitorIdentity{comparable: obj}
		}
		// Do not let an unexpected backend representation reach Go's map hash
		// operation and panic with "hash of unhashable type". Such a value is not
		// a Java reference representation for which this runtime can promise
		// stable identity.
		panic(NewIllegalArgumentException("unsupported non-comparable monitor reference"))
	}
}

type monitorCleanupEntry struct {
	identity monitorIdentity
	epoch    weak.Pointer[monitor]
}

// Managed cleanup keys contain no object, monitor, bound view, or execution.
// Comparable host values retain the existing value-key compatibility caveat.
// A delayed cleanup must not remove a replacement with the same scalar key.
func cleanupMonitorEntry(entry monitorCleanupEntry) {
	monitorsMu.Lock()
	if monitors[entry.identity] == entry.epoch {
		delete(monitors, entry.identity)
	}
	monitorsMu.Unlock()
}

func monitorRecord(obj interface{}) *monitor {
	identity := monitorIdentityFor(obj)
	monitorsMu.Lock()
	defer monitorsMu.Unlock()
	m := monitors[identity].Value()
	if m == nil {
		m = &monitor{anchor: obj}
		m.legacyCond = sync.NewCond(&m.legacyMu)
		epoch := weak.Make(m)
		monitors[identity] = epoch
		runtime.AddCleanup(m, cleanupMonitorEntry, monitorCleanupEntry{identity, epoch})
	}
	runtime.KeepAlive(obj)
	return m
}

// monitorFor returns the lock for obj. Retained for callers (and tests) that
// only need the mutex.
func monitorFor(obj interface{}) *sync.Mutex {
	return &monitorRecord(obj).legacyMu
}

// nilMonitorReference reports whether obj represents Java null. Transpiled
// references are commonly pointers, but Java arrays are Go slices and an
// interface can carry a typed nil of either kind. Checking those forms before
// consulting the monitor registry both preserves Java's exception and avoids a
// raw Go panic for unhashable nil slices.
func nilMonitorReference(obj interface{}) bool {
	return javaReferenceIsNull(obj)
}

func requireNonNullMonitorReference(obj interface{}, operation string) {
	if nilMonitorReference(obj) {
		panic(NewNullPointerException(operation + " on null"))
	}
}

func requireExecution(execution *Execution) {
	if execution == nil {
		panic(NewIllegalArgumentException("nil Java execution context"))
	}
}

// MonitorEnter retains the original non-reentrant monitor entry point for Go
// callers and generated code produced before explicit Execution propagation.
// Newly generated Java code uses MonitorEnterExecution.
func MonitorEnter(obj interface{}) *sync.Mutex {
	requireNonNullMonitorReference(obj, "monitor operation")
	m := monitorFor(obj)
	m.Lock()
	runtime.KeepAlive(obj)
	return m
}

// MonitorExit releases a legacy monitor previously acquired with MonitorEnter.
func MonitorExit(m *sync.Mutex) {
	if m != nil {
		m.Unlock()
	}
}

// MonitorEnterExecution acquires obj's intrinsic monitor for execution. A
// second entry by the same execution is reentrant; entries by different
// executions remain mutually exclusive. The returned guard is normally
// released with defer.
func MonitorEnterExecution(execution *Execution, obj interface{}) *MonitorGuard {
	requireExecution(execution)
	requireNonNullMonitorReference(obj, "monitor operation")
	m := monitorRecord(obj)
	m.mu.Lock()
	if m.owner == execution {
		m.depth++
		m.mu.Unlock()
		return &MonitorGuard{monitor: m, execution: execution}
	}
	m.mu.Unlock()

	// Every different execution, as well as every legacy caller, competes for
	// the same physical mutex. The previous explicit owner clears its logical
	// state before releasing this mutex, so ownership is empty once acquired.
	m.legacyMu.Lock()
	m.mu.Lock()
	m.owner = execution
	m.depth = 1
	m.mu.Unlock()
	runtime.KeepAlive(obj)
	return &MonitorGuard{monitor: m, execution: execution}
}

// MonitorExitExecution releases a monitor previously acquired with
// MonitorEnterExecution.
func MonitorExitExecution(guard *MonitorGuard) {
	if guard == nil {
		return
	}
	m := guard.monitor
	if m == nil {
		panic(NewIllegalStateException("invalid monitor guard"))
	}

	m.mu.Lock()
	if guard.released || m.owner != guard.execution || m.depth == 0 {
		m.mu.Unlock()
		panic(NewIllegalStateException("monitor exit without ownership"))
	}
	guard.released = true
	m.depth--
	releasePhysical := false
	if m.depth == 0 {
		m.owner = nil
		releasePhysical = true
	}
	m.mu.Unlock()
	if releasePhysical {
		m.legacyMu.Unlock()
	}
	// Released generated guards must not extend the Java allocation's lifetime.
	guard.monitor = nil
	guard.execution = nil
	runtime.KeepAlive(m)
}

// MonitorWait retains the original wait implementation for legacy generated
// code. The caller must hold the mutex returned by MonitorEnter.
func MonitorWait(obj interface{}) {
	requireNonNullMonitorReference(obj, "wait")
	m := monitorRecord(obj)
	m.legacyCond.Wait()
	runtime.KeepAlive(m)
	runtime.KeepAlive(obj)
}

// MonitorNotify retains the original notify implementation for legacy
// generated code.
func MonitorNotify(obj interface{}) {
	requireNonNullMonitorReference(obj, "notify")
	m := monitorRecord(obj)
	m.legacyCond.Signal()
	runtime.KeepAlive(m)
	runtime.KeepAlive(obj)
}

// MonitorNotifyAll retains the original notifyAll implementation for legacy
// generated code.
func MonitorNotifyAll(obj interface{}) {
	requireNonNullMonitorReference(obj, "notifyAll")
	m := monitorRecord(obj)
	m.legacyCond.Broadcast()
	runtime.KeepAlive(m)
	runtime.KeepAlive(obj)
}

// MonitorWaitExecution implements Object.wait(): the caller must hold obj's
// monitor for execution; it
// atomically releases every reentrant acquisition, blocks until notified, then
// re-acquires the monitor and restores the original recursion depth. Timed
// wait(millis) is not modelled and falls back to an untimed wait.
func MonitorWaitExecution(execution *Execution, obj interface{}) {
	requireExecution(execution)
	requireNonNullMonitorReference(obj, "wait")
	m := monitorRecord(obj)

	m.mu.Lock()
	if m.owner != execution || m.depth == 0 {
		m.mu.Unlock()
		panic(NewIllegalMonitorStateException("wait without monitor ownership"))
	}
	savedDepth := m.depth
	m.owner = nil
	m.depth = 0
	m.mu.Unlock()

	// The outermost explicit entry owns legacyMu, so Cond.Wait atomically makes
	// the monitor available to legacy and explicit competitors and re-acquires it
	// before returning after notification.
	m.legacyCond.Wait()

	m.mu.Lock()
	m.owner = execution
	m.depth = savedDepth
	m.mu.Unlock()
	runtime.KeepAlive(m)
	runtime.KeepAlive(obj)
}

// MonitorNotifyExecution implements Object.notify(): wake one waiter on obj's
// monitor.
func MonitorNotifyExecution(execution *Execution, obj interface{}) {
	requireExecution(execution)
	requireNonNullMonitorReference(obj, "notify")
	m := monitorRecord(obj)
	m.mu.Lock()
	if m.owner != execution || m.depth == 0 {
		m.mu.Unlock()
		panic(NewIllegalMonitorStateException("notify without monitor ownership"))
	}
	m.legacyCond.Signal()
	m.mu.Unlock()
	runtime.KeepAlive(m)
	runtime.KeepAlive(obj)
}

// MonitorNotifyAllExecution implements Object.notifyAll(): wake all waiters on
// obj's monitor.
func MonitorNotifyAllExecution(execution *Execution, obj interface{}) {
	requireExecution(execution)
	requireNonNullMonitorReference(obj, "notifyAll")
	m := monitorRecord(obj)
	m.mu.Lock()
	if m.owner != execution || m.depth == 0 {
		m.mu.Unlock()
		panic(NewIllegalMonitorStateException("notifyAll without monitor ownership"))
	}
	m.legacyCond.Broadcast()
	m.mu.Unlock()
	runtime.KeepAlive(m)
	runtime.KeepAlive(obj)
}

type classMonitorReference struct {
	name string
}

// ClassMonitorEnter retains the original class-level monitor entry point.
func ClassMonitorEnter(className string) *sync.Mutex {
	return MonitorEnter(classMonitorReference{name: className})
}

// ClassMonitorEnterExecution acquires the class-level monitor named by
// className, used to
// lower a `static synchronized` method (which in Java locks the Class object).
// The name is the generated Go type name, unique per class within the program.
func ClassMonitorEnterExecution(execution *Execution, className string) *MonitorGuard {
	return MonitorEnterExecution(execution, classMonitorReference{name: className})
}

package stdjava

import (
	"reflect"
	"sync"
	"sync/atomic"
	"time"
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

// ConcurrentHashMap is a mutex-guarded map mirroring the subset of
// java.util.concurrent.ConcurrentHashMap that transpiled code commonly uses.
// Keys and values are generic; a sync.RWMutex guards the backing map. A
// dedicated type (rather than sync.Map) keeps the get/put/size API close to
// Java and preserves typed values without per-call assertions.
type ConcurrentHashMap[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewConcurrentHashMap[K comparable, V any]() *ConcurrentHashMap[K, V] {
	return &ConcurrentHashMap[K, V]{m: make(map[K]V)}
}

// Put stores value under key and returns the previous value (or the zero value)
// matching Java's Map.put return contract.
func (c *ConcurrentHashMap[K, V]) Put(key K, value V) V {
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := c.m[key]
	c.m[key] = value
	return prev
}

// Get returns the value for key, or the zero value if absent. (The two-result
// form is GetOk for callers that need presence.)
func (c *ConcurrentHashMap[K, V]) Get(key K) V {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.m[key]
}

func (c *ConcurrentHashMap[K, V]) GetOk(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[key]
	return v, ok
}

func (c *ConcurrentHashMap[K, V]) ContainsKey(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.m[key]
	return ok
}

func (c *ConcurrentHashMap[K, V]) Remove(key K) V {
	c.mu.Lock()
	defer c.mu.Unlock()
	prev := c.m[key]
	delete(c.m, key)
	return prev
}

func (c *ConcurrentHashMap[K, V]) Size() int32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return int32(len(c.m))
}

// Runnable is the Go counterpart of java.lang.Runnable: anything with a Run()
// method. Anonymous Runnable classes and Thread subclasses (whose run() override
// is generated as Run()) satisfy it, so they can be handed to a Thread directly.
type Runnable interface {
	Run()
}

// runnableFunc adapts a plain func() (a lambda or method reference) to Runnable.
type runnableFunc func()

func (f runnableFunc) Run() {
	if f != nil {
		f()
	}
}

// Thread mirrors the subset of java.lang.Thread used by transpiled code: it holds
// a Runnable and runs it in a goroutine on Start(), with Join() blocking until it
// finishes. Java's Thread is far richer (priorities, interruption, daemon status,
// fairness); those are out of scope and documented as such.
type Thread struct {
	run  Runnable
	done chan struct{}
	once sync.Once
}

// NewThread builds a Thread from a Runnable. The argument is either a plain
// func() (the lambda / method-reference form) or a value implementing Runnable
// (an anonymous Runnable class). Other values produce a Thread that does
// nothing when started.
func NewThread(runnable any) *Thread {
	return &Thread{run: asRunnable(runnable), done: make(chan struct{})}
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
	default:
		return nil
	}
}

// NewThreadBase backs a `class X extends Thread` subclass. The generated
// constructor passes the subclass instance (which provides the run() override as
// Run()) so that Start() dispatches to it. It is embedded as the *Thread field of
// the subclass struct.
func NewThreadBase(self Runnable) *Thread {
	return &Thread{run: self, done: make(chan struct{})}
}

// Start runs the thread's Runnable in a goroutine. Calling Start more than once
// is a no-op after the first, approximating Java's IllegalThreadStateException
// without panicking. There is no scheduling fairness or interruption.
func (t *Thread) Start() {
	t.once.Do(func() {
		go func() {
			defer close(t.done)
			if t.run != nil {
				t.run.Run()
			}
		}()
	})
}

// Join blocks until the thread's Runnable has finished. A Thread that was never
// started returns immediately, since its done channel is closed by Start.
func (t *Thread) Join() {
	<-t.done
}

// ThreadSleep mirrors Thread.sleep(millis). The Java method takes milliseconds.
func ThreadSleep(millis int64) {
	time.Sleep(time.Duration(millis) * time.Millisecond)
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
// has no per-object lock, so we maintain a registry that associates a
// *sync.Mutex with each object identity (its pointer). monitorFor returns the
// same mutex for the same object across calls, giving `synchronized (obj) {}`
// true mutual exclusion on that object.
//
// LIMITATION: monitors are keyed by the runtime pointer of the value passed in,
// so the synchronized argument must be a reference type (the common case:
// `this`, a lock object, a field). Synchronizing on a value type would key on a
// transient boxed copy and not exclude correctly; such uses are rare and not
// modelled. The registry never releases monitors, matching the fact that an
// object's monitor lives as long as the object.
// monitor pairs an object's lock with the condition variable that backs
// wait/notify/notifyAll. The cond is lazily created over the same mutex so that
// waiting atomically releases the lock and re-acquires it on wake, matching Java.
type monitor struct {
	mu   sync.Mutex
	cond *sync.Cond

	// anchor keeps identity-bearing reference storage alive for as long as its
	// monitor is registered. In particular, a slice monitor is keyed by its
	// backing-storage address; retaining the slice prevents that address from
	// being recycled for an unrelated Java array.
	anchor interface{}
}

// monitorIdentity is a comparable description of a Java reference. Most
// generated references are already comparable Go values and use comparable
// directly. Java arrays are represented as Go slices, which cannot be map
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
	monitors   = map[monitorIdentity]*monitor{}
)

func monitorIdentityFor(obj interface{}) monitorIdentity {
	value := reflect.ValueOf(obj)
	if value.Type().Comparable() {
		return monitorIdentity{comparable: obj}
	}

	switch value.Kind() {
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
		// runtime identity and the monitor anchor keeps it alive.
		return monitorIdentity{reference: value.Type(), data: value.Pointer()}
	default:
		// Do not let an unexpected backend representation reach Go's map hash
		// operation and panic with "hash of unhashable type". Such a value is not
		// a Java reference representation for which this runtime can promise
		// stable identity.
		panic(NewIllegalArgumentException("unsupported non-comparable monitor reference"))
	}
}

func monitorRecord(obj interface{}) *monitor {
	identity := monitorIdentityFor(obj)
	monitorsMu.Lock()
	defer monitorsMu.Unlock()
	m, ok := monitors[identity]
	if !ok {
		m = &monitor{anchor: obj}
		m.cond = sync.NewCond(&m.mu)
		monitors[identity] = m
	}
	return m
}

// monitorFor returns the lock for obj. Retained for callers (and tests) that
// only need the mutex.
func monitorFor(obj interface{}) *sync.Mutex {
	return &monitorRecord(obj).mu
}

// nilMonitorReference reports whether obj represents Java null. Transpiled
// references are commonly pointers, but Java arrays are Go slices and an
// interface can carry a typed nil of either kind. Checking those forms before
// consulting the monitor registry both preserves Java's exception and avoids a
// raw Go panic for unhashable nil slices.
func nilMonitorReference(obj interface{}) bool {
	if obj == nil {
		return true
	}

	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func requireNonNullMonitorReference(obj interface{}, operation string) {
	if nilMonitorReference(obj) {
		panic(NewNullPointerException(operation + " on null"))
	}
}

// MonitorEnter acquires the intrinsic monitor for obj and returns it, mirroring
// the entry of a `synchronized (obj)` block. The returned mutex is passed to
// MonitorExit (typically via defer) to release it.
func MonitorEnter(obj interface{}) *sync.Mutex {
	requireNonNullMonitorReference(obj, "monitor operation")
	m := monitorFor(obj)
	m.Lock()
	return m
}

// MonitorExit releases a monitor previously acquired with MonitorEnter.
func MonitorExit(m *sync.Mutex) {
	if m != nil {
		m.Unlock()
	}
}

// MonitorWait implements Object.wait(): the caller must hold obj's monitor; it
// atomically releases the lock and blocks until notified, then re-acquires the
// lock before returning.
//
// LIMITATION: Java's wait can wake spuriously and is always used in a loop that
// rechecks a condition; sync.Cond.Wait has the same contract, so transpiled code
// that follows the `while (!cond) obj.wait();` idiom behaves correctly. Timed
// wait(millis) is not modelled and falls back to an untimed wait.
func MonitorWait(obj interface{}) {
	requireNonNullMonitorReference(obj, "wait")
	monitorRecord(obj).cond.Wait()
}

// MonitorNotify implements Object.notify(): wake one waiter on obj's monitor.
func MonitorNotify(obj interface{}) {
	requireNonNullMonitorReference(obj, "notify")
	monitorRecord(obj).cond.Signal()
}

// MonitorNotifyAll implements Object.notifyAll(): wake all waiters on obj's
// monitor.
func MonitorNotifyAll(obj interface{}) {
	requireNonNullMonitorReference(obj, "notifyAll")
	monitorRecord(obj).cond.Broadcast()
}

// ClassMonitorEnter acquires the class-level monitor named by className, used to
// lower a `static synchronized` method (which in Java locks the Class object).
// The name is the generated Go type name, unique per class within the program.
func ClassMonitorEnter(className string) *sync.Mutex {
	return MonitorEnter(className)
}

// ExecutorService is a minimal fixed-size worker pool mirroring the
// ExecutorService methods transpiled code commonly uses: submit a Runnable,
// shutdown, and awaitTermination. Tasks are plain func() values.
type ExecutorService struct {
	tasks   chan func()
	wg      sync.WaitGroup
	workers sync.WaitGroup
	once    sync.Once
}

// NewFixedThreadPool mirrors Executors.newFixedThreadPool(n): it starts n worker
// goroutines that drain a task queue.
func NewFixedThreadPool(n int32) *ExecutorService {
	if n < 1 {
		n = 1
	}
	e := &ExecutorService{tasks: make(chan func(), 64)}
	for i := int32(0); i < n; i++ {
		e.workers.Add(1)
		go func() {
			defer e.workers.Done()
			for task := range e.tasks {
				func() {
					defer e.wg.Done()
					if task != nil {
						task()
					}
				}()
			}
		}()
	}
	return e
}

// Submit enqueues a task for execution by the pool. The task is either a func()
// (lambda / method reference) or a value implementing Runnable (an anonymous
// Runnable class), matching the forms ExecutorService.submit accepts in Java.
func (e *ExecutorService) Submit(task any) {
	r := asRunnable(task)
	e.wg.Add(1)
	e.tasks <- func() {
		if r != nil {
			r.Run()
		}
	}
}

// Shutdown stops accepting new tasks and lets the workers drain the queue.
// Calling it more than once is safe.
func (e *ExecutorService) Shutdown() {
	e.once.Do(func() { close(e.tasks) })
}

// AwaitTermination blocks until all submitted tasks have completed. It must be
// preceded by Shutdown for the workers to exit; it waits for both queued work
// and worker shutdown.
func (e *ExecutorService) AwaitTermination() {
	e.wg.Wait()
	e.workers.Wait()
}

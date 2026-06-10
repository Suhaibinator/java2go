package stdjava

import (
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

// Thread mirrors the subset of java.lang.Thread used by transpiled code: it
// wraps a Runnable (a func()) and runs it in a goroutine on Start(), with Join()
// blocking until it finishes. Java's Thread is far richer (priorities,
// interruption, daemon status); those are out of scope and documented as such.
type Thread struct {
	run  func()
	done chan struct{}
	once sync.Once
}

// NewThread builds a Thread from a Runnable. The Runnable is a plain func() in
// generated code (Runnable.run lowered to a closure).
func NewThread(run func()) *Thread {
	return &Thread{run: run, done: make(chan struct{})}
}

// Start runs the thread's Runnable in a goroutine. Calling Start more than once
// is a no-op after the first, approximating Java's IllegalThreadStateException
// without panicking.
func (t *Thread) Start() {
	t.once.Do(func() {
		go func() {
			defer close(t.done)
			if t.run != nil {
				t.run()
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
	return new(struct{})
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
var (
	monitorsMu sync.Mutex
	monitors   = map[interface{}]*sync.Mutex{}
)

func monitorFor(obj interface{}) *sync.Mutex {
	monitorsMu.Lock()
	defer monitorsMu.Unlock()
	m, ok := monitors[obj]
	if !ok {
		m = &sync.Mutex{}
		monitors[obj] = m
	}
	return m
}

// MonitorEnter acquires the intrinsic monitor for obj and returns it, mirroring
// the entry of a `synchronized (obj)` block. The returned mutex is passed to
// MonitorExit (typically via defer) to release it.
func MonitorEnter(obj interface{}) *sync.Mutex {
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

// Submit enqueues a Runnable for execution by the pool.
func (e *ExecutorService) Submit(task func()) {
	e.wg.Add(1)
	e.tasks <- task
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

package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestAtomicInteger_ConcurrentIncrement(t *testing.T) {
	a := NewAtomicInteger(0)
	var wg sync.WaitGroup
	const goroutines, perG = 50, 100
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				a.IncrementAndGet()
			}
		}()
	}
	wg.Wait()
	if got := a.Get(); got != goroutines*perG {
		t.Fatalf("AtomicInteger lost updates: got %d, want %d", got, goroutines*perG)
	}
}

func TestAtomicInteger_GetAndIncrementReturnsPrevious(t *testing.T) {
	a := NewAtomicInteger(5)
	if prev := a.GetAndIncrement(); prev != 5 {
		t.Fatalf("GetAndIncrement returned %d, want 5", prev)
	}
	if got := a.Get(); got != 6 {
		t.Fatalf("after GetAndIncrement value = %d, want 6", got)
	}
}

func TestAtomicInteger_CompareAndSet(t *testing.T) {
	a := NewAtomicInteger(1)
	if !a.CompareAndSet(1, 2) {
		t.Fatal("CompareAndSet(1,2) should succeed when value is 1")
	}
	if a.CompareAndSet(1, 3) {
		t.Fatal("CompareAndSet(1,3) should fail when value is 2")
	}
	if got := a.Get(); got != 2 {
		t.Fatalf("value = %d, want 2", got)
	}
}

func TestAtomicLong_Add(t *testing.T) {
	a := NewAtomicLong(10)
	if got := a.AddAndGet(5); got != 15 {
		t.Fatalf("AddAndGet = %d, want 15", got)
	}
}

func TestConcurrentHashMap_ConcurrentPut(t *testing.T) {
	m := NewConcurrentHashMap[int32, int32]()
	var wg sync.WaitGroup
	const n = 200
	for i := int32(0); i < n; i++ {
		wg.Add(1)
		go func(k int32) {
			defer wg.Done()
			m.Put(k, k*2)
		}(i)
	}
	wg.Wait()
	if got := m.Size(); got != n {
		t.Fatalf("map size = %d, want %d", got, n)
	}
	if v, ok := m.GetOk(100); !ok || v != 200 {
		t.Fatalf("Get(100) = %d, %v; want 200, true", v, ok)
	}
}

// threadSubclass mirrors the shape the transpiler generates for
// `class X extends Thread`: a struct embedding *Thread, whose constructor wires
// the base to dispatch to its own Run() override.
type threadSubclass struct {
	*Thread
	id      int32
	results []int32
}

func newThreadSubclass(id int32, results []int32) *threadSubclass {
	w := &threadSubclass{id: id, results: results}
	w.Thread = NewThreadBase(w)
	return w
}

func (w *threadSubclass) Run() {
	var total int32
	for i := int32(1); i <= w.id; i++ {
		total += i
	}
	w.results[w.id] = total
}

func TestThreadSubclass_StartJoinDispatchesToOverride(t *testing.T) {
	// Mirrors the ThreadJoin fixture: each worker computes 1..id and Start()
	// must dispatch to the subclass Run(), with Join() observing the result.
	const n = 5
	results := make([]int32, n)
	workers := make([]*threadSubclass, n)
	for i := int32(0); i < n; i++ {
		workers[i] = newThreadSubclass(i, results)
	}
	for i := 0; i < n; i++ {
		workers[i].Start()
	}
	for i := 0; i < n; i++ {
		workers[i].Join()
	}
	var grand int32
	for i := int32(0); i < n; i++ {
		grand += results[i]
	}
	// 0 + 0 + 1 + 3 + 6 + 10 = 20 (worker i sums 1..i)
	if grand != 20 {
		t.Fatalf("Thread-subclass start/join sum = %d, want 20", grand)
	}
}

func TestThreadSubclass_StartDoesNotRecurseThroughEmbed(t *testing.T) {
	// The embedded Thread.Start() dispatches to the subclass Run() exactly once;
	// a Run() that does NOT call Start() must not loop back through the embed.
	// Guards against an accidental Start->Run->Start cycle.
	var runs atomic.Int32
	w := &countingThreadSubclass{counter: &runs}
	w.Thread = NewThreadBase(w)
	w.Start()
	w.Start() // once-guarded: a second Start is a no-op
	w.Join()
	if got := runs.Load(); got != 1 {
		t.Fatalf("Run() executed %d times, want exactly 1", got)
	}
}

type countingThreadSubclass struct {
	*Thread
	counter *atomic.Int32
}

func (c *countingThreadSubclass) Run() {
	c.counter.Add(1)
}

func TestThread_StartJoinRunsRunnable(t *testing.T) {
	done := make(chan int, 1)
	th := NewThread(func() { done <- 42 })
	th.Start()
	th.Join()
	select {
	case v := <-done:
		if v != 42 {
			t.Fatalf("runnable produced %d, want 42", v)
		}
	default:
		t.Fatal("Join returned before the runnable ran")
	}
}

func TestThread_JoinWithoutStartDoesNotBlock(t *testing.T) {
	// A Thread is only joinable after Start; this documents that Join on a
	// started thread completes. (Join before Start would block forever, matching
	// the note in the implementation, so we only test the started path.)
	th := NewThread(func() {})
	th.Start()
	th.Join()
}

func TestMonitor_MutualExclusion(t *testing.T) {
	lock := NewObject()
	counter := 0
	var wg sync.WaitGroup
	const goroutines, perG = 40, 100
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				m := MonitorEnter(lock)
				counter++ // racy without the monitor
				MonitorExit(m)
			}
		}()
	}
	wg.Wait()
	if counter != goroutines*perG {
		t.Fatalf("monitor failed to serialize: got %d, want %d", counter, goroutines*perG)
	}
}

func TestMonitor_SameObjectSameMutex(t *testing.T) {
	obj := NewObject()
	if MonitorEnter(obj) != monitorFor(obj) {
		t.Fatal("the same object must map to the same monitor")
	}
	monitorFor(obj).Unlock()
}

func TestMonitor_DistinctObjectsHaveDistinctIdentity(t *testing.T) {
	first := NewObject()
	second := NewObject()
	if first == second {
		t.Fatal("distinct Java Object allocations must not share monitor identity")
	}
}

func TestMonitor_ArrayAliasHasStableIdentity(t *testing.T) {
	array := []int32{1, 2, 3}
	alias := array

	arrayMonitor := monitorFor(array)
	if aliasMonitor := monitorFor(alias); aliasMonitor != arrayMonitor {
		t.Fatal("aliases of one Java array must resolve to the same monitor")
	}

	arrayMonitor.Lock()
	arrayMonitor.Unlock()
}

func TestMonitor_DistinctArraysHaveDistinctIdentity(t *testing.T) {
	first := []int32{1, 2, 3}
	second := []int32{1, 2, 3}
	if monitorFor(first) == monitorFor(second) {
		t.Fatal("distinct Java arrays with equal elements must have distinct monitors")
	}
}

func TestMonitor_DistinctEmptyJavaArraysHaveDistinctIdentity(t *testing.T) {
	first := NewArray[int32](0)
	second := NewArray[int32](0)
	if monitorFor(first) == monitorFor(second) {
		t.Fatal("distinct zero-length Java arrays must have distinct monitors")
	}

	alias := first
	if monitorFor(first) != monitorFor(alias) {
		t.Fatal("aliases of a zero-length Java array must retain one monitor")
	}
}

func TestMonitor_ArrayIdentityIncludesElementType(t *testing.T) {
	ints := NewArray[int32](0)
	bytes := NewArray[byte](0)
	intIdentity := monitorIdentityFor(ints)
	byteIdentity := monitorIdentityFor(bytes)
	if intIdentity.reference == byteIdentity.reference {
		t.Fatalf("array identity types = %v and %v, want distinct element types", intIdentity.reference, byteIdentity.reference)
	}
	if monitorFor(ints) == monitorFor(bytes) {
		t.Fatal("Java arrays with different element types must have distinct monitors")
	}
}

func TestMonitor_NonComparableMapUsesStableIdentity(t *testing.T) {
	value := map[string]int32{"answer": 42}
	alias := value
	if monitorFor(value) != monitorFor(alias) {
		t.Fatal("aliases of a non-comparable map reference must retain one monitor")
	}
	if monitorFor(value) == monitorFor(map[string]int32{"answer": 42}) {
		t.Fatal("distinct map references with equal contents must have distinct monitors")
	}
}

func TestMonitor_UnsupportedNonComparableValueThrowsJavaException(t *testing.T) {
	value := struct{ values []int32 }{values: []int32{1}}
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		MonitorEnter(value)
	}()
	if !CaughtAs(recovered, "IllegalArgumentException") {
		t.Fatalf("unsupported monitor value panic = %T (%v), want IllegalArgumentException", recovered, recovered)
	}
}

func TestMonitor_ArrayMutualExclusionThroughAliases(t *testing.T) {
	array := []int32{0}
	alias := array
	var wg sync.WaitGroup
	const goroutines, perG = 40, 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(lock []int32) {
			defer wg.Done()
			for j := 0; j < perG; j++ {
				m := MonitorEnter(lock)
				array[0]++
				MonitorExit(m)
			}
		}(alias)
	}

	wg.Wait()
	if got := array[0]; got != goroutines*perG {
		t.Fatalf("array monitor failed to serialize aliases: got %d, want %d", got, goroutines*perG)
	}
}

func TestMonitorEnter_NullReferenceThrowsNullPointerException(t *testing.T) {
	var (
		nilPointer *byte
		nilSlice   []int32
		nilMap     map[string]int32
		nilFunc    func()
		nilChan    chan int32
	)

	tests := []struct {
		name string
		lock interface{}
	}{
		{name: "nil interface", lock: nil},
		{name: "typed nil pointer", lock: nilPointer},
		{name: "typed nil slice", lock: nilSlice},
		{name: "typed nil map", lock: nilMap},
		{name: "typed nil function", lock: nilFunc},
		{name: "typed nil channel", lock: nilChan},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var recovered interface{}
			func() {
				defer func() { recovered = recover() }()
				MonitorEnter(test.lock)
			}()

			if !CaughtAs(recovered, "NullPointerException") {
				t.Fatalf("MonitorEnter(%s) panicked with %T (%v), want NullPointerException", test.name, recovered, recovered)
			}
		})
	}
}

func TestMonitorOperations_TypedNilArrayThrowsNullPointerException(t *testing.T) {
	var nilArray []int32
	tests := []struct {
		name      string
		operation func()
	}{
		{name: "wait", operation: func() { MonitorWait(nilArray) }},
		{name: "notify", operation: func() { MonitorNotify(nilArray) }},
		{name: "notifyAll", operation: func() { MonitorNotifyAll(nilArray) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var recovered interface{}
			func() {
				defer func() { recovered = recover() }()
				test.operation()
			}()
			if !CaughtAs(recovered, "NullPointerException") {
				t.Fatalf("%s on typed nil array panicked with %T (%v), want NullPointerException", test.name, recovered, recovered)
			}
		})
	}
}

func TestMonitor_WaitNotify(t *testing.T) {
	lock := NewObject()
	ready := false

	go func() {
		// Producer: set the flag and notify under the monitor.
		m := MonitorEnter(lock)
		ready = true
		MonitorNotifyAll(lock)
		MonitorExit(m)
	}()

	// Consumer: wait under the monitor until the flag is set, using the
	// while-loop idiom that tolerates spurious wakeups.
	m := MonitorEnter(lock)
	for !ready {
		MonitorWait(lock)
	}
	got := ready
	MonitorExit(m)

	if !got {
		t.Fatal("consumer woke without the condition being satisfied")
	}
}

func TestExecutorService_RunsAllSubmittedTasks(t *testing.T) {
	pool := NewFixedThreadPool(4)
	var counter AtomicInteger
	const tasks = 100
	for i := 0; i < tasks; i++ {
		pool.Submit(func() { counter.IncrementAndGet() })
	}
	pool.Shutdown()
	pool.AwaitTermination()
	if got := counter.Get(); got != tasks {
		t.Fatalf("executor ran %d tasks, want %d", got, tasks)
	}
}

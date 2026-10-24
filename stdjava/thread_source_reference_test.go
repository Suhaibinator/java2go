package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

type threadSourceReferenceFixture struct {
	native *Thread
	info   *ObjectInfo
}

func (source *threadSourceReferenceFixture) JavaObjectInfo() *ObjectInfo { return source.info }

// A source Thread must use its nominal provider before the legacy Go adapter.
func (*threadSourceReferenceFixture) Run() { panic("public Go method shape selected") }
func threadSourceReferenceFixtureFor(t *testing.T, native *Thread) *threadSourceReferenceFixture {
	t.Helper()
	id := TypeID("threadsource." + t.Name())
	RegisterJavaType(id, ThreadTypeID)
	RegisterJavaSourceType(id)
	source := &threadSourceReferenceFixture{native: native}
	source.info = NewObjectInfo(id, func(requested TypeID) any {
		switch requested {
		case id, ObjectTypeID:
			return source
		case ThreadTypeID, RunnableTypeID:
			return native
		}
		return nil
	})
	return source
}
func threadSourceExpectPanic(t *testing.T, call func()) any {
	t.Helper()
	var result any
	func() { defer func() { result = recover() }(); call() }()
	if result == nil {
		t.Fatal("expected Java exception")
	}
	return result
}
func threadSourceAssertUnlocked(t *testing.T, native *Thread) {
	t.Helper()
	if !native.interruptMu.TryLock() {
		t.Error("Java callback entered under native lock")
		return
	}
	native.interruptMu.Unlock()
}

func TestThreadSourceReferenceIdentityViews(t *testing.T) {
	native := NewThreadNamed(nil, "source-name")
	source := threadSourceReferenceFixtureFor(t, native)
	beforeDone := native.done
	if result := BindThreadSourceReference(native, source.info, ThreadSourceMethods{}); result != native || native.done != beforeDone {
		t.Fatal("binding replaced existing lifecycle state")
	}
	if native.JavaObjectInfo() != source.info || native.JavaDynamicTypeID() != source.info.DynamicType() {
		t.Fatal("native view lost source metadata")
	}
	if !JavaReferenceEqual(source, native) || ObjectHashCodeExecution(NewExecution(), source) != ObjectHashCodeExecution(NewExecution(), native) {
		t.Fatal("source/native allocation identity diverged")
	}
	if ObjectView[*Thread](source, ThreadTypeID) != native || ObjectView[*threadSourceReferenceFixture](native, source.info.DynamicType()) != source {
		t.Fatal("canonical/source round trip changed view")
	}
	if ObjectView[Runnable](native, RunnableTypeID) != native {
		t.Fatal("Runnable view changed state")
	}
	array := NewReferenceArray(1, ThreadTypeID)
	ReferenceArraySet(array, 0, source)
	if ReferenceArrayGet[*Thread](array, 0, ThreadTypeID) != native {
		t.Fatal("array projection changed state")
	}
	execution := NewExecution()
	guard := MonitorEnterExecution(execution, source)
	if !ThreadHoldsLockExecution(execution, native) {
		t.Error("source/native monitor diverged")
	}
	MonitorExitExecution(guard)
	if native.GetName() != "source-name" || native.started.Load() || native.IsAlive() {
		t.Fatal("binding changed constructor/lifecycle")
	}
	if ObjectView[*Thread](nil, ThreadTypeID) != nil {
		t.Fatal("null projection allocated")
	}
	plain := NewThread(nil)
	if plain.JavaObjectInfo() != nil || plain.JavaDynamicTypeID() != ThreadTypeID || JavaReferenceEqual(plain, native) {
		t.Fatal("native identity contract changed")
	}
}

func TestThreadSourceReferenceBindingValidationAndImmutableSlots(t *testing.T) {
	native := NewThread(nil)
	source := threadSourceReferenceFixtureFor(t, native)
	if _, ok := threadSourceExpectPanic(t, func() { BindThreadSourceReference(nil, source.info, ThreadSourceMethods{}) }).(NullPointerException); !ok {
		t.Fatal("null state must throw NPE")
	}
	if _, ok := threadSourceExpectPanic(t, func() { BindThreadSourceReference(native, nil, ThreadSourceMethods{}) }).(IllegalArgumentException); !ok {
		t.Fatal("missing identity must fail")
	}
	bad := NewObjectInfo(ObjectTypeID, nil)
	if _, ok := threadSourceExpectPanic(t, func() { BindThreadSourceReference(native, bad, ThreadSourceMethods{}) }).(IllegalArgumentException); !ok {
		t.Fatal("non-Thread identity admitted")
	}
	calls := 0
	methods := ThreadSourceMethods{Run: func(*Execution) { calls++ }}
	BindThreadSourceReference(native, source.info, methods)
	methods.Run = func(*Execution) { t.Error("mutable caller bundle retained") }
	if _, ok := threadSourceExpectPanic(t, func() { BindThreadSourceReference(native, source.info, methods) }).(IllegalStateException); !ok {
		t.Fatal("duplicate binding accepted")
	}
	ThreadRunExecution(NewExecution(), native)
	if calls != 1 || native.JavaObjectInfo() != source.info {
		t.Fatal("failed rebinding changed published identity/callback")
	}
	started := NewThread(nil)
	started.Start()
	started.Join()
	startedSource := threadSourceReferenceFixtureFor(t, started)
	if _, ok := threadSourceExpectPanic(t, func() { BindThreadSourceReference(started, startedSource.info, ThreadSourceMethods{}) }).(IllegalStateException); !ok {
		t.Fatal("binding after start accepted")
	}
	if started.JavaObjectInfo() != nil {
		t.Fatal("failed binding published identity")
	}
}

func TestThreadSourceReferenceInterruptObserverCallerAndDefaults(t *testing.T) {
	callerThread := NewThread(nil)
	caller := &Execution{thread: callerThread}
	local := NewThreadLocal[int32]()
	local.Set(caller, 41)
	native := NewThread(nil)
	source := threadSourceReferenceFixtureFor(t, native)
	calls, observations := 0, 0
	withSuper, throwing := false, false
	sentinel := &struct{ label string }{"abrupt"}
	BindThreadSourceReference(native, source.info, ThreadSourceMethods{
		Interrupt: func(execution *Execution) {
			threadSourceAssertUnlocked(t, native)
			if execution != caller || ThreadCurrentThread(execution) != callerThread || local.Get(execution) != 41 {
				t.Error("interrupt lost caller Execution")
			}
			calls++
			if throwing {
				panic(sentinel)
			}
			if withSuper {
				ThreadInterruptDefaultExecution(execution, native)
			}
		},
		IsInterrupted: func(execution *Execution) bool {
			threadSourceAssertUnlocked(t, native)
			if execution != caller {
				t.Error("observer lost caller Execution")
			}
			observations++
			return !ThreadIsInterruptedDefaultExecution(execution, native)
		},
	})
	ThreadInterruptExecution(caller, native)
	ThreadInterruptExecution(caller, native)
	if calls != 2 || ThreadIsInterruptedDefaultExecution(caller, native) || !ThreadIsInterruptedExecution(caller, native) {
		t.Fatal("no-super/virtual observer semantics changed")
	}
	withSuper = true
	ThreadInterruptExecution(caller, native)
	ThreadInterruptExecution(caller, native)
	if calls != 4 || !ThreadIsInterruptedDefaultExecution(caller, native) || ThreadIsInterruptedExecution(caller, native) {
		t.Fatal("true native bypass/observer semantics changed")
	}
	throwing = true
	for _, pending := range []bool{true, false} {
		if !pending {
			native.consumeInterrupt()
		}
		if threadSourceExpectPanic(t, func() { ThreadInterruptExecution(caller, native) }) != sentinel || ThreadIsInterruptedDefaultExecution(caller, native) != pending {
			t.Fatal("abrupt override changed identity/flag")
		}
	}
	ThreadInterruptDefaultExecution(caller, native)
	if !ThreadInterruptedExecution(&Execution{thread: native}) || ThreadInterruptedExecution(&Execution{thread: native}) || observations != 2 {
		t.Fatal("static interrupted dispatched observer or failed to clear current state")
	}
}

func TestThreadSourceReferenceRunTargetAndExecution(t *testing.T) {
	caller := &Execution{thread: NewThread(nil)}
	local := NewThreadLocal[int32]()
	local.Set(caller, 41)
	targets, overrides := 0, 0
	native := NewThread(func(execution *Execution) {
		if execution != caller || local.Get(execution) != 41 {
			t.Error("constructor target lost direct caller")
		}
		targets++
	})
	source := threadSourceReferenceFixtureFor(t, native)
	sentinel := &struct{ marker byte }{1}
	throwing := false
	BindThreadSourceReference(native, source.info, ThreadSourceMethods{Run: func(execution *Execution) {
		threadSourceAssertUnlocked(t, native)
		if execution != caller || ThreadCurrentThread(execution) != ThreadCurrentThread(caller) {
			t.Error("run changed logical caller")
		}
		overrides++
		if throwing {
			panic(sentinel)
		}
		ThreadRunDefaultExecution(execution, native)
	}})
	ThreadRunExecution(caller, native)
	RunRunnableExecution(caller, source)
	RunRunnableExecution(caller, native)
	if overrides != 3 || targets != 3 {
		t.Fatal("sourceRun/native target mixed or promoted recursion")
	}
	ThreadRunDefaultExecution(caller, native)
	if targets != 4 || overrides != 3 {
		t.Fatal("super run re-entered source override")
	}
	throwing = true
	if threadSourceExpectPanic(t, func() { ThreadRunExecution(caller, native) }) != sentinel || targets != 4 || overrides != 4 {
		t.Fatal("run abrupt identity/order changed")
	}
	empty := NewThread(nil)
	emptySource := threadSourceReferenceFixtureFor(t, empty)
	BindThreadSourceReference(empty, emptySource.info, ThreadSourceMethods{})
	ThreadRunExecution(caller, empty)
	ThreadRunDefaultExecution(caller, empty)
	RunRunnableExecution(caller, emptySource)
	if empty.started.Load() {
		t.Fatal("no-target direct run started thread")
	}
}

func TestThreadSourceReferenceStartWorkerAndDirect(t *testing.T) {
	caller := &Execution{thread: NewThread(nil)}
	local := NewThreadLocal[int32]()
	local.Set(caller, 41)
	native := NewThread(func(*Execution) { t.Error("source run incorrectly used native Runnable target") })
	source := threadSourceReferenceFixtureFor(t, native)
	starts := 0
	launch := false
	seen := make(chan *Execution, 1)
	BindThreadSourceReference(native, source.info, ThreadSourceMethods{
		Start: func(execution *Execution) {
			threadSourceAssertUnlocked(t, native)
			if execution != caller {
				t.Error("start override lost caller")
			}
			starts++
			if launch {
				ThreadStartDefaultExecution(execution, native)
			}
		},
		Run: func(execution *Execution) {
			threadSourceAssertUnlocked(t, native)
			if ThreadCurrentThread(execution) != native || execution == caller || local.Get(execution) != 0 {
				t.Error("worker inherited caller token/ThreadLocal or wrong state")
			}
			seen <- execution
		},
	})
	ThreadStartExecution(caller, native)
	if starts != 1 || native.started.Load() {
		t.Fatal("source start without super launched")
	}
	launch = true
	ThreadStartExecution(caller, native)
	native.Join()
	if starts != 2 || !native.started.Load() || native.IsAlive() {
		t.Fatal("source super start lifecycle changed")
	}
	worker := <-seen
	if ThreadCurrentThread(worker).JavaObjectInfo() != source.info {
		t.Fatal("worker lost shared source identity")
	}
	if _, ok := threadSourceExpectPanic(t, func() { ThreadStartDefaultExecution(caller, native) }).(IllegalThreadStateException); !ok {
		t.Fatal("native one-shot lost")
	}
	if starts != 2 {
		t.Fatal("native start bypass re-entered override")
	}
}

func TestThreadSourceReferenceRuntimeEntryGuards(t *testing.T) {
	for _, helper := range []func(*Execution, *Thread){ThreadRunExecution, ThreadRunDefaultExecution, ThreadStartExecution, ThreadStartDefaultExecution} {
		if _, ok := threadSourceExpectPanic(t, func() { helper(nil, nil) }).(IllegalArgumentException); !ok {
			t.Fatal("missing token admitted")
		}
		if _, ok := threadSourceExpectPanic(t, func() { helper(NewExecution(), nil) }).(NullPointerException); !ok {
			t.Fatal("null receiver not Java NPE")
		}
	}
}

func TestThreadSourceReferenceConcurrentPublishedCallbacks(t *testing.T) {
	native := NewThread(nil)
	source := threadSourceReferenceFixtureFor(t, native)
	var calls atomic.Int32
	BindThreadSourceReference(native, source.info, ThreadSourceMethods{
		Run:           func(*Execution) { calls.Add(1) },
		Interrupt:     func(*Execution) { calls.Add(1) },
		IsInterrupted: func(*Execution) bool { calls.Add(1); return true },
	})
	var wait sync.WaitGroup
	for range 16 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			execution := NewExecution()
			ThreadRunExecution(execution, native)
			ThreadInterruptExecution(execution, native)
			if !ThreadIsInterruptedExecution(execution, native) {
				t.Error("published observer lost")
			}
			if native.JavaObjectInfo() != source.info || ObjectView[*Thread](source, ThreadTypeID) != native {
				t.Error("published identity/view lost")
			}
		}()
	}
	wait.Wait()
	if calls.Load() != 48 || ThreadIsInterruptedDefaultExecution(NewExecution(), native) {
		t.Fatal("lost callbacks or native state changed without super")
	}
}

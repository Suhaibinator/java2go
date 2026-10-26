package stdjava

import (
	"sync"
	"testing"
)

func TestThreadNativeStateFlagClearingExecution(t *testing.T) {
	first, second := NewThread(nil), NewThread(nil)
	caller, other := &Execution{thread: first}, &Execution{thread: second}
	if ThreadIsInterruptedExecution(caller, first) || ThreadInterruptedExecution(caller) {
		t.Fatal("fresh flag set")
	}
	ThreadInterruptExecution(caller, first)
	ThreadInterruptExecution(caller, first)
	signal := first.interruptChannel()
	for i := 0; i < 2; i++ {
		if !ThreadIsInterruptedExecution(caller, first) {
			t.Fatal("observer cleared pending flag")
		}
	}
	if ThreadInterruptedExecution(other) || ThreadIsInterruptedExecution(other, second) {
		t.Fatal("cleared another Execution thread")
	}
	ThreadOnSpinWaitExecution(caller)
	if first.interruptChannel() != signal || !ThreadIsInterruptedExecution(caller, first) {
		t.Fatal("hint changed pending state")
	}
	if !ThreadInterruptedExecution(caller) || ThreadInterruptedExecution(caller) {
		t.Fatal("clear was not exactly once")
	}
	reset := first.interruptChannel()
	if reset == signal {
		t.Fatal("consume did not replace signal")
	}
	select {
	case <-reset:
		t.Fatal("cleared signal remained closed")
	default:
	}
	ThreadInterruptExecution(caller, first)
	select {
	case <-reset:
	default:
		t.Fatal("reinterruption did not close replacement signal")
	}
	first.Start()
	first.Join()
	if first.IsAlive() || !ThreadIsInterruptedExecution(caller, first) {
		t.Fatal("termination changed retained flag")
	}
	if !ThreadInterruptedExecution(caller) || ThreadInterruptedExecution(caller) {
		t.Fatal("retained flag clear changed")
	}
}

func TestThreadNativeStateVirtualObserverAndDefault(t *testing.T) {
	target := NewThread(nil)
	caller := &Execution{thread: target}
	local := NewThreadLocal[int32]()
	local.Set(caller, 41)
	calls := 0
	target.isInterruptedExecution = func(execution *Execution) bool {
		calls++
		if execution != caller || ThreadCurrentThread(execution) != target || local.Get(execution) != 41 {
			t.Fatal("observer replaced caller")
		}
		if ThreadIsInterruptedDefaultExecution(execution, target) {
			t.Fatal("override implicitly set flag")
		}
		return true
	}
	first := ThreadIsInterruptedExecution(caller, target)
	second := ThreadIsInterruptedExecution(caller, target)
	if !first || !second || calls != 2 {
		t.Fatal("virtual observer skipped call")
	}
	if ThreadInterruptedExecution(caller) || calls != 2 || ThreadIsInterruptedDefaultExecution(caller, target) {
		t.Fatal("raw clear/default called override")
	}
	sentinel := &struct{ marker int }{73}
	target.isInterruptedExecution = func(execution *Execution) bool {
		calls++
		_ = ThreadIsInterruptedDefaultExecution(execution, target)
		panic(sentinel)
	}
	ThreadInterruptDefaultExecution(caller, target)
	func() {
		defer func() {
			if recover() != sentinel {
				t.Fatal("observer abrupt identity changed")
			}
		}()
		ThreadIsInterruptedExecution(caller, target)
	}()
	if calls != 3 || !ThreadIsInterruptedDefaultExecution(caller, target) || !ThreadInterruptedExecution(caller) || ThreadInterruptedExecution(caller) || calls != 3 {
		t.Fatal("throwing observer or raw clear changed flag/callback")
	}
}

func TestThreadNativeStateNullAndHint(t *testing.T) {
	caller := NewExecution()
	for _, observer := range []func(*Execution, *Thread) bool{ThreadIsInterruptedExecution, ThreadIsInterruptedDefaultExecution} {
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Fatal("nil observer did not throw Java NPE")
				}
			}()
			observer(caller, nil)
		}()
		func() {
			defer func() {
				if _, ok := recover().(IllegalArgumentException); !ok {
					t.Fatal("nil Execution not rejected")
				}
			}()
			observer(nil, NewThread(nil))
		}()
	}
	for _, service := range []func(*Execution){func(e *Execution) { _ = ThreadInterruptedExecution(e) }, ThreadOnSpinWaitExecution} {
		func() {
			defer func() {
				if _, ok := recover().(IllegalArgumentException); !ok {
					t.Fatal("nil Execution not rejected")
				}
			}()
			service(nil)
		}()
	}
	thread := NewThread(nil)
	execution := &Execution{thread: thread}
	local := NewThreadLocal[int32]()
	local.Set(execution, 97)
	thread.Interrupt()
	signal := thread.interruptChannel()
	ThreadOnSpinWaitExecution(execution)
	if ThreadCurrentThread(execution) != thread || local.Get(execution) != 97 || thread.interruptChannel() != signal || !ThreadInterruptedExecution(execution) {
		t.Fatal("hint changed Execution/thread/local/flag")
	}
}

func TestThreadNativeStateConcurrentObserverAndClear(t *testing.T) {
	thread := NewThread(nil)
	execution := &Execution{thread: thread}
	thread.Interrupt()
	var wait sync.WaitGroup
	var lock sync.Mutex
	cleared := 0
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_ = ThreadIsInterruptedExecution(execution, thread)
			ThreadOnSpinWaitExecution(execution)
			if ThreadInterruptedExecution(execution) {
				lock.Lock()
				cleared++
				lock.Unlock()
			}
		}()
	}
	wait.Wait()
	if cleared != 1 || ThreadIsInterruptedExecution(execution, thread) {
		t.Fatal("concurrent clear duplicated/lost status")
	}
}

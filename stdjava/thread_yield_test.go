package stdjava

import "testing"

func TestThreadYieldPreservesExecutionState(t *testing.T) {
	thread := newNamedThread("yield-state-witness")
	execution := &Execution{thread: thread}
	local := NewThreadLocal[int32]()
	local.Set(execution, 41)
	thread.Interrupt()
	signal := thread.interruptChannel()
	for attempt := 0; attempt < 10; attempt++ {
		ThreadYieldExecution(execution)
	}
	if ThreadCurrentThread(execution) != thread {
		t.Fatal("yield replaced the current Java thread")
	}
	if got := local.Get(execution); got != 41 {
		t.Fatalf("yield changed thread-local value to %d", got)
	}
	if thread.interruptChannel() != signal {
		t.Fatal("yield replaced the pending interrupt signal")
	}
	select {
	case <-signal:
	default:
		t.Fatal("yield cleared interruption")
	}
	if !thread.consumeInterrupt() || thread.consumeInterrupt() {
		t.Fatal("yield lost or duplicated interruption")
	}
}

func TestThreadYieldRequiresExecution(t *testing.T) {
	defer func() {
		if _, ok := recover().(IllegalStateException); !ok {
			t.Fatal("missing caller execution must fail before yielding")
		}
	}()
	ThreadYieldExecution(nil)
}

package stdjava

import (
	"fmt"
	"strings"
	"sync"
	"testing"
)

// The state sequence matches campaign/reproducers/jdk-thread-interrupt/native-state-held,
// frozen from three actual JDK21 runs before the runtime service was added.
func TestThreadInterruptExecutionJDKStates(t *testing.T) {
	execution := NewExecution()
	fresh := NewThread(nil)
	status := func(thread *Thread) bool {
		thread.interruptMu.Lock()
		defer thread.interruptMu.Unlock()
		return thread.interrupted
	}
	var out strings.Builder
	fmt.Fprintf(&out, "new-initial:%t:%t\n", status(fresh), fresh.IsAlive())
	ThreadInterruptExecution(execution, fresh)
	signal := fresh.interruptChannel()
	ThreadInterruptExecution(execution, fresh)
	if signal != fresh.interruptChannel() {
		t.Fatal("repeated request replaced signal")
	}
	fmt.Fprintf(&out, "new-repeated:%t:%t\n", status(fresh), fresh.IsAlive())
	fresh.Start()
	fresh.Join()
	fmt.Fprintf(&out, "terminated-retained:%t:%t\n", status(fresh), fresh.IsAlive())
	ended := NewThread(nil)
	ended.Start()
	ended.Join()
	ThreadInterruptExecution(execution, ended)
	ThreadInterruptExecution(execution, ended)
	fmt.Fprintf(&out, "terminated-request:%t:%t\n", status(ended), ended.IsAlive())
	ready, release := make(chan struct{}), make(chan struct{})
	worker := NewThread(func(workerExecution *Execution) {
		close(ready)
		<-release
		current := ThreadCurrentThread(workerExecution)
		fmt.Fprintf(&out, "alive:%t:%t:%t:%t\n", status(current), current.consumeInterrupt(), current.consumeInterrupt(), status(current))
	})
	worker.Start()
	<-ready
	ThreadInterruptExecution(execution, worker)
	ThreadInterruptExecution(execution, worker)
	close(release)
	worker.Join()
	current := ThreadCurrentThread(execution)
	ThreadInterruptExecution(execution, current)
	ThreadInterruptExecution(execution, current)
	fmt.Fprintf(&out, "current:%t:%t:%t:%t\n", status(current), current.consumeInterrupt(), current.consumeInterrupt(), status(current))
	ThreadInterruptExecution(execution, current)
	fmt.Fprintf(&out, "reset:%t:%t\n", current.consumeInterrupt(), current.consumeInterrupt())
	func() {
		defer func() {
			failure, ok := recover().(NullPointerException)
			if !ok {
				t.Fatal("null must throw canonical NPE")
			}
			fmt.Fprintf(&out, "null:java.lang.%s\n", failure.ThrowableTypeName())
		}()
		ThreadInterruptExecution(execution, nil)
	}()
	want := "new-initial:false:false\nnew-repeated:true:false\nterminated-retained:true:false\nterminated-request:true:false\nalive:true:true:false:false\ncurrent:true:true:false:false\nreset:true:false\nnull:java.lang.NullPointerException\n"
	if out.String() != want {
		t.Fatalf("JDK state difference:\n%s", out.String())
	}
}

func TestThreadInterruptExecutionOverrideBoundary(t *testing.T) {
	callerThread := NewThread(nil)
	execution := &Execution{thread: callerThread}
	target := NewThread(nil)
	local := NewThreadLocal[int32]()
	local.Set(execution, 41)
	calls := 0
	target.interruptExecution = func(caller *Execution) {
		calls++
		if caller != execution || ThreadCurrentThread(caller) != callerThread || local.Get(caller) != 41 {
			t.Fatal("callback changed caller execution")
		}
	}
	ThreadInterruptExecution(execution, target)
	ThreadInterruptExecution(execution, target)
	if calls != 2 || target.consumeInterrupt() {
		t.Fatal("override without super altered flag or skipped repeated call")
	}
	target.interruptExecution = func(caller *Execution) { calls++; ThreadInterruptDefaultExecution(caller, target) }
	ThreadInterruptExecution(execution, target)
	signal := target.interruptChannel()
	ThreadInterruptExecution(execution, target)
	if calls != 4 || target.interruptChannel() != signal || !target.consumeInterrupt() || target.consumeInterrupt() {
		t.Fatal("default path not reentrant/once")
	}
	sentinel := &struct{ marker string }{"abrupt"}
	target.interruptExecution = func(*Execution) { panic(sentinel) }
	for _, pending := range []bool{false, true} {
		if pending {
			ThreadInterruptDefaultExecution(execution, target)
		}
		func() {
			defer func() {
				if recover() != sentinel {
					t.Fatal("abrupt identity replaced")
				}
			}()
			ThreadInterruptExecution(execution, target)
		}()
		if target.consumeInterrupt() != pending {
			t.Fatal("throwing override changed pending flag")
		}
	}
	target.interruptExecution = func(*Execution) { t.Fatal("default invoked override") }
	ThreadInterruptDefaultExecution(execution, target)
	if !target.consumeInterrupt() {
		t.Fatal("default did not interrupt")
	}
}

func TestThreadInterruptExecutionConcurrentRequests(t *testing.T) {
	execution := NewExecution()
	target := NewThread(nil)
	var wait sync.WaitGroup
	for i := 0; i < 16; i++ {
		wait.Add(1)
		go func() { defer wait.Done(); ThreadInterruptExecution(execution, target) }()
	}
	wait.Wait()
	if !target.consumeInterrupt() || target.consumeInterrupt() {
		t.Fatal("concurrent requests lost/duplicated flag")
	}
}

func TestThreadInterruptExecutionRequiresTokenAndCanonicalNull(t *testing.T) {
	for _, helper := range []func(*Execution, *Thread){ThreadInterruptExecution, ThreadInterruptDefaultExecution} {
		func() {
			defer func() {
				if _, ok := recover().(IllegalArgumentException); !ok {
					t.Fatal("missing caller execution must fail")
				}
			}()
			helper(nil, nil)
		}()
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Fatal("native nil receiver escaped Java NPE")
				}
			}()
			helper(NewExecution(), nil)
		}()
	}
}

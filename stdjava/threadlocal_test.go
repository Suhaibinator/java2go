package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestThreadLocalExecutionIsolation(t *testing.T) {
	var initialized atomic.Int32
	local := ThreadLocalWithInitial[*Execution](NewSupplierFuncAdapter(func(execution *Execution) *Execution { initialized.Add(1); return execution }))
	var workers sync.WaitGroup
	for range 24 {
		workers.Go(func() {
			execution := NewExecution()
			if local.Get(execution) != execution {
				t.Error("initializer received another execution")
			}
			local.Set(execution, nil)
			if local.Get(execution) != nil {
				t.Error("stored null was reinitialized")
			}
			local.Remove(execution)
			if len(execution.threadLocals) != 0 {
				t.Error("remove retained value")
			}
			if local.Get(execution) != execution {
				t.Error("remove did not reinitialize")
			}
		})
	}
	workers.Wait()
	if initialized.Load() != 48 {
		t.Fatalf("initializations=%d", initialized.Load())
	}
}

func TestThreadLocalInitializerFailureAndReentrantSet(t *testing.T) {
	execution := NewExecution()
	calls := 0
	local := ThreadLocalWithInitial[string](NewSupplierFuncAdapter(func(*Execution) string {
		calls++
		if calls == 1 {
			panic(NewIllegalStateException("retry"))
		}
		return "initialized"
	}))
	func() {
		defer func() {
			if failure := recover(); failure == nil || !CaughtAs(failure, "IllegalStateException") {
				t.Fatalf("initialization failure=%v", failure)
			}
		}()
		local.Get(execution)
	}()
	if value := local.Get(execution); value != "initialized" || calls != 2 {
		t.Fatalf("value=%q calls=%d", value, calls)
	}
	var reentrant *ThreadLocal[string]
	reentrant = ThreadLocalWithInitial[string](NewSupplierFuncAdapter(func(e *Execution) string { reentrant.Set(e, "temporary"); return "returned" }))
	if value := reentrant.Get(execution); value != "returned" || reentrant.Get(execution) != "returned" {
		t.Fatalf("reentrant initializer=%q", value)
	}
}

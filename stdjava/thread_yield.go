package stdjava

import "runtime"

// ThreadYieldExecution is a scheduling hint made by the current logical Java
// execution. It never changes that execution's thread, interruption or local
// state and supplies no ordering, fairness or completion guarantee.
func ThreadYieldExecution(execution *Execution) {
	if execution == nil {
		panic(NewIllegalStateException("Thread.yield requires a logical execution"))
	}
	runtime.Gosched()
}

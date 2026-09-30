package stdjava

import "testing"

func TestExecutorReplacementThreadLifecycle(t *testing.T) {
	pool := NewFixedThreadPool(1)
	oldThread := make(chan *Thread, 1)
	pool.Execute(NewRunnableFuncAdapter(func(execution *Execution) {
		oldThread <- ThreadCurrentThread(execution)
		panic(NewIllegalStateException("replace worker"))
	}))
	replacement := SubmitCallable(pool, NewCallableFuncAdapter(func(execution *Execution) *Thread {
		return ThreadCurrentThread(execution)
	})).Get()
	old := <-oldThread
	if old == replacement || old.IsAlive() || !replacement.IsAlive() {
		t.Errorf("replacement lifecycle: same=%v, old alive=%v, new alive=%v", old == replacement, old.IsAlive(), replacement.IsAlive())
	}
	pool.Shutdown()
	pool.AwaitTermination()
	if replacement.IsAlive() {
		t.Error("replacement remained alive after pool termination")
	}
}

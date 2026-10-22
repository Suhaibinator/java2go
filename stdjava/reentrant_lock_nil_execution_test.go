package stdjava

import "testing"

func TestReentrantLock180NilExecutionRefusal(t *testing.T) {
	lock := NewReentrantLock()
	for _, action := range []func(){
		func() { ReentrantLockLockExecution(nil, lock) },
		func() { ReentrantLockUnlockExecution(nil, lock) },
		func() { ReentrantLockGetHoldCountExecution(nil, lock) },
		func() { ReentrantLockIsHeldByCurrentThreadExecution(nil, lock) },
	} {
		reentrantLock180Panic(t, "IllegalArgumentException", action)
	}
	if ReentrantLockIsLocked(lock) {
		t.Fatal("nil execution created fallback ownership")
	}
}

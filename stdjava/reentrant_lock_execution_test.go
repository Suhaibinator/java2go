package stdjava

import (
	"sync"
	"testing"
	"time"
)

func reentrantLock180Panic(t *testing.T, want string, action func()) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil || !CaughtAs(got, want) {
			t.Fatalf("expected %s, got %T %v", want, got, got)
		}
	}()
	action()
}

func reentrantLock180Wait(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("lock/monitor handoff did not complete")
	}
}

func TestReentrantLock180ExecutionOwnership(t *testing.T) {
	lock := NewReentrantLock()
	owner, other := NewExecution(), NewExecution()
	if ReentrantLockIsLocked(lock) || ReentrantLockGetHoldCountExecution(owner, lock) != 0 {
		t.Fatal("new lock held")
	}
	ReentrantLockLockExecution(owner, lock)
	ReentrantLockLockExecution(owner, lock)
	if ReentrantLockGetHoldCountExecution(owner, lock) != 2 || !ReentrantLockIsHeldByCurrentThreadExecution(owner, lock) || ReentrantLockGetHoldCountExecution(other, lock) != 0 || ReentrantLockIsHeldByCurrentThreadExecution(other, lock) {
		t.Fatal("logical Java execution ownership/reentrancy lost")
	}
	reentrantLock180Panic(t, "IllegalMonitorStateException", func() { ReentrantLockUnlockExecution(other, lock) })
	if ReentrantLockGetHoldCountExecution(owner, lock) != 2 {
		t.Fatal("wrong-owner unlock changed state")
	}
	ReentrantLockUnlockExecution(owner, lock)
	if ReentrantLockGetHoldCountExecution(owner, lock) != 1 || !ReentrantLockIsLocked(lock) {
		t.Fatal("first recursive unlock released lock")
	}
	ReentrantLockUnlockExecution(owner, lock)
	reentrantLock180Panic(t, "IllegalMonitorStateException", func() { ReentrantLockUnlockExecution(owner, lock) })
	if ReentrantLockIsLocked(lock) {
		t.Fatal("last unlock did not release")
	}
}

func TestReentrantLock180CrossExecutionVisibility(t *testing.T) {
	lock := NewReentrantLock()
	owner := NewExecution()
	ReentrantLockLockExecution(owner, lock)
	ReentrantLockLockExecution(owner, lock)
	ready, done := make(chan struct{}), make(chan struct{})
	value, observed := 0, 0
	go func() {
		other := NewExecution()
		close(ready)
		ReentrantLockLockExecution(other, lock)
		observed = value
		ReentrantLockUnlockExecution(other, lock)
		close(done)
	}()
	<-ready
	value = 17
	ReentrantLockUnlockExecution(owner, lock)
	value = 41
	ReentrantLockUnlockExecution(owner, lock)
	reentrantLock180Wait(t, done)
	if observed != 41 {
		t.Fatalf("final unlock must publish state, got %d", observed)
	}
	var workers sync.WaitGroup
	for range 8 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			execution := NewExecution()
			for range 100 {
				ReentrantLockLockExecution(execution, lock)
				value++
				ReentrantLockUnlockExecution(execution, lock)
			}
		}()
	}
	complete := make(chan struct{})
	go func() { workers.Wait(); close(complete) }()
	reentrantLock180Wait(t, complete)
	if value != 841 {
		t.Fatalf("mutual exclusion lost updates: %d", value)
	}
}

func TestReentrantLock180IntrinsicMonitorIndependence(t *testing.T) {
	lock := NewReentrantLock()
	owner := NewExecution()
	guard := MonitorEnterExecution(owner, lock)
	done := make(chan struct{})
	go func() {
		other := NewExecution()
		ReentrantLockLockExecution(other, lock)
		ReentrantLockUnlockExecution(other, lock)
		close(done)
	}()
	reentrantLock180Wait(t, done)
	MonitorExitExecution(guard)
	ReentrantLockLockExecution(owner, lock)
	done = make(chan struct{})
	go func() {
		guard := MonitorEnterExecution(NewExecution(), lock)
		MonitorExitExecution(guard)
		close(done)
	}()
	reentrantLock180Wait(t, done)
	ReentrantLockUnlockExecution(owner, lock)
}

func TestReentrantLock180FinallyAndNull(t *testing.T) {
	lock, owner := NewReentrantLock(), NewExecution()
	reentrantLock180Panic(t, "IllegalStateException", func() {
		ReentrantLockLockExecution(owner, lock)
		defer ReentrantLockUnlockExecution(owner, lock)
		panic(NewIllegalStateException("checkpoint"))
	})
	if ReentrantLockIsLocked(lock) {
		t.Fatal("abrupt finally left lock held")
	}
	var absent *JavaReentrantLock
	for _, action := range []func(){
		func() { ReentrantLockLockExecution(owner, absent) },
		func() { ReentrantLockUnlockExecution(owner, absent) },
		func() { ReentrantLockGetHoldCountExecution(owner, absent) },
		func() { ReentrantLockIsHeldByCurrentThreadExecution(owner, absent) },
		func() { ReentrantLockIsLocked(absent) },
	} {
		reentrantLock180Panic(t, "NullPointerException", action)
	}
}

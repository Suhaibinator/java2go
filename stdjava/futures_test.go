package stdjava

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func futurePanic(t *testing.T, kind string, call func()) any {
	t.Helper()
	var failure any
	func() { defer func() { failure = recover() }(); call() }()
	if failure == nil || !CaughtAs(failure, kind) {
		t.Fatalf("panic = %v; want %s", failure, kind)
	}
	return failure
}

func TestFutureResultFailureAndWorkerSurvival(t *testing.T) {
	pool := NewFixedThreadPool(1)
	defer func() { pool.Shutdown(); pool.AwaitTermination() }()
	cause := NewIllegalStateException("failed")
	failed := SubmitCallable(pool, NewPlainCallableFuncAdapter(func() int { panic(cause) }))
	failure := futurePanic(t, "ExecutionException", func() { failed.Get() })
	if !sameThrowableIdentity(GetCause(failure), cause) {
		t.Fatal("lost original Java cause")
	}
	native := SubmitCallable(pool, NewPlainCallableFuncAdapter(func() int { panic("native failure") }))
	failure = futurePanic(t, "ExecutionException", func() { native.Get() })
	if !CaughtAs(GetCause(failure), "RuntimeException") {
		t.Fatal("native cause is not Throwable")
	}
	arithmetic := SubmitCallable(pool, NewPlainCallableFuncAdapter(func() int { zero := 0; return 1 / zero }))
	failure = futurePanic(t, "ExecutionException", func() { arithmetic.Get() })
	if !CaughtAs(GetCause(failure), "ArithmeticException") {
		t.Fatal("arithmetic panic lost Java type")
	}
	value := SubmitCallable(pool, NewPlainCallableFuncAdapter(func() int { return 42 }))
	if value.GetTimed(1, SECONDS) != 42 || !value.IsDone() || value.IsCancelled() || value.Cancel(true) {
		t.Fatal("bad terminal success state")
	}
}

func TestFutureCancellationAndExecutorTimeout(t *testing.T) {
	pool := NewFixedThreadPool(1)
	release := make(chan struct{})
	started := make(chan struct{})
	var ran atomic.Bool
	running := pool.Submit(func() { close(started); <-release })
	<-started
	queued := pool.Submit(func() { ran.Store(true) })
	futurePanic(t, "TimeoutException", func() { queued.GetTimed(0, NANOSECONDS) })
	futurePanic(t, "TimeoutException", func() { queued.GetTimed(1, MILLISECONDS) })
	futurePanic(t, "NullPointerException", func() { queued.GetTimed(0, nil) })
	if !queued.Cancel(false) || !queued.IsDone() || !queued.IsCancelled() || queued.Cancel(true) {
		t.Fatal("bad cancellation state")
	}
	futurePanic(t, "CancellationException", func() { queued.Get() })
	if !running.Cancel(true) {
		t.Fatal("running task could not be cancelled")
	}
	futurePanic(t, "CancellationException", func() { running.GetTimed(1, SECONDS) })
	pool.Shutdown()
	if !pool.IsShutdown() || pool.IsTerminated() || pool.AwaitTerminationTimed(0, SECONDS) || pool.AwaitTerminationTimed(1, MILLISECONDS) {
		t.Fatal("terminated before running task returned")
	}
	futurePanic(t, "NullPointerException", func() { pool.AwaitTerminationTimed(0, nil) })
	close(release)
	if !pool.AwaitTerminationTimed(1, SECONDS) || ran.Load() {
		t.Fatal("cancelled queued task ran or pool failed to terminate")
	}
	futurePanic(t, "RejectedExecutionException", func() { pool.Submit(func() {}) })
	futurePanic(t, "RejectedExecutionException", func() { pool.Execute(func() {}) })
}

func TestExecutorSubmitShutdownRaces(t *testing.T) {
	for round := 0; round < 20; round++ {
		pool := NewFixedThreadPool(4)
		var submitters sync.WaitGroup
		var accepted, completed atomic.Int32
		for i := 0; i < 8; i++ {
			submitters.Add(1)
			go func() {
				defer submitters.Done()
				for j := 0; j < 40; j++ {
					func() {
						defer func() {
							if failure := recover(); failure != nil && !CaughtAs(failure, "RejectedExecutionException") {
								t.Errorf("submission panic: %v", failure)
							}
						}()
						pool.Submit(func() { completed.Add(1) })
						accepted.Add(1)
					}()
				}
			}()
		}
		pool.Shutdown()
		submitters.Wait()
		if !pool.AwaitTerminationTimed(2, SECONDS) {
			t.Fatal("shutdown hung")
		}
		if completed.Load() != accepted.Load() {
			t.Fatalf("accepted %d, completed %d", accepted.Load(), completed.Load())
		}
	}
}

func TestFutureConcurrentGetCancelAndCompletion(t *testing.T) {
	pool := NewFixedThreadPool(4)
	defer func() { pool.Shutdown(); pool.AwaitTermination() }()
	for i := 0; i < 100; i++ {
		future := SubmitCallable(pool, NewPlainCallableFuncAdapter(func() int { return 7 }))
		var callers sync.WaitGroup
		for j := 0; j < 8; j++ {
			callers.Add(1)
			go func() {
				defer callers.Done()
				defer func() {
					if failure := recover(); failure != nil && !CaughtAs(failure, "CancellationException") {
						t.Errorf("get panic: %v", failure)
					}
				}()
				if result := future.GetTimed(2, SECONDS); result != 7 {
					t.Errorf("value = %d", result)
				}
			}()
		}
		future.Cancel(false)
		callers.Wait()
		if !future.IsDone() {
			t.Fatal("future not terminal")
		}
	}
}

func TestExecutorUnboundedQueueAndTimeUnitSaturation(t *testing.T) {
	pool := NewFixedThreadPool(1)
	release := make(chan struct{})
	pool.Submit(func() { <-release })
	// More than the old 64-slot queue must be accepted without blocking.
	for i := 0; i < 1000; i++ {
		pool.Submit(func() {})
	}
	pool.Shutdown()
	close(release)
	if !pool.AwaitTerminationTimed(2, SECONDS) {
		t.Fatal("queue did not drain")
	}
	if DAYS.duration(math.MaxInt64) != time.Duration(math.MaxInt64) {
		t.Fatal("timeout overflow")
	}
	if SECONDS.duration(-1) != 0 {
		t.Fatal("negative timeout must poll")
	}
	futurePanic(t, "IllegalArgumentException", func() { NewFixedThreadPool(0) })
	futurePanic(t, "NullPointerException", func() { pool.Submit(nil) })
}

func TestThreadLifecycleAndTimedJoin(t *testing.T) {
	release := make(chan struct{})
	thread := NewThread(func() { <-release })
	thread.Join()
	if thread.IsAlive() {
		t.Fatal("unstarted thread is alive")
	}
	thread.Start()
	thread.JoinTimed(1)
	if !thread.IsAlive() {
		t.Fatal("join timeout killed thread")
	}
	futurePanic(t, "IllegalThreadStateException", thread.Start)
	futurePanic(t, "IllegalArgumentException", func() { thread.JoinTimed(-1) })
	futurePanic(t, "IllegalArgumentException", func() { thread.JoinTimed(0, 1000000) })
	close(release)
	thread.Join()
	if thread.IsAlive() {
		t.Fatal("joined thread is alive")
	}
}

package stdjava

import (
	"sync"
	"sync/atomic"
	"testing"
)

func recoverClassInitializationPanic(fn func()) (recovered interface{}) {
	defer func() {
		recovered = recover()
	}()
	fn()
	return nil
}

func TestClassInitialization_SuccessRunsExactlyOnce(t *testing.T) {
	initialization := NewClassInitialization("tests.Success")
	firstExecution := NewExecution()
	secondExecution := NewExecution()
	var calls int

	initialization.Ensure(firstExecution, func(received *Execution) {
		calls++
		if received != firstExecution {
			t.Fatalf("initializer received execution %p, want %p", received, firstExecution)
		}
	})
	initialization.Ensure(firstExecution, func(*Execution) { calls++ })
	initialization.Ensure(secondExecution, func(*Execution) { calls++ })

	if calls != 1 {
		t.Fatalf("initializer ran %d times, want exactly once", calls)
	}
}

func TestClassInitialization_SameExecutionRecursionReturns(t *testing.T) {
	initialization := NewClassInitialization("tests.Recursive")
	execution := NewExecution()
	var outerCalls, recursiveBodyCalls int

	initialization.Ensure(execution, func(received *Execution) {
		outerCalls++
		initialization.Ensure(received, func(*Execution) {
			recursiveBodyCalls++
		})
	})

	if outerCalls != 1 {
		t.Fatalf("outer initializer ran %d times, want 1", outerCalls)
	}
	if recursiveBodyCalls != 0 {
		t.Fatalf("recursive initializer body ran %d times, want 0", recursiveBodyCalls)
	}
}

func TestClassInitialization_MutualRecursionOnSameExecutionReturns(t *testing.T) {
	first := NewClassInitialization("tests.First")
	second := NewClassInitialization("tests.Second")
	execution := NewExecution()
	var firstCalls, secondCalls, recursiveFirstBodyCalls int

	first.Ensure(execution, func(received *Execution) {
		firstCalls++
		second.Ensure(received, func(received *Execution) {
			secondCalls++
			first.Ensure(received, func(*Execution) {
				recursiveFirstBodyCalls++
			})
		})
	})

	if firstCalls != 1 {
		t.Fatalf("first class initializer ran %d times, want 1", firstCalls)
	}
	if secondCalls != 1 {
		t.Fatalf("second class initializer ran %d times, want 1", secondCalls)
	}
	if recursiveFirstBodyCalls != 0 {
		t.Fatalf("recursive first-class body ran %d times, want 0", recursiveFirstBodyCalls)
	}

	first.Ensure(NewExecution(), func(*Execution) { firstCalls++ })
	second.Ensure(NewExecution(), func(*Execution) { secondCalls++ })
	if firstCalls != 1 || secondCalls != 1 {
		t.Fatalf("completed mutually recursive classes reran: first=%d second=%d", firstCalls, secondCalls)
	}
}

func TestClassInitialization_DifferentExecutionsWaitForSuccess(t *testing.T) {
	initialization := NewClassInitialization("tests.ConcurrentSuccess")
	entered := make(chan struct{})
	release := make(chan struct{})
	const waiters = 32

	var calls atomic.Int32
	var group sync.WaitGroup
	group.Add(waiters + 1)
	go func() {
		defer group.Done()
		initialization.Ensure(NewExecution(), func(*Execution) {
			calls.Add(1)
			close(entered)
			<-release
		})
	}()
	<-entered

	for index := 0; index < waiters; index++ {
		go func() {
			defer group.Done()
			initialization.Ensure(NewExecution(), func(*Execution) {
				calls.Add(1)
			})
		}()
	}
	close(release)
	group.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("concurrent initializer ran %d times, want 1", got)
	}
}

func TestClassInitialization_NonErrorIsWrappedThenClassIsErroneous(t *testing.T) {
	initialization := NewClassInitialization("tests.Broken")
	original := NewIllegalStateException("initializer failed")

	first := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), func(*Execution) {
			panic(original)
		})
	})
	if !CaughtAs(first, "ExceptionInInitializerError") {
		t.Fatalf("first active use panicked with %T (%v), want ExceptionInInitializerError", first, first)
	}
	if cause := GetCause(first); !sameThrowableIdentity(cause, original) {
		t.Fatalf("first-use cause = %T (%v), want original initializer throwable", cause, cause)
	}

	bodyCalled := false
	second := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), func(*Execution) {
			bodyCalled = true
		})
	})
	if bodyCalled {
		t.Fatal("an erroneous class attempted to run its initializer again")
	}
	if !CaughtAs(second, "NoClassDefFoundError") {
		t.Fatalf("second active use panicked with %T (%v), want NoClassDefFoundError", second, second)
	}
	secondCause := GetCause(second)
	if !CaughtAs(secondCause, "ExceptionInInitializerError") {
		t.Fatalf("second-use cause = %T (%v), want ExceptionInInitializerError summary", secondCause, secondCause)
	}
	if sameThrowableIdentity(secondCause, first) {
		t.Fatal("second-use summary must be distinct from the owner-visible failure")
	}
	if cause := GetCause(secondCause); cause != nil {
		t.Fatalf("second-use summary cause = %T (%v), want nil", cause, cause)
	}

	third := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), nil)
	})
	if !CaughtAs(third, "NoClassDefFoundError") {
		t.Fatalf("third active use panicked with %T (%v), want NoClassDefFoundError", third, third)
	}
	if cause := GetCause(third); !sameThrowableIdentity(cause, secondCause) {
		t.Fatalf("third-use cause = %T (%v), want shared second-use summary", cause, cause)
	}
}

func TestClassInitialization_ErrorPropagatesUnwrapped(t *testing.T) {
	initialization := NewClassInitialization("tests.Error")
	original := NewAssertionError("fatal initializer")

	first := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), func(*Execution) {
			panic(original)
		})
	})
	if !CaughtAs(first, "AssertionError") {
		t.Fatalf("first active use panicked with %T (%v), want AssertionError", first, first)
	}
	if !sameThrowableIdentity(first, original) {
		t.Fatal("first active use did not propagate the original Error unchanged")
	}

	second := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), func(*Execution) {
			t.Fatal("erroneous class reran its initializer")
		})
	})
	if !CaughtAs(second, "NoClassDefFoundError") {
		t.Fatalf("second active use panicked with %T (%v), want NoClassDefFoundError", second, second)
	}
	secondCause := GetCause(second)
	if !CaughtAs(secondCause, "ExceptionInInitializerError") {
		t.Fatalf("second-use cause = %T (%v), want ExceptionInInitializerError summary", secondCause, secondCause)
	}
	if sameThrowableIdentity(secondCause, original) {
		t.Fatal("second-use summary must be distinct from the owner-visible Error")
	}
	if cause := GetCause(secondCause); cause != nil {
		t.Fatalf("second-use summary cause = %T (%v), want nil", cause, cause)
	}
}

func TestClassInitialization_WaitersObserveErroneousClass(t *testing.T) {
	initialization := NewClassInitialization("tests.ConcurrentFailure")
	entered := make(chan struct{})
	release := make(chan struct{})
	ownerResult := make(chan interface{}, 1)
	const waiters = 32

	go func() {
		ownerResult <- recoverClassInitializationPanic(func() {
			initialization.Ensure(NewExecution(), func(*Execution) {
				close(entered)
				<-release
				panic(NewRuntimeException("boom"))
			})
		})
	}()
	<-entered

	results := make(chan interface{}, waiters)
	launched := make(chan struct{}, waiters)
	var group sync.WaitGroup
	group.Add(waiters)
	for index := 0; index < waiters; index++ {
		go func() {
			defer group.Done()
			launched <- struct{}{}
			results <- recoverClassInitializationPanic(func() {
				initialization.Ensure(NewExecution(), func(*Execution) {
					t.Error("waiter ran the initializer body")
				})
			})
		}()
	}
	for index := 0; index < waiters; index++ {
		<-launched
	}
	close(release)
	group.Wait()
	close(results)

	first := <-ownerResult
	if !CaughtAs(first, "ExceptionInInitializerError") {
		t.Fatalf("initializing execution got %T (%v), want ExceptionInInitializerError", first, first)
	}
	var sharedSummary interface{}
	for result := range results {
		if !CaughtAs(result, "NoClassDefFoundError") {
			t.Fatalf("waiting execution got %T (%v), want NoClassDefFoundError", result, result)
		}
		cause := GetCause(result)
		if !CaughtAs(cause, "ExceptionInInitializerError") {
			t.Fatalf("waiter cause = %T (%v), want ExceptionInInitializerError summary", cause, cause)
		}
		if sameThrowableIdentity(cause, first) {
			t.Fatal("waiter summary must be distinct from the owner-visible failure")
		}
		if nested := GetCause(cause); nested != nil {
			t.Fatalf("waiter summary cause = %T (%v), want nil", nested, nested)
		}
		if sharedSummary == nil {
			sharedSummary = cause
		} else if !sameThrowableIdentity(cause, sharedSummary) {
			t.Fatal("waiters did not share one later-use initialization summary")
		}
	}

	later := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), nil)
	})
	if !CaughtAs(later, "NoClassDefFoundError") {
		t.Fatalf("later active use got %T (%v), want NoClassDefFoundError", later, later)
	}
	if cause := GetCause(later); !sameThrowableIdentity(cause, sharedSummary) {
		t.Fatalf("later-use cause = %T (%v), want shared waiter summary", cause, cause)
	}
}

func TestClassInitialization_NormalizesNonJavaPanicBeforeWrapping(t *testing.T) {
	initialization := NewClassInitialization("tests.NativeFailure")
	first := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), func(*Execution) {
			panic("native failure")
		})
	})
	if !CaughtAs(first, "ExceptionInInitializerError") {
		t.Fatalf("first active use panicked with %T (%v), want ExceptionInInitializerError", first, first)
	}
	cause := GetCause(first)
	if !CaughtAs(cause, "RuntimeException") || GetMessage(cause) != "native failure" {
		t.Fatalf("normalized cause = %T (%v), want RuntimeException(native failure)", cause, cause)
	}
}

func TestClassInitialization_NilExecutionFailsWithoutStarting(t *testing.T) {
	initialization := NewClassInitialization("tests.NilExecution")
	called := false
	recovered := recoverClassInitializationPanic(func() {
		initialization.Ensure(nil, func(*Execution) {
			called = true
		})
	})
	if called {
		t.Fatal("initializer body ran with a nil execution")
	}
	if !CaughtAs(recovered, "IllegalArgumentException") {
		t.Fatalf("nil execution panicked with %T (%v), want IllegalArgumentException", recovered, recovered)
	}

	initialization.Ensure(NewExecution(), func(*Execution) {
		called = true
	})
	if !called {
		t.Fatal("nil-execution rejection incorrectly poisoned the class")
	}
}

func TestClassInitialization_NilStateFailsDeterministically(t *testing.T) {
	var initialization *ClassInitialization
	recovered := recoverClassInitializationPanic(func() {
		initialization.Ensure(NewExecution(), nil)
	})
	if !CaughtAs(recovered, "IllegalArgumentException") {
		t.Fatalf("nil state panicked with %T (%v), want IllegalArgumentException", recovered, recovered)
	}
}

func TestClassInitialization_ZeroValueCanInitialize(t *testing.T) {
	var initialization ClassInitialization
	calls := 0
	initialization.Ensure(NewExecution(), func(*Execution) { calls++ })
	initialization.Ensure(NewExecution(), func(*Execution) { calls++ })
	if calls != 1 {
		t.Fatalf("zero-value initializer ran %d times, want 1", calls)
	}
}

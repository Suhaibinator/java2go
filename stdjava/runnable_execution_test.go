package stdjava

import "testing"

func TestRunRunnableExecutionAdaptsExecutionLambda(t *testing.T) {
	execution := NewExecution()
	var received *Execution

	RunRunnableExecution(execution, func(actual *Execution) {
		received = actual
	})

	if received != execution {
		t.Fatalf("execution lambda received %p, want caller token %p", received, execution)
	}
}

type collidingExecutionRunnable struct {
	publicCalls   int
	decoyCalls    int
	hiddenCalls   int
	receivedToken *Execution
}

func (r *collidingExecutionRunnable) Run() {
	r.publicCalls++
}

// RunJava2goExecution represents the colliding user-declared Java member. Its
// signature deliberately prevents it from satisfying executionRunnable.
func (r *collidingExecutionRunnable) RunJava2goExecution() {
	r.decoyCalls++
}

// RunJava2goExecution1 represents the collision-safe hidden implementation
// emitted for Java's actual Runnable.run method.
func (r *collidingExecutionRunnable) RunJava2goExecution1(execution *Execution) {
	r.hiddenCalls++
	r.receivedToken = execution
}

func TestRunRunnableExecutionFindsCollisionSafeHiddenMethod(t *testing.T) {
	execution := NewExecution()
	runnable := &collidingExecutionRunnable{}

	RunRunnableExecution(execution, runnable)

	if runnable.hiddenCalls != 1 || runnable.receivedToken != execution {
		t.Fatalf("hidden execution method calls/token = %d/%p, want 1/%p", runnable.hiddenCalls, runnable.receivedToken, execution)
	}
	if runnable.publicCalls != 0 || runnable.decoyCalls != 0 {
		t.Fatalf("fallback or colliding method ran: public=%d decoy=%d", runnable.publicCalls, runnable.decoyCalls)
	}
}

type plainRunnable struct {
	calls int
}

func (r *plainRunnable) Run() {
	r.calls++
}

func TestRunRunnableExecutionFallsBackForExternalRunnable(t *testing.T) {
	runnable := &plainRunnable{}
	RunRunnableExecution(NewExecution(), runnable)
	if runnable.calls != 1 {
		t.Fatalf("external Runnable calls = %d, want 1", runnable.calls)
	}
}

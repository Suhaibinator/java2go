package stdjava

import "sync"

// ClassInitialization coordinates the one-time, demand-driven initialization
// of one generated Java class. Java class initialization differs from sync.Once
// in two important ways: recursive use by the execution currently performing
// initialization succeeds immediately, and a failed initialization permanently
// marks the class erroneous.
//
// A ClassInitialization must not be copied after its first use.
type ClassInitialization struct {
	mu            sync.Mutex
	condition     *sync.Cond
	state         classInitializationState
	owner         *Execution
	laterUseCause interface{}
	name          string
}

type classInitializationState uint8

const (
	classUninitialized classInitializationState = iota
	classInitializing
	classInitialized
	classErroneous
)

// NewClassInitialization creates the initialization state for the Java class
// identified by javaBinaryName (for example, "example.Outer$Nested").
func NewClassInitialization(javaBinaryName string) *ClassInitialization {
	initialization := &ClassInitialization{name: javaBinaryName}
	initialization.condition = sync.NewCond(&initialization.mu)
	return initialization
}

// Ensure initializes the class on its first active use.
//
// If initialization is already in progress on the same logical Java execution,
// Ensure returns immediately. A different execution waits for that attempt to
// finish. Successful initialization is never repeated. If body completes
// abruptly, a Java Error is propagated unchanged; every other throwable is
// wrapped in ExceptionInInitializerError. Later active uses fail with
// NoClassDefFoundError values that share a cause-less
// ExceptionInInitializerError summary of the failed initialization.
func (initialization *ClassInitialization) Ensure(execution *Execution, body func(*Execution)) {
	if execution == nil {
		panic(NewIllegalArgumentException("class initialization requires a non-nil execution"))
	}
	if initialization == nil {
		panic(NewIllegalArgumentException("class initialization state must not be nil"))
	}

	initialization.mu.Lock()
	initialization.ensureConditionLocked()
	for {
		switch initialization.state {
		case classUninitialized:
			initialization.state = classInitializing
			initialization.owner = execution
			initialization.mu.Unlock()

			failure := runClassInitializer(execution, body)
			if failure == nil {
				initialization.mu.Lock()
				initialization.owner = nil
				initialization.state = classInitialized
				initialization.condition.Broadcast()
				initialization.mu.Unlock()
				return
			}

			failure = NormalizePanic(failure)
			if !CaughtAs(failure, "Error") {
				failure = NewExceptionInInitializerError(failure)
			}

			initialization.mu.Lock()
			initialization.owner = nil
			initialization.laterUseCause = NewExceptionInInitializerError("")
			initialization.state = classErroneous
			initialization.condition.Broadcast()
			initialization.mu.Unlock()
			panic(failure)

		case classInitializing:
			if initialization.owner == execution {
				initialization.mu.Unlock()
				return
			}
			initialization.condition.Wait()

		case classInitialized:
			initialization.mu.Unlock()
			return

		case classErroneous:
			name := initialization.name
			cause := initialization.laterUseCause
			initialization.mu.Unlock()
			panic(NewNoClassDefFoundErrorWithCause("Could not initialize class "+name, cause))
		}
	}
}

// ensureConditionLocked makes the zero value usable as a convenience for
// runtime tests and defensive callers. Generated code uses
// NewClassInitialization so its Java binary name is retained in diagnostics.
func (initialization *ClassInitialization) ensureConditionLocked() {
	if initialization.condition == nil {
		initialization.condition = sync.NewCond(&initialization.mu)
	}
}

func runClassInitializer(execution *Execution, body func(*Execution)) (failure interface{}) {
	defer func() {
		failure = recover()
	}()
	if body != nil {
		body(execution)
	}
	return nil
}

package stdjava

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

// TimeUnit carries the nanosecond scale of a java.util.concurrent.TimeUnit.
type TimeUnit struct{ nanos int64 }

var (
	NANOSECONDS  = &TimeUnit{1}
	MICROSECONDS = &TimeUnit{1000}
	MILLISECONDS = &TimeUnit{1000000}
	SECONDS      = &TimeUnit{1000000000}
	MINUTES      = &TimeUnit{60 * 1000000000}
	HOURS        = &TimeUnit{3600 * 1000000000}
	DAYS         = &TimeUnit{86400 * 1000000000}
)

func (u *TimeUnit) duration(n int64) time.Duration {
	if u == nil {
		panic(NewNullPointerException("time unit"))
	}
	if n <= 0 {
		return 0
	}
	if n > math.MaxInt64/u.nanos {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(n * u.nanos)
}

// Callable preserves execution identity across worker callbacks and direct calls.
type Callable[T any] interface{ Call() T }
type CallableFuncAdapter[T any] struct{ call func(*Execution) T }

func NewCallableFuncAdapter[T any](call func(*Execution) T) *CallableFuncAdapter[T] {
	return &CallableFuncAdapter[T]{call}
}
func NewPlainCallableFuncAdapter[T any](call func() T) *CallableFuncAdapter[T] {
	return NewCallableFuncAdapter(func(_ *Execution) T { return call() })
}
func (c *CallableFuncAdapter[T]) Call() T                             { return c.call(NewExecution()) }
func (c *CallableFuncAdapter[T]) CallJava2goExecution(e *Execution) T { return c.call(e) }
func (*CallableFuncAdapter[T]) JavaDynamicTypeID() TypeID             { return "Callable" }
func CallCallableExecution[T any](e *Execution, task Callable[T]) T {
	ReferenceRequireNonNull(task)
	if aware, ok := task.(interface{ CallJava2goExecution(*Execution) T }); ok {
		return aware.CallJava2goExecution(e)
	}
	return task.Call()
}

// Future publishes exactly one terminal state. Cancellation wakes getters even
// when a running task cannot yet return. Go goroutines cannot be interrupted;
// cancel(true) requests cancellation but does not forcibly stop a task body.
type Future[T any] struct {
	mu                   sync.Mutex
	done                 chan struct{}
	completed, cancelled bool
	value                T
	failure              any
}

func newFuture[T any]() *Future[T] { return &Future[T]{done: make(chan struct{})} }
func (f *Future[T]) finish(value T, failure any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.completed {
		return
	}
	failure = NormalizePanic(failure)
	f.value, f.failure, f.completed = value, failure, true
	close(f.done)
}
func (f *Future[T]) Cancel(_ bool) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.completed {
		return false
	}
	f.completed, f.cancelled = true, true
	close(f.done)
	return true
}
func (f *Future[T]) IsDone() bool      { f.mu.Lock(); defer f.mu.Unlock(); return f.completed }
func (f *Future[T]) IsCancelled() bool { f.mu.Lock(); defer f.mu.Unlock(); return f.cancelled }
func (f *Future[T]) result() T {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cancelled {
		panic(NewCancellationException(""))
	}
	if f.failure != nil {
		panic(NewExecutionException(f.failure))
	}
	return f.value
}
func (f *Future[T]) Get() T { <-f.done; return f.result() }
func (f *Future[T]) GetTimed(timeout int64, unit *TimeUnit) T {
	duration := unit.duration(timeout)
	select {
	case <-f.done:
		return f.result()
	default:
	}
	if duration == 0 {
		panic(NewTimeoutException(""))
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-f.done:
		return f.result()
	case <-timer.C:
		if f.IsDone() {
			return f.result()
		}
		panic(NewTimeoutException(""))
	}
}

// ExecutorService uses an unbounded queue guarded by the same mutex as
// shutdown. Submissions never block behind a full channel and shutdown cannot
// close a channel while a producer sends to it.
type ExecutorService struct {
	mu         sync.Mutex
	available  *sync.Cond
	queue      []func(*Execution)
	shutdown   bool
	workers    sync.WaitGroup
	terminated chan struct{}
}

func NewFixedThreadPool(n int32) *ExecutorService {
	if n <= 0 {
		panic(NewIllegalArgumentException("pool size must be positive"))
	}
	e := &ExecutorService{terminated: make(chan struct{})}
	e.available = sync.NewCond(&e.mu)
	e.workers.Add(int(n))
	for i := int32(0); i < n; i++ {
		go e.worker()
	}
	go func() { e.workers.Wait(); close(e.terminated) }()
	return e
}
func (e *ExecutorService) worker() {
	defer e.workers.Done()
	execution := NewExecution()
	for {
		e.mu.Lock()
		for len(e.queue) == 0 && !e.shutdown {
			e.available.Wait()
		}
		if len(e.queue) == 0 {
			e.mu.Unlock()
			return
		}
		task := e.queue[0]
		e.queue[0] = nil
		e.queue = e.queue[1:]
		e.mu.Unlock()
		// An execute task's uncaught exception ends the Java worker. Continuing
		// this goroutine models the pool's replacement worker without crashing
		// the Go process; submitted tasks capture failure in their Future.
		func() {
			defer func() {
				if failure := recover(); failure != nil {
					fmt.Fprintln(os.Stderr, "Exception in executor worker:", failure)
					execution = NewExecution()
				}
			}()
			task(execution)
		}()
	}
}
func (e *ExecutorService) enqueue(task func(*Execution)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.shutdown {
		panic(NewRejectedExecutionException("executor is shut down"))
	}
	e.queue = append(e.queue, task)
	e.available.Signal()
}
func SubmitCallable[T any](e *ExecutorService, task Callable[T]) *Future[T] {
	ReferenceRequireNonNull(task)
	return submitFuture(e, func(execution *Execution) T { return CallCallableExecution(execution, task) })
}
func submitFuture[T any](e *ExecutorService, call func(*Execution) T) *Future[T] {
	f := newFuture[T]()
	e.enqueue(func(execution *Execution) {
		if f.IsCancelled() {
			return
		}
		var value T
		defer func() { f.finish(value, recover()) }()
		value = call(execution)
	})
	return f
}
func SubmitRunnableResult[T any](e *ExecutorService, task any, result T) *Future[T] {
	ReferenceRequireNonNull(task)
	return submitFuture(e, func(execution *Execution) T { RunRunnableExecution(execution, task); return result })
}
func (e *ExecutorService) Submit(task any) *Future[any] {
	return SubmitRunnableResult[any](e, task, nil)
}
func (e *ExecutorService) Execute(task any) {
	ReferenceRequireNonNull(task)
	e.enqueue(func(execution *Execution) { RunRunnableExecution(execution, task) })
}
func (e *ExecutorService) Shutdown() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.shutdown = true
	e.available.Broadcast()
}
func (e *ExecutorService) IsShutdown() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.shutdown }
func (e *ExecutorService) IsTerminated() bool {
	select {
	case <-e.terminated:
		return true
	default:
		return false
	}
}

// AwaitTermination retains the old Go-facing blocking entry point.
func (e *ExecutorService) AwaitTermination() { <-e.terminated }
func (e *ExecutorService) AwaitTerminationTimed(timeout int64, unit *TimeUnit) bool {
	duration := unit.duration(timeout)
	if e.IsTerminated() {
		return true
	}
	if duration == 0 {
		return false
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-e.terminated:
		return true
	case <-timer.C:
		return e.IsTerminated()
	}
}

func reportUncaughtTaskException() {
	if failure := recover(); failure != nil {
		fmt.Fprintln(os.Stderr, "Exception in thread:", failure)
	}
}

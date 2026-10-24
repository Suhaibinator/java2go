package stdjava

import (
	"sync"
	"time"
)

// CountDownLatch releases all current and future waiters once its count reaches
// zero. Closing released publishes writes preceding the final CountDown.
type CountDownLatch struct {
	mu       sync.Mutex
	count    int32
	released chan struct{}
}

func NewCountDownLatch(count int32) *CountDownLatch {
	if count < 0 {
		panic(NewIllegalArgumentException("count < 0"))
	}
	latch := &CountDownLatch{count: count, released: make(chan struct{})}
	if count == 0 {
		close(latch.released)
	}
	return latch
}

func (l *CountDownLatch) CountDown() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.count > 0 {
		l.count--
		if l.count == 0 {
			close(l.released)
		}
	}
}

func (l *CountDownLatch) GetCount() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int64(l.count)
}

func (l *CountDownLatch) Await() { <-l.released }

func (l *CountDownLatch) AwaitTimed(timeout int64, unit *TimeUnit) bool {
	duration := unit.duration(timeout)
	select {
	case <-l.released:
		return true
	default:
	}
	if duration <= 0 {
		return false
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-l.released:
		return true
	case <-timer.C:
		return false
	}
}

// AwaitExecution implements interruptible waiting for generated Java calls.
// The public Go-facing Await remains a non-interruptible convenience method.
func (l *CountDownLatch) AwaitExecution(execution *Execution) {
	thread := ThreadCurrentThread(execution)
	if thread.consumeInterrupt() {
		panic(newThrowableBase("InterruptedException", ""))
	}
	select {
	case <-thread.interruptChannel():
		thread.consumeInterrupt()
		panic(newThrowableBase("InterruptedException", ""))
	case <-l.released:
	}
}

func (l *CountDownLatch) AwaitTimedExecution(execution *Execution, timeout int64, unit *TimeUnit) bool {
	duration := unit.duration(timeout)
	thread := ThreadCurrentThread(execution)
	if thread.consumeInterrupt() {
		panic(newThrowableBase("InterruptedException", ""))
	}
	select {
	case <-l.released:
		return true
	default:
	}
	if duration <= 0 {
		return false
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-thread.interruptChannel():
		thread.consumeInterrupt()
		panic(newThrowableBase("InterruptedException", ""))
	case <-l.released:
		return true
	case <-timer.C:
		return false
	}
}

func (*CountDownLatch) JavaDynamicTypeID() TypeID { return "CountDownLatch" }

func init() { RegisterJavaType("CountDownLatch", ObjectTypeID) }

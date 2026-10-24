package stdjava

import "sync"

const ReentrantLockTypeID TypeID = "java.util.concurrent.locks.ReentrantLock"

// JavaReentrantLock owns a distinct, stable private monitor. Synchronizing on
// this Java allocation uses its public identity and does not acquire this lock.
// Each guard belongs to the explicit Java Execution which acquired it.
type JavaReentrantLock struct {
	anchor struct{ marker byte }
	mu     sync.Mutex
	owner  *Execution
	guards []*MonitorGuard
}

func init() {
	RegisterJavaType(ReentrantLockTypeID, ObjectTypeID, "java.util.concurrent.locks.Lock", SerializableTypeID)
}

func NewReentrantLock() *JavaReentrantLock { return &JavaReentrantLock{} }

func (*JavaReentrantLock) JavaDynamicTypeID() TypeID { return ReentrantLockTypeID }

func ReentrantLockLockExecution(execution *Execution, lock *JavaReentrantLock) {
	requireExecution(execution)
	ReferenceRequireNonNull(lock)
	guard := MonitorEnterExecution(execution, &lock.anchor)
	lock.mu.Lock()
	defer lock.mu.Unlock()
	lock.owner = execution
	lock.guards = append(lock.guards, guard)
}

func ReentrantLockUnlockExecution(execution *Execution, lock *JavaReentrantLock) {
	requireExecution(execution)
	ReferenceRequireNonNull(lock)
	lock.mu.Lock()
	defer lock.mu.Unlock()
	if lock.owner != execution || len(lock.guards) == 0 {
		// Sync.tryRelease throws the no-argument JDK exception. Preserve its
		// null message and canonical throwable identity, rather than a host error.
		panic(IllegalMonitorStateException{newExceptionConstructorBaseExecution(execution, "IllegalMonitorStateException")})
	}
	last := len(lock.guards) - 1
	guard := lock.guards[last]
	lock.guards[last] = nil
	lock.guards = lock.guards[:last]
	if last == 0 {
		lock.owner = nil
	}
	MonitorExitExecution(guard)
}

func ReentrantLockGetHoldCountExecution(execution *Execution, lock *JavaReentrantLock) int32 {
	requireExecution(execution)
	ReferenceRequireNonNull(lock)
	lock.mu.Lock()
	defer lock.mu.Unlock()
	if lock.owner != execution {
		return 0
	}
	return int32(len(lock.guards))
}

func ReentrantLockIsHeldByCurrentThreadExecution(execution *Execution, lock *JavaReentrantLock) bool {
	requireExecution(execution)
	ReferenceRequireNonNull(lock)
	lock.mu.Lock()
	defer lock.mu.Unlock()
	return lock.owner == execution
}

func ReentrantLockIsLocked(lock *JavaReentrantLock) bool {
	ReferenceRequireNonNull(lock)
	lock.mu.Lock()
	defer lock.mu.Unlock()
	return lock.owner != nil
}

package stdjava

import (
	"runtime"
	"sync"
	"testing"
	"time"
	"weak"
)

type monitorLifetime121Object struct {
	payload []byte
}

// The allocation has pointers and exceeds Go's tiny-allocation threshold.
// This control tests Go retention; it makes no Java GC timing assertion.
func monitorLifetime121Released() weak.Pointer[monitorLifetime121Object] {
	value := &monitorLifetime121Object{payload: make([]byte, 4096)}
	guard := MonitorEnterExecution(NewExecution(), value)
	MonitorExitExecution(guard)
	identity := weak.Make(value)
	runtime.KeepAlive(value)
	return identity
}

func monitorLifetime121Eventually(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		runtime.Gosched()
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("lifetime condition did not become true within the bounded Go GC control")
}

func TestMonitorLifetime121ReleasedObjectIsCollectible(t *testing.T) {
	identity := monitorLifetime121Released()
	monitorLifetime121Eventually(t, func() bool { return identity.Value() == nil })
}

func monitorLifetime121RetainedGuard() (weak.Pointer[monitorLifetime121Object], *MonitorGuard) {
	value := &monitorLifetime121Object{payload: make([]byte, 4096)}
	guard := MonitorEnterExecution(NewExecution(), value)
	MonitorExitExecution(guard)
	identity := weak.Make(value)
	runtime.KeepAlive(value)
	return identity, guard
}

func TestMonitorLifetime121ReleasedGuardDoesNotRetainObject(t *testing.T) {
	identity, guard := monitorLifetime121RetainedGuard()
	monitorLifetime121Eventually(t, func() bool { return identity.Value() == nil })
	if !guard.released || guard.monitor != nil || guard.execution != nil {
		t.Fatal("released guard still has active lifetime roots")
	}
	runtime.KeepAlive(guard)
}

type monitorLifetime121Source struct {
	*ObjectInfo
	payload []byte
}

func monitorLifetime121SourceReleased() weak.Pointer[monitorLifetime121Source] {
	value := &monitorLifetime121Source{payload: make([]byte, 4096)}
	value.ObjectInfo = NewObjectInfo("monitor.lifetime.Source", func(TypeID) any { return value })
	guard := MonitorEnterExecution(NewExecution(), value)
	MonitorExitExecution(guard)
	identity := weak.Make(value)
	runtime.KeepAlive(value)
	return identity
}

func TestMonitorLifetime121BoundObjectInfoViewIsCollectible(t *testing.T) {
	identity := monitorLifetime121SourceReleased()
	monitorLifetime121Eventually(t, func() bool { return identity.Value() == nil })
}

func TestMonitorLifetime121ActiveAndLegacyHandlesPinObject(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		value := &monitorLifetime121Object{payload: make([]byte, 4096)}
		identity := weak.Make(value)
		var guard *MonitorGuard
		var mutex *sync.Mutex
		if legacy {
			mutex = MonitorEnter(value)
		} else {
			guard = MonitorEnterExecution(NewExecution(), value)
		}
		value = nil
		for range 4 {
			runtime.GC()
			if identity.Value() == nil {
				t.Fatal("active monitor handle failed to retain its object")
			}
		}
		if legacy {
			MonitorExit(mutex)
			mutex = nil
		} else {
			MonitorExitExecution(guard)
		}
		monitorLifetime121Eventually(t, func() bool { return identity.Value() == nil })
		runtime.KeepAlive(guard)
		runtime.KeepAlive(mutex)
	}
}

func TestMonitorLifetime121StaleCleanupCannotRemoveReplacement(t *testing.T) {
	value := &monitorLifetime121Object{payload: make([]byte, 4096)}
	original := monitorRecord(value)
	identity := monitorIdentityFor(value)
	originalEpoch := weak.Make(original)
	replacement := &monitor{anchor: value}
	replacement.legacyCond = sync.NewCond(&replacement.legacyMu)
	replacementEpoch := weak.Make(replacement)
	monitorsMu.Lock()
	monitors[identity] = replacementEpoch
	monitorsMu.Unlock()
	cleanupMonitorEntry(monitorCleanupEntry{identity, originalEpoch})
	if got := monitorRecord(value); got != replacement {
		t.Fatal("stale cleanup erased the new allocation epoch")
	}
	cleanupMonitorEntry(monitorCleanupEntry{identity, replacementEpoch})
	monitorsMu.Lock()
	_, present := monitors[identity]
	monitorsMu.Unlock()
	if present {
		t.Fatal("matching cleanup failed to remove its entry")
	}
	runtime.KeepAlive(original)
	runtime.KeepAlive(replacement)
	runtime.KeepAlive(value)
}

func TestMonitorLifetime121CanonicalAndNativeIdentities(t *testing.T) {
	first := NewJavaStringUTF16([]uint16{'x'})
	second := NewJavaStringUTF16([]uint16{'x'})
	if monitorFor(first) == monitorFor(second) {
		t.Fatal("canonical String allocations collapsed to content identity")
	}
	if monitorFor(first) != monitorFor(first) {
		t.Fatal("canonical String alias lost its monitor")
	}
	if monitorFor("legacy") != monitorFor(string([]byte("legacy"))) {
		t.Fatal("legacy native-string value comparison changed")
	}
	throwable := NewRuntimeException("identity")
	copy := throwable
	if monitorFor(throwable) != monitorFor(&copy) {
		t.Fatal("throwable state aliases lost their monitor")
	}
	for _, value := range []any{first, NewReferenceArray(0, ObjectTypeID), NewPrimitiveArray[int32](0, PrimitiveIntTypeID), NewObject(), &monitorLifetime121Object{}} {
		identity := monitorIdentityFor(value)
		if identity.comparable != nil || identity.data == 0 || identity.reference == nil {
			t.Fatalf("managed pointer identity retained a Go reference: %+v", identity)
		}
	}
	legacyHost := struct{ value *JavaString }{first}
	if monitorFor(legacyHost) != monitorFor(legacyHost) {
		t.Fatal("legacy comparable host value lost its existing identity semantics")
	}
}

func monitorLifetime121Batch(count int) ([]weak.Pointer[monitorLifetime121Object], []monitorIdentity) {
	values := make([]*monitorLifetime121Object, count)
	refs := make([]weak.Pointer[monitorLifetime121Object], count)
	keys := make([]monitorIdentity, count)
	for index := range values {
		value := &monitorLifetime121Object{payload: make([]byte, 1024)}
		values[index] = value
		guard := MonitorEnterExecution(NewExecution(), value)
		MonitorExitExecution(guard)
		refs[index] = weak.Make(value)
		keys[index] = monitorIdentityFor(value)
	}
	runtime.KeepAlive(values)
	return refs, keys
}

func TestMonitorLifetime121BoundedBatchReclaimsObjectsAndEntries(t *testing.T) {
	refs, keys := monitorLifetime121Batch(2048)
	monitorLifetime121Eventually(t, func() bool {
		for _, ref := range refs {
			if ref.Value() != nil {
				return false
			}
		}
		monitorsMu.Lock()
		defer monitorsMu.Unlock()
		for _, key := range keys {
			if _, present := monitors[key]; present {
				return false
			}
		}
		return true
	})
}

func TestMonitorLifetime121ConcurrentLookupPreservesExclusionAcrossGC(t *testing.T) {
	value := &monitorLifetime121Object{payload: make([]byte, 4096)}
	var workers sync.WaitGroup
	var count int
	for range 8 {
		workers.Go(func() {
			execution := NewExecution()
			for range 256 {
				guard := MonitorEnterExecution(execution, value)
				count++
				inner := MonitorEnterExecution(execution, value)
				MonitorExitExecution(inner)
				MonitorExitExecution(guard)
			}
		})
	}
	workers.Go(func() {
		for range 12 {
			runtime.GC()
		}
	})
	workers.Wait()
	if count != 8*256 {
		t.Fatalf("lost monitor exclusion: count=%d", count)
	}
	runtime.KeepAlive(value)
}

func TestMonitorLifetime121BlockedEntrantRetainsAllocation(t *testing.T) {
	value := &monitorLifetime121Object{payload: make([]byte, 4096)}
	identity := weak.Make(value)
	owner := MonitorEnterExecution(NewExecution(), value)
	started := make(chan struct{})
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func(value *monitorLifetime121Object) {
		close(started)
		guard := MonitorEnterExecution(NewExecution(), value)
		close(entered)
		<-release
		MonitorExitExecution(guard)
		close(done)
	}(value)
	value = nil
	<-started
	runtime.GC()
	select {
	case <-entered:
		t.Fatal("entrant acquired a monitor already owned by another execution")
	default:
	}
	MonitorExitExecution(owner)
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("blocked entrant did not acquire the released monitor")
	}
	for range 4 {
		runtime.GC()
		if identity.Value() == nil {
			t.Fatal("entrant lost the allocation while holding its monitor")
		}
	}
	close(release)
	<-done
	monitorLifetime121Eventually(t, func() bool { return identity.Value() == nil })
}

func TestMonitorLifetime121WaitNotifyKeepsOneMonitorAcrossGC(t *testing.T) {
	value := &monitorLifetime121Object{payload: make([]byte, 4096)}
	identity := weak.Make(value)
	ready := make(chan struct{})
	done := make(chan bool, 1)
	go func(value *monitorLifetime121Object) {
		execution := NewExecution()
		outer := MonitorEnterExecution(execution, value)
		inner := MonitorEnterExecution(execution, value)
		close(ready)
		MonitorWaitExecution(execution, value)
		MonitorExitExecution(inner)
		retainedOwnership := ThreadHoldsLockExecution(execution, value)
		MonitorExitExecution(outer)
		done <- retainedOwnership
	}(value)
	value = nil
	<-ready
	for range 4 {
		runtime.GC()
		if identity.Value() == nil {
			t.Fatal("waiter lost its allocation while the physical monitor was released")
		}
	}
	func() {
		value := identity.Value()
		execution := NewExecution()
		guard := MonitorEnterExecution(execution, value)
		MonitorNotifyAllExecution(execution, value)
		MonitorExitExecution(guard)
	}()
	select {
	case owned := <-done:
		if !owned {
			t.Fatal("wait did not restore the complete reentrant depth")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("notify did not resume the waiter on its original monitor")
	}
	monitorLifetime121Eventually(t, func() bool { return identity.Value() == nil })
}

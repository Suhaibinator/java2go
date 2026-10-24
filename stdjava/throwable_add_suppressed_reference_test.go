package stdjava

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func suppressionFacadePanic(execution *Execution, primary, suppressed any) (failure any) {
	defer func() { failure = recover() }()
	JavaThrowableAddSuppressedExecution(execution, primary, suppressed)
	return nil
}
func suppressionMonitorWaiter(execution *Execution, primary, suppressed any, evaluated chan<- bool, result chan<- any) {
	evaluated <- true
	result <- suppressionFacadePanic(execution, primary, suppressed)
}
func waitSuppressionBlocked(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		size := 16 * 1024
		var stack []byte
		for {
			stack = make([]byte, size)
			n := runtime.Stack(stack, true)
			stack = stack[:n]
			if n < size {
				break
			}
			size *= 2
			if size > 8*1024*1024 {
				t.Fatal("goroutine evidence too large")
			}
		}
		for _, block := range strings.Split(string(stack), "\n\n") {
			if strings.Contains(block, "suppressionMonitorWaiter") && strings.Contains(block, "JavaThrowableAddSuppressedExecution") && strings.Contains(block, "MonitorEnterExecution") && (strings.Contains(block, "[sync.Mutex.Lock]") || strings.Contains(block, "internal/sync.(*Mutex).lockSlow")) {
				return
			}
		}
		runtime.Gosched()
	}
	t.Fatal("same worker never observed blocked in facade Java monitor acquisition")
}
func TestJavaThrowableAddSuppressedMonitorAndCleanup(t *testing.T) {
	for _, nullArg := range []bool{false, true} {
		primary := NewJavaGeneralSecurityExceptionMessage(nil)
		other := NewJavaExceptionMessage(nil)
		var argument any = other
		if nullArg {
			var typedNull *Exception
			argument = typedNull
		}
		execution := NewExecution()
		guard := MonitorEnterExecution(execution, primary)
		released := false
		completed := false
		evaluated := make(chan bool, 1)
		result := make(chan any, 1)
		go suppressionMonitorWaiter(NewExecution(), primary, argument, evaluated, result)
		defer func() {
			if !released {
				MonitorExitExecution(guard)
			}
			if !completed {
				select {
				case <-result:
				case <-time.After(2 * time.Second):
				}
			}
		}()
		<-evaluated
		if monitorRecord(primary) != guard.monitor {
			t.Fatal("facade primary has different Java monitor identity")
		}
		waitSuppressionBlocked(t)
		select {
		case <-result:
			t.Fatal("body ran before held monitor release")
		default:
		}
		MonitorExitExecution(guard)
		released = true
		var failure any
		select {
		case failure = <-result:
			completed = true
		case <-time.After(2 * time.Second):
			t.Fatal("worker failed to complete after release")
		}
		if nullArg {
			if _, ok := failure.(NullPointerException); !ok {
				t.Fatalf("typed null did not reject canonically: %T", failure)
			}
			if JavaThrowableMessageDefault(failure) != JavaStringLiteralUTF16([]uint16{67, 97, 110, 110, 111, 116, 32, 115, 117, 112, 112, 114, 101, 115, 115, 32, 97, 32, 110, 117, 108, 108, 32, 101, 120, 99, 101, 112, 116, 105, 111, 110, 46}) {
				t.Fatal("null rejection message not exact interned JDK literal")
			}
			if GetCause(failure) != nil {
				t.Fatal("null rejection invented cause")
			}
			if len(GetSuppressed(primary)) != 0 {
				t.Fatal("null rejection appended argument")
			}
		} else {
			if failure != nil {
				t.Fatalf("nonnull append failed: %T", failure)
			}
			got := GetSuppressed(primary)
			if len(got) != 1 || !sameThrowableIdentity(got[0], other) {
				t.Fatal("suppression identity lost")
			}
		}
		peer := MonitorEnterExecution(NewExecution(), primary)
		MonitorExitExecution(peer)
	}
}
func TestJavaThrowableAddSuppressedReentrantBodyIdentity(t *testing.T) {
	primary := NewJavaGeneralSecurityExceptionMessage(nil)
	other := NewJavaExceptionMessage(nil)
	execution := NewExecution()
	outer := MonitorEnterExecution(execution, primary)
	defer MonitorExitExecution(outer)
	JavaThrowableAddSuppressedExecution(execution, primary, other)
	failure := suppressionFacadePanic(execution, primary, primary)
	if _, ok := failure.(IllegalArgumentException); !ok {
		t.Fatalf("self rejection %T", failure)
	}
	if JavaThrowableMessageDefault(failure) != JavaStringLiteralUTF16([]uint16{83, 101, 108, 102, 45, 115, 117, 112, 112, 114, 101, 115, 115, 105, 111, 110, 32, 110, 111, 116, 32, 112, 101, 114, 109, 105, 116, 116, 101, 100}) || !sameThrowableIdentity(GetCause(failure), primary) {
		t.Fatal("self rejection exact interned message/cause lost")
	}
	var absent *Exception
	failure = suppressionFacadePanic(execution, primary, absent)
	if _, ok := failure.(NullPointerException); !ok {
		t.Fatalf("typed null rejection %T", failure)
	}
	outer.monitor.mu.Lock()
	owner, depth := outer.monitor.owner, outer.monitor.depth
	outer.monitor.mu.Unlock()
	if owner != execution || depth != 1 {
		t.Fatal("facade changed caller-held reentrant depth or owner")
	}
	snapshot := GetSuppressed(primary)
	snapshot[0] = nil
	if len(GetSuppressed(primary)) != 1 || !sameThrowableIdentity(GetSuppressed(primary)[0], other) {
		t.Fatal("suppressed snapshot aliased carrier storage")
	}
}
func TestJavaThrowableAddSuppressedNullableReceiverBeforeMonitor(t *testing.T) {
	var primary *GeneralSecurityException
	var argument *Exception
	failure := suppressionFacadePanic(NewExecution(), primary, argument)
	if _, ok := failure.(NullPointerException); !ok {
		t.Fatalf("null receiver accepted: %T", failure)
	}
}

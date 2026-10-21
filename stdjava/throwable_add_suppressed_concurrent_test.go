package stdjava

import (
	"runtime"
	"sync"
	"testing"
)

// This exercises concurrent carrier snapshots and synchronized append. It does
// not claim the preexisting getSuppressed body acquires Java's object monitor.
func TestJavaThrowableAddSuppressedConcurrentAppendSnapshots(t *testing.T) {
	const count = 24
	primary := NewJavaGeneralSecurityExceptionMessage(nil)
	expected := make(map[*throwableState]bool, count)
	causes := make([]Exception, count)
	for i := range causes {
		causes[i] = NewJavaExceptionMessage(nil)
		expected[causes[i].ThrowableIdentity()] = true
	}
	start := make(chan struct{})
	finished := make(chan struct{})
	snapshotDone := make(chan struct{})
	var writers sync.WaitGroup
	writers.Add(count)
	for _, cause := range causes {
		go func(cause Exception) {
			defer writers.Done()
			<-start
			JavaThrowableAddSuppressedExecution(NewExecution(), primary, cause)
		}(cause)
	}
	go func() {
		defer close(snapshotDone)
		<-start
		for {
			snapshot := GetSuppressed(primary)
			if len(snapshot) > count {
				panic("suppressed snapshot grew beyond append count")
			}
			for _, value := range snapshot {
				if value == nil {
					panic("suppressed snapshot contains null")
				}
			}
			select {
			case <-finished:
				return
			default:
				runtime.Gosched()
			}
		}
	}()
	close(start)
	writers.Wait()
	close(finished)
	<-snapshotDone
	snapshot := GetSuppressed(primary)
	if len(snapshot) != count {
		t.Fatalf("lost concurrent append: %d", len(snapshot))
	}
	seen := make(map[*throwableState]bool, count)
	for _, value := range snapshot {
		carrier, ok := value.(interface{ ThrowableIdentity() *throwableState })
		if !ok || !expected[carrier.ThrowableIdentity()] || seen[carrier.ThrowableIdentity()] {
			t.Fatal("suppression identity duplicate or lost")
		}
		seen[carrier.ThrowableIdentity()] = true
	}
	snapshot[0] = nil
	if GetSuppressed(primary)[0] == nil {
		t.Fatal("snapshot write mutated shared suppression state")
	}
}

package stdjava

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// This compares direct runtime monitor semantics to a frozen, explicitly
// executed JDK21 scheduling oracle. GC timing is absent from both sequences.
func TestMonitorJDK21Lifecycle121SchedulingOracle(t *testing.T) {
	want, err := os.ReadFile("testdata/monitor_lifecycle121/jdk21.stdout")
	if err != nil {
		t.Fatal(err)
	}
	var results []string
	record := func(label string, value bool) {
		results = append(results, fmt.Sprintf("%s:%t", label, value))
	}
	base := &campaignMonitorBase{NewObjectInfo("campaign.MonitorDerived", nil)}
	leaf := &campaignMonitorDerived{base}
	execution := NewExecution()
	record("initial", ThreadHoldsLockExecution(execution, leaf))
	outer := MonitorEnterExecution(execution, base)
	record("base", ThreadHoldsLockExecution(execution, base))
	record("leaf", ThreadHoldsLockExecution(execution, leaf))
	inner := MonitorEnterExecution(execution, leaf)
	record("reentrant", ThreadHoldsLockExecution(execution, base))
	MonitorExitExecution(inner)
	otherDone := make(chan struct{})
	go func() {
		record("other", ThreadHoldsLockExecution(NewExecution(), leaf))
		close(otherDone)
	}()
	<-otherDone
	record("outer", ThreadHoldsLockExecution(execution, leaf))
	MonitorExitExecution(outer)
	record("released", ThreadHoldsLockExecution(execution, base))
	func() {
		defer func() {
			if !CaughtAs(recover(), "NullPointerException") {
				t.Fatal("synchronized null did not throw NullPointerException")
			}
			results = append(results, "null:NullPointerException")
		}()
		MonitorEnterExecution(execution, nil)
	}()
	func() {
		defer func() {
			if !CaughtAs(recover(), "IllegalStateException") {
				t.Fatal("wrong abrupt-completion exception")
			}
			record("abrupt", ThreadHoldsLockExecution(execution, base))
		}()
		guard := MonitorEnterExecution(execution, leaf)
		defer MonitorExitExecution(guard)
		panic(NewIllegalStateException("abrupt"))
	}()
	ready := make(chan struct{})
	waiterDone := make(chan struct{})
	notified := false
	go func() {
		waiterExecution := NewExecution()
		outer := MonitorEnterExecution(waiterExecution, base)
		inner := MonitorEnterExecution(waiterExecution, leaf)
		close(ready)
		for !notified {
			MonitorWaitExecution(waiterExecution, leaf)
		}
		record("wait-restored", ThreadHoldsLockExecution(waiterExecution, base))
		MonitorExitExecution(inner)
		record("wait-outer", ThreadHoldsLockExecution(waiterExecution, leaf))
		MonitorExitExecution(outer)
		close(waiterDone)
	}()
	<-ready
	guard := MonitorEnterExecution(execution, leaf)
	record("notifier", ThreadHoldsLockExecution(execution, base))
	notified = true
	MonitorNotifyAllExecution(execution, base)
	MonitorExitExecution(guard)
	<-waiterDone
	record("after-wait", ThreadHoldsLockExecution(execution, base))
	got := strings.Join(results, "\n") + "\n"
	if got != string(want) {
		t.Fatalf("runtime differs from the deterministic JDK21 oracle:\ngot:\n%swant:\n%s", got, want)
	}
}

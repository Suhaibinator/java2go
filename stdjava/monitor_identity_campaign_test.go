package stdjava

import "testing"

type campaignMonitorBase struct{ *ObjectInfo }
type campaignMonitorDerived struct{ *campaignMonitorBase }

func TestCampaignMonitorCanonicalViews(t *testing.T) {
	base := &campaignMonitorBase{NewObjectInfo("campaign.MonitorDerived", nil)}
	derived := &campaignMonitorDerived{base}
	other := &campaignMonitorDerived{&campaignMonitorBase{NewObjectInfo("campaign.MonitorDerived", nil)}}
	if !JavaReferenceEqual(base, derived) {
		t.Fatal("fixture must share Java reference identity")
	}
	if monitorRecord(base) != monitorRecord(derived) {
		t.Fatal("base and derived Java views have different monitors")
	}
	if monitorRecord(other) == monitorRecord(base) {
		t.Fatal("distinct allocations share a monitor")
	}
	execution := NewExecution()
	outer := MonitorEnterExecution(execution, base)
	defer MonitorExitExecution(outer)
	inner := MonitorEnterExecution(execution, derived)
	if !ThreadHoldsLockExecution(execution, derived) || ThreadHoldsLockExecution(NewExecution(), base) {
		t.Fatal("alias ownership is incorrect")
	}
	MonitorNotifyExecution(execution, derived)
	MonitorNotifyAllExecution(execution, derived)
	MonitorExitExecution(inner)
	if !ThreadHoldsLockExecution(execution, base) {
		t.Fatal("alias reentry lost outer ownership")
	}
}

func TestCampaignMonitorJavaNull(t *testing.T) {
	var typedNil *campaignMonitorDerived
	for name, value := range map[string]any{"nil": nil, "typed": typedNil, "String": NullString()} {
		for operation, call := range map[string]func(any){
			"enter":       func(v any) { guard := MonitorEnterExecution(NewExecution(), v); MonitorExitExecution(guard) },
			"holdsLock":   func(v any) { ThreadHoldsLockExecution(NewExecution(), v) },
			"wait":        func(v any) { MonitorWaitExecution(NewExecution(), v) },
			"notify":      func(v any) { MonitorNotifyExecution(NewExecution(), v) },
			"notifyAll":   func(v any) { MonitorNotifyAllExecution(NewExecution(), v) },
			"legacyEnter": func(v any) { guard := MonitorEnter(v); MonitorExit(guard) },
		} {
			t.Run(name+"/"+operation, func(t *testing.T) {
				var failure any
				func() { defer func() { failure = recover() }(); call(value) }()
				if !CaughtAs(failure, "NullPointerException") {
					t.Fatalf("got %T (%v), want NullPointerException", failure, failure)
				}
			})
		}
	}
}

func TestCampaignMonitorObservationDoesNotRegister(t *testing.T) {
	const runs = 32
	objects := make([]any, runs+1)
	for i := range objects {
		objects[i] = NewObject()
	}
	execution := NewExecution()
	monitorsMu.Lock()
	before := len(monitors)
	monitorsMu.Unlock()
	index := 0
	allocations := testing.AllocsPerRun(runs, func() {
		if ThreadHoldsLockExecution(execution, objects[index]) {
			t.Fatal("fresh object reported owned")
		}
		index++
	})
	monitorsMu.Lock()
	after := len(monitors)
	monitorsMu.Unlock()
	if after != before {
		t.Errorf("observation registered %d monitors", after-before)
	}
	if allocations != 0 {
		t.Errorf("fresh observation allocated %g times, want zero", allocations)
	}
}

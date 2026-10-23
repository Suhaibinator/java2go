package stdjava

import (
	"runtime"
	"testing"
	"weak"
)

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
	for _, tombstones := range []bool{false, true} {
		name := "absent"
		if tombstones {
			name = "empty-weak-entry"
		}
		t.Run(name, func(t *testing.T) {
			objects := make([]any, runs+1)
			keys := make([]monitorIdentity, len(objects))
			unique := make(map[monitorIdentity]bool, len(objects))
			for i := range objects {
				objects[i] = NewObject()
				keys[i] = monitorIdentityFor(objects[i])
				if keys[i].data == 0 || keys[i].reference == nil || keys[i].comparable != nil || unique[keys[i]] {
					t.Fatal("fresh retained object identity is not unique/nonretaining")
				}
				unique[keys[i]] = true
			}
			defer runtime.KeepAlive(objects)
			var empty weak.Pointer[monitor]
			monitorsMu.Lock()
			liveBaseline := -1
			for i, key := range keys {
				if old, present := monitors[key]; present && old.Value() != nil {
					liveBaseline = i
					break
				}
			}
			if liveBaseline < 0 {
				// A recycled scalar address may still have an expired old epoch pending
				// cleanup. Clear only that expired premise; never erase a live monitor.
				for _, key := range keys {
					delete(monitors, key)
					if tombstones {
						monitors[key] = empty
					}
				}
			}
			monitorsMu.Unlock()
			if liveBaseline >= 0 {
				t.Fatalf("fresh retained object %d already has a live monitor", liveBaseline)
			}
			t.Cleanup(func() {
				monitorsMu.Lock()
				for _, key := range keys {
					delete(monitors, key)
				}
				monitorsMu.Unlock()
				runtime.KeepAlive(objects)
			})
			execution := NewExecution()
			var changed, owned, identityChanged [runs + 1]bool
			index := 0
			allocations := testing.AllocsPerRun(runs, func() {
				object := objects[index]
				owned[index] = ThreadHoldsLockExecution(execution, object)
				identityChanged[index] = monitorIdentityFor(object) != keys[index]
				monitorsMu.Lock()
				changed[index] = campaignMonitorObservationChangedLocked(keys[index], tombstones)
				monitorsMu.Unlock()
				index++
			})
			runtime.KeepAlive(objects)
			monitorsMu.Lock()
			for i, key := range keys {
				changed[i] = changed[i] || campaignMonitorObservationChangedLocked(key, tombstones)
			}
			monitorsMu.Unlock()
			if index != len(objects) {
				t.Fatalf("observation count=%d want=%d", index, len(objects))
			}
			for i := range objects {
				if owned[i] {
					t.Errorf("fresh object %d reported owned", i)
				}
				if identityChanged[i] {
					t.Errorf("retained object %d changed identity", i)
				}
				if changed[i] {
					t.Errorf("observation changed watched monitor identity %d", i)
				}
			}
			if allocations != 0 {
				t.Errorf("fresh observation allocated %g times, want zero", allocations)
			}
		})
	}
}

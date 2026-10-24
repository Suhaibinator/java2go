package stdjava

import (
	"runtime"
	"testing"
	"weak"
)

// Caller holds monitorsMu. Both the measured observation test and controls
// use this exact witness, including expired epochs and missing tombstones.
func campaignMonitorObservationChangedLocked(key monitorIdentity, tombstone bool) bool {
	epoch, present := monitors[key]
	if tombstone {
		return !present || epoch != (weak.Pointer[monitor]{})
	}
	return present
}

// Publish an idle weak epoch without installing an asynchronous cleanup.
// Only scalar identity/weak epoch escape: after return no strong monitor is
// retained. A manual matching cleanup models an already eligible callback;
// this fixture makes no claim about actual GC or cleanup scheduling.
func campaignMonitorWitnessIdleEntry(object any) monitorCleanupEntry {
	record := &monitor{anchor: object}
	entry := monitorCleanupEntry{monitorIdentityFor(object), weak.Make(record)}
	monitorsMu.Lock()
	monitors[entry.identity] = entry.epoch
	monitorsMu.Unlock()
	runtime.KeepAlive(record)
	return entry
}

func TestCampaignMonitorObservationWitnessCleanupControls(t *testing.T) {
	objects := []any{NewObject(), NewObject()}
	watched, unrelatedKey := monitorIdentityFor(objects[0]), monitorIdentityFor(objects[1])
	if watched == unrelatedKey {
		t.Fatal("control object identities collapsed")
	}
	defer runtime.KeepAlive(objects)
	monitorsMu.Lock()
	live := false
	for _, key := range []monitorIdentity{watched, unrelatedKey} {
		if old, present := monitors[key]; present && old.Value() != nil {
			live = true
		}
	}
	if !live {
		delete(monitors, watched)
		delete(monitors, unrelatedKey)
	}
	monitorsMu.Unlock()
	if live {
		t.Fatal("fresh control key already has a live monitor")
	}
	t.Cleanup(func() {
		monitorsMu.Lock()
		delete(monitors, watched)
		delete(monitors, unrelatedKey)
		monitorsMu.Unlock()
		runtime.KeepAlive(objects)
	})

	// Exact old predicate: unrelated deletion alone looks like registration.
	// Count capture and the controlled delete are one locked transaction, so
	// other pending cleanups cannot perturb this legacy counterexample.
	unrelated := campaignMonitorWitnessIdleEntry(objects[1])
	if ThreadHoldsLockExecution(NewExecution(), objects[0]) {
		t.Fatal("fresh control owned")
	}
	monitorsMu.Lock()
	before := len(monitors)
	fixtureOK := monitors[unrelated.identity] == unrelated.epoch
	delete(monitors, unrelated.identity) // controlled model transition, not a helper call
	after := len(monitors)
	changed := campaignMonitorObservationChangedLocked(watched, false)
	monitorsMu.Unlock()
	if !fixtureOK || before-after != 1 || changed || before == after {
		t.Fatal("unrelated-cleanup counterexample was not deterministic")
	}

	// A watched insertion and unrelated deletion cancel in global length.
	// The exact per-identity witness must reject that registration regardless.
	unrelated = campaignMonitorWitnessIdleEntry(objects[1])
	injected := &monitor{anchor: objects[0]}
	injectedEpoch := weak.Make(injected)
	monitorsMu.Lock()
	before = len(monitors)
	_, alreadyPresent := monitors[watched]
	fixtureOK = monitors[unrelated.identity] == unrelated.epoch
	monitors[watched] = injectedEpoch
	delete(monitors, unrelated.identity)
	after = len(monitors)
	changed = campaignMonitorObservationChangedLocked(watched, false)
	monitorsMu.Unlock()
	if alreadyPresent || !fixtureOK || before != after || !changed {
		t.Fatal("balanced cleanup masked watched registration")
	}
	runtime.KeepAlive(injected)
	monitorsMu.Lock()
	delete(monitors, watched)
	monitorsMu.Unlock()

	// Expired/zero weak metadata is still an insertion for an absent identity.
	var empty weak.Pointer[monitor]
	monitorsMu.Lock()
	monitors[watched] = empty
	insertedMetadata := campaignMonitorObservationChangedLocked(watched, false)
	unchangedTombstone := !campaignMonitorObservationChangedLocked(watched, true)
	monitorsMu.Unlock()
	if !insertedMetadata || !unchangedTombstone {
		t.Fatal("witness inspected only live weak.Value")
	}

	// An accidental monitor can disappear after its registration. It cannot
	// restore the expected zero epoch: missing sentinel remains a discrepancy.
	registered := campaignMonitorWitnessIdleEntry(objects[0])
	monitorsMu.Lock()
	changed = campaignMonitorObservationChangedLocked(watched, true)
	monitorsMu.Unlock()
	cleanupMonitorEntry(registered) // real helper takes its own lock; not called under monitorsMu
	monitorsMu.Lock()
	removedChanged := campaignMonitorObservationChangedLocked(watched, true)
	_, present := monitors[watched]
	monitorsMu.Unlock()
	if !changed || !removedChanged || present {
		t.Fatal("own cleanup masked a changed watched epoch")
	}

	// Separately exercise the actual epoch-checked cleanup helper. No total
	// counts span its lock acquisition; only this retained identity is checked.
	original := campaignMonitorWitnessIdleEntry(objects[1])
	replacement := campaignMonitorWitnessIdleEntry(objects[1])
	cleanupMonitorEntry(original)
	monitorsMu.Lock()
	epoch, present := monitors[unrelatedKey]
	monitorsMu.Unlock()
	if !present || epoch != replacement.epoch {
		t.Fatal("stale cleanup removed replacement")
	}
	cleanupMonitorEntry(replacement)
	monitorsMu.Lock()
	_, present = monitors[unrelatedKey]
	monitorsMu.Unlock()
	if present {
		t.Fatal("matching cleanup did not remove owned entry")
	}
}

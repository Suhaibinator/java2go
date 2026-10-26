package stdjava

import "testing"

type objectFinalizeOverrideProbe struct {
	*ObjectInfo
	calls int
}

func (p *objectFinalizeOverrideProbe) FinalizeJava2goExecution(*Execution) {
	p.calls++
	panic("canonical Object body dispatched a source override")
}

func TestObjectDefaultFinalizeBypassesOverridesAndPreservesMonitor(t *testing.T) {
	execution := NewExecution()
	value := &objectFinalizeOverrideProbe{ObjectInfo: NewObjectInfo("direct.Finalize", nil)}
	guard := MonitorEnterExecution(execution, value)
	func() {
		defer MonitorExitExecution(guard)
		ObjectDefaultFinalizeExecution(execution, value)
		if value.calls != 0 || !ThreadHoldsLockExecution(execution, value) {
			t.Fatal("empty Object body invoked an override or changed the caller monitor")
		}
	}()
	if ThreadHoldsLockExecution(execution, value) {
		t.Fatal("caller monitor was not released")
	}
}

func TestObjectDefaultFinalizeRejectsNullAndMissingExecution(t *testing.T) {
	for _, value := range []any{nil, (*objectFinalizeOverrideProbe)(nil)} {
		expectBoxedException(t, "NullPointerException", func() {
			ObjectDefaultFinalizeExecution(NewExecution(), value)
		})
	}
	expectBoxedException(t, "IllegalArgumentException", func() {
		ObjectDefaultFinalizeExecution(nil, NewObjectInfo("direct.Finalize", nil))
	})
}

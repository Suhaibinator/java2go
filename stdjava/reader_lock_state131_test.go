package stdjava

import "testing"

func TestReaderBaseLockConstruction131(t *testing.T) {
	base := NewReaderBase()
	if base.Lock != base {
		t.Errorf("default lock must be allocation itself: %T", base.Lock)
	}
	sentinel := new(int)
	explicit := NewReaderBase(sentinel)
	if explicit.Lock != sentinel {
		t.Fatal("explicit lock identity changed")
	}
	var typedNil *int
	for _, lock := range []any{nil, typedNil} {
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Error("explicit null must throw NullPointerException")
				}
			}()
			NewReaderBase(lock)
			t.Error("explicit null accepted")
		}()
	}
}
func TestReaderBasePureStateView131(t *testing.T) {
	base := NewReaderBase()
	state, ok := any(base).(interface{ JavaReaderState() *ReaderBase })
	if !ok {
		t.Fatal("native Reader has no base state view")
	}
	first := state.JavaReaderState()
	second := state.JavaReaderState()
	if first != base || second != base {
		t.Fatal("state view replaced allocation")
	}
	base.Lock = nil
	if state.JavaReaderState().Lock != nil {
		t.Fatal("state lookup reset legal null lock")
	}
}

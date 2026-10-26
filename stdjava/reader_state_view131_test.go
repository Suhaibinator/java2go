package stdjava

import "testing"

type readerStateViewProbe131 struct {
	*ReaderBase
	calls int
}

func (p *readerStateViewProbe131) JavaReaderState() *ReaderBase { p.calls++; return p.ReaderBase }

type readerNoStateProbe131 struct{}

func (*readerNoStateProbe131) JavaReaderMarker()    {}
func (*readerNoStateProbe131) JavaCloseableMarker() {}
func TestReaderStatePureLookupAndFailures131(t *testing.T) {
	base := NewReaderBase()
	probe := &readerStateViewProbe131{ReaderBase: base}
	if ReaderState(probe) != base || probe.calls != 1 {
		t.Fatal("state view not evaluated once")
	}
	base.Lock = nil
	if ReaderState(probe) != base || probe.calls != 2 || base.Lock != nil {
		t.Fatal("state view allocated or reset lock")
	}
	for _, value := range []Reader{nil, (*ReaderBase)(nil), (*readerStateViewProbe131)(nil), &readerStateViewProbe131{}} {
		func() {
			defer func() {
				if _, ok := recover().(NullPointerException); !ok {
					t.Error("null receiver/carrier must throw NPE")
				}
			}()
			ReaderState(value)
			t.Error("null accepted")
		}()
	}
	func() {
		defer func() {
			if _, ok := recover().(ClassCastException); !ok {
				t.Error("invalid host adapter must remain visible")
			}
		}()
		ReaderState(&readerNoStateProbe131{})
		t.Error("missing adapter accepted")
	}()
}

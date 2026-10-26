package stdjava

import "testing"

// The historical physical StringReader class lacked Java's Reader protocol.
// This is a boundary RED independent of missing new factory symbols. Legacy
// host APIs keep their own behavior; admission must not dereference nil state.
func TestStringReaderReferenceProtocolAdmission(t *testing.T) {
	reader := NewStringReader("")
	if _, ok := any(reader).(Reader); !ok {
		t.Fatal("physical StringReader lacks Java Reader/Closeable protocol")
	}
	var absent *StringReader
	if _, ok := any(absent).(Reader); !ok {
		t.Fatal("typed nil StringReader loses Reader view")
	}
}

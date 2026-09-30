package stdjava

import "testing"

func TestStringBuilderAppendAndString(t *testing.T) {
	sb := NewStringBuilder()
	sb.Append("ab").Append(1).Append(true)
	if got := sb.String(); got != "ab1true" {
		t.Errorf("String = %q, want ab1true", got)
	}
	if got := sb.Length(); got != 7 {
		t.Errorf("Length = %d, want 7", got)
	}
}

func TestStringBuilderInsertReverseDelete(t *testing.T) {
	sb := NewStringBuilderString("bcd")
	sb.Insert(0, "a")
	if got := sb.String(); got != "abcd" {
		t.Errorf("after insert = %q, want abcd", got)
	}
	sb.DeleteCharAt(1)
	if got := sb.String(); got != "acd" {
		t.Errorf("after delete = %q, want acd", got)
	}
	sb.Reverse()
	if got := sb.String(); got != "dca" {
		t.Errorf("after reverse = %q, want dca", got)
	}
}

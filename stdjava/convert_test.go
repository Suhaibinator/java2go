package stdjava

import "testing"

func TestParseNumbers(t *testing.T) {
	if got := ParseInt("42"); got != 42 {
		t.Errorf("ParseInt = %d, want 42", got)
	}
	if got := ParseLong("9000000000"); got != 9000000000 {
		t.Errorf("ParseLong = %d, want 9000000000", got)
	}
	if got := ParseDouble("3.5"); got != 3.5 {
		t.Errorf("ParseDouble = %v, want 3.5", got)
	}
}

func TestParseIntPanicsOnBadInput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Errorf("ParseInt(bad) did not panic")
		}
	}()
	ParseInt("not-a-number")
}

func TestParseBoolean(t *testing.T) {
	if !ParseBoolean("true") || !ParseBoolean("TRUE") || !ParseBoolean("True") {
		t.Errorf("ParseBoolean(true variants) = false")
	}
	if ParseBoolean("yes") || ParseBoolean("1") || ParseBoolean("") {
		t.Errorf("ParseBoolean(non-true) = true")
	}
}

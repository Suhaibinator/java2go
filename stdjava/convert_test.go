package stdjava

import (
	"math"
	"testing"
)

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

func TestParseIntegerRadixAndJavaErrors(t *testing.T) {
	if ParseByte("-80", 16) != -128 || ParseShort("7fff", 16) != 32767 ||
		ParseInt("+101", 2) != 5 || ParseLong("-8000000000000000", 16) != math.MinInt64 ||
		ParseInt("１２３") != 123 || ParseInt("١٢٣") != 123 || ParseInt("ＦＦ", 16) != 255 {
		t.Fatal("integer parser radix, signs, or Java digits differ")
	}
	for _, test := range []struct {
		name  string
		parse func()
	}{
		{"byte overflow", func() { ParseByte("128") }},
		{"short overflow", func() { ParseShort("32768") }},
		{"int overflow", func() { ParseInt("2147483648") }},
		{"long overflow", func() { ParseLong("9223372036854775808") }},
		{"whitespace", func() { ParseInt(" 1") }},
		{"underscores", func() { ParseInt("1_0") }},
		{"empty", func() { ParseInt("") }},
		{"null", func() { ParseInt(NullString()) }},
		{"radix too small", func() { ParseInt("1", 1) }},
		{"radix too big", func() { ParseInt("1", 37) }},
	} {
		t.Run(test.name, func(t *testing.T) { expectBoxedException(t, "NumberFormatException", test.parse) })
	}
}

func TestParseFloatingJavaForms(t *testing.T) {
	if ParseDouble(" \t1.25d\n") != 1.25 || ParseFloat("0x1.8p1F") != 3 ||
		!math.IsNaN(ParseDouble("-NaN")) || !math.IsInf(ParseDouble("+Infinity"), 1) ||
		!math.IsInf(ParseDouble("1e10000"), 1) || ParseDouble("1e-10000") != 0 ||
		!math.Signbit(ParseDouble("-0.0")) {
		t.Fatal("floating parser Java syntax, overflow, or signed zero differs")
	}
	for _, text := range []string{"Inf", "infinity", "NaNf", "Infinityd", "1_0.0", "", "1.0 ff"} {
		t.Run(text, func(t *testing.T) {
			expectBoxedException(t, "NumberFormatException", func() { ParseDouble(text) })
		})
	}
	expectBoxedException(t, "NullPointerException", func() { ParseFloat(NullString()) })
	if ParseBoolean(NullString()) {
		t.Fatal("Boolean.parseBoolean(null) must be false")
	}
}

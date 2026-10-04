package stdjava

import (
	"os"
	"strings"
	"sync"
	"testing"
)

func currencyJDKCodeOracle(t *testing.T) map[string]bool {
	t.Helper()
	bytes, err := os.ReadFile("testdata/currency_jdk21_177/jdk-uppercase-codes.stdout")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(bytes)), "\n")
	if lines[len(lines)-1] != "counts:232:17344" {
		t.Fatalf("incomplete JDK triple oracle: %q", lines[len(lines)-1])
	}
	codes := map[string]bool{}
	for _, code := range lines[:len(lines)-1] {
		if len(code) != 3 || codes[code] {
			t.Fatalf("invalid oracle code %q", code)
		}
		codes[code] = true
	}
	if len(codes) != 232 {
		t.Fatalf("oracle selected %d codes", len(codes))
	}
	return codes
}

func TestCurrencyCompleteJDKStringValidation(t *testing.T) {
	want := currencyJDKCodeOracle(t)
	for first := uint16('A'); first <= 'Z'; first++ {
		for second := uint16('A'); second <= 'Z'; second++ {
			for third := uint16('A'); third <= 'Z'; third++ {
				units := []uint16{first, second, third}
				code := string([]byte{byte(first), byte(second), byte(third)})
				if currencyCodeValidAt(units, 1743480000001) != want[code] {
					t.Fatalf("JDK complete-code mismatch for %s", code)
				}
			}
		}
	}
}

func TestCurrencyUTF16DoesNotNarrowOrNormalize(t *testing.T) {
	want := currencyJDKCodeOracle(t)
	for position := 0; position < 3; position++ {
		for unit := 0; unit <= 65535; unit++ {
			units := []uint16{'U', 'S', 'D'}
			units[position] = uint16(unit)
			expected := unit < 128 && want[string([]byte{byte(units[0]), byte(units[1]), byte(units[2])})]
			if currencyCodeValidAt(units, 1743480000001) != expected {
				t.Fatalf("position%d UTF16unit%04x incorrectly admitted or rejected", position, unit)
			}
		}
	}
	for _, units := range [][]uint16{nil, {'U'}, {'U', 'S'}, {'U', 'S', 'D', 'D'}, {0xd83d, 0xde00, 'D'}, {'U', 'S', 0xd800}, {'U', 'S', 0xdfff}, {'U', 0x301, 'D'}} {
		if currencyCodeValidAt(units, 1743480000001) {
			t.Fatalf("malformed UTF16 admitted: %x", units)
		}
	}
}

func currencyPanicClass(t *testing.T, class string, operation func()) {
	t.Helper()
	defer func() {
		if failure := recover(); failure == nil || !CaughtAs(failure, class) {
			t.Fatalf("wanted %s, got %#v", class, failure)
		}
	}()
	operation()
}

func TestCurrencyDateBoundaryAndCachedWinner(t *testing.T) {
	const boundary = int64(1743480000000)
	for _, delta := range []int64{-1, 0, 1} {
		if got := currencyCodeValidAt([]uint16{'X', 'C', 'G'}, boundary+delta); got != (delta >= 0) {
			t.Fatalf("XCG boundary%d: %v", delta, got)
		}
		if !currencyCodeValidAt([]uint16{'A', 'N', 'G'}, boundary+delta) {
			t.Fatalf("ANG simple fallback lost at%d", delta)
		}
	}
	for _, special := range currencySpecialCases {
		if special.cutover == 1<<63-1 {
			units := []uint16{uint16(special.oldCode[0]), uint16(special.oldCode[1]), uint16(special.oldCode[2])}
			for _, now := range []int64{special.cutover - 1, special.cutover} {
				if !currencyCodeValidAt(units, now) {
					t.Fatalf("Long.MAX_VALUE sentinel lost %s", special.oldCode)
				}
			}
		}
	}
	var cache currencyInstanceCache
	text := NewJavaStringUTF16([]uint16{'X', 'C', 'G'})
	currencyPanicClass(t, "IllegalArgumentException", func() { cache.get(text, func() int64 { return boundary - 1 }) })
	if len(cache.values) != 0 {
		t.Fatal("failed lookup was cached")
	}
	first := cache.get(text, func() int64 { return boundary })
	second := cache.get(CopyJavaString(text), func() int64 { t.Fatal("cache hit reevaluated clock"); return boundary - 1 })
	if first != second || first.GetCurrencyCode() != text {
		t.Fatal("cached identity or stored input was replaced")
	}
}

func TestCurrencyNullInvalidAndNominalIdentity(t *testing.T) {
	var cache currencyInstanceCache
	clock := func() int64 { return 1743480000001 }
	currencyPanicClass(t, "NullPointerException", func() { cache.get(nil, func() int64 { t.Fatal("null read clock"); return 0 }) })
	currencyPanicClass(t, "IllegalArgumentException", func() { cache.get(NewJavaStringUTF16([]uint16{'Z', 'Z', 'Z'}), clock) })
	currencyPanicClass(t, "NullPointerException", func() { (*JavaCurrency)(nil).GetCurrencyCode() })
	currencyPanicClass(t, "NullPointerException", func() { (*JavaCurrency)(nil).StringJava2goExecution(NewExecution()) })
	text := NewJavaStringUTF16([]uint16{'C', 'H', 'F'})
	first := cache.get(text, clock)
	second := cache.get(CopyJavaString(text), clock)
	if first != second || first.GetCurrencyCode() != text || first.StringJava2goExecution(NewExecution()) != text {
		t.Fatal("singleton or String identity changed")
	}
	if !ObjectInstanceOf(first, ObjectTypeID) || !ObjectInstanceOf(first, SerializableTypeID) || ObjectInstanceOf(first, ComparableTypeID) {
		t.Fatal("Currency nominal ancestry mismatch")
	}
	if ObjectGetClass(first) != ClassLiteral("java.util.Currency") || ObjectGetClass(first) != ObjectGetClass(second) {
		t.Fatal("Currency class identity mismatch")
	}
	if !ObjectEqualsExecution(NewExecution(), first, second) || ObjectEqualsExecution(NewExecution(), first, text) {
		t.Fatal("Currency default Object equality changed")
	}
	if ObjectHashCodeExecution(NewExecution(), first) != ObjectHashCodeExecution(NewExecution(), second) {
		t.Fatal("Currency identity hash inconsistent")
	}
}

func TestCurrencyConcurrentSingletonPublication(t *testing.T) {
	var cache currencyInstanceCache
	const count = 16
	start := make(chan struct{})
	var joined sync.WaitGroup
	texts := make([]*JavaString, count)
	results := make([]*JavaCurrency, count)
	for index := range texts {
		texts[index] = NewJavaStringUTF16([]uint16{'N', 'O', 'K'})
		joined.Add(1)
		go func(i int) {
			defer joined.Done()
			<-start
			results[i] = cache.get(texts[i], func() int64 { return 1743480000001 })
		}(index)
	}
	close(start)
	joined.Wait()
	winner := results[0]
	storedInput := false
	for i, result := range results {
		if result != winner {
			t.Fatalf("publication escaped losing candidate%d", i)
		}
		storedInput = storedInput || winner.GetCurrencyCode() == texts[i]
	}
	if !storedInput || winner.StringJava2goExecution(NewExecution()) != winner.GetCurrencyCode() {
		t.Fatal("winning String publication lost")
	}
}

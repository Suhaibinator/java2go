package stdjava

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestStringSplitLiteralFrozenJDK21(t *testing.T) {
	raw, err := os.ReadFile("testdata/string_split_literal_cost93/contract.stdout")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		fields := strings.Split(line, "\t")
		if fields[0] == "CALL" {
			continue
		}
		if len(fields) != 9 {
			t.Fatalf("invalid JDK row %q", line)
		}
		count++
		t.Run(fields[0], func(t *testing.T) {
			text, regex := splitProbeUnits(fields[1]), splitProbeUnits(fields[2])
			limit, err := strconv.ParseInt(fields[3], 10, 32)
			if err != nil {
				t.Fatal(err)
			}
			var result *ReferenceArray
			var caught any
			call := func() *ReferenceArray {
				if fields[4] == "true" {
					return JavaStringSplitArray(text, regex)
				}
				return JavaStringSplitArray(text, regex, int32(limit))
			}
			func() { defer func() { caught = recover() }(); result = call() }()
			if fields[5] == "EX" {
				failure, ok := caught.(Throwable)
				if !ok || failure.ThrowableTypeName() != fields[6] {
					t.Fatalf("JDK %s; Go %#v", fields[6], caught)
				}
				return
			}
			if caught != nil {
				t.Fatalf("JDK success; Go panic %v", caught)
			}
			n, err := strconv.Atoi(fields[6])
			if err != nil {
				t.Fatal(err)
			}
			if result == nil || result.componentType != StringTypeID || len(result.elements) != n {
				t.Fatal("canonical String[]/length differs")
			}
			encoded := "-"
			if n != 0 {
				parts := make([]string, n)
				for i, value := range result.elements {
					text, ok := value.(*JavaString)
					if !ok || text == nil {
						t.Fatalf("noncanonical String %#v", value)
					}
					parts[i] = splitProbeEncode(text)
				}
				encoded = strings.Join(parts, "/")
			}
			if encoded != fields[7] {
				t.Fatalf("UTF16 %q; JDK %q", encoded, fields[7])
			}
			identity := n == 1 && result.elements[0] == text
			if strconv.FormatBool(identity) != fields[8] {
				t.Fatal("receiver identity differs")
			}
			again := call()
			if result == again {
				t.Fatal("result array reused")
			}
			if n != 0 {
				saved := again.elements[0]
				result.elements[0] = nil
				if again.elements[0] != saved {
					t.Fatal("result storage shared")
				}
			}
		})
	}
	if count != 225 {
		t.Fatalf("case closure %d", count)
	}
}

var stringSplitAllocationSink *ReferenceArray

// The input and regex are prepared outside measurement. A literal split with
// fixed result cardinality must not allocate matching state per UTF16 position.
// These budgets count allocations, not wall time, and preserve consumed output.
func TestStringSplitLiteralAllocationBudget(t *testing.T) {
	for _, row := range []struct {
		name    string
		text    string
		maximum float64
		fields  int
	}{
		{"short-no-match", strings.Repeat("λ", 64), 8, 1},
		{"long-no-match", strings.Repeat("λ", 4096), 8, 1},
		{"short-six-fields", strings.Repeat("a|", 5) + "b", 24, 6},
		{"long-six-fields", strings.Repeat(strings.Repeat("λ", 512)+"|", 5) + strings.Repeat("b", 512), 24, 6},
	} {
		t.Run(row.name, func(t *testing.T) {
			text := JavaStringFromHostUTF8(row.text)
			regex := JavaStringFromHostUTF8(`\|`)
			allocations := testing.AllocsPerRun(100, func() { stringSplitAllocationSink = JavaStringSplitArray(text, regex) })
			result := stringSplitAllocationSink
			if result == nil || len(result.elements) != row.fields || result.componentType != StringTypeID {
				t.Fatal("consumed result differs")
			}
			if row.fields == 1 && result.elements[0] != text {
				t.Fatal("no-match receiver identity lost")
			}
			t.Logf("literal split allocations/call %.0f, frozen maximum %.0f, UTF16 units %d, fields %d", allocations, row.maximum, text.Length(), row.fields)
			if allocations > row.maximum {
				t.Fatalf("literal matcher cost grows with input: %.0f allocations > %.0f", allocations, row.maximum)
			}
		})
	}
}

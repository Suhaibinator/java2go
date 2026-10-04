package stdjava

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"testing"
	"unicode/utf16"
)

func TestCharAtCandidateEveryUTF16Unit(t *testing.T) {
	// Every nonsurrogate BMP unit is directly representable by this native ABI.
	for unit := rune(0); unit <= 0xffff; unit++ {
		if unit >= 0xd800 && unit <= 0xdfff {
			continue
		}
		if got := StringCharAt(string(unit), 0); got != unit {
			t.Fatalf("BMP unit %04x: got %04x", unit, got)
		}
	}
	// All high/low surrogate unit values are observed inside valid pairs.
	// This does not pretend that native UTF8 stores isolated Java surrogates.
	for offset := rune(0); offset < 1024; offset++ {
		for _, value := range []rune{0x10000 + (offset << 10), 0x10000 + offset} {
			high, low := utf16.EncodeRune(value)
			text := string(value)
			if StringCharAt(text, 0) != high || StringCharAt(text, 1) != low {
				t.Fatalf("supplementary %x: wrong UTF16 pair", value)
			}
		}
	}
}

func TestCharAtCandidateNativeDecodingCompatibility(t *testing.T) {
	values := []string{
		"", "\x00", "ASCII", "éλ\ufffd", "a😀b\U0010ffff\x00",
		"\xff", "x\xc0\x80y", "\xed\xa0\x80", "\xf0\x9f\x98", "😀\xffλ",
		strings.Repeat("aλ😀\x00", 100), NullString(),
	}
	for value := 0; value < 256; value++ {
		values = append(values, string([]byte{'x', byte(value), 'y'}))
	}
	for _, value := range values {
		// Existing untouched decoder is the native-ABI compatibility reference.
		// Malformed UTF8 and the raw null sentinel are host-helper cases only.
		units := StringChars(value)
		for index, want := range units {
			if got := StringCharAt(value, int32(index)); got != want {
				t.Fatalf("native bytes=%x unit=%d got=%x want=%x", value, index, got, want)
			}
		}
	}
}

func charAtCandidatePanic(call func()) (kind reflect.Type, message string) {
	defer func() {
		if value := recover(); value != nil {
			kind, message = reflect.TypeOf(value), GetMessage(value)
		}
	}()
	call()
	return
}

func TestCharAtCandidateInvalidIndexFallback(t *testing.T) {
	for _, value := range []string{"", "a", "a\x00λ😀", "\xff😀", NullString()} {
		length := int32(len(StringChars(value)))
		for _, index := range []int32{math.MinInt32, -1, length, length + 1, math.MaxInt32} {
			// The JVM-verified String contract supersedes the old native slice
			// panic comparison. Host malformed-byte/null-sentinel cases still
			// use the existing StringChars unit count; caller null checks remain
			// independently covered by TestCharAtCandidateCallerNullBoundary.
			wantType := reflect.TypeOf(StringIndexOutOfBoundsException{})
			wantMessage := fmt.Sprintf("Index %d out of bounds for length %d", index, length)
			gotType, gotMessage := charAtCandidatePanic(func() { _ = StringCharAt(value, index) })
			if gotType != wantType || gotMessage != wantMessage {
				t.Fatalf("fallback bytes=%x index=%d got=%v/%q want=%v/%q", value, index, gotType, gotMessage, wantType, wantMessage)
			}
		}
	}
}

func TestCharAtCandidateCallerNullBoundary(t *testing.T) {
	for _, value := range []any{nil, NullString()} {
		for _, index := range []int32{-1, 0, math.MaxInt32} {
			func() {
				defer func() {
					if result := recover(); !CaughtAs(result, "NullPointerException") {
						t.Fatalf("caller boundary must keep NullPointerException, got %T", result)
					}
				}()
				_ = StringCharAt(StringRequireNonNull(value), index)
			}()
		}
	}
}

func TestCharAtCandidateConcurrentObservations(t *testing.T) {
	value := "sharedλ😀\x00\U0010ffff"
	want := StringChars(value)
	failures := make(chan string, 8)
	var workers sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for repeat := 0; repeat < 32; repeat++ {
				for index, unit := range want {
					if StringCharAt(value, int32(index)) != unit {
						failures <- "concurrent code-unit mismatch"
						return
					}
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}

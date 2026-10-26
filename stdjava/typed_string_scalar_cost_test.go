package stdjava

import (
	"fmt"
	"sync"
	"testing"
)

func typedStringRecovered(f func()) (value any) {
	defer func() { value = recover() }()
	f()
	return nil
}

func typedStringExpectNPE(t *testing.T, label string, f func()) {
	t.Helper()
	caught := typedStringRecovered(f)
	exception, ok := caught.(NullPointerException)
	if !ok || exception.Message() != "null reference" {
		t.Fatalf("%s: got %T %v, want NullPointerException with original message", label, caught, caught)
	}
}

func TestTypedJavaStringScalarNullAndBounds(t *testing.T) {
	var absent *JavaString
	typedStringExpectNPE(t, "adapter typed nil", func() { RequireJavaString(absent) })
	typedStringExpectNPE(t, "length typed nil", func() { absent.Length() })
	for _, index := range []int32{-1, 0, 1, 2147483647} {
		typedStringExpectNPE(t, fmt.Sprintf("nil charAt(%d)", index), func() { absent.CharAt(index) })
	}
	text := NewJavaStringUTF16([]uint16{0, 0xD800, 0xDC00, 0xFFFF})
	if RequireJavaString(text) != text {
		t.Fatal("nonnull adapter changed object identity")
	}
	for _, index := range []int32{-1, text.Length(), 2147483647} {
		caught := typedStringRecovered(func() { text.CharAt(index) })
		exception, ok := caught.(StringIndexOutOfBoundsException)
		want := fmt.Sprintf("Index %d out of bounds for length 4", index)
		if !ok || exception.Message() != want {
			t.Fatalf("charAt(%d): got %T %v, want original bounds exception/message", index, caught, caught)
		}
	}
	empty := NewJavaStringUTF16(nil)
	if empty.Length() != 0 {
		t.Fatal("empty length changed")
	}
	if _, ok := typedStringRecovered(func() { empty.CharAt(0) }).(StringIndexOutOfBoundsException); !ok {
		t.Fatal("empty charAt(0) must reject its index")
	}
	// Legacy concrete String sentinel and erased typed nil remain distinct
	// boundaries; a typed-receiver optimization must preserve both contracts.
	typedStringExpectNPE(t, "erased typed nil", func() { ReferenceRequireNonNull(any(absent)) })
	typedStringExpectNPE(t, "legacy String sentinel", func() { ReferenceRequireNonNull(NullString()) })
	if !StringIsNull(NullString()) || StringIsNull("") || StringIsNull("null") || StringIsNull(absent) {
		t.Fatal("legacy String sentinel recognition changed")
	}
	for _, value := range []string{"", "null", "\xffordinary"} {
		if ReferenceRequireNonNull(value) != value {
			t.Fatal("ordinary concrete String reference changed")
		}
	}
}

func TestTypedJavaStringScalarUTF16CopiesAndConcurrency(t *testing.T) {
	want := []uint16{0, 'A', 0xD800, 0xDC00, 0xD83D, 0xDE00, 0xFFFF}
	input := append([]uint16(nil), want...)
	text := NewJavaStringUTF16(input)
	copy := CopyJavaString(text)
	input[0] = 'x'
	exported := text.UTF16Copy()
	exported[1] = 'y'
	if copy == text || RequireJavaString(text) != text || text.Length() != int32(len(want)) || copy.Length() != text.Length() {
		t.Fatal("identity, copied length, or UTF16 unit count changed")
	}
	for index, unit := range want {
		if text.CharAt(int32(index)) != rune(unit) || copy.CharAt(int32(index)) != rune(unit) {
			t.Fatalf("constructor/export alias or UTF16 unit changed at %d", index)
		}
	}
	var workers sync.WaitGroup
	failures := make(chan bool, 8)
	for worker := 0; worker < 8; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for repeat := 0; repeat < 200; repeat++ {
				for index, unit := range want {
					if RequireJavaString(text).Length() != int32(len(want)) || RequireJavaString(text).CharAt(int32(index)) != rune(unit) {
						failures <- true
						return
					}
				}
			}
		}()
	}
	workers.Wait()
	close(failures)
	for range failures {
		t.Fatal("concurrent immutable scalar read changed")
	}
}

var typedJavaStringScalarCostSink uint64

func BenchmarkTypedJavaStringScalarBoundary(b *testing.B) {
	lengths := []int{0, 1, 16, 64, 256, 1024}
	unitPattern := []uint16{0, 'A', 0xD800, 0xDC00, 0xD83D, 0xDE00, 0xFFFF, 0xFEFF}
	texts := make([]*JavaString, len(lengths))
	unitsPerIteration := 0
	for item, length := range lengths {
		units := make([]uint16, length)
		for index := range units {
			units[index] = unitPattern[(item+index)%len(unitPattern)]
		}
		texts[item] = NewJavaStringUTF16(units)
		unitsPerIteration += length
	}
	b.ReportAllocs()
	b.ResetTimer()
	var checksum uint64
	for repeat := 0; repeat < b.N; repeat++ {
		for _, text := range texts {
			for index := int32(0); index < RequireJavaString(text).Length(); index++ {
				checksum = checksum*31 + uint64(RequireJavaString(text).CharAt(index)) + uint64(repeat)
			}
		}
	}
	b.StopTimer()
	typedJavaStringScalarCostSink = checksum
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/float64(unitsPerIteration), "ns/unit")
}

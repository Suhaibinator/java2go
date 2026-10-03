package stdjava

import (
	"fmt"
	"slices"
	"sync"
	"testing"
	"unicode/utf16"
)

func TestCI111SQLNativeCompatibilityAndReferenceFormatting(t *testing.T) {
	original := TimeZoneGetDefault()
	TimeZoneSetDefault(TimeZoneGetTimeZone("UTC"))
	t.Cleanup(func() { TimeZoneSetDefault(original) })
	for _, sample := range []struct {
		name, input string
		native      func(string) DateValue
		reference   func(*JavaString) DateValue
	}{
		{"Date", "2023-2-29", func(s string) DateValue { return SQLDateValueOf(s) }, func(s *JavaString) DateValue { return SQLDateValueOfJavaString(s) }},
		{"Time", "25:61:61", func(s string) DateValue { return SQLTimeValueOf(s) }, func(s *JavaString) DateValue { return SQLTimeValueOfJavaString(s) }},
		{"Timestamp", " 2024-01-01 01:02:03.00100 ", func(s string) DateValue { return SQLTimestampValueOf(s) }, func(s *JavaString) DateValue { return SQLTimestampValueOfJavaString(s) }},
	} {
		t.Run(sample.name, func(t *testing.T) {
			host := sample.native(sample.input)
			input := NewJavaStringUTF16(utf16.Encode([]rune(sample.input)))
			value := sample.reference(input)
			if host.GetTime() != value.GetTime() || host.JavaDynamicTypeID() != value.JavaDynamicTypeID() {
				t.Fatal("native compatibility entrypoint changed SQL value")
			}
			if !slices.Equal(input.UTF16Copy(), utf16.Encode([]rune(sample.input))) {
				t.Fatal("parser mutated immutable input")
			}
			first := JavaStringValueOfExecution(NewExecution(), value)
			second := JavaStringValueOfExecution(NewExecution(), value)
			if first == second || !first.Equals(second) {
				t.Fatal("SQL Object formatting must produce fresh equal Strings")
			}
			text := host.(interface{ String() string }).String()
			if !slices.Equal(first.UTF16Copy(), utf16.Encode([]rune(text))) {
				t.Fatal("Java and native SQL numeric formatting disagree")
			}
			for _, nullable := range []func(){func() { sample.native(NullString()) }, func() { sample.reference(nil) }} {
				func() {
					defer func() {
						failure := recover()
						if _, ok := failure.(IllegalArgumentException); !ok {
							t.Fatalf("null input: got %T", failure)
						}
						message := JavaThrowableMessageDefault(failure)
						if sample.name == "Timestamp" {
							if message == nil || !slices.Equal(message.UTF16Copy(), []uint16{'n', 'u', 'l', 'l', ' ', 's', 't', 'r', 'i', 'n', 'g'}) {
								t.Fatal("Timestamp null diagnostic changed")
							}
						} else if message != nil {
							t.Fatal("Date/Time null diagnostic must have a null message")
						}
					}()
					nullable()
				}()
			}
		})
	}
}

func TestCI111SQLSharedImmutableInputRace(t *testing.T) {
	input := NewJavaStringUTF16([]uint16{'2', '0', '2', '4', '-', '0', '1', '-', '0', '1', ' ', '0', '1', ':', '0', '2', ':', '0', '3', '.', '0', '0', '1'})
	baseline := SQLTimestampValueOfJavaString(input)
	want := baseline.StringJava2goExecution(NewExecution())
	errors := make(chan error, 32)
	var workers sync.WaitGroup
	for i := 0; i < cap(errors); i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for repeat := 0; repeat < 20; repeat++ {
				value := SQLTimestampValueOfJavaString(input)
				if value.GetTime() != baseline.GetTime() || value.GetNanos() != baseline.GetNanos() || !value.StringJava2goExecution(NewExecution()).Equals(want) {
					errors <- fmt.Errorf("shared immutable SQL input changed observations")
					return
				}
			}
		}()
	}
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

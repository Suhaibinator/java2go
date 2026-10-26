package stdjava

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

type longParseActualCase struct {
	Label string `json:"label"`
	Input []uint16 `json:"input"`
	Radix int32 `json:"radix"`
	UseDefault bool `json:"use_default"`
	Outcome string `json:"outcome"`
	Value int64 `json:"value"`
	ExceptionClass string `json:"exception_class"`
	Message []uint16 `json:"message"`
}

func longParseActualCases(t *testing.T) []longParseActualCase {
	t.Helper()
	data, err := os.ReadFile("testdata/long_parse_actual_jdk21.json")
	if err != nil { t.Fatal(err) }
	var fixture struct { Cases []longParseActualCase `json:"cases"` }
	if err := json.Unmarshal(data, &fixture); err != nil { t.Fatal(err) }
	if len(fixture.Cases) != 29 { t.Fatalf("actual case count=%d, want29", len(fixture.Cases)) }
	return fixture.Cases
}

func longParseActualRun(c longParseActualCase) (value int64, thrown any) {
	defer func() { thrown = recover() }()
	var input *JavaString
	if c.Input != nil { input = NewJavaStringUTF16(c.Input) }
	if c.UseDefault { return JavaLongParseLong(input), nil }
	return JavaLongParseLong(input, c.Radix), nil
}

// Values and inputs come from 26 unchanged LongFlow cases plus the three actual
// static-import/default-radix observations, never from Go's parser.
func TestCampaignJavaLongParseLongActualJDK21Values(t *testing.T) {
	count := 0
	for _, c := range longParseActualCases(t) {
		if c.Outcome != "ok" { continue }
		count++
		value, thrown := longParseActualRun(c)
		if thrown != nil || value != c.Value { t.Errorf("%s: value=%d thrown=%T, actual JDK value=%d", c.Label, value, thrown, c.Value) }
	}
	if count != 13 { t.Fatalf("actual success count=%d, want13", count) }
}

// Exact UTF16 error units include isolated surrogate and NUL inputs. Nil input
// wins invalid radix; no native string conversion observes these messages.
func TestCampaignJavaLongParseLongActualJDK21Exceptions(t *testing.T) {
	count := 0
	for _, c := range longParseActualCases(t) {
		if c.Outcome != "error" { continue }
		count++
		_, thrown := longParseActualRun(c)
		error, ok := thrown.(NumberFormatException)
		if !ok { t.Errorf("%s: thrown=%T, actual JDK %s", c.Label, thrown, c.ExceptionClass); continue }
		if c.ExceptionClass != "java.lang.NumberFormatException" || error.ThrowableTypeName() != "NumberFormatException" { t.Errorf("%s: wrong nominal exception type", c.Label) }
		message := JavaThrowableMessageDefault(error)
		if message == nil || !reflect.DeepEqual(message.UTF16Copy(), c.Message) { t.Errorf("%s: UTF16 message differs from actual JVM units", c.Label) }
	}
	if count != 16 { t.Fatalf("actual exception count=%d, want16", count) }
}

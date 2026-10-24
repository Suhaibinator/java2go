package stdjava

import (
	"slices"
	"testing"
)

func TestCampaignJoiningUTF16AndFreshIdentity(t *testing.T) {
	text := func(units ...uint16) *JavaString { return NewJavaStringUTF16(units) }
	for _, test := range []struct {
		name                      string
		parts                     []*JavaString
		separator, prefix, suffix *JavaString
		want                      []uint16
	}{
		{"empty", nil, text(), text(), text(), nil},
		{"empty wrapped", nil, text('|'), text('['), text(']'), []uint16{'[', ']'}},
		{"single", []*JavaString{text('A', 0xd800, 0, 0xdfff)}, text('|'), text(), text(), []uint16{'A', 0xd800, 0, 0xdfff}},
		{"null", []*JavaString{nil}, text('|'), text(), text(), []uint16{'n', 'u', 'l', 'l'}},
		{"pair joins", []*JavaString{text(0xd83d), text(0xde00)}, text(), text(), text(), []uint16{0xd83d, 0xde00}},
		{"pair separated", []*JavaString{text(0xd83d), text(0xde00)}, text(0), text(0xdfff), text(0xd800), []uint16{0xdfff, 0xd83d, 0, 0xde00, 0xd800}},
		{"empty parts", []*JavaString{text(), text(), text()}, text('|'), text('['), text(']'), []uint16{'[', '|', '|', ']'}},
		{"mixed", []*JavaString{text('A', 0xe000), nil, text(0xd800, 0, 0xdfff)}, text('|'), text('['), text(']'), []uint16{'[', 'A', 0xe000, '|', 'n', 'u', 'l', 'l', '|', 0xd800, 0, 0xdfff, ']'}},
	} {
		t.Run(test.name, func(t *testing.T) {
			stream := NewStream(test.parts...)
			first := JavaStringStreamJoining(stream, test.separator, test.prefix, test.suffix)
			second := JavaStringStreamJoining(stream, test.separator, test.prefix, test.suffix)
			if first == nil || second == nil || first == second || !first.Equals(second) || !slices.Equal(first.UTF16Copy(), test.want) {
				t.Fatalf("joining content/identity: first=%x want=%x", first.UTF16Copy(), test.want)
			}
			for _, input := range append(append([]*JavaString{}, test.parts...), test.separator, test.prefix, test.suffix) {
				if input != nil && first == input {
					t.Fatal("joining returned an input reference")
				}
			}
			if exported := first.UTF16Copy(); len(exported) != 0 {
				exported[0] ^= 0xffff
			}
			if !slices.Equal(first.UTF16Copy(), test.want) || !slices.Equal(second.UTF16Copy(), test.want) {
				t.Fatal("export mutation changed retained joining results")
			}
		})
	}
}

func TestCampaignJoiningDoesNotAliasInputs(t *testing.T) {
	units := []uint16{'A', 0xd800, 0, 0xdfff}
	input := NewJavaStringUTF16(units)
	prefix := NewJavaStringUTF16([]uint16{'['})
	suffix := NewJavaStringUTF16([]uint16{']'})
	result := JavaStringStreamJoining(NewStream(input), JavaStringLiteralUTF16(nil), prefix, suffix)
	units[0] = 'X'
	input.UTF16Copy()[0] = 'Y'
	prefix.UTF16Copy()[0] = 'Z'
	suffix.UTF16Copy()[0] = 'Q'
	if !slices.Equal(result.UTF16Copy(), []uint16{'[', 'A', 0xd800, 0, 0xdfff, ']'}) || input.CharAt(0) != 'A' {
		t.Fatal("joining changed source or result storage independence")
	}
}

func TestCampaignJoiningNullParametersEvenWhenEmpty(t *testing.T) {
	empty := JavaStringLiteralUTF16(nil)
	for _, arguments := range [][3]*JavaString{{nil, empty, empty}, {empty, nil, empty}, {empty, empty, nil}} {
		func() {
			defer func() {
				if recovered := recover(); !CaughtAs(recovered, "NullPointerException") {
					t.Fatalf("null joining parameter: %v", recovered)
				}
			}()
			JavaStringStreamJoining(NewStream[*JavaString](), arguments[0], arguments[1], arguments[2])
		}()
	}
}

var campaignJoiningResultSink *JavaString
var campaignJoiningHashSink int32

func BenchmarkCampaignJoiningOwnedUTF16(b *testing.B) {
	for _, test := range []struct {
		name               string
		count, width, mode int
		wrapped            bool
	}{
		{"empty", 0, 0, 0, false}, {"emptyWrapped", 0, 0, 0, true},
		{"singleASCII", 1, 16, 0, false}, {"eightASCII", 8, 16, 0, true},
		{"128ASCII", 128, 16, 0, true}, {"eightLongBMP", 8, 1024, 1, true},
		{"128Mixed", 128, 16, 2, true}, {"eightLongMixed", 8, 1024, 2, true},
	} {
		for _, delimiter := range []struct {
			name  string
			units []uint16
		}{{"emptyDelimiter", nil}, {"pipeDelimiter", []uint16{'|'}}} {
			b.Run(test.name+"/"+delimiter.name, func(b *testing.B) {
				parts := make([]*JavaString, test.count)
				patterns := [][]uint16{{'a', 'b', 'c', 'd'}, {0xe000, 0x96ea, 0x03bb, 0x0100}, {0, 0xd83d, 0xde00, 0xd800, 0xdfff, 'Z'}}
				for part := range parts {
					if test.mode == 2 && part%4 == 3 {
						continue
					}
					units := make([]uint16, test.width)
					for index := range units {
						units[index] = patterns[test.mode][index%len(patterns[test.mode])]
					}
					parts[part] = NewJavaStringUTF16(units)
				}
				separator, prefix, suffix := NewJavaStringUTF16(delimiter.units), NewJavaStringUTF16(nil), NewJavaStringUTF16(nil)
				if test.wrapped {
					prefix, suffix = NewJavaStringUTF16([]uint16{'['}), NewJavaStringUTF16([]uint16{']'})
				}
				stream := NewStream(parts...)
				want := JavaStringStreamJoining(stream, separator, prefix, suffix)
				b.ReportAllocs()
				b.ResetTimer()
				for index := 0; index < b.N; index++ {
					result := JavaStringStreamJoining(stream, separator, prefix, suffix)
					campaignJoiningResultSink = result
					campaignJoiningHashSink = result.HashCode()
				}
				b.StopTimer()
				if !campaignJoiningResultSink.Equals(want) {
					b.Fatal("joining benchmark output changed")
				}
			})
		}
	}
}

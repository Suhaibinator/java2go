package stdjava

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

type integerParseReferenceCase struct {
	label      string
	units      []uint16
	radix      int32
	useDefault bool
}

func integerParseReferenceCases() []integerParseReferenceCase {
	cases := []integerParseReferenceCase{
		{"null", nil, 10, false},
		{"null.lowradix", nil, 1, false},
		{"null.highradix", nil, 37, false},
		{"null.default", nil, 10, true},
		{"empty.10", []uint16{}, 10, false},
		{"empty.16", []uint16{}, 16, false},
		{"empty.36", []uint16{}, 36, false},
		{"empty.default", []uint16{}, 10, true},
		{"plus.10", []uint16{0x2b}, 10, false},
		{"plus.16", []uint16{0x2b}, 16, false},
		{"plus.36", []uint16{0x2b}, 36, false},
		{"plus.default", []uint16{0x2b}, 10, true},
		{"minus.10", []uint16{0x2d}, 10, false},
		{"minus.16", []uint16{0x2d}, 16, false},
		{"minus.36", []uint16{0x2d}, 36, false},
		{"minus.default", []uint16{0x2d}, 10, true},
		{"zero.10", []uint16{0x30}, 10, false},
		{"zero.16", []uint16{0x30}, 16, false},
		{"zero.36", []uint16{0x30}, 36, false},
		{"zero.default", []uint16{0x30}, 10, true},
		{"pluszero.10", []uint16{0x2b, 0x30}, 10, false},
		{"pluszero.16", []uint16{0x2b, 0x30}, 16, false},
		{"pluszero.36", []uint16{0x2b, 0x30}, 36, false},
		{"pluszero.default", []uint16{0x2b, 0x30}, 10, true},
		{"minuszero.10", []uint16{0x2d, 0x30}, 10, false},
		{"minuszero.16", []uint16{0x2d, 0x30}, 16, false},
		{"minuszero.36", []uint16{0x2d, 0x30}, 36, false},
		{"minuszero.default", []uint16{0x2d, 0x30}, 10, true},
		{"leading.10", []uint16{0x30, 0x30, 0x30, 0x31, 0x32, 0x33}, 10, false},
		{"leading.16", []uint16{0x30, 0x30, 0x30, 0x31, 0x32, 0x33}, 16, false},
		{"leading.36", []uint16{0x30, 0x30, 0x30, 0x31, 0x32, 0x33}, 36, false},
		{"leading.default", []uint16{0x30, 0x30, 0x30, 0x31, 0x32, 0x33}, 10, true},
		{"doubleplus.10", []uint16{0x2b, 0x2b, 0x31}, 10, false},
		{"doubleplus.16", []uint16{0x2b, 0x2b, 0x31}, 16, false},
		{"doubleplus.36", []uint16{0x2b, 0x2b, 0x31}, 36, false},
		{"doubleplus.default", []uint16{0x2b, 0x2b, 0x31}, 10, true},
		{"doubleminus.10", []uint16{0x2d, 0x2d, 0x31}, 10, false},
		{"doubleminus.16", []uint16{0x2d, 0x2d, 0x31}, 16, false},
		{"doubleminus.36", []uint16{0x2d, 0x2d, 0x31}, 36, false},
		{"doubleminus.default", []uint16{0x2d, 0x2d, 0x31}, 10, true},
		{"mixedsign.10", []uint16{0x2b, 0x2d, 0x31}, 10, false},
		{"mixedsign.16", []uint16{0x2b, 0x2d, 0x31}, 16, false},
		{"mixedsign.36", []uint16{0x2b, 0x2d, 0x31}, 36, false},
		{"mixedsign.default", []uint16{0x2b, 0x2d, 0x31}, 10, true},
		{"space.10", []uint16{0x20, 0x31}, 10, false},
		{"space.16", []uint16{0x20, 0x31}, 16, false},
		{"space.36", []uint16{0x20, 0x31}, 36, false},
		{"space.default", []uint16{0x20, 0x31}, 10, true},
		{"trailing.10", []uint16{0x31, 0x20}, 10, false},
		{"trailing.16", []uint16{0x31, 0x20}, 16, false},
		{"trailing.36", []uint16{0x31, 0x20}, 36, false},
		{"trailing.default", []uint16{0x31, 0x20}, 10, true},
		{"tab.10", []uint16{0x9, 0x31}, 10, false},
		{"tab.16", []uint16{0x9, 0x31}, 16, false},
		{"tab.36", []uint16{0x9, 0x31}, 36, false},
		{"tab.default", []uint16{0x9, 0x31}, 10, true},
		{"newline.10", []uint16{0x31, 0xa}, 10, false},
		{"newline.16", []uint16{0x31, 0xa}, 16, false},
		{"newline.36", []uint16{0x31, 0xa}, 36, false},
		{"newline.default", []uint16{0x31, 0xa}, 10, true},
		{"nul.10", []uint16{0x31, 0x0}, 10, false},
		{"nul.16", []uint16{0x31, 0x0}, 16, false},
		{"nul.36", []uint16{0x31, 0x0}, 36, false},
		{"nul.default", []uint16{0x31, 0x0}, 10, true},
		{"underscore.10", []uint16{0x31, 0x5f, 0x30}, 10, false},
		{"underscore.16", []uint16{0x31, 0x5f, 0x30}, 16, false},
		{"underscore.36", []uint16{0x31, 0x5f, 0x30}, 36, false},
		{"underscore.default", []uint16{0x31, 0x5f, 0x30}, 10, true},
		{"prefix.10", []uint16{0x30, 0x78, 0x31, 0x30}, 10, false},
		{"prefix.16", []uint16{0x30, 0x78, 0x31, 0x30}, 16, false},
		{"prefix.36", []uint16{0x30, 0x78, 0x31, 0x30}, 36, false},
		{"prefix.default", []uint16{0x30, 0x78, 0x31, 0x30}, 10, true},
		{"nonbreaking.10", []uint16{0xa0, 0x31}, 10, false},
		{"nonbreaking.16", []uint16{0xa0, 0x31}, 16, false},
		{"nonbreaking.36", []uint16{0xa0, 0x31}, 36, false},
		{"nonbreaking.default", []uint16{0xa0, 0x31}, 10, true},
		{"fullwidth.10", []uint16{0xff11, 0xff12, 0xff13}, 10, false},
		{"fullwidth.16", []uint16{0xff11, 0xff12, 0xff13}, 16, false},
		{"fullwidth.36", []uint16{0xff11, 0xff12, 0xff13}, 36, false},
		{"fullwidth.default", []uint16{0xff11, 0xff12, 0xff13}, 10, true},
		{"arabic.10", []uint16{0x661, 0x662, 0x663}, 10, false},
		{"arabic.16", []uint16{0x661, 0x662, 0x663}, 16, false},
		{"arabic.36", []uint16{0x661, 0x662, 0x663}, 36, false},
		{"arabic.default", []uint16{0x661, 0x662, 0x663}, 10, true},
		{"devanagari.10", []uint16{0x967, 0x968, 0x969}, 10, false},
		{"devanagari.16", []uint16{0x967, 0x968, 0x969}, 16, false},
		{"devanagari.36", []uint16{0x967, 0x968, 0x969}, 36, false},
		{"devanagari.default", []uint16{0x967, 0x968, 0x969}, 10, true},
		{"fullwidthletters.10", []uint16{0xff26, 0xff46}, 10, false},
		{"fullwidthletters.16", []uint16{0xff26, 0xff46}, 16, false},
		{"fullwidthletters.36", []uint16{0xff26, 0xff46}, 36, false},
		{"fullwidthletters.default", []uint16{0xff26, 0xff46}, 10, true},
		{"asciiletters.10", []uint16{0x5a, 0x7a}, 10, false},
		{"asciiletters.16", []uint16{0x5a, 0x7a}, 16, false},
		{"asciiletters.36", []uint16{0x5a, 0x7a}, 36, false},
		{"asciiletters.default", []uint16{0x5a, 0x7a}, 10, true},
		{"nonasciisign.10", []uint16{0x2212, 0x31}, 10, false},
		{"nonasciisign.16", []uint16{0x2212, 0x31}, 16, false},
		{"nonasciisign.36", []uint16{0x2212, 0x31}, 36, false},
		{"nonasciisign.default", []uint16{0x2212, 0x31}, 10, true},
		{"high.10", []uint16{0xd800}, 10, false},
		{"high.16", []uint16{0xd800}, 16, false},
		{"high.36", []uint16{0xd800}, 36, false},
		{"high.default", []uint16{0xd800}, 10, true},
		{"low.10", []uint16{0xdc00}, 10, false},
		{"low.16", []uint16{0xdc00}, 16, false},
		{"low.36", []uint16{0xdc00}, 36, false},
		{"low.default", []uint16{0xdc00}, 10, true},
		{"highmiddle.10", []uint16{0x31, 0xd800, 0x32}, 10, false},
		{"highmiddle.16", []uint16{0x31, 0xd800, 0x32}, 16, false},
		{"highmiddle.36", []uint16{0x31, 0xd800, 0x32}, 36, false},
		{"highmiddle.default", []uint16{0x31, 0xd800, 0x32}, 10, true},
		{"lowmiddle.10", []uint16{0x31, 0xdc00, 0x32}, 10, false},
		{"lowmiddle.16", []uint16{0x31, 0xdc00, 0x32}, 16, false},
		{"lowmiddle.36", []uint16{0x31, 0xdc00, 0x32}, 36, false},
		{"lowmiddle.default", []uint16{0x31, 0xdc00, 0x32}, 10, true},
		{"pairdigit.10", []uint16{0xd835, 0xdfd9}, 10, false},
		{"pairdigit.16", []uint16{0xd835, 0xdfd9}, 16, false},
		{"pairdigit.36", []uint16{0xd835, 0xdfd9}, 36, false},
		{"pairdigit.default", []uint16{0xd835, 0xdfd9}, 10, true},
		{"maximum.10", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x37}, 10, false},
		{"maximum.16", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x37}, 16, false},
		{"maximum.36", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x37}, 36, false},
		{"maximum.default", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x37}, 10, true},
		{"minimum.10", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 10, false},
		{"minimum.16", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 16, false},
		{"minimum.36", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 36, false},
		{"minimum.default", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 10, true},
		{"overpositive.10", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 10, false},
		{"overpositive.16", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 16, false},
		{"overpositive.36", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 36, false},
		{"overpositive.default", []uint16{0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x38}, 10, true},
		{"overnegative.10", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x39}, 10, false},
		{"overnegative.16", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x39}, 16, false},
		{"overnegative.36", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x39}, 36, false},
		{"overnegative.default", []uint16{0x2d, 0x32, 0x31, 0x34, 0x37, 0x34, 0x38, 0x33, 0x36, 0x34, 0x39}, 10, true},
		{"longzero.10", []uint16{0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31}, 10, false},
		{"longzero.16", []uint16{0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31}, 16, false},
		{"longzero.36", []uint16{0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31}, 36, false},
		{"longzero.default", []uint16{0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x31}, 10, true},
		{"invalidradix.-2147483648", []uint16{0x31}, -2147483648, false},
		{"invalidradix.-1", []uint16{0x31}, -1, false},
		{"invalidradix.0", []uint16{0x31}, 0, false},
		{"invalidradix.1", []uint16{0x31}, 1, false},
		{"invalidradix.37", []uint16{0x31}, 37, false},
		{"invalidradix.2147483647", []uint16{0x31}, 2147483647, false}}
	for radix := int32(2); radix <= 36; radix++ {
		for _, value := range []int64{2147483647, -2147483648, 2147483648, -2147483649} {
			ascii := strconv.FormatInt(value, int(radix))
			units := make([]uint16, len(ascii))
			for i := range ascii {
				units[i] = uint16(ascii[i])
			}
			cases = append(cases, integerParseReferenceCase{fmt.Sprintf("boundary.%d.%d", radix, value), units, radix, false})
		}
	}
	return cases
}
func integerParseReferenceRun(c integerParseReferenceCase) (value int32, thrown any) {
	defer func() { thrown = recover() }()
	var input *JavaString
	if c.units != nil {
		input = NewJavaStringUTF16(c.units)
	}
	if c.useDefault {
		return JavaIntegerParseInt(input), nil
	}
	return JavaIntegerParseInt(input, c.radix), nil
}
func TestCampaignJavaIntegerParseIntValuesJDK21(t *testing.T) {
	var out strings.Builder
	out.WriteString("digits36=")
	for cp := 0; cp <= 0xffff; cp++ {
		value, thrown := integerParseReferenceRun(integerParseReferenceCase{units: []uint16{uint16(cp)}, radix: 36})
		if thrown == nil {
			fmt.Fprintf(&out, "%x:%d,", cp, value)
		} else if _, ok := thrown.(NumberFormatException); !ok {
			t.Fatalf("unit%x threw %T:%v", cp, thrown, thrown)
		}
	}
	out.WriteByte('\n')
	for _, c := range integerParseReferenceCases() {
		value, thrown := integerParseReferenceRun(c)
		if thrown == nil {
			fmt.Fprintf(&out, "%s=value:%d\n", c.label, value)
		} else {
			error, ok := thrown.(NumberFormatException)
			if !ok {
				t.Fatalf("%s threw %T:%v", c.label, thrown, thrown)
			}
			fmt.Fprintf(&out, "%s=%s\n", c.label, error.ThrowableTypeName())
		}
	}
	want, err := os.ReadFile("testdata/integer_parse_values_jdk21.txt")
	if err != nil {
		t.Fatal(err)
	}
	gotLines, wantLines := strings.Split(out.String(), "\n"), strings.Split(string(want), "\n")
	if len(gotLines) != len(wantLines) {
		t.Fatalf("linecount Go%d/JDK%d", len(gotLines), len(wantLines))
	}
	for i := range wantLines {
		if gotLines[i] != wantLines[i] {
			t.Errorf("line%d Go:%s JDK21:%s", i+1, gotLines[i], wantLines[i])
		}
	}
}

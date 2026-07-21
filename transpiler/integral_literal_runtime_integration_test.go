package transpiler

import (
	"strings"
	"testing"
)

func TestNonDecimalIntegralLiteralsUseJavaSignedWidths(t *testing.T) {
	src := `
public class IntegralLiteralProgram {
    public static int hexIntNegOne() { return 0xFFFF_FFFF; }
    public static int hexIntMin() { return 0x8000_0000; }
    public static int binaryIntNegOne() { return 0b1111_1111_1111_1111_1111_1111_1111_1111; }
    public static int binaryIntMin() { return 0b1000_0000_0000_0000_0000_0000_0000_0000; }
    public static int octalIntNegOne() { return 037_777_777_777; }
    public static int octalIntMin() { return 020_000_000_000; }

    public static long hexLongNegOne() { return 0xFFFF_FFFF_FFFF_FFFFL; }
    public static long hexLongMin() { return 0x8000_0000_0000_0000L; }
    public static long hexLongUnsignedIntMax() { return 0xFFFF_FFFFL; }
    public static long binaryLongNegOne() {
        return 0b1111_1111_1111_1111_1111_1111_1111_1111_1111_1111_1111_1111_1111_1111_1111_1111L;
    }
    public static long binaryLongMin() {
        return 0b1000_0000_0000_0000_0000_0000_0000_0000_0000_0000_0000_0000_0000_0000_0000_0000l;
    }
    public static long octalLongNegOne() { return 01_777_777_777_777_777_777_777L; }
    public static long octalLongMin() { return 01_000_000_000_000_000_000_000L; }
}
`

	out := renderGoFileFromJava(t, src)
	for _, signed := range []string{
		"int32(-1)",
		"int32(-2147483648)",
		"int64(-1)",
		"int64(-9223372036854775808)",
		"int64(4294967295)",
	} {
		if !strings.Contains(out, signed) {
			t.Fatalf("expected generated signed-width literal %q, got:\n%s", signed, out)
		}
	}

	runGoTestInTempModule(t, out, `
package main

import "testing"

func TestIntLiteralValues(t *testing.T) {
    tests := []struct {
        name string
        got  int32
        want int32
    }{
        {"hex negative one", HexIntNegOne(), -1},
        {"hex minimum", HexIntMin(), -2147483648},
        {"binary negative one", BinaryIntNegOne(), -1},
        {"binary minimum", BinaryIntMin(), -2147483648},
        {"octal negative one", OctalIntNegOne(), -1},
        {"octal minimum", OctalIntMin(), -2147483648},
    }
    for _, test := range tests {
        if test.got != test.want {
            t.Errorf("%s = %d, want %d", test.name, test.got, test.want)
        }
    }
}

func TestLongLiteralValues(t *testing.T) {
    tests := []struct {
        name string
        got  int64
        want int64
    }{
        {"hex negative one", HexLongNegOne(), -1},
        {"hex minimum", HexLongMin(), -9223372036854775808},
        {"hex unsigned int maximum", HexLongUnsignedIntMax(), 4294967295},
        {"binary negative one", BinaryLongNegOne(), -1},
        {"binary minimum", BinaryLongMin(), -9223372036854775808},
        {"octal negative one", OctalLongNegOne(), -1},
        {"octal minimum", OctalLongMin(), -9223372036854775808},
    }
    for _, test := range tests {
        if test.got != test.want {
            t.Errorf("%s = %d, want %d", test.name, test.got, test.want)
        }
    }
}
`)
}

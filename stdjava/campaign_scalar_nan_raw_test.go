package stdjava

import (
	"math"
	"testing"
)

func TestCampaignScalarNaNRawNative(t *testing.T) {
	for _, text := range []string{"NaN", "+NaN", "-NaN", " NaN\t"} {
		value := JavaStringFromHostUTF8(text)
		if got := math.Float64bits(JavaDoubleParseDouble(value)); got != 0x7ff8000000000000 {
			t.Fatalf("Double.parseDouble(%q) bits=%x want JDK21 canonical NaN", text, got)
		}
		if got := math.Float32bits(JavaFloatParseFloat(value)); got != 0x7fc00000 {
			t.Fatalf("Float.parseFloat(%q) bits=%x want JDK21 canonical NaN", text, got)
		}
	}
}

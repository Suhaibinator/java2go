package transpiler

import (
	"fmt"
	"testing"
)

func TestCampaignStringFormatBareNullFixedArrayABI(t *testing.T) {
	const source = `public class StringFormatBareNullFixed {
 public static String run(){return String.format("%s",null)+":"+String.format("%s:%s",null);}
 public static void main(String[] args){System.out.print(run());}
}`
	want := campaignRuntimeJavaOracle(t, "StringFormatBareNullFixed", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestCanonicalABI(t *testing.T){if got:=Run();!got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatalf("generated bare-null result differs from exact JVM stream")}}
`, want))
}

func TestCampaignStringFormatBareNullExpandedABI(t *testing.T) {
	const source = `public class StringFormatBareNullExpanded {
 public static String run(){return String.format("%s:%s","x",null);}
 public static void main(String[] args){System.out.print(run());}
}`
	want := campaignRuntimeJavaOracle(t, "StringFormatBareNullExpanded", source)
	generated := renderGoFileFromJava(t, source)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestCanonicalABI(t *testing.T){if got:=Run();!got.Equals(j.JavaStringFromHostUTF8(%q)){t.Fatalf("generated bare-null result differs from exact JVM stream")}}
`, want))
}

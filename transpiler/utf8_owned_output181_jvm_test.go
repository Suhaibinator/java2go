package transpiler

// Portable final-test shape; no host/campaign/fixture environment.
import (
	_ "embed"
	"fmt"
	"testing"
	"unicode/utf16"
)

//go:embed testdata/byte_encoding181/ByteEncodingBoundary181.java
var utf8OwnedOutput181Java string

func TestUTF8OwnedOutput181JVM(t *testing.T) {
	want := campaignRuntimeJavaOracle(t, "ByteEncodingBoundary181", utf8OwnedOutput181Java)
	t.Logf("JDK21 encoding181 oracle: %q", want)
	generated := renderGoFileFromJava(t, utf8OwnedOutput181Java)
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";"slices")
func TestEncoding181(t *testing.T){
 got:=Run();if got==nil{t.Fatal("null result")}
 want:=%#v
 if !slices.Equal(got.UTF16Copy(),want){t.Fatalf("JDK UTF16 %%x != generated %%x",want,got.UTF16Copy())}
}
`, utf16.Encode([]rune(want))))
}

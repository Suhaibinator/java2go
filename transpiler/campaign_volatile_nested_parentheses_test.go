package transpiler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The fixture and test are frozen before any production fix. The future causal
// RED/GREEN gate must capture JDK21 output before generating Go, and must retain
// this exact fixture, including all receiver/RHS and cleanup observations.
func TestVolatileNestedParenthesesJDK21(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("testdata", "volatile_nested_parentheses", "VolatileNestedParenthesesProbe.java"))
	if err != nil {
		t.Fatal(err)
	}
	want := campaignRuntimeJavaOracle(t, "VolatileNestedParenthesesProbe", string(source))
	t.Logf("JDK21 nested volatile parentheses oracle: %q", want)
	generated := renderGoFileFromJava(t, string(source))
	runGoTestInTempModule(t, generated, fmt.Sprintf(`package main
import("testing";j "github.com/NickyBoy89/java2go/stdjava")
func TestNestedVolatileParentheses(t *testing.T){value:=Run();encoded:=j.JavaStringGetBytes(value,j.UTF_8).Elements;bytes:=make([]byte,len(encoded));for i,b:=range encoded{bytes[i]=byte(b)};if got:=string(bytes);got!=%q{t.Fatalf("JDK %%q != Go %%q",%q,got)}}`, want, want))
}

package transpiler

import (
	"fmt"
	"strings"
	"testing"
)

// A source method's existing Go-operation bindings remain valid when its body
// never emits those operations. Global callable candidates need one inventory
// per hygiene pass, regardless of how many such bindings the project contains.
func TestLocalIdentifierGlobalCandidateWork(t *testing.T) {
	var source strings.Builder
	for owner := 0; owner < 16; owner++ {
		fmt.Fprintf(&source, "class Noise%d{", owner)
		for method := 0; method < 16; method++ {
			fmt.Fprintf(&source, "static int method%d(){return 0;}", method)
		}
		source.WriteString("}")
	}
	source.WriteString("class Work{")
	for method := 0; method < 16; method++ {
		fmt.Fprintf(&source, "void use%d(int len,int cap,int copy,int append,int panic,int recover,int close,int complex){int value=0;}", method)
	}
	source.WriteString("}")
	helper := setupParseHelper(t, source.String())
	work := helper.File.Symbols.FindClassScope("Work")
	if work == nil {
		t.Fatal("work declaration absent")
	}
	for pass := 0; pass < 3; pass++ {
		resolveLocalIdentifierHygiene()
		count := 0
		for _, method := range work.Methods {
			for _, parameter := range method.Parameters {
				if parameter.Name != parameter.OriginalName {
					t.Fatalf("unused body operation %s unnecessarily renamed %s", parameter.OriginalName, parameter.Name)
				}
				count++
			}
		}
		if count != 128 {
			t.Fatalf("parameters=%d want128", count)
		}
	}
}

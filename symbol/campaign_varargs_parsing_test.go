package symbol_test

import (
	"github.com/NickyBoy89/java2go/parsing"
	"testing"
)

func TestCampaignModifiedVarargsSymbols(t *testing.T) {
	for _, modifier := range []string{"", "final ", "@Deprecated ", "@Deprecated final "} {
		t.Run(modifier, func(t *testing.T) {
			source := `public class Varargs { public Varargs(` + modifier + `Object... args) {} public static <T> void accept(` + modifier + `T... values) {} }`
			file := parsing.SourceFile{Name: "Varargs.java", Source: []byte(source)}
			if err := file.ParseAST(); err != nil {
				t.Fatal(err)
			}
			if file.Ast.HasError() {
				t.Fatal("invalid Java parse")
			}
			scope := file.ParseSymbols().BaseClass
			for _, method := range scope.Methods {
				if len(method.Parameters) != 1 {
					t.Fatalf("parameter count for %s: %d", method.OriginalName, len(method.Parameters))
				}
				param := method.Parameters[0]
				wantName, wantType := "args", "Object"
				if method.OriginalName == "accept" {
					wantName, wantType = "values", "T"
				}
				if param.OriginalName != wantName || param.OriginalType != wantType {
					t.Fatalf("%s: parameter = %s %s; want %s %s", method.OriginalName, param.OriginalType, param.OriginalName, wantType, wantName)
				}
			}
		})
	}
}

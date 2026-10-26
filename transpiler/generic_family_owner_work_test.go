package transpiler

import (
	"fmt"
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// All calls use existing APIs. Coverage records visits to the ordinary owner
// search, independently of returned plans or elapsed time.
func TestGenericFamilyOwnerLookupWork(t *testing.T) {
	resolveImport205State(t)
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	var files []parsing.SourceFile
	for index := 0; index < 32; index++ {
		file := parsing.SourceFile{Name: fmt.Sprintf("FamilyOwner%d.java", index), Source: []byte(fmt.Sprintf("package ownerwork; class Cell%d<T>{} class Use%d { Cell%d<?> value; }", index, index, index))}
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		file.ParseSymbols()
		symbol.AddSymbolsToPackage(file.Symbols)
		files = append(files, file)
	}
	ResolveFiles(files)
	for sample := 0; sample < 3; sample++ {
		plans := discoverCanonicalGenericFamilies(Ctx{})
		if len(plans) != 32 {
			t.Fatalf("sample%d families=%d want32", sample, len(plans))
		}
		seen := map[*symbol.ClassScope]bool{}
		for _, plan := range plans {
			if len(plan.members) != 1 || len(plan.binders) != 1 || len(plan.locals) != 0 || len(plan.anonymous) != 0 {
				t.Fatal("family admission/storage changed")
			}
			for owner := range plan.members {
				if seen[owner] {
					t.Fatal("duplicate family")
				}
				seen[owner] = true
			}
		}
	}
}

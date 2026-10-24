package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
)

// These regressions inspect actual registered source scopes. They use no new
// API so the partial helper builds and fails semantically before the schema fix.
func TestReaderNativeCloneIndependentRootRejectsDetachedAliases(t *testing.T) {
	for _, row := range []struct{ name, java string }{
		{"empty", `abstract class R extends java.io.Reader{}`},
		{"allSymbolAliases", `class R extends java.io.Reader{int state;public int read(char[] b,int o,int n){return -1;}public void close(){}}`},
	} {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, row.java)
			owner := resolveClassScopeByQualifiedName(helper.Ctx, "R")
			if owner == nil {
				t.Fatal("actual registered source owner missing")
			}
			if _, ok := sourceNativeObjectCloneCapability(owner, helper.Ctx); !ok {
				t.Fatal("same-tree native capability rejected")
			}
			twin := parsing.SourceFile{Name: "ByteIdenticalTwin.java", Source: []byte(row.java)}
			if err := twin.ParseAST(); err != nil {
				t.Fatal(err)
			}
			other := twin.ParseSymbols().BaseClass
			oldClass, oldFields, oldMethods := owner.Class, owner.Fields, owner.Methods
			owner.Class, owner.Fields, owner.Methods = other.Class, other.Fields, other.Methods
			t.Cleanup(func() { owner.Class, owner.Fields, owner.Methods = oldClass, oldFields, oldMethods })
			// FileScope.BaseClass and every retained symbol can alias the replacement.
			// An independent original parse anchor must still reject this foreign tree.
			if _, ok := sourceNativeObjectCloneCapability(owner, helper.Ctx); ok {
				t.Fatal("detached byte-identical alias graph certified without independent original parse root")
			}
		})
	}
}

func TestReaderNativeCloneIndependentRootRejectsSyntheticFile(t *testing.T) {
	java := `abstract class R extends java.io.Reader{}`
	helper := setupParseHelper(t, java)
	owner := resolveClassScopeByQualifiedName(helper.Ctx, "R")
	synthetic := &symbol.FileScope{Source: []byte(java), Package: helper.Ctx.currentFile.Package, BaseClass: owner, TopLevelClasses: []*symbol.ClassScope{owner}, Imports: map[string]string{}}
	pkg := symbol.GlobalScope.Packages[synthetic.Package]
	if pkg == nil {
		t.Fatal("actual source package missing")
	}
	old := pkg.Files
	pkg.Files = map[string]*symbol.FileScope{"Synthetic.java": synthetic}
	t.Cleanup(func() { pkg.Files = old })
	ctx := Ctx{currentClass: owner, currentFile: synthetic}
	if _, ok := sourceNativeObjectCloneCapability(owner, ctx); ok {
		t.Fatal("hand-built FileScope without parse anchor certified")
	}
}

func TestReaderNativeCloneIndependentRootEmptyDerived(t *testing.T) {
	helper := setupParseHelper(t, `abstract class Empty extends java.io.Reader{}class R extends Empty{public int read(char[]b,int o,int n){return -1;}public void close(){}}`)
	owner := resolveClassScopeByQualifiedName(helper.Ctx, "R")
	root := resolveClassScopeByQualifiedName(helper.Ctx, "Empty")
	if owner == nil || root == nil {
		t.Fatal("actual empty native source root missing")
	}
	cap, ok := sourceNativeObjectCloneCapability(owner, helper.Ctx)
	if !ok || cap.root != root {
		t.Fatal("empty source/native root lost across genuine source inheritance")
	}
}

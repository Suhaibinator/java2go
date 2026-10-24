package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/parsing"
)

func TestReaderNativeCloneCapabilityDeclarationShields(t *testing.T) {
	for _, row := range []struct {
		name, source string
		native       bool
	}{
		{"canonical", `class R extends java.io.Reader{public int read(char[]b,int o,int n){return -1;}public void close(){}}`, true},
		{"namedImport", `import java.io.Reader;class R extends Reader{public int read(char[]b,int o,int n){return -1;}public void close(){}}`, true},
		{"sourceReader", `class Reader{}class R extends Reader{}`, false},
		{"foreignReader", `import foreign.Reader;class R extends Reader{}`, false},
		{"unknown", `class R extends unknown.Reader{}`, false},
		{"binder", `class R<Reader> extends Reader{}`, false},
		{"capturedBinder", `class Outer<Reader>{class R extends Reader{}}`, false},
		{"otherNative", `class R extends java.io.StringReader{R(){super("");}}`, false},
	} {
		t.Run(row.name, func(t *testing.T) {
			helper := setupParseHelper(t, row.source)
			owner := resolveClassScopeByQualifiedName(helper.Ctx, "R")
			if owner == nil {
				owner = resolveClassScopeByQualifiedName(helper.Ctx, "Outer.R")
			}
			if owner == nil {
				t.Fatal("actual parser owner missing")
			}
			_, got := sourceNativeObjectCloneCapability(owner, helper.Ctx)
			if got != row.native {
				t.Fatalf("actual native declaration capability=%t want %t", got, row.native)
			}
		})
	}
}

func TestReaderNativeCloneCapabilityRejectsDetachedDeclarations(t *testing.T) {
	java := `class R extends java.io.Reader{public int read(char[]b,int o,int n){return -1;}public void close(){}}`
	helper := setupParseHelper(t, java)
	owner := resolveClassScopeByQualifiedName(helper.Ctx, "R")
	if owner == nil {
		t.Fatal("actual owner missing")
	}
	if _, ok := sourceNativeObjectCloneCapability(owner, helper.Ctx); !ok {
		t.Fatal("same-tree canonical declaration rejected")
	}
	detached := *owner
	if _, ok := sourceNativeObjectCloneCapability(&detached, helper.Ctx); ok {
		t.Fatal("unregistered copied source scope certified")
	}
	second := parsing.SourceFile{Name: "Detached.java", Source: []byte(java)}
	if err := second.ParseAST(); err != nil {
		t.Fatal(err)
	}
	secondScope := second.ParseSymbols().BaseClass
	definition := *owner.Class
	definition.DeclarationNode = secondScope.Class.DeclarationNode
	previous := owner.Class
	owner.Class = &definition
	t.Cleanup(func() { owner.Class = previous })
	if _, ok := sourceNativeObjectCloneCapability(owner, helper.Ctx); ok {
		t.Fatal("identical-span detached declaration tree certified")
	}
}

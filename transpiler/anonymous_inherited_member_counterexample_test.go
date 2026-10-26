package transpiler

import (
	"os"
	"testing"
)

func TestGenericFamilyAnonymousInheritedMemberTypeStorageRefusal(t *testing.T) {
	b, err := os.ReadFile("testdata/anonymous_inherited_member/AnonymousInheritedMemberTypeBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	h := setupParseHelper(t, string(b))
	if p, err := planGenericFamily(h.File.Symbols.FindClassScope("Carrier"), h.Ctx); p != nil || err == nil {
		t.Fatal("anonymous inherited member type rebound to unrelated top-level source class")
	}
}

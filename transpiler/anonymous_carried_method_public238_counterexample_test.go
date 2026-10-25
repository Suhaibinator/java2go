package transpiler

import (
	"os"
	"testing"
)

func TestPublished238AnonymousCarriedMethodBinderStorageRefusal(t *testing.T) {
	b, err := os.ReadFile("testdata/published_anonymous_carried/PublishedAnonymousCarriedBinderBoundary.java")
	if err != nil {
		t.Fatal(err)
	}
	h := setupParseHelper(t, string(b))
	root := h.File.Symbols.FindClassScope("Carrier")
	if root == nil {
		t.Fatal("missing original named family root")
	}
	if p, err := planGenericFamily(root, h.Ctx); p != nil || err == nil {
		t.Fatal("external method binder escaped complete anonymous storage audit")
	}
}

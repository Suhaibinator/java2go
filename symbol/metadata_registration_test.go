package symbol_test

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestMetadataFileScopeRegistersPackageWithoutClass(t *testing.T) {
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	defer func() {
		if value := recover(); value != nil {
			t.Errorf("metadata registration dereferenced a missing class: %v", value)
		}
	}()
	metadata := &symbol.FileScope{Package: "raw.metadata", Source: []byte("package raw.metadata;")}
	symbol.AddSymbolsToPackage(metadata)
	pkg := symbol.GlobalScope.FindPackage("raw.metadata")
	if pkg == nil {
		t.Fatal("package metadata should declare the package scope")
	}
	if len(pkg.Files) != 0 {
		t.Fatalf("package metadata invented a source class: %#v", pkg.Files)
	}
}

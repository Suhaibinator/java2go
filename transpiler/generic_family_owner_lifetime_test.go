package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGenericFamilyOwnershipQueryLifetime(t *testing.T) {
	helper := setupParseHelper(t, `class Cell<T>{} class Use{Cell<?> value;}`)
	foreignGraph := &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	foreign := &resolutionFileIndex{graph: foreignGraph, files: map[*symbol.ClassScope]*symbol.FileScope{}}
	ctx := helper.Ctx.Clone()
	ctx.sourceOwnership = foreign
	if plans := discoverCanonicalGenericFamilies(ctx); len(plans) != 1 {
		t.Fatalf("families=%d want1", len(plans))
	}
	if ctx.sourceOwnership != foreign || foreign.graph != foreignGraph || len(foreign.files) != 0 {
		t.Fatal("discovery mutated caller ownership")
	}
	if plans := discoverCanonicalGenericFamilies(ctx); len(plans) != 1 {
		t.Fatal("fresh discovery lost family")
	}
	if foreign.graph != foreignGraph || len(foreign.files) != 0 {
		t.Fatal("fresh query retained or reset foreign holder")
	}
}

func TestGenericFamilyOwnershipUniqueRefusals(t *testing.T) {
	priorGlobal, priorActive := symbol.GlobalScope, activeResolutionFiles
	t.Cleanup(func() { symbol.GlobalScope, activeResolutionFiles = priorGlobal, priorActive })
	helper := setupParseHelper(t, `class Cell<T>{class Nested{}} class Use{Cell<?> value;}`)
	index := newResolutionFileIndex()
	if !genericFamilyOwnershipUnique(index) {
		t.Fatal("current ordinary graph refused")
	}
	file := helper.File.Symbols
	top := file.TopLevelClasses[0]
	pkg := symbol.GlobalScope.Packages[file.Package]
	pkg.Files["duplicate"] = &symbol.FileScope{TopLevelClasses: []*symbol.ClassScope{top}}
	if genericFamilyOwnershipUnique(index) {
		t.Fatal("duplicated ordinary class admitted")
	}
	delete(pkg.Files, "duplicate")
	if !genericFamilyOwnershipUnique(index) {
		t.Fatal("restored ordinary graph refused")
	}
	file.TopLevelClasses = append(file.TopLevelClasses, &symbol.ClassScope{})
	if genericFamilyOwnershipUnique(index) {
		t.Fatal("late class admitted by old index")
	}
	file.TopLevelClasses = file.TopLevelClasses[:len(file.TopLevelClasses)-1]
	old := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: map[string]*symbol.PackageScope{}}
	if genericFamilyOwnershipUnique(index) {
		t.Fatal("foreign graph admitted")
	}
	symbol.GlobalScope = old
	if !genericFamilyOwnershipUnique(index) {
		t.Fatal("caller index was mutated on refusal")
	}
	prior := activeResolutionFiles
	activeResolutionFiles = &resolutionFileIndex{graph: old, files: map[*symbol.ClassScope]*symbol.FileScope{}}
	if _, known := sourceGenericDemandOwnershipContext(); known {
		t.Fatal("active ownership mismatch admitted")
	}
	activeResolutionFiles = prior
}

package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func localIdentifierCandidateFixture(t *testing.T) (Ctx, Ctx, *symbol.ClassScope) {
	t.Helper()
	ctx := sourceGenericViewDemandTestContext(t, map[string]string{"Names.java": `class Calls{static void call(){}static void placeholder(){}<T>void generic(T value){}}class Work{void staticUse(int token){Calls.call();}void helperUse(int token){new Calls().generic(null);}}`})
	work := ctx.currentFile.FindClassScope("Work")
	calls := ctx.currentFile.FindClassScope("Calls")
	if work == nil || calls == nil {
		t.Fatal("fixture declarations missing")
	}
	staticCtx := classScopeCtx(work, ctx)
	staticCtx.localScope = work.FindMethod().ByOriginalName("staticUse")[0]
	helperCtx := classScopeCtx(work, ctx)
	helperCtx.localScope = work.FindMethod().ByOriginalName("helperUse")[0]
	return staticCtx, helperCtx, calls
}

func TestLocalIdentifierCallCandidatesFreshResolution(t *testing.T) {
	ctx, _, calls := localIdentifierCandidateFixture(t)
	callee := calls.FindMethod().ByOriginalName("call")[0]
	parameter := ctx.localScope.Parameters[0]
	callee.Name = "Collision"
	if !localIdentifierRequiredByBody("Collision", ctx) {
		t.Fatal("current static selector demand lost")
	}
	saved := localIdentifierReservedGlobals()
	callee.Name = "Different"
	if localIdentifierRequiredByBody("Collision", ctx) {
		t.Fatal("standalone body demand retained stale static selector")
	}
	parameter.Name = "Collision"
	hygienizeLocalScopeWithReserved(ctx, saved)
	if parameter.Name != "Collision" {
		t.Fatal("standalone reservation borrowed an old body-demand candidate")
	}
	callee.Name = "Collision"
	resolveLocalIdentifierHygiene()
	if parameter.Name == "Collision" {
		t.Fatal("fresh resolution pass omitted static call demand")
	}
	if callee.Name != "Collision" || parameter.OriginalName != "token" || parameter.OriginalType != "int" {
		t.Fatal("source or callable metadata changed")
	}
	callee.Name = "Different"
	parameter.Name = "Collision"
	resolveLocalIdentifierHygiene()
	if parameter.Name != "Collision" {
		t.Fatal("later resolution retained earlier callable candidates")
	}
}

func TestLocalIdentifierCallCandidatesFlagsAndAliases(t *testing.T) {
	t.Run("execution selector", func(t *testing.T) {
		ctx, _, calls := localIdentifierCandidateFixture(t)
		callee := calls.FindMethod().ByOriginalName("call")[0]
		placeholder := calls.FindMethod().ByOriginalName("placeholder")[0]
		callee.Name = "Collision"
		name := callee.Name + executionMethodSuffix
		placeholder.Name = name
		if !localIdentifierRequiredByBody(name, ctx) {
			t.Fatal("static execution selector demand lost")
		}
		ctx.localScope.Parameters[0].Name = name
		resolveLocalIdentifierHygiene()
		if ctx.localScope.Parameters[0].Name == name {
			t.Fatal("pass omitted execution selector")
		}
		callee.IsStatic = false
		ctx.localScope.Parameters[0].Name = name
		resolveLocalIdentifierHygiene()
		if ctx.localScope.Parameters[0].Name != name {
			t.Fatal("unrelated same-name callable invented a body demand")
		}
	})
	t.Run("generic helper selectors", func(t *testing.T) {
		_, ctx, calls := localIdentifierCandidateFixture(t)
		callee := calls.FindMethod().ByOriginalName("generic")[0]
		callee.RequiresHelper = true
		callee.HelperName = "ReservedHelper"
		for _, name := range []string{"ReservedHelper", "NewReservedHelper"} {
			if !localIdentifierRequiredByBody(name, ctx) {
				t.Fatal("generic helper demand lost", name)
			}
			ctx.localScope.Parameters[0].Name = name
			resolveLocalIdentifierHygiene()
			if ctx.localScope.Parameters[0].Name == name {
				t.Fatal("pass omitted helper selector", name)
			}
		}
		saved := localIdentifierReservedGlobals()
		callee.RequiresHelper = false
		ctx.localScope.Parameters[0].Name = "ReservedHelper"
		hygienizeLocalScopeWithReserved(ctx, saved)
		if ctx.localScope.Parameters[0].Name != "ReservedHelper" {
			t.Fatal("standalone helper flag mutation was ignored")
		}
		if localIdentifierRequiredByBody("ReservedHelper", ctx) {
			t.Fatal("standalone helper demand retained old flag")
		}
	})
}

func TestLocalIdentifierCallCandidatesForeignGraphFallback(t *testing.T) {
	ctx, _, calls := localIdentifierCandidateFixture(t)
	calls.FindMethod().ByOriginalName("call")[0].Name = "Collision"
	saved := localIdentifierReservedGlobals()
	sourceGenericViewDemandTestContext(t, map[string]string{"Other.java": `class Other{static void unused(){}}`})
	other := symbol.GlobalScope.FindPackage("")
	if other == nil {
		t.Fatal("foreign graph missing")
	}
	for _, file := range other.Files {
		file.FindClassScope("Other").FindMethod().ByOriginalName("unused")[0].Name = "Collision"
	}
	ctx.localScope.Parameters[0].Name = "Collision"
	hygienizeLocalScopeWithReserved(ctx, saved)
	if ctx.localScope.Parameters[0].Name != "Collision" {
		t.Fatal("retained reservation donated old graph callable demand")
	}
	if localIdentifierRequiredByBody("Collision", ctx) {
		t.Fatal("standalone demand did not use current graph")
	}
}

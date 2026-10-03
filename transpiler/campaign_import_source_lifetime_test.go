package transpiler

import "testing"

func TestCampaignImportSourceLifetime(t *testing.T) {
	withCleanDiagnostics(t)
	previousImports, previousOwnership := activeImportSourceInventory, activeResolutionFiles
	t.Cleanup(func() { activeImportSourceInventory = previousImports; activeResolutionFiles = previousOwnership })
	t.Run("explicit-inventory-wins-and-unregistered-file-falls-back", func(t *testing.T) {
		helper := setupParseHelper(t, campaignImportSource("alpha"))
		ctx := helper.Ctx
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		explicit := resolvedSourceInventory(ctx)
		activeImportSourceInventory = &callableSubclassSourceInventory{graph: explicit.graph}
		activeResolutionFiles = newResolutionFileIndex()
		if importSourceInventory(ctx) != explicit {
			t.Fatal("scoped fallback displaced caller inventory")
		}
		fresh := ctx
		fresh.callableSubclasses = nil
		campaignAssertImportText(t, fresh, "alpha")
		if importSourceInventory(fresh) != activeImportSourceInventory {
			t.Fatal("registered current graph did not share import syntax during render")
		}
		activeResolutionFiles.files = nil
		if importSourceInventory(fresh) != nil {
			t.Fatal("unregistered file used scoped syntax authority")
		}
		campaignAssertImportText(t, fresh, "alpha")
	})
	t.Run("nested-graph-render-restores-outer-scope-on-success-and-strict-error", func(t *testing.T) {
		outer := setupParseHelper(t, campaignImportSource("alpha"))
		outer.Ctx.callableSubclasses = &callableSubclassSourceInventory{}
		sentinel := resolvedSourceInventory(outer.Ctx)
		activeImportSourceInventory = sentinel
		sentinelOwnership := newResolutionFileIndex()
		activeResolutionFiles = sentinelOwnership
		inner := setupParseHelper(t, "package q;class Inner{static int answer(){return 37;}}")
		if _, err := convertFileNode(inner.File, inner.Ctx); err != nil {
			t.Fatal(err)
		}
		if activeImportSourceInventory != sentinel || activeResolutionFiles != sentinelOwnership {
			t.Fatal("nested successful render leaked import scope")
		}
		if importSourceInventory(inner.Ctx) != nil {
			t.Fatal("restored outer graph served inner standalone imports")
		}
		bad := setupParseHelper(t, "package r;class Broken{static int answer(){record Local(int value){}return 1;}}")
		diagnostics.mu.Lock()
		previousStrict := diagnostics.strict
		diagnostics.mu.Unlock()
		setStrictMode(true)
		defer setStrictMode(previousStrict)
		if _, err := convertFileNode(bad.File, bad.Ctx); err == nil {
			t.Fatal("local record lost strict rejection")
		}
		if activeImportSourceInventory != sentinel || activeResolutionFiles != sentinelOwnership {
			t.Fatal("nested failed render leaked import scope")
		}
		if importSourceInventory(bad.Ctx) != nil {
			t.Fatal("stale graph served standalone imports")
		}
		// Exercise an ordinary panic after scope installation. A mismatched API
		// source/tree pair must propagate the panic and restore the outer render.
		malformed := inner.File
		malformed.Source = nil
		var caught any
		func() {
			defer func() { caught = recover() }()
			_, _ = convertFileNode(malformed, inner.Ctx)
		}()
		if caught == nil {
			t.Fatal("mismatched source/tree API input did not propagate panic")
		}
		if activeImportSourceInventory != sentinel || activeResolutionFiles != sentinelOwnership {
			t.Fatal("panicking render leaked import scope")
		}
	})
}

package transpiler

// readOnlyStaticMethodImports lends source tuples to internal read-only
// iteration only. Callers must not modify or retain the slice. Namespace,
// accessibility, owner and overload decisions are never stored here. The
// caller-owned staticMethodImports API and its fallback remain unchanged.
func readOnlyStaticMethodImports(ctx Ctx) []staticMethodImport {
	inventory, facts, ok := importSourceFacts(ctx)
	if !ok {
		return staticMethodImportsFromFile(ctx)
	}
	if !facts.importsReady {
		facts.imports = staticMethodImportsFromFile(ctx)
		facts.importsReady = true
		storeImportSourceFacts(inventory, ctx.currentFile, facts)
	}
	return facts.imports
}

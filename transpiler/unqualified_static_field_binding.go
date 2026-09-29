package transpiler

// resolveUnqualifiedStaticField is shared by value lowering and receiver type
// inference. Static fields remain visible through enclosing static classes;
// an intervening local or instance field still hides an outer static field.
// It returns the declaring owner so generic arguments/imports are qualified in
// that owner's context rather than in the nested class consuming the value.
func resolveUnqualifiedStaticField(name string, ctx Ctx) *fieldResolution {
	if identifierHasLocalBinding(name, ctx) {
		return nil
	}
	for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
		resolution := findFieldResolutionInHierarchy(scope, name, ctx)
		if resolution == nil {
			continue
		}
		if !resolution.def.IsStatic {
			return nil
		}
		return resolution
	}
	return resolveStaticImportedField(name, ctx).source
}

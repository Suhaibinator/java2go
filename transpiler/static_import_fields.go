package transpiler

import "github.com/NickyBoy89/java2go/symbol"

type staticImportFieldResolution struct {
	source    *fieldResolution
	intrinsic intrinsicKey
	problem   string
}

// A value binding is independent of any imported type with the same name.
// Locals and lexical fields hide imports; explicit imports precede on-demand
// imports, and duplicate paths to one declaration are not ambiguous.
func resolveStaticImportedField(name string, ctx Ctx) staticImportFieldResolution {
	if identifierHasLocalBinding(name, ctx) {
		return staticImportFieldResolution{}
	}
	for owner := ctx.currentClass; owner != nil; owner = owner.Enclosing {
		if findFieldResolutionInHierarchy(owner, name, ctx) != nil {
			return staticImportFieldResolution{}
		}
	}
	for _, wildcard := range []bool{false, true} {
		var selected staticImportFieldResolution
		found := false
		for _, imported := range staticMethodImports(ctx) {
			if imported.wildcard != wildcard || !wildcard && imported.member != name {
				continue
			}
			var candidate staticImportFieldResolution
			if owner := findQualifiedSourceClass(imported.owner); owner != nil {
				field, ambiguous := importedFieldInHierarchy(owner, name, ctx, map[*symbol.ClassScope]bool{})
				if ambiguous {
					return staticImportFieldResolution{problem: "ambiguous static imported field " + name}
				}
				if field == nil || !field.def.IsStatic || !staticImportMethodAccessible(field.def, field.owner, ctx) {
					continue
				}
				candidate.source = field
			} else {
				owner, registered := canonicalIntrinsicOwner(imported.owner, ctx)
				if !registered || !intrinsicOwnerSupported(owner) {
					continue
				}
				key := intrinsicKey{intrinsicOwnerKey(owner), name}
				if staticFieldIntrinsics[key] == nil {
					continue
				}
				candidate.intrinsic = key
			}
			if found && !sameStaticImportedField(selected, candidate) {
				return staticImportFieldResolution{problem: "ambiguous static imported field " + name}
			}
			selected, found = candidate, true
		}
		if found {
			return selected
		}
	}
	return staticImportFieldResolution{}
}

func sameStaticImportedField(left, right staticImportFieldResolution) bool {
	if left.source != nil || right.source != nil {
		return left.source != nil && right.source != nil && left.source.def == right.source.def
	}
	return left.intrinsic == right.intrinsic
}

func importedFieldInHierarchy(owner *symbol.ClassScope, name string, ctx Ctx, visiting map[*symbol.ClassScope]bool) (*fieldResolution, bool) {
	if owner == nil || visiting[owner] {
		return nil, false
	}
	if field := owner.FindFieldByName(name); field != nil {
		return &fieldResolution{def: field, owner: owner}, false
	}
	visiting[owner] = true
	defer delete(visiting, owner)
	parents := []*symbol.ClassScope{resolveSuperclassScopeInDeclaringContext(ctx, owner)}
	parents = append(parents, resolveImplementedInterfaceScopesInDeclaringContext(ctx, owner)...)
	var found *fieldResolution
	for _, parent := range parents {
		field, ambiguous := importedFieldInHierarchy(parent, name, ctx, visiting)
		if ambiguous {
			return nil, true
		}
		if field == nil || field.def.IsPrivate {
			continue
		}
		// Package members do not survive an inheritance edge across packages.
		if !field.owner.IsInterface && !staticImportMethodHasModifier(field.def, "public") && !staticImportMethodHasModifier(field.def, "protected") && findJavaPackageForClassScope(field.owner) != findJavaPackageForClassScope(owner) {
			continue
		}
		if found != nil && found.def != field.def {
			return nil, true
		}
		found = field
	}
	return found, false
}

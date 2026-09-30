package transpiler

import "github.com/NickyBoy89/java2go/symbol"

// Static imports are resolved in the requested namespace. A field named E
// does not import a type E, while one declaration can import both when the
// owner declares a field and member type with the same name.
func staticImportedSourceType(name string, wildcard bool, ctx Ctx) (*symbol.ClassScope, bool) {
	var found *symbol.ClassScope
	for _, imported := range staticMethodImports(ctx) {
		if imported.wildcard != wildcard || !wildcard && imported.member != name {
			continue
		}
		owner := findQualifiedSourceClass(imported.owner)
		if owner == nil {
			continue
		}
		candidate := memberTypeInHierarchy(owner, name, ctx)
		if !staticImportedTypeAccessible(candidate, ctx) {
			continue
		}
		if found != nil && found != candidate {
			reportUnsupported("ambiguous static imported type "+name, nil, ctx.currentFile.Source, ctx)
			return nil, true
		}
		found = candidate
	}
	return found, found != nil
}

func staticImportedTypeAccessible(member *symbol.ClassScope, ctx Ctx) bool {
	if member == nil || member.Class == nil || member.Enclosing == nil || member.IsInner {
		return false
	}
	declaration := member.Class.DeclarationNode
	if declarationHasModifier(declaration, "private") {
		return false
	}
	if member.Enclosing.IsInterface || declarationHasModifier(declaration, "public") {
		return true
	}
	// A protected member is not accessible merely because the importing class
	// happens to inherit it; import declarations are outside the class body.
	pkg, known := lexicalMemberPackage(member)
	return known && ctx.currentFile != nil && pkg == ctx.currentFile.Package
}

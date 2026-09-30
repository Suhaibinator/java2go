package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
)

// On-demand imports belong to the compilation unit, not the class temporarily
// selected while resolving a hierarchy. They remain uncached because lexical
// declarations and explicit imports may choose a different type in each context.
func ordinaryTypeOnDemandImports(ctx Ctx) []string {
	if ctx.currentFile == nil || ctx.currentFile.BaseClass == nil || ctx.currentFile.BaseClass.Class == nil {
		return nil
	}
	root := ctx.currentFile.BaseClass.Class.DeclarationNode
	if root == nil {
		return nil
	}
	for root.Parent() != nil {
		root = root.Parent()
	}
	seen := map[string]bool{}
	var result []string
	for _, node := range nodeutil.NamedChildrenOf(root) {
		if node.Type() != "import_declaration" || node.NamedChildCount() == 0 {
			continue
		}
		static, wildcard := false, false
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			static = static || child.Type() == "static"
			wildcard = wildcard || child.Content(ctx.currentFile.Source) == "*"
		}
		if static || !wildcard {
			continue
		}
		owner := node.NamedChild(0).Content(ctx.currentFile.Source)
		if !seen[owner] {
			seen[owner] = true
			result = append(result, owner)
		}
	}
	return result
}

// A member reached through a public importing owner can be declared in a
// package-private superclass. Check the selected member itself here; the import
// owner's enclosing path is checked separately before hierarchy lookup.
func ordinaryImportedTypeAccessible(scope *symbol.ClassScope, ctx Ctx) bool {
	if scope == nil || scope.Class == nil || ctx.currentFile == nil {
		return false
	}
	if scope.Class.IsPrivate || declarationHasModifier(scope.Class.DeclarationNode, "private") {
		return false
	}
	if scope.Enclosing != nil && scope.Enclosing.IsInterface || declarationHasModifier(scope.Class.DeclarationNode, "public") {
		return true
	}
	pkg, known := lexicalMemberPackage(scope)
	return known && pkg == ctx.currentFile.Package
}

func importedTypeOwnerAccessible(owner *symbol.ClassScope, ctx Ctx) bool {
	for scope := owner; scope != nil; scope = scope.Enclosing {
		if !ordinaryImportedTypeAccessible(scope, ctx) {
			return false
		}
	}
	return owner != nil
}

// This is the compiler's existing nominal JDK catalog, not a classpath loader.
// Keep external candidates in the same demand tier so a source p.String cannot
// silently capture implicit java.lang.String. No intrinsic owner lookup is used
// here: that lookup itself depends on resolving source types.
func knownExternalDemandType(owner, name string) bool {
	qualified := owner + "." + name
	if intrinsicResultDeclarations[name] == qualified {
		return true
	}
	_, known := intrinsicOwners[name][qualified]
	return known
}

// Ordinary and static on-demand imports form one Java ambiguity tier after
// lexical, explicit import and same-package types. A diamond or repeated import
// selecting the same declaration is one candidate, not an ambiguity.
func onDemandImportedSourceType(name string, ctx Ctx) (*symbol.ClassScope, bool) {
	if ctx.currentFile == nil {
		return nil, false
	}
	candidates := map[string]*symbol.ClassScope{}
	addSource := func(scope *symbol.ClassScope) {
		if scope != nil {
			candidates[qualifiedSourceClassName(scope)] = scope
		}
	}
	for _, ownerPath := range ordinaryTypeOnDemandImports(ctx) {
		if owner := findQualifiedSourceClass(ownerPath); owner != nil {
			if importedTypeOwnerAccessible(owner, ctx) {
				// Ordinary type-on-demand imports expose direct declarations.
				// Inheritance is considered separately for static imports below.
				for _, candidate := range owner.Subclasses {
					if candidate.Class != nil && candidate.Class.OriginalName == name && ordinaryImportedTypeAccessible(candidate, ctx) {
						addSource(candidate)
					}
				}
			}
			continue
		}
		if pkg := symbol.GlobalScope.FindPackage(ownerPath); pkg != nil {
			for _, file := range pkg.Files {
				if file == nil {
					continue
				}
				// Package demand imports expose top-level declarations only.
				for _, scope := range file.TopLevelClasses {
					if scope.Class != nil && scope.Class.OriginalName == name && ordinaryImportedTypeAccessible(scope, ctx) {
						addSource(scope)
					}
				}
			}
		}
		if knownExternalDemandType(ownerPath, name) {
			candidates[ownerPath+"."+name] = nil
		}
	}
	for _, imported := range staticMethodImports(ctx) {
		if !imported.wildcard {
			continue
		}
		owner := findQualifiedSourceClass(imported.owner)
		if !importedTypeOwnerAccessible(owner, ctx) {
			continue
		}
		candidate := memberTypeInHierarchy(owner, name, ctx)
		if staticImportedTypeAccessible(candidate, ctx) {
			addSource(candidate)
		}
	}
	if knownExternalDemandType("java.lang", name) {
		candidates["java.lang."+name] = nil
	}
	if len(candidates) > 1 {
		reportUnsupported("ambiguous imported type "+name, nil, ctx.currentFile.Source, ctx)
		return nil, true
	}
	for _, candidate := range candidates {
		return candidate, true
	}
	return nil, false
}

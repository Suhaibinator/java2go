package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Reflection metadata is a declaration use, not an executable invocation.
// Certify its real file/tree and live late registry without borrowing a caller
// frame, while resolving every TYPE name in the actual declaration context.
func reflectionAnnotationNominalOwner(base string, bindings map[string]*symbol.TypeParamDeclaration, ctx Ctx) string {
	file, scope := ctx.currentFile, ctx.currentClass
	if file == nil || scope == nil || scope.Class == nil || file.SourceInputSnapshot() == nil || file.SourceInputSnapshot().Validate() != nil || !threadSourceNodeWithin(scope.Class.DeclarationNode, file.ParsedRoot()) {
		return ""
	}
	if findFileScopeForClassScope(scope, ctx) != file && !intrinsicLexicalDeclaringFile(scope, file, ctx) {
		return ""
	}
	if owner := declaredExternalNominalOwner(base, bindings, ctx); owner != "" {
		return owner
	}
	first := strings.SplitN(base, ".", 2)[0]
	if bindings[first] != nil || visibleTypeParameterDeclarationForJavaType(first, ctx) != nil || resolveClassScopeByQualifiedName(ctx, first) != nil {
		return ""
	}
	// Static TYPE imports remain authoritative. Unknown membership is not absence.
	for _, imported := range staticMethodImports(ctx) {
		if !imported.wildcard && imported.member != first {
			continue
		}
		owner, known := jdk21QualifiedMemberPath(imported.owner)
		if !known {
			return ""
		}
		if _, presence := lookupJDK21Member(owner, first); presence != jdk21NominalAbsentCertified {
			return ""
		}
	}
	eligible := func(owner string) bool {
		if _, known := reflectionJDKAnnotationPolicies[owner]; !known && owner != "java.lang.annotation.RetentionPolicy" && owner != "java.lang.annotation.ElementType" {
			return false
		}
		declaration, presence := jdk21NominalQualified(owner)
		return presence == jdk21NominalPresent && jdk21NominalEligible(declaration) && declaration.binaryName == owner
	}
	if pkg, explicit := file.Imports[first]; explicit {
		owner, known := jdk21QualifiedMemberPath(pkg + "." + base)
		if known && eligible(owner) {
			return owner
		}
		return ""
	}
	if strings.Contains(base, ".") {
		if !jdk21MemberDemandPrefixAbsent(first, ctx) {
			return ""
		}
		owner, known := jdk21QualifiedMemberPath(base)
		if known && eligible(owner) {
			return owner
		}
		return ""
	}
	candidate := ""
	add := func(owner string) bool {
		if !eligible(owner) || candidate != "" && candidate != owner {
			return false
		}
		candidate = owner
		return true
	}
	if declaration, presence := lookupJDK21TopLevel("java.lang", base); presence == jdk21NominalPresent {
		if !add(declaration.binaryName) {
			return ""
		}
	}
	for _, pkg := range ordinaryTypeOnDemandImports(ctx) {
		if pkg == "java.lang" {
			continue
		}
		if declaration, presence := lookupJDK21TopLevel(pkg, base); presence != jdk21NominalUnknown {
			if presence == jdk21NominalPresent && !add(declaration.binaryName) {
				return ""
			}
			continue
		}
		if owner, known := jdk21QualifiedMemberPath(pkg); known {
			declaration, presence := lookupJDK21Member(owner, base)
			if presence == jdk21NominalUnknown || presence == jdk21NominalPresent && !add(declaration.binaryName) {
				return ""
			}
			continue
		}
		if findQualifiedSourceClass(pkg) != nil || symbol.GlobalScope.FindPackage(pkg) != nil {
			continue
		}
		return ""
	}
	return candidate
}

package transpiler

import "strings"

// This metadata profile covers Java 21 ordinary unnamed-module source, java.base,
// unqualified exports and no preview. It certifies names, never runtime APIs.
const jdk21NominalProfile = "jdk21-java.base-unnamed-no-preview"

type jdk21NominalPresence uint8

const (
	jdk21NominalUnknown jdk21NominalPresence = iota
	jdk21NominalPresent
	jdk21NominalAbsentCertified
)

type jdk21NominalDeclaration struct {
	binaryName                              string
	public, isInterface, preview, synthetic bool
}

type jdk21NominalPackage struct {
	certificate  string
	declarations map[string]jdk21NominalDeclaration
}

// A missing declaration is absence evidence only inside a completely scanned,
// exported package. Ineligible declarations remain Present, with their flags.
func lookupJDK21TopLevel(packageName, simpleName string) (jdk21NominalDeclaration, jdk21NominalPresence) {
	pkg, covered := jdk21NominalPackages[packageName]
	if !covered || pkg.certificate == "" || simpleName == "" || strings.ContainsAny(simpleName, ".[/; ") {
		return jdk21NominalDeclaration{}, jdk21NominalUnknown
	}
	declaration, present := pkg.declarations[simpleName]
	if !present {
		return jdk21NominalDeclaration{}, jdk21NominalAbsentCertified
	}
	return declaration, jdk21NominalPresent
}

func jdk21NominalEligible(declaration jdk21NominalDeclaration) bool {
	return declaration.binaryName != "" && declaration.public && !declaration.preview && !declaration.synthetic
}

func jdk21NominalQualified(owner string) (jdk21NominalDeclaration, jdk21NominalPresence) {
	index := strings.LastIndexByte(owner, '.')
	if index < 0 {
		return jdk21NominalDeclaration{}, jdk21NominalUnknown
	}
	packageName := owner[:index]
	return lookupJDK21TopLevel(packageName, owner[index+1:])
}

func jdk21NominalDeclaringUseOwned(ctx Ctx, first string) bool {
	binder := visibleTypeParameterDeclarationForJavaType(first, ctx)
	return (binder == nil || binder.SourceName != first) && ctx.currentFile != nil && ctx.currentClass != nil && findFileScopeForClassScope(ctx.currentClass, ctx) == ctx.currentFile
}

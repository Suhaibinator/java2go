package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
)

// This descriptor-only service proves the actual JDK TYPE declaration. It
// neither registers a default simple-name owner nor admits a physical source
// carrier. The older reflection carrier allowlist is not this service's proof.
func sourceReflectionRecordComponentDescriptor(javaType string, ctx Ctx) (ast.Expr, bool) {
	if !sourceReflectionRecordNamespaceCurrent(ctx.currentFile) {
		return nil, false
	}
	bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
	owner, proved := jdk21MemberUseOwner(javaType, bindings, ctx)
	if !proved || owner != "java.lang.reflect.RecordComponent" {
		return nil, false
	}
	declaration, presence := jdk21NominalQualified(owner)
	if presence != jdk21NominalPresent || !jdk21NominalEligible(declaration) || declaration.isInterface || declaration.binaryName != owner {
		return nil, false
	}
	return stdjavaQualifiedExpr("RecordComponentTypeID", ctx), true
}

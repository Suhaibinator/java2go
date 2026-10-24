package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/stdjava"
)

// Resolve catch identities independently of concrete constructor availability.
// Source declarations and binders occupy the namespace before runtime lookup.
func throwableCatchDescriptor(javaType string, ctx Ctx) (name string, nominal, catchAll bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if throwableReferenceNamesTypeParameter(base, ctx) {
		return base, false, false
	}
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		return sourceClassRuntimeTypeID(scope, ctx), true, false
	}
	resolved := base
	if !strings.Contains(base, ".") && ctx.currentFile != nil {
		if pkg, imported := ctx.currentFile.Imports[base]; imported {
			resolved = pkg + "." + base
		}
	}
	if resolved == "Object" || resolved == "java.lang.Object" {
		return "java.lang.Object", true, true
	}
	if id, _, known := stdjava.RuntimeThrowableDescriptor(resolved); known {
		return string(id), true, id == stdjava.ThrowableTypeID
	}
	// A foreign qualified/imported declaration with a runtime spelling must
	// never borrow that runtime type's parent or become a universal catch.
	if resolved != stripJavaQualifier(resolved) {
		if _, _, known := stdjava.RuntimeThrowableDescriptor(stripJavaQualifier(resolved)); known || stripJavaQualifier(resolved) == "Object" {
			return resolved, true, false
		}
	}
	// Preserve the legacy unavailable-external-class fallback. Resolved source
	// and runtime descriptors above always have a hierarchy-sensitive guard.
	return stripJavaQualifier(base), false, true
}

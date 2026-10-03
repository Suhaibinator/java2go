package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Reflective failures use the runtime's shared reflectionException value, not a
// distinct Go struct and constructor for each Java class. Keep their nominal
// reference edges separate from builtinExceptionTypes, whose membership also
// enables concrete constructor/type emission.
var reflectiveThrowableReferenceParents = map[string]string{
	"java.lang.ReflectiveOperationException":      "java.lang.Exception",
	"java.lang.ClassNotFoundException":            "java.lang.ReflectiveOperationException",
	"java.lang.NoSuchMethodException":             "java.lang.ReflectiveOperationException",
	"java.lang.NoSuchFieldException":              "java.lang.ReflectiveOperationException",
	"java.lang.IllegalAccessException":            "java.lang.ReflectiveOperationException",
	"java.lang.reflect.InvocationTargetException": "java.lang.ReflectiveOperationException",
}

// Most modeled Throwable classes belong to java.lang. These are the canonical
// owners of the remaining entries in the concrete runtime constructor inventory.
var builtinThrowableReferencePackages = map[string]string{
	"DateTimeException":               "java.time",
	"DateTimeParseException":          "java.time.format",
	"ZoneRulesException":              "java.time.zone",
	"ParseException":                  "java.text",
	"ExecutionException":              "java.util.concurrent",
	"TimeoutException":                "java.util.concurrent",
	"CancellationException":           "java.util.concurrent",
	"RejectedExecutionException":      "java.util.concurrent",
	"IOException":                     "java.io",
	"InvalidPathException":            "java.nio.file",
	"UnsupportedEncodingException":    "java.io",
	"NoSuchAlgorithmException":        "java.security",
	"NoSuchElementException":          "java.util",
	"ConcurrentModificationException": "java.util",
}

func builtinThrowableReferenceQualifiedName(name string) (string, bool) {
	if _, ok := builtinExceptionTypes[name]; ok {
		pkg := builtinThrowableReferencePackages[name]
		if pkg == "" {
			pkg = "java.lang"
		}
		return pkg + "." + name, true
	}
	// Only one canonical owner exists for each of these runtime-owned classes.
	for canonical := range reflectiveThrowableReferenceParents {
		if stripJavaQualifier(canonical) == name {
			return canonical, true
		}
	}
	return "", false
}

// Resolve a Java reference name before walking the external graph. Source
// declarations and explicit imports win over implicit/legacy builtin lookup;
// a qualified foreign class never borrows a JDK parent by simple spelling.
func builtinThrowableReferenceName(javaType string, ctx Ctx) (string, bool) {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if throwableReferenceNamesTypeParameter(base, ctx) {
		return "", false
	}
	if resolveClassScopeByQualifiedName(ctx, base) != nil {
		return "", false
	}
	canonical, known := builtinThrowableReferenceQualifiedName(stripJavaQualifier(base))
	if !known {
		return "", false
	}
	if strings.Contains(base, ".") {
		return canonical, base == canonical
	}
	if ctx.currentFile != nil {
		if pkg, imported := ctx.currentFile.Imports[base]; imported {
			return canonical, pkg+"."+base == canonical
		}
	}
	return canonical, true
}

func builtinThrowableReferenceParent(canonical string) (string, bool) {
	if parent, ok := reflectiveThrowableReferenceParents[canonical]; ok {
		return parent, true
	}
	name := stripJavaQualifier(canonical)
	qualified, known := builtinThrowableReferenceQualifiedName(name)
	if !known || qualified != canonical {
		return "", false
	}
	parent, known := builtinExceptionTypes[name]
	if !known {
		return "", false
	}
	if parent == "" {
		return "", true
	}
	return builtinThrowableReferenceQualifiedName(parent)
}

// Type parameters occupy the Java type namespace before library classes. Leave
// their bounds to generic assignability instead of interpreting the spelling as
// a nominal JDK descriptor (on either side of a conversion).
func throwableReferenceNamesTypeParameter(javaType string, ctx Ctx) bool {
	_, found := resolveReferenceTypeParameter(symbol.JavaType{Original: javaType}, ctx)
	return found
}

// A source type parameter may inherit Throwable operations through a declared
// bound. Follow that declaration (including captured outer binders) instead of
// interpreting its source spelling as a library class.
func throwableBoundReferenceAssignable(actual symbol.JavaType, expectedName string, ctx Ctx, visiting map[typeParameterIdentityKey]bool, sourcePath map[*symbol.ClassScope]bool) bool {
	if binding, found := resolveReferenceTypeParameter(actual, ctx); found {
		key := identityKeyForTypeParameter(binding.parameter)
		if visiting[key] {
			return false
		}
		visiting[key] = true
		defer delete(visiting, key)
		for _, bound := range binding.parameter.Bounds {
			if throwableBoundReferenceAssignable(bound, expectedName, binding.context, visiting, sourcePath) {
				return true
			}
		}
		return false
	}
	base, _ := parseJavaTypeString(strings.TrimSpace(actual.Original))
	// A captured dependency that is absent from this declaration's inventory
	// must not be rebound by spelling to an unrelated nominal class.
	if actual.TypeParameterBindings[base] != nil {
		return false
	}
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		if sourcePath[scope] {
			return false
		}
		sourcePath[scope] = true
		defer delete(sourcePath, scope)
		return throwableBoundReferenceAssignable(symbol.JavaType{Original: scope.Superclass}, expectedName, classScopeCtx(scope, ctx), visiting, sourcePath)
	}
	canonical, known := builtinThrowableReferenceName(base, ctx)
	if !known {
		return false
	}
	seen := map[string]bool{}
	for canonical != "" && !seen[canonical] {
		if canonical == expectedName {
			return true
		}
		seen[canonical] = true
		parent, known := builtinThrowableReferenceParent(canonical)
		if !known {
			return false
		}
		canonical = parent
	}
	return false
}

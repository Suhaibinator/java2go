package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// Java's library interfaces and Number participate in the existing erased
// generic member ABI just like source-declared interfaces. The symbol graph
// does not contain these runtime-provided declarations.
func javaTypeHasInterfaceRepresentation(javaType string, ctx Ctx) bool {
	base, _ := parseJavaTypeString(strings.TrimSpace(javaType))
	if scope := resolveClassScopeByQualifiedName(ctx, base); scope != nil {
		return scope.IsInterface
	}
	switch stripJavaQualifier(base) {
	case "Number", "Comparable", "Serializable", "Cloneable", "Constable", "ConstantDesc", "CharSequence":
		return true
	}
	return false
}

// builtinJavaNumericReference recognizes Number and its six standard concrete
// subclasses. Character and Boolean are objects, but are not Numbers.
func builtinJavaNumericReference(javaType string, ctx Ctx) bool {
	base, arguments := parseJavaTypeString(strings.TrimSpace(javaType))
	if len(arguments) != 0 || strings.HasSuffix(base, "[]") {
		return false
	}
	if !strings.Contains(base, ".") {
		visible := visibleTypeParameterDeclarations(ctx)
		if parameter, found := newTypeParameterLookup(visible).resolve(symbol.JavaType{}, base); found {
			erasure := rawTypeParameterErasure(parameter, visible)
			return erasure != base && builtinJavaNumericReference(erasure, ctx)
		}
	}
	if stripJavaQualifier(base) == "Number" && resolveClassScopeByQualifiedName(ctx, base) == nil {
		return true
	}
	primitive, wrapper := builtinJavaWrapperPrimitive(base, ctx)
	return wrapper && primitive != "boolean" && primitive != "char"
}

// builtinJavaReferenceAssignable provides nominal library relationships that
// are absent from the source-class symbol graph. Comparable<T> is invariant.
func builtinJavaReferenceAssignable(actual, expected string, ctx Ctx) bool {
	if _, rank := javaArrayTypeParts(actual); rank != 0 {
		return false
	}
	if _, rank := javaArrayTypeParts(expected); rank != 0 {
		return false
	}
	actualBase, _ := parseJavaTypeString(strings.TrimSpace(actual))
	expectedBase, expectedArguments := parseJavaTypeString(strings.TrimSpace(expected))
	if resolveClassScopeByQualifiedName(ctx, expectedBase) != nil ||
		resolveClassScopeByQualifiedName(ctx, actualBase) != nil {
		return false
	}
	primitive, wrapper := builtinJavaWrapperPrimitive(actualBase, ctx)
	stringObject := stripJavaQualifier(actualBase) == "String"
	if wrapper && javaInferenceSameType(actual, expected, ctx) {
		return true
	}
	switch stripJavaQualifier(expectedBase) {
	case "Object":
		return len(expectedArguments) == 0 && (wrapper || stringObject || builtinJavaNumericReference(actual, ctx))
	case "Number":
		return len(expectedArguments) == 0 && builtinJavaNumericReference(actual, ctx)
	case "Comparable":
		if !wrapper && !stringObject {
			return false
		}
		if len(expectedArguments) == 0 {
			return true
		}
		if len(expectedArguments) != 1 {
			return false
		}
		argument := strings.TrimSpace(expectedArguments[0])
		if argument == "?" {
			return true
		}
		if bound, ok := strings.CutPrefix(argument, "? extends "); ok {
			return javaInferenceTypeAssignable(actual, bound, ctx)
		}
		if bound, ok := strings.CutPrefix(argument, "? super "); ok {
			return javaInferenceTypeAssignable(bound, actual, ctx)
		}
		return javaInferenceSameType(actual, argument, ctx)
	case "Serializable":
		return len(expectedArguments) == 0 && (wrapper || stringObject || builtinJavaNumericReference(actual, ctx))
	case "Constable":
		return len(expectedArguments) == 0 && (wrapper || stringObject)
	case "ConstantDesc":
		return len(expectedArguments) == 0 && (stringObject || wrapper &&
			(primitive == "int" || primitive == "long" || primitive == "float" || primitive == "double"))
	case "CharSequence":
		return len(expectedArguments) == 0 && stringObject
	}
	return false
}

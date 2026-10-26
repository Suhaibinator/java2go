package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

func init() {
	for _, owner := range []string{"java.lang.reflect.AccessibleObject", "java.lang.reflect.Member", "java.lang.annotation.Annotation"} {
		registerIntrinsicOwner(owner, true)
	}
}
func reflectionSurfaceRuntimeTypeExpr(javaType string, arguments []string, ctx Ctx) (ast.Expr, bool) {
	if len(arguments) != 0 {
		return nil, false
	}
	switch stripJavaQualifier(javaType) {
	case "AccessibleObject", "Member", "Annotation":
	default:
		return nil, false
	}
	owner := platformTypeOwner(javaType, nil, ctx, nil)
	if owner == "" {
		return nil, false
	}
	names := map[string]string{"java.lang.reflect.AccessibleObject": "JavaAccessibleObject", "java.lang.reflect.Member": "JavaMember", "java.lang.annotation.Annotation": "JavaAnnotation"}
	name, ok := names[owner]
	if !ok {
		return nil, false
	}
	return stdjavaQualifiedExpr(name, ctx), true
}

func reflectionSurfaceReferenceAssignable(actual, expected string, ctx Ctx) bool {
	switch stripJavaQualifier(strings.TrimSpace(expected)) {
	case "Object":
		switch stripJavaQualifier(strings.TrimSpace(actual)) {
		case "AccessibleObject", "Member", "Annotation", "Field", "Method", "Constructor":
		default:
			return false
		}
	case "AccessibleObject", "Member", "AnnotatedElement", "Annotation":
	default:
		return false
	}
	expectedBase, expectedArguments := parseJavaTypeString(expected)
	if len(expectedArguments) != 0 {
		return false
	}
	expectedOwner := platformTypeOwner(expectedBase, nil, ctx, nil)
	if expectedOwner == "" && stripJavaQualifier(expectedBase) == "Object" {
		bindings := symbol.VisibleTypeParamBindings(visibleTypeParameterDeclarations(ctx))
		if declaredExternalNominalOwner(expectedBase, bindings, ctx) == "java.lang.Object" {
			expectedOwner = "java.lang.Object"
		}
	}
	if expectedOwner == "" {
		return false
	}
	actualBase, _ := parseJavaTypeString(actual)
	first := strings.SplitN(actualBase, ".", 2)[0]
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: first}, ctx); bound {
		return false
	}
	actualOwner := platformTypeOwner(actualBase, nil, ctx, nil)
	if actualOwner != "" && actualOwner == expectedOwner {
		return true
	}
	if expectedOwner == "java.lang.Object" {
		switch actualOwner {
		case "java.lang.reflect.AccessibleObject", "java.lang.reflect.Member", "java.lang.annotation.Annotation", "java.lang.reflect.Field", "java.lang.reflect.Method", "java.lang.reflect.Constructor":
			return true
		}
		return false
	}
	if expectedOwner == "java.lang.annotation.Annotation" {
		scope := resolveClassScopeByQualifiedName(ctx, actualBase)
		if scope == nil || scope.Class == nil || !scope.IsInterface || scope.IsEnum || scope.Class.DeclarationNode == nil || scope.Class.DeclarationNode.Type() != "annotation_type_declaration" {
			return false
		}
		declaring, owned := supplierSourceDeclarationCtx(scope, ctx)
		if !owned {
			return false
		}
		file := declaring.currentFile
		return file != nil && file.SourceInputSnapshot() != nil && file.SourceInputSnapshot().Validate() == nil && threadSourceNodeWithin(scope.Class.DeclarationNode, file.ParsedRoot())
	}
	if expectedOwner != "java.lang.reflect.AccessibleObject" && expectedOwner != "java.lang.reflect.Member" && expectedOwner != "java.lang.reflect.AnnotatedElement" {
		return false
	}
	if actualOwner == "" {
		return false
	}
	for _, member := range []string{"java.lang.reflect.Field", "java.lang.reflect.Method", "java.lang.reflect.Constructor"} {
		if actualOwner == member {
			return true
		}
	}
	return expectedOwner == "java.lang.reflect.AnnotatedElement" && actualOwner == "java.lang.reflect.AccessibleObject"
}

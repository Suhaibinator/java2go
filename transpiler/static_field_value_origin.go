package transpiler

import (
	"go/ast"
	"strconv"
	"strings"

	"github.com/NickyBoy89/java2go/symbol"
)

// staticFieldDeclaredStorageType separates declaration lookup from output
// imports. Failed proofs publish neither type facts nor speculative imports.
func staticFieldDeclaredStorageType(def *symbol.Definition, owner *symbol.ClassScope, output Ctx) (ast.Expr, bool) {
	if def == nil || owner == nil || output.currentFile == nil {
		return nil, false
	}
	file := findFileScopeForClassScope(owner, output)
	if file == nil {
		return nil, false
	}
	declaring := classScopeCtx(owner, output)
	if declaring.currentFile != file {
		return nil, false
	}
	declaring.expectedType, declaring.expectedTypeRoot = "", nil
	declaring.syntheticTypeParameters = nil
	origin := declaredJavaTypeOrigin(definitionDeclaredJavaType(def), declaring)
	emission := output.Clone()
	emission.importAliases = make(map[string]string, len(output.importAliases))
	for key, value := range output.importAliases {
		emission.importAliases[key] = value
	}
	emission.usedImports = make(map[string]bool, len(output.usedImports))
	for key, value := range output.usedImports {
		emission.usedImports[key] = value
	}
	result, ok := staticFieldOriginStorageType(origin, declaring, emission)
	if !ok {
		return nil, false
	}
	for key, value := range emission.importAliases {
		if output.importAliases != nil {
			output.importAliases[key] = value
		}
	}
	for key, value := range emission.usedImports {
		if output.usedImports != nil {
			output.usedImports[key] = value
		}
	}
	return result, true
}

func staticFieldOriginStorageType(origin inferredJavaTypeOrigin, declaring, output Ctx) (ast.Expr, bool) {
	if !completeJavaTypeOrigin(origin) || origin.wildcard || origin.parameter != nil {
		return nil, false
	}
	if origin.arrayRank > 0 {
		if primitive, ok := primitiveArrayTypeExpr(origin.javaType, output); ok {
			return primitive, true
		}
		if _, _, reified := reifiedSourceReferenceArrayComponent(origin.javaType, declaring); reified {
			return reifiedReferenceArrayTypeExpr(output), true
		}
		return nil, false
	}
	if canonical, ok := canonicalResultPhysicalExpr(origin, output); ok {
		return canonical, true
	}
	if scope := origin.nominalScope; scope != nil {
		if scope.Class == nil || scope.Class.Name == "" || len(scope.TypeParameters) != len(origin.arguments) {
			return nil, false
		}
		arguments := make([]ast.Expr, len(origin.arguments))
		for index, arg := range origin.arguments {
			var ok bool
			arguments[index], ok = staticFieldOriginStorageType(arg, declaring, output)
			if !ok {
				return nil, false
			}
		}
		name := scope.Class.Name
		interfaceView := scope.IsInterface || abstractClassUsesInterfaceView(scope)
		if abstractClassUsesInterfaceView(scope) {
			name += "I"
		}
		result := applyTypeArguments(qualifiedNameExpr(name, findJavaPackageForClassScope(scope), output), arguments)
		if interfaceView {
			return result, true
		}
		return &ast.StarExpr{X: result}, true
	}
	base, _ := javaArrayTypeParts(origin.javaType)
	if _, primitive := javaPrimitiveType(base); primitive {
		return javaTypeStringToGoTypeExpr(base, nil, output), true
	}
	if origin.externalName == "" {
		return nil, false
	}
	// Existing native mappings choose the ABI in the certified declaration
	// context. Internal slots stand for already-proved argument ASTs; source
	// type spellings are never reparsed in the consuming namespace.
	mapping := declaring.Clone()
	mapping.importAliases = map[string]string{}
	mapping.usedImports = map[string]bool{}
	slots := map[string]ast.Expr{}
	names := make([]string, len(origin.arguments))
	for index, arg := range origin.arguments {
		projected, ok := staticFieldOriginStorageType(arg, declaring, output)
		if !ok {
			return nil, false
		}
		names[index] = "__java2goStaticStorageArgument" + strconv.Itoa(index)
		slots[names[index]] = projected
	}
	javaType := origin.externalName
	if len(names) > 0 {
		javaType += "<" + strings.Join(names, ",") + ">"
	}
	mapped := javaTypeStringToGoTypeExpr(javaType, names, mapping)
	return staticFieldNativeStorageExpr(mapped, mapping.importAliases[stdjavaImportPath], slots, output)
}

// Re-emit only a fresh native type AST using the caller's import aliases.
// Unknown source identifiers/selectors are not proof of native storage.
func staticFieldNativeStorageExpr(expr ast.Expr, runtimeAlias string, slots map[string]ast.Expr, output Ctx) (ast.Expr, bool) {
	switch value := expr.(type) {
	case *ast.Ident:
		if slot, ok := slots[value.Name]; ok {
			return slot, true
		}
		switch value.Name {
		case "any", "bool", "int8", "int16", "int32", "int64", "float32", "float64", "rune", "string", "byte":
			return ast.NewIdent(value.Name), true
		}
	case *ast.SelectorExpr:
		if qualifier, ok := value.X.(*ast.Ident); ok && runtimeAlias != "" && qualifier.Name == runtimeAlias {
			return stdjavaQualifiedExpr(value.Sel.Name, output), true
		}
	case *ast.StarExpr:
		if inner, ok := staticFieldNativeStorageExpr(value.X, runtimeAlias, slots, output); ok {
			return &ast.StarExpr{X: inner}, true
		}
	case *ast.IndexExpr:
		inner, ok := staticFieldNativeStorageExpr(value.X, runtimeAlias, slots, output)
		if !ok {
			return nil, false
		}
		index, ok := staticFieldNativeStorageExpr(value.Index, runtimeAlias, slots, output)
		if ok {
			return &ast.IndexExpr{X: inner, Index: index}, true
		}
	case *ast.IndexListExpr:
		inner, ok := staticFieldNativeStorageExpr(value.X, runtimeAlias, slots, output)
		if !ok {
			return nil, false
		}
		arguments := make([]ast.Expr, len(value.Indices))
		for index, arg := range value.Indices {
			arguments[index], ok = staticFieldNativeStorageExpr(arg, runtimeAlias, slots, output)
			if !ok {
				return nil, false
			}
		}
		return &ast.IndexListExpr{X: inner, Indices: arguments}, true
	}
	return nil, false
}

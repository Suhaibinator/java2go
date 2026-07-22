package transpiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unicode"

	"github.com/NickyBoy89/java2go/symbol"
)

var tokens = map[string]token.Token{
	"+": token.ADD,
	"-": token.SUB,
	"*": token.MUL,
	"/": token.QUO,
	"%": token.REM,

	"&": token.AND,
	"|": token.OR,
	"^": token.XOR,
	// Java bitwise complement (~)
	"~":  token.XOR,
	"<<": token.SHL,
	">>": token.SHR,
	"&^": token.AND_NOT,

	"+=": token.ADD_ASSIGN,
	"-=": token.SUB_ASSIGN,
	"*=": token.MUL_ASSIGN,
	"/=": token.QUO_ASSIGN,
	"%=": token.REM_ASSIGN,

	"&=":  token.AND_ASSIGN,
	"|=":  token.OR_ASSIGN,
	"^=":  token.XOR_ASSIGN,
	"<<=": token.SHL_ASSIGN,
	">>=": token.SHR_ASSIGN,
	"&^=": token.AND_NOT_ASSIGN,

	"&&": token.LAND,
	"||": token.LOR,
	"++": token.INC,
	"--": token.DEC,

	"==": token.EQL,
	"<":  token.LSS,
	">":  token.GTR,
	"=":  token.ASSIGN,
	"!":  token.NOT,

	"!=":  token.NEQ,
	"<=":  token.LEQ,
	">=":  token.GEQ,
	":=":  token.DEFINE,
	"...": token.ELLIPSIS,
}

// Maps a token's representation to its token, e.g. "+" -> token.ADD
func StrToToken(input string) token.Token {
	if outToken, known := tokens[input]; known {
		return outToken
	}
	panic(fmt.Errorf("unknown token for [%v]", input))
}

// ShortName returns the short-name representation of a class's name for use
// in methods and construtors
// Ex: Test -> ts
func ShortName(longName string) string {
	if len(longName) == 0 {
		return ""
	}
	return string(unicode.ToLower(rune(longName[0]))) + string(unicode.ToLower(rune(longName[len(longName)-1])))
}

// GenStruct is a utility method for generating the ast representation of
// a struct, given its name and fields
func GenStruct(structName string, structFields *ast.FieldList) ast.Decl {
	return GenStructWithTypeParams(structName, structFields, nil)
}

// GenStructWithTypeParams generates a struct with optional type parameters.
// Each type parameter can optionally include bounds, which are converted into
// Go constraints. Missing bounds default to the "any" constraint.
func GenStructWithTypeParams(structName string, structFields *ast.FieldList, typeParams []symbol.TypeParam) ast.Decl {
	return genStructWithTypeParamsInContext(structName, structFields, typeParams, Ctx{})
}

// genStructWithTypeParamsInContext is the symbol-aware form used by the
// transpiler. The context is required to distinguish an interface upper bound
// (T extends Ranked -> T Ranked) from a class upper bound (T extends Base ->
// T *Base), and to qualify bounds declared in another generated package.
func genStructWithTypeParamsInContext(structName string, structFields *ast.FieldList, typeParams []symbol.TypeParam, ctx Ctx) ast.Decl {
	typeSpec := &ast.TypeSpec{
		Name: &ast.Ident{
			Name: structName,
		},
		Type: &ast.StructType{
			Fields: structFields,
		},
	}

	// Add type parameters if present
	if len(typeParams) > 0 {
		typeSpec.TypeParams = &ast.FieldList{List: makeTypeParamFieldsInContext(typeParams, ctx)}
	}

	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{typeSpec},
	}
}

// GenFuncDeclWithTypeParams creates a function declaration with type parameters.
// This is used for constructors and static methods of generic classes.
func GenFuncDeclWithTypeParams(name string, typeParams []symbol.TypeParam, params, results *ast.FieldList, body *ast.BlockStmt) *ast.FuncDecl {
	return genFuncDeclWithTypeParamsInContext(name, typeParams, params, results, body, Ctx{})
}

func genFuncDeclWithTypeParamsInContext(name string, typeParams []symbol.TypeParam, params, results *ast.FieldList, body *ast.BlockStmt, ctx Ctx) *ast.FuncDecl {
	funcDecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: name},
		Type: &ast.FuncType{
			Params:  params,
			Results: results,
		},
		Body: body,
	}

	// Add type parameters if present
	if len(typeParams) > 0 {
		funcDecl.Type.TypeParams = &ast.FieldList{List: makeTypeParamFieldsInContext(typeParams, ctx)}
	}

	return funcDecl
}

func makeTypeParamFieldsInContext(typeParams []symbol.TypeParam, ctx Ctx) []*ast.Field {
	if len(typeParams) == 0 {
		return nil
	}

	paramNames := symbol.GoTypeParamNames(typeParams)
	parameterLookup := newTypeParameterLookup(typeParams)
	fields := make([]*ast.Field, len(typeParams))
	for i, tp := range typeParams {
		constraint := constraintExprInContext(
			goRepresentableTypeParameterBounds(tp.Bounds, parameterLookup, nil),
			paramNames,
			ctx,
		)
		if ctx.dependentTypeWitnesses != nil && ctx.dependentTypeWitnesses.hasSource(tp.Declaration) {
			// A concrete Java upper bound admits every generated subclass pointer,
			// while Go's *Base constraint admits only the exact Base subobject. The
			// hidden projection witness carries that Java proof instead.
			constraint = &ast.Ident{Name: "any"}
		}
		fields[i] = &ast.Field{
			Names: []*ast.Ident{{Name: tp.EmittedName()}},
			Type:  constraint,
		}
	}
	return fields
}

// goRepresentableTypeParameterBounds replaces a direct type-parameter bound
// with that parameter's own upper bounds. Java permits `T extends B` where B is
// another type parameter; Go intentionally rejects using a type parameter as a
// constraint. Following the bound chain to its concrete/interface erasure is
// the closest representable constraint and preserves the method set available
// through T (for example B extends Root, T extends B becomes T Root).
//
// Bounds whose base is a real type remain intact, including parameterized
// bounds such as Comparable<T>. Intersection bounds are flattened in source
// order and deduplicated.
func goRepresentableTypeParameterBounds(
	bounds []symbol.JavaType,
	parameters typeParameterLookup,
	visiting map[typeParameterIdentityKey]bool,
) []symbol.JavaType {
	if len(bounds) == 0 {
		return nil
	}
	if visiting == nil {
		visiting = make(map[typeParameterIdentityKey]bool, len(parameters.byName))
	}

	var result []symbol.JavaType
	seen := make(map[string]struct{})
	appendUnique := func(bound symbol.JavaType) {
		original := strings.TrimSpace(bound.Original)
		if original == "" {
			return
		}
		resolved := substituteTypeParameterDeclarations(original, bound.TypeParameterBindings)
		if _, duplicate := seen[resolved]; duplicate {
			return
		}
		seen[resolved] = struct{}{}
		bound.Original = original
		result = append(result, bound)
	}

	for _, bound := range bounds {
		original := strings.TrimSpace(bound.Original)
		base, arguments := parseJavaTypeString(original)
		dependencyName := strings.TrimSpace(base)
		dependency, dependent := parameters.resolve(bound, dependencyName)
		if !dependent || len(arguments) != 0 {
			appendUnique(bound)
			continue
		}
		identity := identityKeyForTypeParameter(dependency)
		if visiting[identity] {
			// A direct cycle is not a useful Go constraint. Java's legal recursive
			// form is normally parameterized (T extends Comparable<T>) and takes the
			// non-dependent branch above.
			continue
		}
		visiting[identity] = true
		for _, inherited := range goRepresentableTypeParameterBounds(dependency.Bounds, parameters, visiting) {
			appendUnique(inherited)
		}
		delete(visiting, identity)
	}
	return result
}

func constraintExprInContext(bounds []symbol.JavaType, typeParams []string, ctx Ctx) ast.Expr {
	if len(bounds) == 0 {
		return &ast.Ident{Name: "any"}
	}

	if len(bounds) == 1 {
		javaType := substituteTypeParameterDeclarations(bounds[0].Original, bounds[0].TypeParameterBindings)
		return javaTypeStringToGoTypeExpr(javaType, typeParams, ctx)
	}

	fields := make([]*ast.Field, len(bounds))
	for i, b := range bounds {
		javaType := substituteTypeParameterDeclarations(b.Original, b.TypeParameterBindings)
		fields[i] = &ast.Field{Type: javaTypeStringToGoTypeExpr(javaType, typeParams, ctx)}
	}

	return &ast.InterfaceType{Methods: &ast.FieldList{List: fields}}
}

func genType(remaining []string) ast.Expr {
	if len(remaining) == 1 {
		return &ast.UnaryExpr{
			Op: token.TILDE,
			X:  &ast.Ident{Name: remaining[0]},
		}
	}
	return &ast.BinaryExpr{
		X:  genType(remaining[1:]),
		Op: token.OR,
		Y:  genType(remaining[:1]),
	}
}

func GenTypeInterface(name string, types []string) ast.Decl {
	return &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: name},
				Type: &ast.InterfaceType{
					Methods: &ast.FieldList{
						List: []*ast.Field{
							&ast.Field{
								Type: genType(types),
							},
						},
					},
				},
			},
		},
	}
}

func GenInterface(name string, methods *ast.FieldList, typeParams []symbol.TypeParam) ast.Decl {
	return genInterfaceInContext(name, methods, typeParams, Ctx{})
}

func genInterfaceInContext(name string, methods *ast.FieldList, typeParams []symbol.TypeParam, ctx Ctx) ast.Decl {
	typeSpec := &ast.TypeSpec{
		Name: &ast.Ident{Name: name},
		Type: &ast.InterfaceType{
			Methods: methods,
		},
	}
	if len(typeParams) > 0 {
		typeSpec.TypeParams = &ast.FieldList{List: makeTypeParamFieldsInContext(typeParams, ctx)}
	}

	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{typeSpec},
	}
}

func GenMultiDimArray(elementType ast.Expr, dimensions []ast.Expr, totalRank int, ctx Ctx) ast.Expr {
	if totalRank < len(dimensions) {
		totalRank = len(dimensions)
	}
	if len(dimensions) == 1 {
		// NewArray preserves Java object identity even when the requested length
		// is zero. Its generic length parameter also accepts byte/short/char before
		// unary numeric promotion and raises NegativeArraySizeException directly.
		return javaArrayAllocation(genArrayType(elementType, totalRank), dimensions[0], ctx)
	}

	// Java evaluates every explicit dimension expression exactly once, from
	// left to right, before it checks any result for negativity or allocates any
	// part of the array. Passing the expressions as arguments to the generated
	// function literal gives them that ordering without letting the synthetic
	// names shadow identifiers used by later expressions.
	usedNames := arrayAllocationReservedNames(elementType, ctx)
	dimensionNames := make([]string, len(dimensions))
	dimensionArgs := make([]ast.Expr, len(dimensions))
	callArgs := make([]ast.Expr, len(dimensions))
	for i := range dimensions {
		dimensionNames[i] = uniqueArrayAllocationName("__java2goDimension"+strconv.Itoa(i), usedNames)
		dimensionArgs[i] = &ast.Ident{Name: dimensionNames[i]}
		// JLS 15.10.1 applies unary numeric promotion to each dimension. An
		// explicit conversion is needed for byte, short, and char expressions,
		// because Go does not implicitly widen those values to int32 at a call.
		callArgs[i] = &ast.CallExpr{
			Fun:  &ast.Ident{Name: "int32"},
			Args: []ast.Expr{dimensions[i]},
		}
	}
	arrayName := uniqueArrayAllocationName("__java2goArray", usedNames)
	indexNames := make([]string, len(dimensions)-1)
	for i := range indexNames {
		indexNames[i] = uniqueArrayAllocationName("__java2goIndex"+strconv.Itoa(i), usedNames)
	}

	allocationStatements := make([]ast.Stmt, 0, len(dimensions)+3)
	for _, dimension := range dimensionArgs {
		allocationStatements = append(allocationStatements, &ast.IfStmt{
			Cond: &ast.BinaryExpr{
				X:  dimension,
				Op: token.LSS,
				Y:  &ast.BasicLit{Kind: token.INT, Value: "0"},
			},
			Body: &ast.BlockStmt{List: []ast.Stmt{
				&ast.ExprStmt{X: &ast.CallExpr{
					Fun: &ast.Ident{Name: "panic"},
					// JLS specifies the exception type and timing, but not its
					// implementation-dependent detail message. Keep the synthetic
					// multidimensional path deterministic with an empty detail.
					Args: []ast.Expr{stdjavaCall(ctx, "NewNegativeArraySizeException",
						&ast.BasicLit{Kind: token.STRING, Value: `""`})},
				}},
			}},
		})
	}

	// arr := stdjava.NewArray[[][]int](2)
	base := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{&ast.Ident{Name: arrayName}},
		Rhs: []ast.Expr{
			javaArrayAllocation(genArrayType(elementType, totalRank), dimensionArgs[0], ctx),
		},
	}

	var body, currentDimension *ast.RangeStmt

	for offset := range dimensions[1:] {
		nextDim := &ast.RangeStmt{
			Key: &ast.Ident{Name: indexNames[offset]},
			Tok: token.DEFINE,
			X:   multiArrayAccess(arrayName, indexNames[:offset]),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Tok: token.ASSIGN,
						Lhs: []ast.Expr{multiArrayAccess(arrayName, indexNames[:offset+1])},
						Rhs: []ast.Expr{javaArrayAllocation(genArrayType(elementType, totalRank-(offset+1)), dimensionArgs[offset+1], ctx)},
					},
				},
			},
		}

		if body == nil {
			body = nextDim
			currentDimension = body
		} else {
			currentDimension.Body.List = append(currentDimension.Body.List, nextDim)
			currentDimension = currentDimension.Body.List[len(currentDimension.Body.List)-1].(*ast.RangeStmt)
		}
	}
	allocationStatements = append(allocationStatements, base)
	if body != nil {
		allocationStatements = append(allocationStatements, body)
	}
	allocationStatements = append(allocationStatements,
		&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: arrayName}}})

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Params: &ast.FieldList{List: []*ast.Field{{
					Names: func() []*ast.Ident {
						names := make([]*ast.Ident, len(dimensionNames))
						for i, name := range dimensionNames {
							names[i] = &ast.Ident{Name: name}
						}
						return names
					}(),
					Type: &ast.Ident{Name: "int32"},
				}}},
				Results: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Type: genArrayType(elementType, totalRank),
						},
					},
				},
			},
			Body: &ast.BlockStmt{List: allocationStatements},
		},
		Args: callArgs,
	}
}

func multiArrayAccess(arrName string, dims []string) ast.Expr {
	var arr ast.Expr = &ast.Ident{Name: arrName}
	for _, dim := range dims {
		arr = &ast.IndexExpr{X: arr, Index: &ast.Ident{Name: dim}}
	}
	return arr
}

func genArrayType(elementType ast.Expr, depth int) ast.Expr {
	arrayDims := elementType
	for i := 0; i < depth; i++ {
		arrayDims = &ast.ArrayType{Elt: arrayDims}
	}
	return arrayDims
}

func arrayAllocationReservedNames(elementType ast.Expr, ctx Ctx) map[string]struct{} {
	reserved := map[string]struct{}{
		"int32":   {},
		"make":    {},
		"panic":   {},
		"stdjava": {},
	}
	ast.Inspect(elementType, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && identifier.Name != "" {
			reserved[identifier.Name] = struct{}{}
		}
		return true
	})
	for _, typeParameter := range inScopeTypeParameters(ctx) {
		reserved[typeParameter] = struct{}{}
		reserved[sanitizeGoIdent(typeParameter)] = struct{}{}
	}
	for _, alias := range ctx.importAliases {
		if alias != "" {
			reserved[alias] = struct{}{}
		}
	}
	if ctx.currentFile != nil {
		for _, javaPackage := range ctx.currentFile.Imports {
			if alias := packageAliasFromJavaPackage(javaPackage); alias != "" {
				reserved[alias] = struct{}{}
			}
		}
	}
	return reserved
}

func uniqueArrayAllocationName(base string, used map[string]struct{}) string {
	for suffix := 0; ; suffix++ {
		candidate := base
		if suffix > 0 {
			candidate += "_" + strconv.Itoa(suffix)
		}
		if _, collision := used[candidate]; collision {
			continue
		}
		used[candidate] = struct{}{}
		return candidate
	}
}

// javaArrayAllocation constructs a Java-identity-preserving slice allocation.
// arrayType is the complete slice type; NewArray's type argument is its element
// type so `new int[4][][]` becomes NewArray[[][]int32](4).
func javaArrayAllocation(arrayType ast.Expr, length ast.Expr, ctx Ctx) *ast.CallExpr {
	typedArray, ok := arrayType.(*ast.ArrayType)
	if !ok {
		panic("Java array allocation requires an array type")
	}
	return stdjavaGenericCall(ctx, "NewArray", []ast.Expr{typedArray.Elt}, []ast.Expr{length})
}

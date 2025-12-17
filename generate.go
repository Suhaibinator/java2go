package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
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
		typeSpec.TypeParams = &ast.FieldList{List: makeTypeParamFields(typeParams)}
	}

	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{typeSpec},
	}
}

// GenFuncDeclWithTypeParams creates a function declaration with type parameters.
// This is used for constructors and static methods of generic classes.
func GenFuncDeclWithTypeParams(name string, typeParams []symbol.TypeParam, params, results *ast.FieldList, body *ast.BlockStmt) *ast.FuncDecl {
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
		funcDecl.Type.TypeParams = &ast.FieldList{List: makeTypeParamFields(typeParams)}
	}

	return funcDecl
}

func makeTypeParamFields(typeParams []symbol.TypeParam) []*ast.Field {
	if len(typeParams) == 0 {
		return nil
	}

	paramNames := symbol.TypeParamNames(typeParams)
	fields := make([]*ast.Field, len(typeParams))
	for i, tp := range typeParams {
		fields[i] = &ast.Field{
			Names: []*ast.Ident{{Name: tp.Name}},
			Type:  constraintExpr(tp.Bounds, paramNames),
		}
	}
	return fields
}

func constraintExpr(bounds []symbol.JavaType, typeParams []string) ast.Expr {
	if len(bounds) == 0 {
		return &ast.Ident{Name: "any"}
	}

	if len(bounds) == 1 {
		return javaTypeStringToGoTypeExpr(bounds[0].Original, typeParams)
	}

	fields := make([]*ast.Field, len(bounds))
	for i, b := range bounds {
		fields[i] = &ast.Field{Type: javaTypeStringToGoTypeExpr(b.Original, typeParams)}
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
	typeSpec := &ast.TypeSpec{
		Name: &ast.Ident{Name: name},
		Type: &ast.InterfaceType{
			Methods: methods,
		},
	}
	if len(typeParams) > 0 {
		typeSpec.TypeParams = &ast.FieldList{List: makeTypeParamFields(typeParams)}
	}

	return &ast.GenDecl{
		Tok:   token.TYPE,
		Specs: []ast.Spec{typeSpec},
	}
}

func GenMultiDimArray(arrayType string, dimensions []ast.Expr) ast.Expr {
	if len(dimensions) == 1 {
		return &ast.CallExpr{
			Fun:  &ast.Ident{Name: "make"},
			Args: append([]ast.Expr{&ast.Ident{Name: arrayType}}, dimensions...),
		}
	}

	// arr := make([][][]int, 2)
	base := &ast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []ast.Expr{&ast.Ident{Name: "arr"}},
		Rhs: []ast.Expr{
			makeExpression(genArrayType(arrayType, len(dimensions)), dimensions[0]),
		},
	}

	indexes := []string{"ind"}

	var body, currentDimension *ast.RangeStmt

	for offset := range dimensions[1:] {
		nextDim := &ast.RangeStmt{
			Key: &ast.Ident{Name: indexes[len(indexes)-1]},
			Tok: token.DEFINE,
			X:   multiArrayAccess("arr", indexes[:len(indexes)-1]),
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					&ast.AssignStmt{
						Tok: token.ASSIGN,
						Lhs: []ast.Expr{multiArrayAccess("arr", indexes)},
						Rhs: []ast.Expr{makeExpression(genArrayType(arrayType, len(dimensions)-(offset+1)), dimensions[offset+1])},
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

		indexes = append(indexes, indexes[len(indexes)-1]+strconv.Itoa(offset))
	}

	return &ast.CallExpr{
		Fun: &ast.FuncLit{
			Type: &ast.FuncType{
				Results: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Type: genArrayType(arrayType, len(dimensions)),
						},
					},
				},
			},
			Body: &ast.BlockStmt{
				List: []ast.Stmt{
					base,
					body,
					&ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: "arr"}}},
				},
			},
		},
	}
}

func multiArrayAccess(arrName string, dims []string) ast.Expr {
	var arr ast.Expr = &ast.Ident{Name: arrName}
	for _, dim := range dims {
		arr = &ast.IndexExpr{X: arr, Index: &ast.Ident{Name: dim}}
	}
	return arr
}

func genArrayType(arrayType string, depth int) ast.Expr {
	var arrayDims ast.Expr = &ast.Ident{Name: arrayType}
	for i := 0; i < depth; i++ {
		arrayDims = &ast.ArrayType{Elt: arrayDims}
	}
	return arrayDims
}

// makeExpression constructs an array with the `make` keyword
func makeExpression(dims, expr ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.Ident{Name: "make"},
		Args: []ast.Expr{
			dims,
			expr,
		},
	}
}

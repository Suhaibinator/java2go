package transpiler

import (
 "go/ast"
 "go/token"
)

// A standalone Java main still uses Java's nominal String[] representation.
// This is the same host boundary as the project launcher: exclude the native
// executable name and retain each converted String reference in the array.
func legacyMainArgumentsExpr(ctx Ctx) ast.Expr {
 host := ast.NewIdent("java2goHostArguments")
 array := ast.NewIdent("java2goArguments")
 index := ast.NewIdent("java2goArgumentIndex")
 value := ast.NewIdent("java2goArgumentValue")
 return &ast.CallExpr{
  Fun: &ast.FuncLit{
   Type: &ast.FuncType{
    Params: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{host}, Type: &ast.ArrayType{Elt: ast.NewIdent("string")}}}},
    Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: stdjavaQualifiedExpr("ReferenceArray",ctx)}}}},
   },
   Body: &ast.BlockStmt{List: []ast.Stmt{
    &ast.AssignStmt{Lhs: []ast.Expr{array}, Tok: token.DEFINE, Rhs: []ast.Expr{stdjavaGenericCall(ctx,"NewReferenceArrayOf",[]ast.Expr{javaStringReferenceType(ctx)},[]ast.Expr{callIdent("len",host),stdjavaQualifiedExpr("StringTypeID",ctx)})}},
    &ast.RangeStmt{Key:index,Value:value,Tok:token.DEFINE,X:host,Body:&ast.BlockStmt{List:[]ast.Stmt{&ast.ExprStmt{X:stdjavaCall(ctx,"ReferenceArraySet",array,index,stdjavaCall(ctx,"JavaStringFromHostUTF8",value))}}}},
    &ast.ReturnStmt{Results:[]ast.Expr{array}},
   }},
  },
  Args: []ast.Expr{&ast.SliceExpr{X:qualifiedNameExpr("Args","os",ctx),Low:&ast.BasicLit{Kind:token.INT,Value:"1"}}},
 }
}

package transpiler

import (
 "go/ast"
 "go/token"
 "strings"
 "strconv"

 "github.com/NickyBoy89/java2go/symbol"
)

// A Java process invokes a generic main using its erased signature. Keep the
// source callable method, including its witness parameters, and emit a distinct
// process boundary in the declaring package so private source bound types are
// resolved there rather than in a host launcher.
func genericMainProcessBoundary(def *symbol.Definition, ctx Ctx) bool {
 return projectMain(def) && len(def.TypeParameters)>0 &&
  isBuiltinJavaString(strings.TrimSuffix(definitionJavaType(def.Parameters[0]),"[]"),ctx)
}

func genericMainProcessEntryName(def *symbol.Definition, owner *symbol.ClassScope) string {
 return collisionSafeExecutionIdentifier("Java2goProcessEntry"+symbol.GoIdentifier(def.Name),owner)
}

// Interpret a bound in its captured method declaration context. Following a
// dependent binder uses declaration identity. A concrete bound uses its Java
// raw erasure; the ordinary type producer supplies a generated raw class's own
// erased arguments when Go requires them.
func genericMainErasedJavaTypes(def *symbol.Definition, ctx Ctx) []string {
 lookup:=newTypeParameterLookup(def.TypeParameters)
 visiting:=map[typeParameterIdentityKey]bool{}
 var erase func(symbol.TypeParam) string
 erase=func(parameter symbol.TypeParam) string {
  key:=identityKeyForTypeParameter(parameter)
  if visiting[key] {panic("cyclic generic main bound")}
  if len(parameter.Bounds)==0{return "java.lang.Object"}
  visiting[key]=true;defer delete(visiting,key)
  bound:=parameter.Bounds[0]
  base,args:=parseJavaTypeString(strings.TrimSpace(bound.Original))
  if len(args)==0 {if next,ok:=lookup.resolve(bound,base);ok{return erase(next)}}
  bound.Original=base
  return qualifyDeclaredReferenceType(bound,ctx)
 }
 result:=make([]string,len(def.TypeParameters))
 for i,parameter:=range def.TypeParameters {result[i]=erase(parameter)}
 return result
}

func genericMainProcessEntryDecl(def *symbol.Definition, ctx Ctx) *ast.FuncDecl {
 javaTypes:=genericMainErasedJavaTypes(def,ctx)
 goTypes:=make([]ast.Expr,len(javaTypes))
 byDeclaration:=map[*symbol.TypeParamDeclaration]string{}
 for i,parameter:=range def.TypeParameters {
  goTypes[i]=javaTypeStringToGoTypeExpr(javaTypes[i],inScopeTypeParameters(ctx),ctx)
  byDeclaration[parameter.Declaration]=javaTypes[i]
 }
 body:=[]ast.Stmt{}
 arguments:=[]ast.Expr{}
 for i,edge:=range concreteDependentTypeWitnessEdges(def,ctx) {
  sourceType:=byDeclaration[edge.source.Declaration]
  targetType:=qualifyDeclaredReferenceType(symbol.JavaType{Original:edge.targetJavaType},ctx)
  if edge.targetParameter!=nil {targetType=byDeclaration[edge.targetParameter]}
  projection:=dependentTypeProjectionWitnessExpr(sourceType,targetType,ctx)
  if projection==nil {panic("unresolved erased generic main projection")}
  name:=ast.NewIdent("__java2goProcessWitness"+strconv.Itoa(i+1))
  body=append(body,&ast.AssignStmt{Lhs:[]ast.Expr{name},Tok:token.DEFINE,Rhs:[]ast.Expr{projection}})
  arguments=append(arguments,name)
 }
 arguments=append(arguments,legacyMainArgumentsExpr(ctx))
 body=append(body,&ast.ExprStmt{X:&ast.CallExpr{Fun:&ast.IndexListExpr{X:ast.NewIdent(symbol.GoIdentifier(def.Name)),Indices:goTypes},Args:arguments}})
 return &ast.FuncDecl{Name:ast.NewIdent(genericMainProcessEntryName(def,ctx.currentClass)),Type:&ast.FuncType{Params:&ast.FieldList{}},Body:&ast.BlockStmt{List:body}}
}

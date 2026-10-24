package transpiler

import (
 "go/ast"
 "go/parser"
 "go/token"
 "os"
 "strings"
 "testing"
)

// Hidden dependent-binder witnesses must not be mistaken for the source main
// argument. Full generic entry execution remains a distinct JVM parity gate.
func TestLegacyMainGenericSourceArgumentBinding(t *testing.T) {
 source,err:=os.ReadFile("testdata/generic_legacy_main_witness/Main.java");if err!=nil{t.Fatal(err)}
 generated:=renderGoFileFromJava(t,string(source));file,err:=parser.ParseFile(token.NewFileSet(),"generated.go",generated,0);if err!=nil{t.Fatal(err)}
 actualBindings,hiddenBindings,canonicalGetters,dependentFunctions,actualParameters,witnessParameters,processBoundaries:=0,0,0,0,0,0,0
 ast.Inspect(file,func(n ast.Node)bool{
  if fn,ok:=n.(*ast.FuncDecl);ok {
   if fn.Type.TypeParams!=nil&&fn.Type.TypeParams.NumFields()==2{dependentFunctions++}
   if fn.Type.Params!=nil{for _,field:=range fn.Type.Params.List{for _,name:=range field.Names{if name.Name=="actual"{actualParameters++};if strings.HasPrefix(name.Name,"__java2goDependentWitness"){witnessParameters++}}}}
   if strings.HasPrefix(fn.Name.Name,"Java2goProcessEntry"){if fn.Type.TypeParams!=nil||fn.Type.Params.NumFields()!=0{t.Fatal("generic process wrapper signature is not erased")};processBoundaries++}
  }
  if assign,ok:=n.(*ast.AssignStmt);ok&&assign.Tok==token.DEFINE{for i,lhs:=range assign.Lhs{ident,ok:=lhs.(*ast.Ident);if !ok||i>=len(assign.Rhs){continue};call,ok:=assign.Rhs[i].(*ast.CallExpr);if !ok{continue};if _,ok:=call.Fun.(*ast.FuncLit);!ok||len(call.Args)!=1{continue};slice,ok:=call.Args[0].(*ast.SliceExpr);if !ok{continue};sel,ok:=slice.X.(*ast.SelectorExpr);if !ok{continue};pkg,ok:=sel.X.(*ast.Ident);if !ok||pkg.Name!="os"||sel.Sel.Name!="Args"{continue};if ident.Name=="actual"{actualBindings++};if strings.HasPrefix(ident.Name,"__java2goDependentWitness"){hiddenBindings++}}}
  if call,ok:=n.(*ast.CallExpr);ok{if indexed,ok:=call.Fun.(*ast.IndexExpr);ok{if sel,ok:=indexed.X.(*ast.SelectorExpr);ok&&sel.Sel.Name=="ReferenceArrayGet"&&len(call.Args)>0{if a,ok:=call.Args[0].(*ast.Ident);ok&&a.Name=="actual"{canonicalGetters++}}}}
  return true
 })
 if dependentFunctions==0{t.Fatal("generic witness no longer exercises dependent type parameters")}
 if canonicalGetters<1{t.Fatal("generic witness no longer reads source argument array")}
 if actualBindings!=0||hiddenBindings!=0||actualParameters!=2||witnessParameters!=4||processBoundaries!=1{t.Fatalf("LEGACY_MAIN_SOURCE_ARGUMENT_BINDING: source callable/body args were rewritten or witnesses dropped: hostbindings=%d/%d actualparameters=%d witnesses=%d processboundaries=%d want0/0/2/4/1",actualBindings,hiddenBindings,actualParameters,witnessParameters,processBoundaries)}
}

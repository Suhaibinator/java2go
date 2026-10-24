package transpiler

import (
 "go/ast"
 "go/parser"
 "go/token"
 "os"
 "testing"
)

// A smaller producer regression uses the same independently observed source.
// Canonical String[] reads cannot consume the unconverted host []string slice.
func TestLegacyMainArgsCanonicalArrayWitness(t *testing.T){
 source,err:=os.ReadFile("testdata/enclosing_qualifier_witness/Main.java");if err!=nil{t.Fatal(err)}
 generated:=renderGoFileFromJava(t,string(source));file,err:=parser.ParseFile(token.NewFileSet(),"generated.go",generated,0);if err!=nil{t.Fatal(err)}
 hostBindings,getters:=0,0
 ast.Inspect(file,func(n ast.Node)bool{
  if assign,ok:=n.(*ast.AssignStmt);ok{for i,lhs:=range assign.Lhs{ident,ok:=lhs.(*ast.Ident);if !ok||ident.Name!="args"||i>=len(assign.Rhs){continue};rhs:=assign.Rhs[i];if slice,ok:=rhs.(*ast.SliceExpr);ok{rhs=slice.X};if sel,ok:=rhs.(*ast.SelectorExpr);ok{if pkg,ok:=sel.X.(*ast.Ident);ok&&pkg.Name=="os"&&sel.Sel.Name=="Args"{hostBindings++}}}}
  if call,ok:=n.(*ast.CallExpr);ok{if indexed,ok:=call.Fun.(*ast.IndexExpr);ok{if sel,ok:=indexed.X.(*ast.SelectorExpr);ok&&sel.Sel.Name=="ReferenceArrayGet"{if len(call.Args)>0{if a,ok:=call.Args[0].(*ast.Ident);ok&&a.Name=="args"{getters++}}}}}
  return true
 })
 if getters!=1{t.Fatalf("unchanged source main canonical getter count=%d want1",getters)}
 if hostBindings!=0{t.Errorf("LEGACY_MAIN_ARRAY_ABI: direct host args bindings=%d canonical getter requires nominal Java String[]",hostBindings)}
}

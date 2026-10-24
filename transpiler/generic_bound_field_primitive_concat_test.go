package transpiler

import (
 "go/ast"
 "go/parser"
 "go/token"
 "os"
 "testing"
 sitter "github.com/smacker/go-tree-sitter"
)

// This is the complete independently recorded Main, including its main and
// finally workflow. The unit isolates the primitive field type producer that
// caused both observed native process and explicit-call runtime failures.
func TestGenericBoundFieldPrimitiveConcatOriginalSource(t *testing.T) {
 source,err:=os.ReadFile("testdata/generic_legacy_main_witness/Main.java");if err!=nil{t.Fatal(err)}
 t.Run("field-type",func(t *testing.T){
  helper:=setupParseHelper(t,string(source));ctx:=helper.Ctx.Clone()
  ctx.currentClass=helper.File.Symbols.FindClassScope("Main");if ctx.currentClass==nil{t.Fatal("missing Main")};ctx.className=ctx.currentClass.Class.Name
  for _,method:=range ctx.currentClass.Methods{if method.OriginalName=="main"{ctx.localScope=method}}
  if ctx.localScope==nil||len(ctx.localScope.TypeParameters)!=2{t.Fatal("dependent Main witness missing")}
  ctx.dependentTypeWitnesses=planConcreteDependentTypeWitnesses(ctx.localScope,source,ctx)
  var field *sitter.Node
  var walk func(*sitter.Node)
  walk=func(n *sitter.Node){if n==nil{return};if n.Type()=="field_access"&&n.Content(source)=="selected.marker"{if field!=nil{t.Fatal("nonunique marker source field")};field=n};for i:=0;i<int(n.NamedChildCount());i++{walk(n.NamedChild(i))}}
  walk(helper.File.Ast);if field==nil{t.Fatal("original selected.marker missing")}
  if target:=resolveInvocationTarget(field.ChildByFieldName("object"),ctx,source);target==nil||target.classScope==nil||target.classScope.Class.OriginalName!="Root"{t.Fatal("source receiver no longer resolves dependent Root bound")}
  got,ok:=inferExprJavaType(field,ctx,source)
  if !ok||got!="int"{t.Fatalf("BOUND_FIELD_TYPE: selected.marker=%q/%t wantint/true",got,ok)}
 })
 t.Run("primitive-concat",func(t *testing.T){
  generated:=renderGoFileFromJava(t,string(source));file,err:=parser.ParseFile(token.NewFileSet(),"generated.go",generated,0);if err!=nil{t.Fatal(err)}
  primitiveCalls:=0
  ast.Inspect(file,func(n ast.Node)bool{call,ok:=n.(*ast.CallExpr);if !ok{return true};sel,ok:=call.Fun.(*ast.SelectorExpr);if !ok||sel.Sel.Name!="JavaStringValueOfInt"{return true};hasMarker:=false;for _,arg:=range call.Args{ast.Inspect(arg,func(child ast.Node)bool{if field,ok:=child.(*ast.SelectorExpr);ok&&field.Sel.Name=="marker"{hasMarker=true};return true})};if hasMarker{primitiveCalls++};return true})
  if primitiveCalls!=1{t.Fatalf("BOUND_FIELD_PRIMITIVE_STRING: primitive marker conversion count=%d want1; preserve canonical primitive overload",primitiveCalls)}
 })
}

package transpiler

import (
 "go/ast"
 "go/parser"
 "go/token"
 "strings"
 "testing"
)

// Wildcard capture remains a Java reference even when invocation substitution
// spells its result as ?. The Go erased return may contain a typed null.
func TestJavaReferenceNullEqualityWildcardShape(t *testing.T) {
 for _, tc := range []struct{name,expr string;helper bool}{
  {"wildcard-equal","value.read()==null",true},
  {"wildcard-reverse-equal","null==value.read()",true},
  {"wildcard-not-equal","value.read()!=null",true},
  {"wildcard-reverse-not-equal","null!=value.read()",true},
  {"Object-control","(Object)value.read()==null",true},
  {"String-control","(String)value.read()==null",true},
  {"primitive-control","3==3",false},
  {"boolean-control","true!=false",false},
 } {
  t.Run(tc.name,func(t *testing.T){
   generated:=renderGoFileFromJava(t,`class Cell<T>{T read(){return null;}}
    public class NullEqualityShape{static boolean check(Cell<?> value){return `+tc.expr+`;}}`)
   file,err:=parser.ParseFile(token.NewFileSet(),"generated.go",generated,0)
   if err!=nil {t.Fatal(err)}
   has,foundCheck:=false,false
   for _,declaration:=range file.Decls {
    method,ok:=declaration.(*ast.FuncDecl)
    if !ok || !strings.Contains(strings.ToLower(method.Name.Name),"check") {continue}
    foundCheck=true
    // Receiver normalization can also call JavaReferenceEqual inside the
    // read invocation's closure. Inspect only the returned equality itself.
    for _,statement:=range method.Body.List {
     returned,ok:=statement.(*ast.ReturnStmt);if !ok {continue}
     for _,result:=range returned.Results {
      for {
       if parenthesized,ok:=result.(*ast.ParenExpr);ok {result=parenthesized.X;continue}
       if negated,ok:=result.(*ast.UnaryExpr);ok && negated.Op==token.NOT {result=negated.X;continue}
       break
      }
      call,ok:=result.(*ast.CallExpr);if !ok {continue}
      selector,ok:=call.Fun.(*ast.SelectorExpr)
      if ok && selector.Sel.Name=="JavaReferenceEqual" {has=true}
     }
    }
   }
   if !foundCheck {t.Fatalf("generated comparison function missing\n%s",generated)}
   if has!=tc.helper {t.Fatalf("reference-null predicate = %v, want %v\n%s",has,tc.helper,generated)}
  })
 }
}

func TestJavaReferenceNullEqualityWildcardProjectJVM(t *testing.T) {
 runCampaignCompilerStrictProjectOracle(t,map[string]string{
 "pom.xml":`<project><groupId>probe</groupId><artifactId>null-equality</artifactId><version>1</version></project>`,
 "src/main/java/p/Cell.java":`package p;public class Cell<T>{public T value;public int reads;public Cell(T value){this.value=value;}public T read(){reads++;return value;}}`,
 "src/main/java/p/Text.java":`package p;public class Text extends Cell<String>{public Text(String value){super(value);}@Override public String read(){reads++;return value;}}`,
 "src/main/java/app/Main.java":`package app;import p.Cell;import p.Text;public class Main{public static void main(String[]args){Text text=new Text(null);Cell<?> value=(Cell<?>)(Object)text;System.out.println((value.read()==null)+":"+(null==value.read())+":"+(value.read()!=null)+":"+(null!=value.read())+":"+text.reads);text.value="present";System.out.println((value.read()==null)+":"+(null==value.read())+":"+(value.read()!=null)+":"+(null!=value.read())+":"+text.reads);Cell<?> absent=(Cell<?>)(Object)null;System.out.println((absent==null)+":"+(null==absent)+":"+(absent!=null)+":"+(null!=absent));}}`,
 },"app.Main","true:true:false:false:4\nfalse:false:true:true:8\ntrue:true:false:false\n")
}

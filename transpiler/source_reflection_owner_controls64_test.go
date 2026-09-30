package transpiler

import (
 "go/ast"
 "go/parser"
 "go/token"
 "testing"
)

// Source owners and lexical binders with JDK-looking names remain ordinary Java
// calls. Require the execution companion call, so a runtime reflection intrinsic
// cannot silently consume a source method with a different result contract.
func TestSourceReflectionIntrinsicOwners64(t *testing.T) {
 cases:=[]struct{name,source,selector string}{
  {"source_field",`public class ReflectionOwner64 { static class Field { public long getGenericType(){return 71L;} } public static long run(){return new Field().getGenericType();} }`,"GetGenericTypeJava2goExecution"},
  {"source_class",`public class ReflectionOwner64 { static class Class { public long getDeclaredFields(){return 73L;} } public static long run(){return new Class().getDeclaredFields();} }`,"GetDeclaredFieldsJava2goExecution"},
  {"source_parameterized",`public class ReflectionOwner64 { static class ParameterizedType { public long getActualTypeArguments(){return 79L;} } public static long run(){return new ParameterizedType().getActualTypeArguments();} }`,"GetActualTypeArgumentsJava2goExecution"},
  {"lexical_field_binder",`public class ReflectionOwner64 { static class Member { public long getModifiers(){return 83L;} } static class Box<Field extends Member> { Field value; Box(Field value){this.value=value;} long read(){return value.getModifiers();} } public static long run(){return new Box<Member>(new Member()).read();} }`,"GetModifiersJava2goExecution"},
  {"annotation_impostor",`public class ReflectionOwner64 { interface Annotation {long value();} static class UserValue implements Annotation {public long value(){return 89L;}} public static long run(){Annotation value=new UserValue();return value.value();} }`,"ValueJava2goExecution"},
 }
 for _,c:=range cases{t.Run(c.name,func(t *testing.T){
  generated:=renderGoFileFromJava(t,c.source)
  file,err:=parser.ParseFile(token.NewFileSet(),"generated.go",generated,0);if err!=nil{t.Fatalf("invalid generated control: %v\n%s",err,generated)}
  calls:=0
  ast.Inspect(file,func(node ast.Node)bool{call,ok:=node.(*ast.CallExpr);if !ok{return true};selector,ok:=call.Fun.(*ast.SelectorExpr);if ok&&selector.Sel.Name==c.selector{calls++};return true})
  if calls==0{t.Fatalf("source declaration must retain execution dispatch %s:\n%s",c.selector,generated)}
 })}
}

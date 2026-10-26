package transpiler

import "testing"

func TestInheritedFieldSuperclassArgumentProvenanceControls(t *testing.T) {
 for _, test := range []struct{name, owner, base, child, want string; arguments []string}{
  {name:"raw_parent_bound", owner:"app.Use", base:`package p;import origin.Marker;public class Base<T extends Marker>{public T value;}`, child:`package app;public class Use extends p.Base{public static class Marker{}}`, want:"origin.Marker"},
  {name:"multilevel_header", owner:"app.Use", base:`package p;public class Base<T>{public T value;}`, child:`package app;import origin.Marker;public class Use extends q.Middle<Marker>{public static class Marker{}}`, want:"origin.Marker"},
  {name:"array_header", owner:"app.Use", base:`package p;public class Base<T>{public T value;}`, child:`package app;import origin.Marker;public class Use extends p.Base<Marker[]>{public static class Marker{}}`, want:"origin.Marker[]"},
  {name:"shadowed_carried_binder", owner:"app.Outer.Inner", base:`package p;public class Base<T>{public T value;}`, child:`package app;public class Outer<T>{public class Inner<T> extends p.Base<T>{}}`, arguments:[]string{"origin.First","origin.Second"}, want:"origin.Second"},
 } {
  t.Run(test.name,func(t *testing.T){
   childName:="app/Use.java";if test.name=="shadowed_carried_binder"{childName="app/Outer.java"}
   ctx:=sourceGenericViewDemandTestContext(t,map[string]string{
    "origin/Marker.java":`package origin;public class Marker{}`,
    "origin/First.java":`package origin;public class First{}`,
    "origin/Second.java":`package origin;public class Second{}`,
    "p/Base.java":test.base,
    "q/Middle.java":`package q;public class Middle<U extends origin.Marker> extends p.Base<U>{}`,
    childName:test.child,
   })
   owner:=findQualifiedSourceClass(test.owner);if owner==nil{t.Fatal("missing owner")}
   ctx=classScopeCtx(owner,ctx)
   resolution:=findFieldResolutionInHierarchy(owner,"value",ctx);if resolution==nil{t.Fatal("missing inherited field")}
   if test.name=="shadowed_carried_binder" && len(owner.TypeParameters)!=2{t.Fatalf("expected two distinct carried declarations, got %d",len(owner.TypeParameters))}
   got:=instantiatedFieldJavaType(owner,test.arguments,resolution,ctx)
   if got!=test.want{t.Errorf("superclass argument provenance %s = %q, want %q",test.name,got,test.want)}
  })
 }
}

package transpiler

import "testing"

// The inherited source view must keep the namespace where its superclass
// argument was declared; Use's body member Marker cannot capture that edge.
func TestInheritedFieldSuperclassHeaderArgumentIdentity(t *testing.T) {
 ctx := sourceGenericViewDemandTestContext(t, map[string]string{
 "origin/Marker.java": `package origin; public class Marker { public int seed,hits; public Marker(int seed){this.seed=seed;} public int mark(){return seed + ++hits;} }`,
 "p/Base.java": `package p; public class Base<T> { public T value; public Base(T value){this.value=value;} }`,
 "app/Use.java": `package app; import origin.Marker; public class Use extends p.Base<Marker> { public Use(origin.Marker value){super(value);} public int read(){return this.value.mark();} public static class Marker { public int mark(){return -999;} } }`,
 "app/Main.java": `package app; public class Main { public static void main(String[] args){ origin.Marker marker=new origin.Marker(Integer.parseInt(args[0])); Use use=new Use(marker); p.Base<?> wildcard=(p.Base<?>)(Object)use; System.out.println(use.read()+":"+marker.hits+":"+(wildcard.value==marker)); } }`,
 })
 owner := findQualifiedSourceClass("app.Use")
 ctx = classScopeCtx(owner, ctx)
 resolution := findFieldResolutionInHierarchy(owner, "value", ctx)
 if resolution == nil { t.Fatal("missing inherited field") }
 got := instantiatedFieldJavaType(owner, nil, resolution, ctx)
 if got != "origin.Marker" { t.Errorf("inherited superclass header argument = %q, want origin.Marker", got) }
 if resolved := resolveClassScopeByQualifiedName(ctx, got); resolved != findQualifiedSourceClass("origin.Marker") { t.Errorf("inherited field source view resolves in child body namespace") }
}

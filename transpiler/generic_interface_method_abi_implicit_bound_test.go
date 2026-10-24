package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"go/ast"
	"testing"
)

func TestGenericInterfaceABIImplicitBoundNamedPrefixJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-implicit-bound</artifactId><version>1</version></project>`, "src/main/java/prefix/Main.java": `package prefix;
class java{static class lang{static class Object{int code=99;}}}
class Object{}
class Child extends java.lang.Object{}
interface Poly{<T>T id(T value);}
interface Bound{<T extends java.lang.Object>T id(T value);}
class Named implements Poly{public<T>T id(T value){return value;}}
class Bounded implements Bound{public<T extends java.lang.Object>T id(T value){return value;}}
public class Main{static<java>String call(Poly p){return p.id("prefix");}static<Object>Child child(Poly p,Child value){return p.id(value);}public static void main(String[]args){Poly p=new Named();Bound b=new Bounded();Child child=new Child();Child seen=b.id(child);Child same=child(p,child);Child nil=child(p,(Child)null);Integer boxed=p.id(7);System.out.println(call(p)+":"+(seen==child)+":"+seen.code+":"+(same==child)+":"+(nil==null)+":"+boxed);}}
`}, "prefix.Main", "prefix:true:99:true:true:7\n")
}
func TestGenericInterfaceABIImplicitBoundProvenance(t *testing.T) {
	h := setupParseHelper(t, `package prefix;class Object{}class java{static class lang{static class Object{}}}interface Poly{<T>T id(T x);}interface Bound{<T extends java.lang.Object>T id(T x);}`)
	owner := h.File.Symbols.FindClassScope("Poly")
	ctx := h.Ctx.Clone()
	ctx.currentClass = owner
	def := owner.FindMethodByName("id", nil)
	plan := planGenericMethodEntryABI(owner, def, ctx)
	if plan == nil {
		t.Fatal("missing existing scalar entry plan")
	}
	for _, typ := range []ast.Expr{plan.parameterTypes[0], plan.resultType} {
		id, ok := typ.(*ast.Ident)
		if !ok || id.Name != "any" {
			t.Fatalf("implicit declaration bound routed through source Object: %#v", typ)
		}
	}
	bound := h.File.Symbols.FindClassScope("Bound")
	ctx.currentClass = bound
	bd := bound.FindMethodByName("id", nil)
	bp := planGenericMethodEntryABI(bound, bd, ctx)
	if bp == nil {
		t.Fatal("missing existing written bound plan")
	}
	if _, ok := bp.resultType.(*ast.StarExpr); !ok {
		t.Fatalf("written source bound lost pointer ABI: %#v", bp.resultType)
	}
}

func TestGenericInterfaceABIImplicitBoundNegativeOrigins(t *testing.T) {
	h := setupParseHelper(t, `class T2{}interface Owner<T>{<U>T classValue(U x);<U>T2 nominal(U x);<U>U[] array(U[] x);}`)
	owner := h.File.Symbols.FindClassScope("Owner")
	ctx := h.Ctx.Clone()
	ctx.currentClass = owner
	for _, name := range []string{"classValue", "nominal"} {
		d := owner.FindMethodByName(name, nil)
		p := planGenericMethodEntryABI(owner, d, ctx)
		if p == nil {
			t.Fatal("missing existing scalar plan")
		}
		if p.implicitObjectScalar(d) {
			t.Fatalf("%s result acquired implicit method bound", name)
		}
		if !p.implicitObjectScalar(d.Parameters[0]) {
			t.Fatal("method parameter lost exact declaration proof")
		}
	}
	if planGenericMethodEntryABI(owner, owner.FindMethodByName("array", nil), ctx) != nil {
		t.Fatal("array admitted by scalar Object proof")
	}
	synthetic := &symbol.Definition{OriginalType: "U", RequiresHelper: true, TypeParameters: []symbol.TypeParam{{Name: "U"}}}
	if got := genericMethodErasedJavaType(synthetic, "U"); got != "Object" {
		t.Fatalf("ownernil legacy erasure changed: %s", got)
	}
	p := &genericMethodEntryABI{method: synthetic}
	if p.implicitObjectScalar(synthetic) {
		t.Fatal("declarationnil acquired canonical proof")
	}
}

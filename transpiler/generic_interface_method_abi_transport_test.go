package transpiler

import (
	"go/ast"
	"testing"
)

func TestGenericInterfaceABIPrivateBoundJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-transport</artifactId><version>1</version></project>`, "src/main/java/scoped/Main.java": `package scoped;
class Base{} class Child extends Base{}
interface Poly{<T extends Base>T id(T value);default<T extends Base>T round(T value){return id(value);}}
class Named implements Poly{public<T extends Base>T id(T value){Main.calls++;return value;}}
public class Main{static int calls;static<Base>Child call(Poly p,Child value){return p.round(value);}public static void main(String[]args){Poly p=new Named();Child child=new Child();Child first=call(p,child);Child nil=call(p,(Child)null);System.out.println((first==child)+":"+(nil==null)+":"+calls);}}
`}, "scoped.Main", "true:true:2\n")
}
func TestGenericInterfaceABIPublicBoundJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-transport</artifactId><version>1</version></project>`, "src/main/java/scoped/Base.java": `package scoped;public class Base{}
`, "src/main/java/scoped/Child.java": `package scoped;public class Child extends Base{}
`, "src/main/java/scoped/Poly.java": `package scoped;public interface Poly{<T extends Base>T id(T value);default<T extends Base>T round(T value){return id(value);}}
`, "src/main/java/scoped/Named.java": `package scoped;public class Named implements Poly{public<T extends Base>T id(T value){Main.calls++;return value;}}
`, "src/main/java/scoped/Main.java": `package scoped;public class Main{public static int calls;static<Base>Child call(Poly p,Child value){return p.round(value);}public static void main(String[]args){Poly p=new Named();Child child=new Child();Child first=call(p,child);Child nil=call(p,(Child)null);System.out.println((first==child)+":"+(nil==null)+":"+calls);}}
`}, "scoped.Main", "true:true:2\n")
}
func TestGenericInterfaceABIImportedBoundJDK21(t *testing.T) {
	runCampaignCompilerStrictProjectOracle(t, map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>interface-transport</artifactId><version>1</version></project>`, "src/main/java/api/Base.java": `package api;public class Base{}
`, "src/main/java/api/Child.java": `package api;public class Child extends Base{}
`, "src/main/java/api/Poly.java": `package api;public interface Poly{<T extends Base>T id(T value);default<T extends Base>T round(T value){return id(value);}}
`, "src/main/java/impl/Named.java": `package impl;public class Named implements api.Poly{public<T extends api.Base>T id(T value){app.Main.calls++;return value;}}
`, "src/main/java/app/Main.java": `package app;import api.Base;import api.Child;import api.Poly;public class Main{public static int calls;static<Base>Child call(Poly p,Child value){return p.round(value);}public static void main(String[]args){Poly p=new impl.Named();Child child=new Child();Child first=call(p,child);Child nil=call(p,(Child)null);System.out.println((first==child)+":"+(nil==null)+":"+calls);}}
`}, "app.Main", "true:true:2\n")
}
func TestGenericInterfaceABICalleeBoundTransportIdentity(t *testing.T) {
	h := setupParseHelper(t, `class Base{}interface Poly<A>{<T extends Base>T id(T x);<T extends Base>Base fixed(T ignored,Base x);<T extends Base>A classValue(T ignored,A x);<T extends Base>T[] array(T[] x);}`)
	owner := h.File.Symbols.FindClassScope("Poly")
	ctx := h.Ctx.Clone()
	ctx.currentClass = owner
	id := owner.FindMethodByName("id", nil)
	plan := planGenericMethodEntryABI(owner, id, ctx)
	if plan == nil {
		t.Fatal("missing existing bounded scalar plan")
	}
	if _, ok := plan.sourceBoundPhysicalType(id, plan.resultJavaType, plan.resultType, nil, nil, ctx).(*ast.StarExpr); !ok {
		t.Fatal("method-owned scalar bound lost source nominal pointer")
	}
	fixed := owner.FindMethodByName("fixed", nil)
	fp := planGenericMethodEntryABI(owner, fixed, ctx)
	if fp == nil {
		t.Fatal("missing existing mixed plan")
	}
	if fp.sourceBoundPhysicalType(fixed, fp.resultJavaType, fp.resultType, nil, nil, ctx) != nil || fp.sourceBoundPhysicalType(fixed.Parameters[1], fp.parameterJavaTypes[1], fp.parameterTypes[1], nil, nil, ctx) != nil {
		t.Fatal("nominal formal/result acquired method-binder transport")
	}
	class := owner.FindMethodByName("classValue", nil)
	cp := planGenericMethodEntryABI(owner, class, ctx)
	if cp == nil {
		t.Fatal("missing existing class-owned plan")
	}
	if cp.sourceBoundPhysicalType(class, cp.resultJavaType, cp.resultType, nil, nil, ctx) != nil {
		t.Fatal("class-owned result acquired method-binder transport")
	}
	arr := owner.FindMethodByName("array", nil)
	if planGenericMethodEntryABI(owner, arr, ctx) != nil {
		t.Fatal("unproved array ABI admitted")
	}
}

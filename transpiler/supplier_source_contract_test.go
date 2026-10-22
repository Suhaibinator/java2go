package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func supplierTestContract(t *testing.T, owner *symbol.ClassScope, ctx Ctx) sourceNativeFunctionalContract {
	t.Helper()
	for _, c := range sourceNativeFunctionalContracts(owner, ctx) {
		if c.family == "Supplier" {
			return c
		}
	}
	t.Fatal("no proved Supplier contract")
	return sourceNativeFunctionalContract{}
}

func TestSupplierSourceContractOwnedGenericInheritance(t *testing.T) {
	h := setupParseHelper(t, `import java.util.function.Supplier;import java.util.Map;class Base<T> implements Supplier<T>{public T get(){return null;}}class Child extends Base<Map<String,Integer>>{}`)
	child := h.Ctx.currentFile.FindClassScope("Child")
	c := supplierTestContract(t, child, h.Ctx)
	method, owner, _ := resolveSourceNativeSAM(child, c, h.Ctx)
	if !supplierSourceSelectedMethodOwned(child, owner, method, c, h.Ctx) {
		t.Fatal("exact inherited get/binder path refused")
	}
	copyMethod := *method
	if supplierSourceSelectedMethodOwned(child, owner, &copyMethod, c, h.Ctx) {
		t.Fatal("unregistered method copy admitted")
	}
	base := h.Ctx.currentFile.FindClassScope("Base")
	original := base.TypeParameters[0].Declaration
	base.TypeParameters[0].Declaration = &symbol.TypeParamDeclaration{SourceName: "T", GoName: original.GoName}
	defer func() { base.TypeParameters[0].Declaration = original }()
	if supplierSourceContractOwned(child, c, h.Ctx) {
		t.Fatal("same-spelled replacement binder acquired original declaration")
	}
}

func TestSupplierSourceContractRejectsDetachedAndDonatedHeaders(t *testing.T) {
	h := setupParseHelper(t, `import java.util.function.Supplier;import java.util.Map;class NativeSource implements Supplier<Map<String,Integer>>{public Map<String,Integer> get(){return null;}}class Shape{public Map<String,Integer> get(){return null;}}`)
	owner := h.Ctx.currentFile.FindClassScope("NativeSource")
	c := supplierTestContract(t, owner, h.Ctx)
	detached := *owner
	if supplierSourceContractOwned(&detached, c, h.Ctx) {
		t.Fatal("unregistered detached class copy admitted")
	}
	shape := h.Ctx.currentFile.FindClassScope("Shape")
	shape.ImplementedInterfaces = append([]string(nil), owner.ImplementedInterfaces...)
	defer func() { shape.ImplementedInterfaces = nil }()
	if supplierSourceContractOwned(shape, c, h.Ctx) {
		t.Fatal("written source header replaced by donated metadata")
	}
	file := parsing.SourceFile{Name: "Other.java", Source: []byte(`class NativeSource implements java.util.function.Supplier<java.util.Map<String,Integer>>{public java.util.Map<String,Integer> get(){return null;}}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	other := file.ParseSymbols().BaseClass
	original := owner.Class.DeclarationNode
	owner.Class.DeclarationNode = other.Class.DeclarationNode
	defer func() { owner.Class.DeclarationNode = original }()
	if supplierSourceContractOwned(owner, c, h.Ctx) {
		t.Fatal("mutable class alias replaced independent parsed-root provenance")
	}
}

func TestSupplierSourceContractRejectsHomonymsAndUnknownResults(t *testing.T) {
	h := setupParseHelper(t, `class Supplier<T>{}class Shape extends Supplier<String>{public String get(){return "x";}}`)
	shape := h.Ctx.currentFile.FindClassScope("Shape")
	if nativeFunctionalFamily("Supplier<String>", h.Ctx) != "" || isExternalSupplierType("Supplier<String>", h.Ctx) {
		t.Fatal("source homonym acquired canonical Supplier owner")
	}
	c, _ := nativeFunctionalContract("Supplier", []string{"java.lang.String"})
	if supplierSourceContractOwned(shape, c, h.Ctx) {
		t.Fatal("source name/shape acquired native certificate")
	}
	c, _ = nativeFunctionalContract("Supplier", []string{"absent.Unknown"})
	if supplierSourceContractOwned(shape, c, h.Ctx) {
		t.Fatal("unknown result accepted")
	}
}

func TestSupplierSourceContractReorderedNestedBinderOrigins(t *testing.T) {
	h := setupParseHelper(t, `import java.util.function.Supplier;import java.util.*;class Parent<A,B> implements Supplier<Map<B,List<A>>>{public Map<B,List<A>> get(){return null;}}class Child<X,Y> extends Parent<Y,X>{}`)
	child := h.Ctx.currentFile.FindClassScope("Child")
	c := supplierTestContract(t, child, h.Ctx)
	method, owner, _ := resolveSourceNativeSAM(child, c, h.Ctx)
	if !supplierSourceSelectedMethodOwned(child, owner, method, c, h.Ctx) {
		t.Fatal("reordered nested inherited result lost declaration-bound origins")
	}
	header, _ := supplierSourceDeclarationCtx(child, h.Ctx)
	origin := declaredJavaTypeOrigin(symbol.JavaType{Original: c.result, TypeParameterBindings: symbol.VisibleTypeParamBindings(child.TypeParameters)}, header)
	if len(origin.arguments) != 2 || origin.arguments[0].parameter != child.TypeParameters[0].Declaration || len(origin.arguments[1].arguments) != 1 || origin.arguments[1].arguments[0].parameter != child.TypeParameters[1].Declaration {
		t.Fatal("nested result slots shifted or rebound by spelling")
	}
}

func TestSupplierSourceContractSharedDiamondAndCovariantBody(t *testing.T) {
	h := setupParseHelper(t, `import java.util.function.Supplier;import java.util.*;interface Left<T> extends Supplier<T>{}interface Right<T> extends Supplier<T>{}class Parent implements Left<Map<String,Integer>>,Right<Map<String,Integer>>{public Map<String,Integer> get(){return null;}}class Child extends Parent{public TreeMap<String,Integer> get(){return null;}}`)
	child := h.Ctx.currentFile.FindClassScope("Child")
	c := supplierTestContract(t, child, h.Ctx)
	method, owner, _ := resolveSourceNativeSAM(child, c, h.Ctx)
	if method != nil {
		proof, proved := supplierSourceContractProof(child, c, h.Ctx)
		header, _ := supplierSourceDeclarationCtx(owner, h.Ctx)
		actual := substituteJavaTypeOrigin(declaredJavaTypeOrigin(definitionDeclaredJavaType(method), header), proof[owner])
		root, _ := supplierSourceDeclarationCtx(child, h.Ctx)
		expected := declaredJavaTypeOrigin(symbol.JavaType{Original: c.result, TypeParameterBindings: symbol.VisibleTypeParamBindings(child.TypeParameters)}, root)
		cost, exact, applies, handled := declaredOriginConversion(actual, expected, header)
		t.Logf("covariant diagnostic: ownerIsChild=%v writtenResult=%q proved=%v actual=%#v expected=%#v conversion=%d/%v/%v/%v", owner == child, method.OriginalType, proved, actual, expected, cost, exact, applies, handled)
	}
	if owner != child || method == nil || method.OriginalType != "TreeMap<String,Integer>" || !supplierSourceSelectedMethodOwned(child, owner, method, c, h.Ctx) {
		t.Fatal("shared Supplier ancestor or actual covariant override body was lost")
	}
}

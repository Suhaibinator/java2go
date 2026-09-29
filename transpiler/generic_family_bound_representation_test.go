package transpiler

import "testing"

func TestGenericFamilyBoundRepresentationKeepsDeclarationIdentity(t *testing.T) {
	helper := setupParseHelper(t, `
abstract class Base<T>{abstract T get();abstract void put(T value);}
class Child<T extends java.lang.Number> extends Base<T>{
 T value; T get(){return value;} void put(T value){this.value=value;}
 <T> T shadow(T value){return value;}
}
`)
	base := helper.File.Symbols.FindClassScope("Base")
	child := helper.File.Symbols.FindClassScope("Child")
	plan, err := planGenericFamily(base, helper.Ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := plan.representations[base.TypeParameters[0].Declaration].erasure; got != "java.lang.Object" {
		t.Fatalf("parent descriptor = %q", got)
	}
	if got := plan.representations[child.TypeParameters[0].Declaration].erasure; got != "java.lang.Number" {
		t.Fatalf("child descriptor = %q", got)
	}
	for _, method := range child.Methods {
		if method.OriginalName != "shadow" {
			continue
		}
		if _, physical := plan.representations[method.TypeParameters[0].Declaration]; physical {
			t.Fatal("method-owned shadow binder joined the owner storage plan")
		}
		if len(method.TypeParameters[0].Bounds) != 0 {
			t.Fatal("method-owned source bounds changed")
		}
		return
	}
	t.Fatal("missing shadow method")
}

func TestGenericFamilyBoundRepresentationRejectsUnprovenBounds(t *testing.T) {
	for _, source := range []string{
		`class Number{} abstract class Base<T extends Number>{T value;}`,
		`abstract class Base<T extends java.lang.Enum<T>>{T value;}`,
		`abstract class Base<T extends java.lang.Number>{java.util.Map<String,T> values;}`,
	} {
		helper := setupParseHelper(t, source)
		if plan, err := planGenericFamily(helper.File.Symbols.FindClassScope("Base"), helper.Ctx); err == nil || plan != nil {
			t.Fatalf("unproved physical representation admitted: %s", source)
		}
	}
}

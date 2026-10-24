package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// Compare producer ASTs, including the hidden Execution offset; no name-pattern
// assertion can substitute for the declaration's actual allocated receiver ABI.
func rawVirtualActualSignatures(t *testing.T, owner *symbol.ClassScope, method *symbol.Definition, ctx Ctx) (*ast.FuncType, *ast.FuncType, *ast.FuncType) {
	t.Helper()
	declaration := classScopeCtx(owner, ctx)
	public := ParseNode(method.DeclarationNode, declaration.currentFile.Source, declaration).(*ast.Field).Type.(*ast.FuncType)
	generated := generateExecutionCompanionInterface(owner, declaration).(*ast.GenDecl)
	hiddenName := executionImplementationName(method, owner, declaration)
	var hidden *ast.FuncType
	for _, spec := range generated.Specs {
		if iface, ok := spec.(*ast.TypeSpec).Type.(*ast.InterfaceType); ok {
			for _, field := range iface.Methods.List {
				for _, name := range field.Names {
					if name.Name == hiddenName {
						hidden = field.Type.(*ast.FuncType)
					}
				}
			}
		}
	}
	if hidden == nil {
		t.Fatal("actual execution companion method absent")
	}
	return public, hidden, sourceInterfaceVirtualSignature(owner, method, ctx)
}

func rawVirtualAssertTypes(t *testing.T, public, hidden, optional *ast.FuncType) {
	t.Helper()
	if public == nil || hidden == nil || optional == nil || len(hidden.Params.List) != len(public.Params.List)+1 || len(optional.Params.List) != len(hidden.Params.List) {
		t.Fatal("signature/Execution arity differs")
	}
	for i, parameter := range public.Params.List {
		if !reflect.DeepEqual(parameter.Type, hidden.Params.List[i+1].Type) || !reflect.DeepEqual(parameter.Type, optional.Params.List[i+1].Type) {
			t.Fatalf("public/hidden/optional actual parameter %d differs: public=%#v hidden=%#v optional=%#v", i, parameter.Type, hidden.Params.List[i+1].Type, optional.Params.List[i+1].Type)
		}
	}
	if !reflect.DeepEqual(hidden.Params.List[0].Type, optional.Params.List[0].Type) {
		t.Fatal("Execution type changed")
	}
	if !reflect.DeepEqual(public.Results, hidden.Results) || !reflect.DeepEqual(hidden.Results, optional.Results) {
		t.Fatal("result signature changed")
	}
}

const rawVirtualDeclarationSource = `public class ReceiverContracts {
 interface Numbered {int number();}
 static class Box<T extends Numbered>{T value; T swap(T next){T old=value;value=next;return old;}}
 static class Item implements Numbered{public int number(){return 1;}}
 interface Raw{Numbered apply(Box receiver,Numbered next);}
 static Raw reference=Box::swap;
}`

func TestSourceInterfaceRawReceiverSignatureActualDescriptors(t *testing.T) {
	for run := 0; run < 3; run++ {
		t.Run(string(rune('a'+run)), func(t *testing.T) {
			helper := setupParseHelper(t, rawVirtualDeclarationSource)
			owner := overrideBridgeTestScope(t, helper, "Raw")
			method := overrideBridgeTestMethod(t, owner, "apply")
			ctx := classScopeCtx(owner, helper.Ctx)
			target := overrideBridgeTestScope(t, helper, "Box")
			actualView := generateRawUnboundReceiverViewDecl(classScopeCtx(target, ctx)).(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
			for repeat := 0; repeat < 3; repeat++ {
				public, hidden, optional := rawVirtualActualSignatures(t, owner, method, ctx.Clone())
				if !reflect.DeepEqual(public.Params.List[0].Type, actualView.Name) {
					t.Fatal("public SAM no longer uses the actual declaration-owned raw receiver view")
				}
				rawVirtualAssertTypes(t, public, hidden, optional)
			}
		})
	}
}

func TestSourceInterfaceRawReceiverSignatureRefusalBoundaries(t *testing.T) {
	const prefix = `public class Refusals {interface Numbered{int number();}static class Item implements Numbered{public int number(){return 1;}}static class Box<T extends Numbered>{T value; T swap(T next){T old=value;value=next;return old;}}interface Anchor{Numbered apply(Box b,Numbered n);}static Anchor anchor=Box::swap;`
	for _, test := range []struct {
		name, body string
		slot       int
	}{
		{"typed", `interface Target{Numbered apply(Box<Item> b,Numbered n);}`, 0},
		{"array", `interface Target{Numbered apply(Box[] b,Numbered n);}`, 0},
		{"nonfirst", `interface Target{Numbered apply(Numbered n,Box b);}`, 1},
		{"multimethod", `interface Target{Numbered apply(Box b,Numbered n);int other();}`, 0},
		{"sourceImplementation", `interface Target{Numbered apply(Box b,Numbered n);}static class Implementation implements Target{public Numbered apply(Box b,Numbered n){return (Numbered)b.value;}}`, 0},
		{"sourceWrongNominal", `static class Plain{Numbered value;}interface Target{Numbered apply(Plain b,Numbered n);}`, 0},
		{"qualifiedWrongOwner", `static class Other{static class Box<T>{T value;}}interface Target{Numbered apply(Other.Box b,Numbered n);}`, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, prefix+test.body+`}`)
			owner := overrideBridgeTestScope(t, helper, "Target")
			method := overrideBridgeTestMethod(t, owner, "apply")
			ctx := classScopeCtx(owner, helper.Ctx)
			javaType := qualifyJavaTypeInDeclaringContext(definitionParameterJavaSignatureType(method, test.slot), owner)
			if rawUnboundReceiverViewScopeForSAMParameter(owner, method, test.slot, javaType, ctx) != nil {
				t.Fatal("negative declaration was admitted by unchanged chooser")
			}
			public, hidden, optional := rawVirtualActualSignatures(t, owner, method, ctx)
			rawVirtualAssertTypes(t, public, hidden, optional)
		})
	}
	for _, test := range []struct{ name, source string }{
		{"missingReference", `public class Missing{static class Box<T>{T swap(T v){return v;}}interface Target{Object apply(Box b,Object n);}}`},
		{"boundReference", `public class Bound{static class Box<T>{T swap(T v){return v;}}interface Target{Object apply(Box b,Object n);}interface BoundOp{Object apply(Object n);}static Box box=new Box();static BoundOp bound=box::swap;}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			owner := overrideBridgeTestScope(t, helper, "Target")
			method := overrideBridgeTestMethod(t, owner, "apply")
			ctx := classScopeCtx(owner, helper.Ctx)
			if rawUnboundReceiverViewScopeForSAMParameter(owner, method, 0, method.Parameters[0].OriginalType, ctx) != nil {
				t.Fatal("missing raw unbound site admitted")
			}
			public, hidden, optional := rawVirtualActualSignatures(t, owner, method, ctx)
			rawVirtualAssertTypes(t, public, hidden, optional)
		})
	}
}

func TestSourceInterfaceRawReceiverSignatureBinderAdmission(t *testing.T) {
	for _, source := range []string{
		`public class Shadow{static class Box<T>{T swap(T v){return v;}}interface Target<Box>{Box apply(Box b);} }`,
		`public class Shadow{static class Box<T>{T swap(T v){return v;}}interface Target{<Box> Box apply(Box b);} }`,
	} {
		helper := setupParseHelper(t, source)
		owner := overrideBridgeTestScope(t, helper, "Target")
		method := overrideBridgeTestMethod(t, owner, "apply")
		if planSourceInterfaceVirtualEntry(owner, method, helper.Ctx) != nil {
			t.Fatal("class/method binder admission changed")
		}
	}
}

func TestSourceInterfaceRawReceiverSignatureQualifiedConsumer(t *testing.T) {
	sources := map[string]string{
		"owner/Numbered.java": `package owner;public interface Numbered{int number();}`,
		"owner/Box.java":      `package owner;public class Box<T extends Numbered>{T value;public T swap(T next){T old=value;value=next;return old;}}`,
		"owner/Raw.java":      `package owner;public interface Raw{Numbered apply(Box receiver,Numbered next);}`,
		"app/Consumer.java":   `package app;import owner.Raw;public class Consumer<Box>{static Raw reference=owner.Box::swap;static class Other{static class Box<T>{T value;}}}`,
	}
	ctx := sourceGenericViewDemandTestContext(t, sources)
	owner := findQualifiedSourceClass("owner.Raw")
	method := overrideBridgeTestMethod(t, owner, "apply")
	target := findQualifiedSourceClass("owner.Box")
	declaration := classScopeCtx(owner, ctx)
	public, hidden, optional := rawVirtualActualSignatures(t, owner, method, declaration)
	rawVirtualAssertTypes(t, public, hidden, optional)
	view := generateRawUnboundReceiverViewDecl(classScopeCtx(target, ctx)).(*ast.GenDecl).Specs[0].(*ast.TypeSpec)
	consumer := classScopeCtx(findQualifiedSourceClass("app.Consumer"), ctx)
	expected := qualifiedNameExpr(view.Name.Name, findJavaPackageForClassScope(target), consumer)
	first := sourceInterfaceVirtualSignature(owner, method, consumer)
	if !reflect.DeepEqual(first.Params.List[1].Type, expected) {
		t.Fatal("consumer import/binder replaced actual declaring owner receiver ABI")
	}
	for repeat := 0; repeat < 3; repeat++ {
		next := sourceInterfaceVirtualSignature(owner, method, consumer.Clone())
		if !reflect.DeepEqual(first, next) {
			t.Fatal("repeated consumer signature/cache changed allocated AST")
		}
	}
}

package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

const generics472PublicFooSource = `package gen.integration5; public class Foo { public int marker; public Foo(int marker){this.marker=marker;} }
`

const generics472UnknownFooSource = `
package gen.integration5;
public class Utils {
    static <T> T id(T value) { return value; }

    public static void test() {
        Foo f = null;
        Foo g = Utils.<Foo>id(f);
    }
}
`

func generics472ParseWithPublicFoo(t *testing.T, source string) *ParseHelper {
	return genericsFixtureParseWithPublicFoo(t, "Utils.java", source, generics472PublicFooSource)
}

func genericsInstanceParseWithPublicFoo(t *testing.T, source string) *ParseHelper {
	return genericsFixtureParseWithPublicFoo(t, "Box.java", source, genericsInstancePublicFooSource)
}

func genericsFixtureParseWithPublicFoo(t *testing.T, sourceName, source, supplemental string) *ParseHelper {
	t.Helper()
	previous := symbol.GlobalScope
	symbol.GlobalScope = &symbol.GlobalSymbols{Packages: make(map[string]*symbol.PackageScope)}
	t.Cleanup(func() { symbol.GlobalScope = previous })
	foo := parsing.SourceFile{Name: "Foo.java", Source: []byte(supplemental)}
	utils := parsing.SourceFile{Name: sourceName, Source: []byte(source)}
	for _, file := range []*parsing.SourceFile{&foo, &utils} {
		if err := file.ParseAST(); err != nil {
			t.Fatal(err)
		}
		symbol.AddSymbolsToPackage(file.ParseSymbols())
	}
	ResolveFile(foo)
	ResolveFile(utils)
	return &ParseHelper{File: utils, Ctx: Ctx{currentFile: utils.Symbols, currentClass: utils.Symbols.BaseClass}}
}

func TestExplicitStaticGeneric472UnknownNominalRejected(t *testing.T) {
	h := setupParseHelper(t, generics472UnknownFooSource)
	owner := h.File.Symbols.FindClassScope("Utils")
	if owner == nil {
		t.Fatal("original owner missing")
	}
	identity, caller := owner.FindMethodByName("id", nil), owner.FindMethodByName("test", nil)
	if identity == nil || caller == nil || len(identity.TypeParameters) != 1 || identity.TypeParameters[0].Declaration == nil {
		t.Fatal("original method binder lost before unknown actual proof")
	}
	ctx := h.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, caller
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	if invocation == nil {
		t.Fatal("original invocation missing")
	}
	var actual inferredJavaTypeOrigin
	typ, known := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, h.File.Source, &actual)
	if !known || typ != "Foo" || !actual.unresolved || completeJavaTypeOrigin(actual) || actual.parameter != nil || actual.nominalScope != nil || actual.externalName != "" {
		t.Fatalf("undeclared Foo gained identity: %s %+v", typ, actual)
	}
	formal := methodFormalOrigins(identity, owner, nil, ctx)
	if len(formal) != 1 || formal[0].parameter != identity.TypeParameters[0].Declaration {
		t.Fatal("callee-owned T declaration lost")
	}
	if _, _, applies, handled := declaredOriginConversion(actual, formal[0], ctx); !handled || applies {
		t.Fatal("unresolved actual gained permissive generic conversion")
	}
	if selected := findBestMethodInHierarchy(owner, "id", invocation.ChildByFieldName("arguments"), false, true, ctx, h.File.Source); selected != nil {
		t.Fatal("unknown Foo must not select a source generic declaration")
	}
}

const genericsInstancePublicFooSource = `package gen.integration4; public class Foo { public int marker; public Foo(int marker){this.marker=marker;} }
`

const genericsInstanceUnknownFooSource = `
package gen.integration4;
public class Box<T> {
    public <R> R identity(R value) { return value; }

    public static Foo callFoo(Box<Foo> box, Foo value) {
        return box.identity(value);
    }

    public static <X> X callGeneric(Box<X> box, X value) {
        return box.identity(value);
    }
}
`

func TestInstanceGenericIntegrationUnknownNominalRejected(t *testing.T) {
	h := setupParseHelper(t, genericsInstanceUnknownFooSource)
	owner := h.File.Symbols.FindClassScope("Box")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("original Box owner binder missing")
	}
	identity, caller := owner.FindMethodByName("identity", nil), owner.FindMethodByName("callFoo", nil)
	if identity == nil || caller == nil || len(identity.TypeParameters) != 1 || identity.TypeParameters[0].Declaration == nil || identity.TypeParameters[0].Declaration == owner.TypeParameters[0].Declaration {
		t.Fatal("original method R/owner T identities lost")
	}
	ctx := h.Ctx.Clone()
	ctx.currentClass, ctx.localScope = owner, caller
	invocation := findNode(caller.DeclarationNode, "method_invocation")
	if invocation == nil {
		t.Fatal("original instance invocation missing")
	}
	var actual inferredJavaTypeOrigin
	typ, known := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, h.File.Source, &actual)
	if !known || typ != "Foo" || !actual.unresolved || completeJavaTypeOrigin(actual) || actual.parameter != nil || actual.nominalScope != nil || actual.externalName != "" {
		t.Fatalf("undeclared instance Foo gained identity: %s %+v", typ, actual)
	}
	formal := methodFormalOrigins(identity, owner, nil, ctx)
	if len(formal) != 1 || formal[0].parameter != identity.TypeParameters[0].Declaration {
		t.Fatal("callee-owned R lost")
	}
	if _, _, applies, handled := declaredOriginConversion(actual, formal[0], ctx); !handled || applies {
		t.Fatal("unknown instance actual gained conversion")
	}
	target := resolveInvocationTarget(invocation.ChildByFieldName("object"), ctx, h.File.Source)
	if target == nil || target.classScope != owner {
		t.Fatal("valid source Box receiver disappeared")
	}
	if selected, _ := findBestMethodForInvocationTarget(target, "identity", invocation.ChildByFieldName("arguments"), true, false, ctx, h.File.Source); selected != nil {
		t.Fatal("unknown Foo must not select instance generic declaration")
	}
}

package transpiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestErasedSelfAliasSameDeclarationIsNotWidening(t *testing.T) {
	helper := setupParseHelper(t, `class Token {} final class Box<Token> { Token item; Token get(){return item;} }`)
	owner := helper.File.Symbols.FindClassScope("Box")
	if owner == nil || len(owner.TypeParameters) != 1 {
		t.Fatal("missing actual source binder")
	}
	parameter := owner.TypeParameters[0]
	if parameter.Declaration == nil || parameter.Name == parameter.EmittedName() {
		t.Fatal("control needs a real declaration with distinct source/emitted names")
	}
	ctx := classScopeCtx(owner, helper.Ctx)
	for _, actual := range []string{parameter.Name, parameter.EmittedName()} {
		for _, expected := range []string{parameter.Name, parameter.EmittedName()} {
			if javaDependentTypeParameterAssignable(actual, expected, ctx) {
				t.Errorf("same declaration aliases %s -> %s must not emit a distinct-slot bridge", actual, expected)
			}
		}
	}
}

func TestErasedSelfAliasDistinctDeclarationWideningPreserved(t *testing.T) {
	helper := setupParseHelper(t, `class Root {} class Other {} class B {} class T {}
class Outer<B extends Root,T extends B> { <B extends Other> void method(T value,B unrelated) {} }
class Chain<A extends Root,B extends A,C extends B> {}`)
	owner := helper.File.Symbols.FindClassScope("Outer")
	ctx := classScopeCtx(owner, helper.Ctx)
	ctx.localScope = owner.FindMethodByName("method", nil)
	if ctx.localScope == nil || len(ctx.localScope.TypeParameters) != 1 {
		t.Fatal("missing method-owned shadow")
	}
	outerB, outerT, methodB := owner.TypeParameters[0], owner.TypeParameters[1], ctx.localScope.TypeParameters[0]
	if outerB.Declaration == methodB.Declaration {
		t.Fatal("distinct declarations collapsed")
	}
	if !javaDependentTypeParameterAssignable(outerT.EmittedName(), outerB.EmittedName(), ctx) {
		t.Fatal("genuine T extends outer B widening lost")
	}
	if javaDependentTypeParameterAssignable(outerT.EmittedName(), methodB.EmittedName(), ctx) {
		t.Fatal("unrelated method B stole outer bound")
	}
	chain := helper.File.Symbols.FindClassScope("Chain")
	chainCtx := classScopeCtx(chain, helper.Ctx)
	if !javaDependentTypeParameterAssignable(chain.TypeParameters[2].EmittedName(), chain.TypeParameters[0].EmittedName(), chainCtx) {
		t.Fatal("transitive C extends B extends A widening lost")
	}
}

func TestErasedSelfAliasRankAndUnrelatedRefusal(t *testing.T) {
	helper := setupParseHelper(t, `class Token {} class Box<Token> {}`)
	owner := helper.File.Symbols.FindClassScope("Box")
	ctx := classScopeCtx(owner, helper.Ctx)
	parameter := owner.TypeParameters[0]
	for _, pair := range [][2]string{
		{parameter.Name + "[]", parameter.EmittedName()},
		{parameter.Name, parameter.EmittedName() + "[]"},
		{"Missing", parameter.EmittedName()},
		{"java.lang.String", parameter.EmittedName()},
		{"Box<String>", parameter.EmittedName()},
	} {
		if javaDependentTypeParameterAssignable(pair[0], pair[1], ctx) {
			t.Errorf("unsupported or unrelated %q -> %q admitted", pair[0], pair[1])
		}
	}
}

func runErasedSelfAliasFullProject(t *testing.T, fixture, mainClass string) {
	t.Helper()
	root := filepath.Join("testdata", "erased_self_alias", fixture)
	files := map[string]string{"pom.xml": "<project><modelVersion>4.0.0</modelVersion><groupId>probe</groupId><artifactId>erased-self-alias</artifactId><version>1</version></project>"}
	if err := filepath.WalkDir(filepath.Join(root, "src"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectObservations(t, files, mainClass, []campaignCompilerProjectObservation{{name: "repeat1", stdout: string(want)}, {name: "repeat2", stdout: string(want)}, {name: "repeat3", stdout: string(want)}})
}

func TestErasedSelfAliasOriginalShadowJDK21(t *testing.T) {
	runErasedSelfAliasFullProject(t, "original-shadow", "anonymousshadow.Shadow")
}

func TestErasedSelfAliasFullControlJDK21(t *testing.T) {
	runErasedSelfAliasFullProject(t, "full-control", "erasedalias.Main")
}

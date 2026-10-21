package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"os"
	"path/filepath"
	"testing"
)

func binderOwnershipSources(t *testing.T) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, name := range []string{"BinderOwnershipOracle.java", "Cell.java"} {
		bytes, err := os.ReadFile(filepath.Join("testdata", "class_owner_binder129", name))
		if err != nil {
			t.Fatal(err)
		}
		files[name] = string(bytes)
	}
	return files
}

func TestRawGenericParameterDeclarationBindersAreNotSourceClasses(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, binderOwnershipSources(t))
	owner := findQualifiedSourceClass("binderproof.BinderOwnershipOracle")
	ctx = classScopeCtx(owner, ctx)
	for _, name := range []string{"method", "bounded", "binderArray", "importedBinder"} {
		t.Run(name, func(t *testing.T) {
			methods := owner.FindMethod().ByOriginalName(name)
			if len(methods) != 1 {
				t.Fatal("missing original method")
			}
			method := methods[0]
			param := method.Parameters[0]
			declaration := method.TypeParameters[0].Declaration
			if param.DirectTypeParameter != declaration || param.TypeParameterBindings[method.TypeParameters[0].Name] != declaration {
				t.Fatal("source binder lost declaration identity")
			}
			before := param.OriginalType
			synthetic, rewritten := synthesizeRawGenericFunctionParameters(method, ctx)
			if len(synthetic) != 0 || len(rewritten) != 0 {
				t.Fatalf("bound source variable rewritten as nominal class: %#v %#v", synthetic, rewritten)
			}
			if param.OriginalType != before || param.DirectTypeParameter != declaration {
				t.Fatal("synthesis mutated original binder")
			}
			legacyParam := *param
			legacyParam.DirectTypeParameter = nil
			legacyParam.TypeParameterBindings = nil
			legacy := *method
			legacy.Parameters = []*symbol.Definition{&legacyParam}
			synthetic, rewritten = synthesizeRawGenericFunctionParameters(&legacy, ctx)
			if len(synthetic) != 0 || len(rewritten) != 0 {
				t.Fatalf("method declaration fallback lost legacy variable: %#v %#v", synthetic, rewritten)
			}
		})
	}
}

func TestRawGenericParameterTrueSourceClassesKeepSynthesis(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, binderOwnershipSources(t))
	owner := findQualifiedSourceClass("binderproof.BinderOwnershipOracle")
	ctx = classScopeCtx(owner, ctx)
	for _, name := range []string{"raw", "wildcard", "sourceArray", "qualified", "importedRaw", "importedWildcard"} {
		t.Run(name, func(t *testing.T) {
			method := owner.FindMethod().ByOriginalName(name)[0]
			param := method.Parameters[0]
			if param.DirectTypeParameter != nil {
				t.Fatal("real source formal acquired a direct binder")
			}
			before := param.OriginalType
			_, rewritten := synthesizeRawGenericFunctionParameters(method, ctx)
			if rewritten[param.OriginalName] == "" {
				t.Fatal("raw/wildcard source synthesis was suppressed")
			}
			if param.OriginalType != before || param.DirectTypeParameter != nil {
				t.Fatal("synthesis mutated nominal source definition")
			}
		})
	}
}

func TestRawGenericParameterClassAndMethodShadowIdentity(t *testing.T) {
	ctx := sourceGenericViewDemandTestContext(t, binderOwnershipSources(t))
	owner := findQualifiedSourceClass("binderproof.BinderOwnershipOracle.Holder")
	ctx = classScopeCtx(owner, ctx)
	echo := owner.FindMethod().ByOriginalName("echo")[0]
	shadow := owner.FindMethod().ByOriginalName("shadow")[0]
	classDeclaration := owner.TypeParameters[0].Declaration
	methodDeclaration := shadow.TypeParameters[0].Declaration
	if classDeclaration == methodDeclaration || echo.Parameters[0].DirectTypeParameter != classDeclaration || shadow.Parameters[0].DirectTypeParameter != methodDeclaration {
		t.Fatal("same-named class/method declarations collapsed")
	}
	for _, method := range []*symbol.Definition{echo, shadow} {
		synthetic, rewritten := synthesizeRawGenericFunctionParameters(method, ctx)
		if len(synthetic) != 0 || len(rewritten) != 0 {
			t.Fatal("instance class/method binder changed")
		}
	}
}

func TestRawGenericParameterBinderFullJDK21Parity(t *testing.T) {
	files := map[string]string{"pom.xml": `<project><modelVersion>4.0.0</modelVersion><groupId>binderproof</groupId><artifactId>declaration-identity</artifactId><version>1</version></project>`}
	for name, source := range binderOwnershipSources(t) {
		files[filepath.Join("src/main/java", name)] = source
	}
	expected, err := os.ReadFile(filepath.Join("testdata", "class_owner_binder129", "expected.stdout"))
	if err != nil {
		t.Fatal(err)
	}
	runCampaignCompilerStrictProjectOracle(t, files, "binderproof.BinderOwnershipOracle", string(expected))
}

package transpiler

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestSyntheticInheritedDeclaringPackageGuards(t *testing.T) {
	ctx, scope := reader132ActualLateScope(t)
	if pkg, known := syntheticInheritedDeclaringPackage(scope, ctx); !known || pkg != ctx.currentFile.Package {
		t.Fatal("actual registry lost known declaring package", pkg, known)
	}
	if pkg, known := syntheticInheritedDeclaringPackage(scope.Enclosing, ctx); !known || pkg != ctx.currentFile.Package {
		t.Fatal("ordinary declaration lost package", pkg, known)
	}
	copied := *scope
	if _, known := syntheticInheritedDeclaringPackage(&copied, ctx); known {
		t.Fatal("copied synthetic scope acquired package")
	}
	missing := ctx.Clone()
	missing.anonymousClasses = nil
	if _, known := syntheticInheritedDeclaringPackage(scope, missing); known {
		t.Fatal("missing registry acquired package")
	}
	wrong := ctx.Clone()
	foreign := *ctx.currentFile
	wrong.currentFile = &foreign
	if _, known := syntheticInheritedDeclaringPackage(scope, wrong); known {
		t.Fatal("wrong current file acquired package")
	}
	if _, known := syntheticInheritedDeclaringPackage(nil, ctx); known {
		t.Fatal("nil scope acquired unnamed package")
	}
}

func TestSyntheticInheritedMethodNonOverrideGuards(t *testing.T) {
	ctx, scope := reader132ActualLateScope(t)
	for _, tc := range []struct {
		name   string
		mutate func(*symbol.Definition, *symbol.Definition)
	}{
		{"private-source", func(m, i *symbol.Definition) { m.IsPrivate = true }},
		{"static-source", func(m, i *symbol.Definition) { m.IsStatic = true }},
		{"private-ancestor", func(m, i *symbol.Definition) { i.IsPrivate = true }},
		{"static-ancestor", func(m, i *symbol.Definition) { i.IsStatic = true }},
		{"constructor", func(m, i *symbol.Definition) { m.Constructor = true }},
		{"different-name", func(m, i *symbol.Definition) { i.OriginalName = "other" }},
		{"different-arity", func(m, i *symbol.Definition) { i.Parameters = []*symbol.Definition{{OriginalType: "Object"}} }},
		{"different-type", func(m, i *symbol.Definition) {
			m.Parameters = []*symbol.Definition{{OriginalType: "int"}}
			i.Parameters = []*symbol.Definition{{OriginalType: "Object"}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &symbol.Definition{OriginalName: "choose", OriginalType: "int"}
			i := &symbol.Definition{OriginalName: "choose", OriginalType: "int"}
			tc.mutate(m, i)
			if syntheticInheritedMethodOverrides(scope, m, scope.Enclosing, i, ctx) {
				t.Fatal("nonoverride admitted")
			}
		})
	}
}

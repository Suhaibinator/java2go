package symbol_test

import (
	"github.com/NickyBoy89/java2go/symbol"
	"testing"
)

func TestResolveDefinitionActualLexicalOwner(t *testing.T) {
	files := registeredResolutionFiles(t, map[string]string{
		"app/Use.java":         "package app;\nimport foreign.Integer;\nclass Sibling {static class Integer {}}\npublic class Use {\n Integer imported(Integer value){return value;}\n static class LocalOwner {\n  static class Integer {}\n  Integer enclosing(Integer value){return value;}\n }\n public static void main(String[] args){System.out.println(new Use().imported(new foreign.Integer()).getClass().getName()+\":\"+new LocalOwner().enclosing(new LocalOwner.Integer()).getClass().getName());}\n}\n",
		"foreign/Integer.java": "package foreign;\npublic class Integer {}\n",
	})
	file := files["app/Use.java"]
	use := file.FindClassScope("Use")
	local := use.FindClassScope("LocalOwner")
	imported := files["foreign/Integer.java"].TopLevelClasses[0]
	enclosing := local.Subclasses[0]
	for _, row := range []struct {
		name          string
		owner, target *symbol.ClassScope
		method        string
	}{
		{"sibling_nested_must_not_shadow_import", use, imported, "imported"},
		{"actual_enclosing_nested_shadows_import", local, enclosing, "enclosing"},
	} {
		t.Run(row.name, func(t *testing.T) {
			defer registeredResolutionMustNotPanic(t)
			method := row.owner.FindMethodByName(row.method, nil)
			if method == nil || method.DeclarationNode == nil || len(method.Parameters) != 1 {
				t.Fatal("actual method/parameter declaration missing")
			}
			formal := method.Parameters[0]
			if formal.OriginalType != "Integer" {
				t.Fatal("actual retained Java formal changed")
			}
			if !symbol.ResolveDefinition(formal, file) || formal.Type != "*"+row.target.Class.Name {
				t.Fatalf("resolved %q; expected actual lexical/imported declaration %p named %q", formal.Type, row.target, row.target.Class.Name)
			}
		})
	}
	t.Run("unknown_owner_does_not_borrow_sibling_nested", func(t *testing.T) {
		defer registeredResolutionMustNotPanic(t)
		original := use.FindMethodByName("imported", nil).Parameters[0]
		detached := *original
		before := detached.Type
		if symbol.ResolveDefinition(&detached, file) || detached.Type != before {
			t.Fatal("unowned formal claimed a nested or imported declaration")
		}
	})
}

func TestRegisteredClassLookupCyclesAndAmbiguity(t *testing.T) {
	files := registeredResolutionFiles(t, map[string]string{"p/Outer.java": `package p;public class Outer {static class Inner {}}`, "p/Other.java": `package p;public class Other {static class Inner {}}`})
	pkg := symbol.GlobalScope.FindPackage("p")
	t.Run("distinct_nested_members_are_ambiguous", func(t *testing.T) {
		if pkg.FindClass("Inner") != nil {
			t.Fatal("ambiguous member name resolved by iteration order")
		}
	})
	t.Run("cyclic_subclasses_terminate_missing_lookup", func(t *testing.T) {
		root := files["p/Outer.java"].BaseClass
		original := root.Subclasses
		defer func() { root.Subclasses = original }()
		root.Subclasses = append(root.Subclasses, root)
		if pkg.FindClass("Missing") != nil {
			t.Fatal("cycle invented member")
		}
	})
	t.Run("repeated_scope_is_one_declaration", func(t *testing.T) {
		root := files["p/Outer.java"].BaseClass
		file := files["p/Outer.java"]
		original := file.TopLevelClasses
		defer func() { file.TopLevelClasses = original }()
		file.TopLevelClasses = append(file.TopLevelClasses, root)
		if pkg.FindClass("Outer") != root {
			t.Fatal("same registered pointer became ambiguous")
		}
	})
	t.Run("converted_primitive_name_is_not_Java_identity", func(t *testing.T) {
		local := registeredResolutionFiles(t, map[string]string{"q/Use.java": `package q;class int32 {} public class Use {void accept(int value){}}`})["q/Use.java"]
		formal := local.FindClassScope("Use").FindMethodByName("accept", nil).Parameters[0]
		before := formal.Type
		if symbol.ResolveDefinition(formal, local) || formal.Type != before {
			t.Fatal("primitive display name matched source class")
		}
	})
}

package transpiler

import (
	"os"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestGenericDateBoundPhysicalRepresentation(t *testing.T) {
	for _, test := range []struct {
		name, source, erasure, physical string
		refused                         bool
	}{
		{"canonical_import", `import java.util.Date; class Child<D extends Date>{D stored;}`, "java.util.Date", "DateValue", false},
		{"canonical_qualified", `class Child<D extends java.util.Date>{D stored;}`, "java.util.Date", "DateValue", false},
		{"source_shadow", `class Holder{abstract static class Date{} static class Child<D extends Date>{D stored;}}`, "", "", true},
		{"binder_shadow", `class Holder<Date>{class Child<D extends Date>{D stored;}}`, "java.lang.Object", "any", false},
		{"foreign_sql_bound", `import java.sql.Date;class Child<D extends Date>{D stored;}`, "", "", true},
		{"unknown_external", `class Child<D extends java.net.URI>{D stored;}`, "", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			helper := setupParseHelper(t, test.source)
			var child *symbol.ClassScope
			for _, scope := range allSourceClassScopes() {
				if scope.Class != nil && scope.Class.OriginalName == "Child" {
					child = scope
					break
				}
			}
			if child == nil {
				t.Fatal("missing exact source Child declaration")
			}
			plan := &genericFamilyPlan{representations: map[*symbol.TypeParamDeclaration]genericFamilyBinderRepresentation{}, binders: map[*symbol.TypeParamDeclaration]struct{}{}}
			err := plan.addBinderRepresentations(child, helper.Ctx)
			if test.refused {
				if err == nil {
					t.Fatal("unproven source/foreign bound admitted")
				}
				return
			}
			if err != nil {
				t.Fatalf("declared native bound representation: %v", err)
			}
			parameters := child.OwnTypeParameters()
			if len(parameters) != 1 || parameters[0].Declaration == nil {
				t.Fatal("missing own binder identity")
			}
			representation, ok := plan.representations[parameters[0].Declaration]
			if !ok || representation.erasure != test.erasure || representation.goName != test.physical || representation.sourceBound != nil {
				t.Fatalf("wrong declaration-owned representation: %#v", representation)
			}
		})
	}
}

func TestGenericDateBoundFamilyFactoryAdmission(t *testing.T) {
	source, err := os.ReadFile("testdata/generic_date_bound/Main.java")
	if err != nil {
		t.Fatal(err)
	}
	helper := setupParseHelper(t, string(source))
	base := helper.File.Symbols.FindClassScope("Base")
	child := helper.File.Symbols.FindClassScope("DateChild")
	factory := helper.File.Symbols.FindClassScope("Factory")
	if base == nil || child == nil || factory == nil {
		t.Fatal("missing exact fixture declarations")
	}
	plan, err := planGenericFamily(base, helper.Ctx)
	if err != nil || plan == nil {
		t.Fatalf("Date-bearing source family requires complete physical plan: %v", err)
	}
	if _, ok := plan.members[child]; !ok {
		t.Fatal("Date-bearing subtype escaped family closure")
	}
	parameters := child.OwnTypeParameters()
	if len(parameters) != 1 || plan.representations[parameters[0].Declaration].goName != "DateValue" {
		t.Fatal("Date subtype storage lost native reference protocol")
	}
	var method *symbol.Definition
	for _, candidate := range factory.Methods {
		if candidate.OriginalName == "create" {
			method = candidate
		}
	}
	if method == nil || len(method.TypeParameters) != 1 || method.TypeParameters[0].Declaration == nil || !strings.Contains(method.OriginalType, "Base<") {
		t.Fatal("missing generic factory declaration")
	}
	if !canonicalGenericMethodSignature(method, classScopeCtx(factory, helper.Ctx)) {
		t.Fatal("generic factory carrier refused after complete family proof")
	}
}

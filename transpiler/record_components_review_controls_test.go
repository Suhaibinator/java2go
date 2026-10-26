package transpiler

import (
	"bytes"
	"go/printer"
	"go/token"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestRecordComponentReviewCarriedBoundMutation(t *testing.T) {
	h := setupParseHelper(t, `class Host {record R<T extends Number>(T count){}}`)
	owner := h.File.Symbols.FindClassScope("R")
	ctx := classScopeCtx(owner, h.Ctx)
	before, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("owned bounded record profile rejected before mutation")
	}
	var old bytes.Buffer
	if err := printer.Fprint(&old, token.NewFileSet(), before[0].typ); err != nil {
		t.Fatal(err)
	}
	owner.TypeParameters[0].Bounds = []symbol.JavaType{{Original: "Object"}}
	after, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if known {
		var changed bytes.Buffer
		if err := printer.Fprint(&changed, token.NewFileSet(), after[0].typ); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("mutable carried Java bound was accepted: before=%s after=%s", old.String(), changed.String())
	}
}
func TestRecordComponentReviewNominalImportMutation(t *testing.T) {
	f := demandPredicate170Setup(t, map[string]string{
		"a/Token.java":   `package a; public class Token {}`,
		"b/Token.java":   `package b; public class Token {}`,
		"records/R.java": `package records; import a.Token; public record R(Token token){}`,
	}, true)
	var owner *symbol.ClassScope
	var file *symbol.FileScope
	for _, source := range f.files {
		if source.Symbols.Package == "records" {
			file = source.Symbols
			owner = file.FindClassScope("R")
		}
	}
	if owner == nil {
		t.Fatal("record declaration missing")
	}
	ctx := classScopeCtx(owner, f.ctx)
	before, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		t.Fatal("owned imported component rejected before mutation")
	}
	var old bytes.Buffer
	if err := printer.Fprint(&old, token.NewFileSet(), before[0].typ); err != nil {
		t.Fatal(err)
	}
	file.Imports["Token"] = "b"
	after, known := sourceReflectionRecordComponentPlan(owner, ctx)
	if !known {
		return
	} // Refusal is a safe result of a mutated derived namespace.
	var changed bytes.Buffer
	if err := printer.Fprint(&changed, token.NewFileSet(), after[0].typ); err != nil {
		t.Fatal(err)
	}
	if changed.String() != old.String() {
		t.Fatalf("mutable import rebound original header type: before=%s after=%s", old.String(), changed.String())
	}
}

func TestRecordComponentReviewCarriedParameterIdentity(t *testing.T) {
	for _, mutation := range []string{"name", "declaration", "source-name", "bound-binding", "removed", "extra", "go-rename"} {
		t.Run(mutation, func(t *testing.T) {
			h := setupParseHelper(t, `class Host {record R<T extends Number>(T count){}}`)
			owner := h.File.Symbols.FindClassScope("R")
			ctx := classScopeCtx(owner, h.Ctx)
			if _, known := sourceReflectionRecordComponentPlan(owner, ctx); !known {
				t.Fatal("genuine profile rejected")
			}
			switch mutation {
			case "name":
				owner.TypeParameters[0].Name = "Other"
			case "declaration":
				copied := *owner.TypeParameters[0].Declaration
				owner.TypeParameters[0].Declaration = &copied
			case "source-name":
				owner.TypeParameters[0].Declaration.SourceName = "Other"
			case "bound-binding":
				owner.TypeParameters[0].Bounds[0].TypeParameterBindings = map[string]*symbol.TypeParamDeclaration{"Number": owner.TypeParameters[0].Declaration}
			case "removed":
				owner.TypeParameters = nil
			case "extra":
				owner.TypeParameters = append(owner.TypeParameters, owner.TypeParameters[0])
			case "go-rename":
				owner.TypeParameters[0].Declaration.GoName = "RenamedJavaBinder"
			}
			_, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if known != (mutation == "go-rename") {
				t.Fatalf("profile known=%v mutation=%s", known, mutation)
			}
		})
	}
}

func TestRecordComponentReviewNamespaceCurrentness(t *testing.T) {
	for _, mutation := range []string{"unchanged", "missing-import", "extra-import", "package", "go-name"} {
		t.Run(mutation, func(t *testing.T) {
			h := setupParseHelper(t, `package records; import java.lang.String; import java.util.*; import static java.lang.Math.*; enum Host {ONE; record R(String text){}}`)
			owner := h.File.Symbols.FindClassScope("R")
			ctx := classScopeCtx(owner, h.Ctx)
			if _, known := sourceReflectionRecordComponentPlan(owner, ctx); !known {
				t.Fatal("genuine enum-nested namespace rejected")
			}
			switch mutation {
			case "missing-import":
				delete(h.File.Symbols.Imports, "String")
			case "extra-import":
				h.File.Symbols.Imports["Other"] = "foreign"
			case "package":
				h.File.Symbols.Package = "foreign"
			case "go-name":
				owner.Class.Name = "RenamedRecord"
			}
			_, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if known != (mutation == "unchanged" || mutation == "go-name") {
				t.Fatalf("namespace known=%v mutation=%s", known, mutation)
			}
		})
	}
}

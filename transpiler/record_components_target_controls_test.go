package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/printer"
	"go/token"
	"testing"
)

func TestRecordComponentReviewNominalTargetPackageMutation(t *testing.T) {
	for _, header := range []string{"Token token", "T token"} {
		t.Run(header, func(t *testing.T) {
			parameters := ""
			if header == "T token" {
				parameters = "<T extends Token>"
			}
			f := demandPredicate170Setup(t, map[string]string{"a/Token.java": `package a;public class Token{}`, "b/Token.java": `package b;public class Token{}`, "records/R.java": `package records;import a.Token;public record R` + parameters + `(` + header + `){}`}, true)
			var owner *symbol.ClassScope
			var target *symbol.FileScope
			for _, file := range f.files {
				if file.Symbols.Package == "records" {
					owner = file.Symbols.FindClassScope("R")
				}
				if file.Symbols.Package == "a" {
					target = file.Symbols
				}
			}
			if owner == nil || target == nil {
				t.Fatal("genuine graph missing")
			}
			ctx := classScopeCtx(owner, f.ctx)
			before, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if !known {
				t.Fatal("genuine nominal origin rejected")
			}
			var old bytes.Buffer
			if err := printer.Fprint(&old, token.NewFileSet(), before[0].typ); err != nil {
				t.Fatal(err)
			}
			target.Package = "b"
			after, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if !known {
				return
			}
			var changed bytes.Buffer
			if err := printer.Fprint(&changed, token.NewFileSet(), after[0].typ); err != nil {
				t.Fatal(err)
			}
			if changed.String() != old.String() {
				t.Fatalf("mutated nominal target changed descriptor: before=%s after=%s", old.String(), changed.String())
			}
		})
	}
}

func TestRecordComponentReviewNominalTargetOwnership(t *testing.T) {
	for _, mutation := range []string{"genuine", "unused-file", "go-name", "declaration-donation", "source-stale"} {
		t.Run(mutation, func(t *testing.T) {
			f := demandPredicate170Setup(t, map[string]string{"a/Token.java": `package a;public class Token{}`, "b/Token.java": `package b;public class Token{}`, "records/R.java": `package records;import a.Token;public record R<T extends U,U extends Token>(T token){}`}, true)
			var owner, target, other *symbol.ClassScope
			var targetFile, otherFile *symbol.FileScope
			for _, file := range f.files {
				switch file.Symbols.Package {
				case "records":
					owner = file.Symbols.FindClassScope("R")
				case "a":
					target = file.Symbols.FindClassScope("Token")
					targetFile = file.Symbols
				case "b":
					other = file.Symbols.FindClassScope("Token")
					otherFile = file.Symbols
				}
			}
			if owner == nil || target == nil || other == nil {
				t.Fatal("source graph incomplete")
			}
			ctx := classScopeCtx(owner, f.ctx)
			if _, known := sourceReflectionRecordComponentPlan(owner, ctx); !known {
				t.Fatal("genuine chained bound rejected")
			}
			switch mutation {
			case "unused-file":
				otherFile.Package = "unrelated"
			case "go-name":
				target.Class.Name = "RenamedToken"
			case "declaration-donation":
				target.Class.DeclarationNode = other.Class.DeclarationNode
			case "source-stale":
				targetFile.Source[0] ^= 1
			}
			_, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if known != (mutation == "genuine" || mutation == "unused-file" || mutation == "go-name") {
				t.Fatalf("nominal known=%v mutation=%s", known, mutation)
			}
		})
	}
}

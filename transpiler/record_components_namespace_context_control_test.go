package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/printer"
	"go/token"
	"testing"
)

func TestRecordComponentReviewCallerRegistryNamespace(t *testing.T) {
	for _, header := range []string{"Token token", "T token"} {
		for _, registry := range []string{"anonymous", "local"} {
			t.Run(header+"/"+registry, func(t *testing.T) {
				parameters := ""
				if header == "T token" {
					parameters = "<T extends Token>"
				}
				f := demandPredicate170Setup(t, map[string]string{"a/Token.java": `package a;public class Token{}`, "b/Token.java": `package b;public class Token{}`, "records/R.java": `package records;import a.Token;public record R` + parameters + `(` + header + `){}`}, true)
				var owner, foreign *symbol.ClassScope
				for _, file := range f.files {
					if file.Symbols.Package == "records" {
						owner = file.Symbols.FindClassScope("R")
					}
					if file.Symbols.Package == "b" {
						foreign = file.Symbols.FindClassScope("Token")
					}
				}
				if owner == nil || foreign == nil {
					t.Fatal("source graph missing")
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
				if registry == "anonymous" {
					ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{{}: {structName: "Token", scope: foreign}}
				} else {
					ctx.localClasses = map[string]*localClassInfo{"Token": {scope: foreign}}
				}
				after, known := sourceReflectionRecordComponentPlan(owner, ctx)
				if !known {
					return
				}
				var changed bytes.Buffer
				if err := printer.Fprint(&changed, token.NewFileSet(), after[0].typ); err != nil {
					t.Fatal(err)
				}
				if changed.String() != old.String() {
					t.Fatalf("caller registry replaced imported declaration: before=%s after=%s", old.String(), changed.String())
				}
			})
		}
	}
}

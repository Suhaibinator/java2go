package transpiler

import (
	"bytes"
	"github.com/NickyBoy89/java2go/symbol"
	"go/printer"
	"go/token"
	"testing"
)

func TestRecordComponentReviewNominalRegistryIdentity(t *testing.T) {
	for _, header := range []string{"Token token", "T token"} {
		t.Run(header, func(t *testing.T) {
			parameters := ""
			if header == "T token" {
				parameters = "<T extends Token>"
			}
			f := demandPredicate170Setup(t, map[string]string{"a/Token.java": `package a;public class Token{}`, "records/R.java": `package records;import a.Token;public record R` + parameters + `(` + header + `){}`}, true)
			var owner, target *symbol.ClassScope
			for _, file := range f.files {
				if file.Symbols.Package == "records" {
					owner = file.Symbols.FindClassScope("R")
				}
				if file.Symbols.Package == "a" {
					target = file.Symbols.FindClassScope("Token")
				}
			}
			if owner == nil || target == nil {
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
			ctx.localClasses = map[string]*localClassInfo{"unrelated": {scope: target, dynamicTypeID: "foreign.Counterfeit"}}
			after, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if !known {
				return
			}
			var changed bytes.Buffer
			if err := printer.Fprint(&changed, token.NewFileSet(), after[0].typ); err != nil {
				t.Fatal(err)
			}
			if changed.String() != old.String() {
				t.Fatalf("local registry replaced named identity: before=%s after=%s", old.String(), changed.String())
			}
		})
	}
}

func TestRecordComponentReviewNamedRegistryConflicts(t *testing.T) {
	for _, mode := range []string{"matching", "empty", "unrelated", "mixed-conflict"} {
		t.Run(mode, func(t *testing.T) {
			f := demandPredicate170Setup(t, map[string]string{"a/Token.java": `package a;public class Token{}`, "records/R.java": `package records;import a.Token;public record R(Token token){}`}, true)
			var owner, target *symbol.ClassScope
			for _, file := range f.files {
				if file.Symbols.Package == "records" {
					owner = file.Symbols.FindClassScope("R")
				}
				if file.Symbols.Package == "a" {
					target = file.Symbols.FindClassScope("Token")
				}
			}
			ctx := classScopeCtx(owner, f.ctx)
			if _, known := sourceReflectionRecordComponentPlan(owner, ctx); !known {
				t.Fatal("genuine profile rejected")
			}
			ctx.localClasses = map[string]*localClassInfo{"matching": {scope: target, dynamicTypeID: "a.Token"}}
			switch mode {
			case "empty":
				ctx.localClasses["matching"].dynamicTypeID = ""
			case "unrelated":
				ctx.localClasses["other"] = &localClassInfo{scope: owner, dynamicTypeID: "unrelated.Foreign"}
			case "mixed-conflict":
				ctx.localClasses["other"] = &localClassInfo{scope: target, dynamicTypeID: "foreign.Counterfeit"}
			}
			_, known := sourceReflectionRecordComponentPlan(owner, ctx)
			if known != (mode != "mixed-conflict") {
				t.Fatalf("registry known=%v mode=%s", known, mode)
			}
		})
	}
}

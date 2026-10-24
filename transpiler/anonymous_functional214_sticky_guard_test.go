package transpiler

import (
	"go/ast"
	"reflect"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

// Every listed deletion/donation starts from an actual completed anonymous
// allocation. The negative origin fact must precede all mutable routing data.
func TestCI214AnonymousFunctionalStickyOriginRouting(t *testing.T) {
	for _, family := range []string{"Function", "Supplier"} {
		for _, mode := range []string{"plain removal", "named local donation", "named graph donation", "copied scope local donation", "copied info local donation", "copied certificate local donation", "unmarked local impersonation", "unmarked graph impersonation"} {
			t.Run(family+"/"+mode, func(t *testing.T) {
				source, header := anonymousFunctional214Source(family)
				ctx, _, _, original := anonymousFunctional209Lower(t, source, header)
				certificate := original.functional
				contract := anonymousFunctionalContract(certificate)
				if !original.scope.HasAnonymousSourceOrigin() {
					t.Fatal("actual anonymous allocation was not marked before publication")
				}
				copied := *original.scope
				if !copied.HasAnonymousSourceOrigin() {
					t.Fatal("scope value copy lost negative origin")
				}
				key, _ := anonymousClassSourceKey(certificate.allocation)
				delete(ctx.anonymousClasses, key)
				retained, scope := original, original.scope
				donor := ctx.currentFile.FindClassScope("Donor")
				if donor == nil || donor.HasAnonymousSourceOrigin() {
					t.Fatal("actual named donor premise failed")
				}
				switch mode {
				case "plain removal":
				case "named local donation":
					scope.Class.DeclarationNode = donor.Class.DeclarationNode
					ctx.localClasses["donor"] = &localClassInfo{scope: scope}
				case "named graph donation":
					scope.Class.DeclarationNode = donor.Class.DeclarationNode
					scope.Class.OriginalName = "Donor"
					ctx.currentFile.TopLevelClasses[0] = scope
				case "copied scope local donation":
					copyClass := *scope.Class
					copied.Class = &copyClass
					copied.Class.DeclarationNode = donor.Class.DeclarationNode
					scope = &copied
					copyInfo := *original
					copyInfo.scope = scope
					retained = &copyInfo
					ctx.localClasses["donor"] = &localClassInfo{scope: scope}
				case "copied info local donation":
					copyInfo := *original
					retained = &copyInfo
					scope.Class.DeclarationNode = donor.Class.DeclarationNode
					ctx.localClasses["donor"] = &localClassInfo{scope: scope}
				case "copied certificate local donation":
					copyCertificate := *certificate
					original.functional = &copyCertificate
					scope.Class.DeclarationNode = donor.Class.DeclarationNode
					ctx.localClasses["donor"] = &localClassInfo{scope: scope}
				case "unmarked local impersonation", "unmarked graph impersonation":
					// This new literal is not a value copy of the marked scope. Unmarkedness
					// still grants no named declaration: a cache/graph donation is insufficient.
					scope = &symbol.ClassScope{Class: &symbol.Definition{OriginalName: "Donor", Name: "Donor", DeclarationNode: donor.Class.DeclarationNode}, Enclosing: ctx.currentClass, ImplementedInterfaces: append([]string(nil), original.scope.ImplementedInterfaces...), Methods: append([]*symbol.Definition(nil), original.scope.Methods...), TypeParameters: append([]symbol.TypeParam(nil), original.scope.TypeParameters...)}
					if mode == "unmarked local impersonation" {
						ctx.localClasses["Donor"] = &localClassInfo{scope: scope, structName: "Donor"}
					} else {
						ctx.currentFile.TopLevelClasses[0] = scope
					}
					copyInfo := *original
					copyInfo.scope = scope
					retained = &copyInfo
				}
				before := anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls})
				beforeCounter := *ctx.anonClassCounter
				beforeMap := map[anonymousClassKey]*anonymousClassInfo{}
				for k, v := range ctx.anonymousClasses {
					beforeMap[k] = v
				}
				query := ctx.Clone()
				query.currentClass = scope
				query.localScope = nil
				if sourceNativeFunctionalContracts(scope, query) != nil || sourceNativeFunctionalViewExpr(scope, contract, ast.NewIdent("receiver"), query) != nil {
					t.Fatal("removed-cache or donated scope acquired named native fallback")
				}
				if _, owned := supplierSourceDeclarationCtx(scope, query); owned {
					t.Fatal("donation acquired named declaration ownership")
				}
				if supplierSourceSelectedMethodOwned(scope, scope, certificate.selected, contract, query) {
					t.Fatal("donation acquired selected Supplier body")
				}
				if anonymousClassConstructionExpr(certificate.allocation, certificate.typ, retained, nil, ctx.currentFile.Source, ctx) != nil {
					t.Fatal("retained poisoned scope constructed after cache removal")
				}
				if anonymousFunctionalAST(&ast.File{Name: ast.NewIdent("snapshot"), Decls: *ctx.hoistedDecls}) != before || *ctx.anonClassCounter != beforeCounter || !reflect.DeepEqual(ctx.anonymousClasses, beforeMap) {
					t.Fatal("direct refusal changed hoists counter or cache")
				}
			})
		}
	}
}

// The AST-only entry can reprove the actual allocation after synthetic deletion.
// It must create new info/scope/certificate and cannot reuse/export the poisoned
// retained scope. This does not claim class-ID equivalence under cache corruption.
func TestCI214AnonymousFunctionalFreshReproofAfterRemoval(t *testing.T) {
	for _, family := range []string{"Function", "Supplier"} {
		for _, mode := range []string{"plain removal", "named local donation"} {
			t.Run(family+"/"+mode, func(t *testing.T) {
				source, header := anonymousFunctional214Source(family)
				ctx, _, _, old := anonymousFunctional209Lower(t, source, header)
				certificate := old.functional
				contract := anonymousFunctionalContract(certificate)
				key, _ := anonymousClassSourceKey(certificate.allocation)
				delete(ctx.anonymousClasses, key)
				if mode == "named local donation" {
					old.scope.Class.DeclarationNode = ctx.currentFile.FindClassScope("Donor").Class.DeclarationNode
					ctx.localClasses["donor"] = &localClassInfo{scope: old.scope}
				}
				if sourceNativeFunctionalViewExpr(old.scope, contract, ast.NewIdent("oldReceiver"), ctx) != nil {
					t.Fatal("old removed scope exported before reproof")
				}
				before := len(*ctx.hoistedDecls)
				counter := *ctx.anonClassCounter
				result := lowerAnonymousClassToStruct(certificate.allocation, certificate.typ, certificate.body, ctx.currentFile.Source, ctx)
				fresh := ctx.anonymousClasses[key]
				if result == nil || fresh == nil || fresh == old || fresh.scope == old.scope || fresh.functional == certificate || fresh.functional == nil || !fresh.scope.HasAnonymousSourceOrigin() || !fresh.functional.sealed || len(*ctx.hoistedDecls) <= before || *ctx.anonClassCounter != counter+1 {
					t.Fatal("AST entry did not build an entirely fresh proved allocation layout")
				}
				query := ctx.Clone()
				query.currentClass = fresh.scope
				query.localScope = nil
				if !anonymousFunctionalValid(fresh.functional, query, true) || !anonymousFunctionalEqual(fresh.scope.Class.DeclarationNode, certificate.allocation) || sourceNativeFunctionalViewExpr(fresh.scope, anonymousFunctionalContract(fresh.functional), ast.NewIdent("freshReceiver"), query) == nil {
					t.Fatal("fresh source/layout/session proof missing")
				}
				if sourceNativeFunctionalContracts(old.scope, query) != nil || sourceNativeFunctionalViewExpr(old.scope, contract, ast.NewIdent("oldReceiver"), query) != nil || anonymousClassConstructionExpr(certificate.allocation, certificate.typ, old, nil, ctx.currentFile.Source, ctx) != nil {
					t.Fatal("fresh reproof revived poisoned original scope")
				}
			})
		}
	}
}

func TestCI214AnonymousFunctionalActualLocalNamedRouting(t *testing.T) {
	for _, family := range []string{"Function", "Supplier"} {
		t.Run(family, func(t *testing.T) {
			edge, method := "java.util.function.Function<String,String>", `public String apply(String value){return value;}`
			if family == "Supplier" {
				edge, method = "java.util.function.Supplier<String>", `public String get(){return "value";}`
			}
			h := setupParseHelper(t, `class Main{static Object make(){class Local implements `+edge+`{`+method+`}return new Local();}}`)
			ctx := h.Ctx.Clone()
			ctx.currentClass = ctx.currentFile.FindClassScope("Main")
			ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("make")[0]
			ctx.className = ctx.currentClass.Class.Name
			var hoists []ast.Decl
			counter := 0
			ctx.hoistedDecls = &hoists
			ctx.anonClassCounter = &counter
			ctx.anonymousClasses = map[anonymousClassKey]*anonymousClassInfo{}
			ctx.localClasses = map[string]*localClassInfo{}
			ctx.executionContextName = "execution"
			node := findNode(ctx.localScope.DeclarationNode.ChildByFieldName("body"), "class_declaration")
			if node == nil {
				t.Fatal("actual local named declaration missing")
			}
			hoistLocalClass(node, h.File.Source, ctx)
			info := ctx.localClasses["Local"]
			if info == nil || info.scope == nil || info.scope.HasAnonymousSourceOrigin() || !anonymousFunctionalActualNamed(info.scope, ctx) {
				t.Fatal("actual local named declaration routing failed")
			}
			if _, anonymous := anonymousFunctionalLookup(info.scope, ctx); anonymous {
				t.Fatal("actual local named scope acquired anonymous origin")
			}
			query := classScopeCtx(info.scope, ctx)
			contracts := sourceNativeFunctionalContracts(info.scope, query)
			if family == "Function" && (len(contracts) != 1 || sourceNativeFunctionalViewExpr(info.scope, contracts[0], ast.NewIdent("receiver"), query) == nil) {
				t.Fatal("existing named local Function path changed")
			}
			// Supplier's existing named predicate requires direct registered-file
			// ownership. A local source scope does not acquire that authority from this
			// routing marker repair; its old predicate remains byteexact and in force.
			if family == "Supplier" {
				if _, owned := supplierSourceDeclarationCtx(info.scope, query); owned {
					t.Fatal("local named routing widened Supplier declaration ownership")
				}
			}
		})
	}
}

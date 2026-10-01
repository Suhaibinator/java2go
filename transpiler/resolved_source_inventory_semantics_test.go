package transpiler

import (
	"github.com/NickyBoy89/java2go/parsing"
	"testing"
)

func TestResolvedSourceInventory_SelectorsKeepOverloadsAndGraphLifetime(t *testing.T) {
	helper := setupParseHelper(t, `class Base {public void Collision(){} public void Collision(int value){} public void Collision0(){}} class Use {Object make(){return new Object(){public void Anonymous(){} public void Java2goInstallBaseSubobjectFrom42617365(){}};} void local(){class Local {public void LocalMember(){}}}}`)
	ctx := helper.Ctx.Clone()
	ctx.callableSubclasses = &callableSubclassSourceInventory{}
	for _, selector := range []string{"Collision", "Collision0", "Collision00", "CollisionJava2goExecution", "Anonymous", "LocalMember"} {
		if !sourceMethodIdentifierExists(selector) || !sourceMethodIdentifierExists(selector, ctx) {
			t.Fatalf("source selector %s disappeared from cached/uncached collision checks", selector)
		}
	}
	if sourceMethodIdentifierExists("Absent", ctx) {
		t.Fatal("invented source selector")
	}
	base := helper.File.Symbols.FindClassScope("Base")
	if classSubobjectInstallerName(base) != "Java2goInstallBaseSubobjectFrom426173651" || classSubobjectInstallerName(base) != classSubobjectInstallerName(base, ctx) {
		t.Fatal("structural inventory changed installer collision allocation")
	}
	clone := ctx.Clone()
	clone.genericFamilies = nil
	if !sourceMethodIdentifierExists("Anonymous", clone) {
		t.Fatal("analysis clone lost anonymous selector")
	}
	fresh := setupParseHelper(t, `class Base {public void Fresh(){}} class Use {}`)
	freshCtx := fresh.Ctx.Clone()
	freshCtx.callableSubclasses = ctx.callableSubclasses
	if sourceMethodIdentifierExists("Anonymous", freshCtx) || sourceMethodIdentifierExists("Collision", freshCtx) || !sourceMethodIdentifierExists("Fresh", freshCtx) {
		t.Fatal("separate source graph reused source selectors")
	}
}

func TestResolvedSourceInventory_DemandsKeepLexicalBindersAndHeaderContexts(t *testing.T) {
	for _, source := range []string{
		`package app;public class Use{static class p{static class Cell<T>{}} p.Cell<?> own;}`,
		`package app;public class Use{<p>void f(){p.Cell<?> unknown;}}`,
		`package app;public class Use extends p.Cell<String>{static class p{static class Cell<T>{}}}`,
		`package app;public class Use{void f(){class Cell<T>{}Cell<?> own;} p.Cell<?> packageView;}`,
	} {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{"p/Cell.java": `package p;public class Cell<T>{T value;}`, "app/Use.java": source})
		want := sourceGenericViewDemandSeeds(ctx)
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		for attempt := 0; attempt < 2; attempt++ {
			clone := ctx.Clone()
			clone.genericFamilies = nil
			got := sourceGenericViewDemandSeeds(clone)
			if len(got) != len(want) {
				t.Fatalf("demand count=%d want%d for %s", len(got), len(want), source)
			}
			for index, seed := range got {
				if seed != want[index] {
					t.Fatalf("cached demand changed source declaration identity for %s", source)
				}
			}
		}
	}
}

func TestResolvedSourceInventory_AnonymousParametersKeepCommentAndKeywordRoles(t *testing.T) {
	file := parsing.SourceFile{Name: "Use.java", Source: []byte(`class Use {public void Empty(){int type=0;} public void Commented(/*empty*/){int type=0;} public void Keywords(int type, /*between*/int type_){int local=type;}}`)}
	if err := file.ParseAST(); err != nil {
		t.Fatal(err)
	}
	methods := anonymousClassMethods(file.Ast.NamedChild(0).ChildByFieldName("body"))
	if len(methods) != 3 {
		t.Fatal("legal method AST fixture lost a declaration")
	}
	for _, method := range methods {
		name := method.ChildByFieldName("name").Content(file.Source)
		parameters := anonymousMethodParameters(method, file.Source)
		switch name {
		case "Empty", "Commented":
			if parameters != nil {
				t.Fatalf("empty/comment-only syntax invented parameters for %s", name)
			}
		case "Keywords":
			if len(parameters) != 2 || parameters[0].OriginalName != "type" || parameters[0].Name != "type__0" || parameters[1].OriginalName != "type_" || parameters[1].Name != "type_" {
				t.Fatalf("comment/keyword collision changed source parameter identities: %+v", parameters)
			}
		default:
			t.Fatalf("unexpected fixture method %s", name)
		}
	}
}

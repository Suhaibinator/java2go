package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestSourceQueryInventory_ProjectContextsKeepLifetimeAndLexicalSemantics(t *testing.T) {
	t.Run("file-binders-shadowing-and-fresh-admission", func(t *testing.T) {
		ctx := sourceGenericViewDemandTestContext(t, map[string]string{
			"p/Cell.java":  `package p;public class Cell<T>{T value;}`,
			"q/Cell.java":  `package q;public class Cell<T>{java.util.List<T> unsupported;}`,
			"app/Use.java": `package app;public class Use{p.Cell<?> allowed;q.Cell<?> rejected;<p>void shadow(){p.Cell<?> unknown;}void local(){class Cell<T>{}Cell<?> own;}}`,
		})
		allowed, rejected := findQualifiedSourceClass("p.Cell"), findQualifiedSourceClass("q.Cell")
		if allowed == nil || rejected == nil || allowed == rejected || allowed.TypeParameters[0].Declaration == rejected.TypeParameters[0].Declaration {
			t.Fatal("same-spelled source classes must retain distinct declarations")
		}
		want := sourceGenericViewDemandSeeds(ctx)
		if len(want) != 2 {
			t.Fatal("lexical shadows must not invent additional package demands")
		}
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		for _, active := range []*symbol.ClassScope{allowed, rejected} {
			fileCtx := ctx.Clone()
			fileCtx.currentFile = findFileScopeForClassScope(active)
			fileCtx.currentClass = active
			fileCtx.genericFamilies = &genericFamilyAnalysis{}
			got := sourceGenericViewDemandSeeds(fileCtx)
			if len(got) != len(want) {
				t.Fatal("shared syntax changed demand count across file contexts")
			}
			for index, seed := range got {
				if seed != want[index] {
					t.Fatal("shared syntax changed a lexical source declaration")
				}
			}
			if plan, err := planGenericFamily(allowed, fileCtx); err != nil || plan == nil {
				t.Fatalf("valid source family lost its fresh admission audit: %v", err)
			}
			if plan, err := planGenericFamily(rejected, fileCtx); err == nil || plan != nil {
				t.Fatal("structural source facts supplied authority to rejectable List storage")
			}
		}
	})
	t.Run("supplied-project-context-resets-on-fresh-graph", func(t *testing.T) {
		first := setupParseHelper(t, `class Base<T>{public T get(){return null;}}class Use{Base<String> make(){return new Base<String>(){public void OldSelector(){}};}}`)
		ctx := first.Ctx.Clone()
		ctx.callableSubclasses = &callableSubclassSourceInventory{}
		if _, err := convertFileNode(first.File, ctx); err != nil {
			t.Fatal(err)
		}
		if !sourceMethodIdentifierExists("OldSelector", ctx) || !classHasUnmodeledCallableSubclass(first.File.Symbols.FindClassScope("Base"), ctx) {
			t.Fatal("first project's anonymous source identity disappeared")
		}
		firstGraph := ctx.callableSubclasses.graph
		fresh := setupParseHelper(t, `class Base<T>{public T get(){return null;}public void FreshSelector(){}}`)
		freshCtx := fresh.Ctx.Clone()
		freshCtx.callableSubclasses = ctx.callableSubclasses
		freshCtx.genericFamilies = &genericFamilyAnalysis{}
		if _, err := convertFileNode(fresh.File, freshCtx); err != nil {
			t.Fatal(err)
		}
		if ctx.callableSubclasses.graph == firstGraph || ctx.callableSubclasses.graph != symbol.GlobalScope || sourceMethodIdentifierExists("OldSelector", freshCtx) || !sourceMethodIdentifierExists("FreshSelector", freshCtx) || classHasUnmodeledCallableSubclass(fresh.File.Symbols.FindClassScope("Base"), freshCtx) {
			t.Fatal("supplied project context retained a previous graph's syntax")
		}
	})
	t.Run("standalone-context-does-not-share-project-facts", func(t *testing.T) {
		helper := setupParseHelper(t, `class Cell<T>{public T get(){return null;}}`)
		project := helper.Ctx.Clone()
		project.callableSubclasses = &callableSubclassSourceInventory{}
		if _, err := convertFileNode(helper.File, project); err != nil {
			t.Fatal(err)
		}
		if !sourceMethodIdentifierExists("Get", project) {
			t.Fatal("project source selector was not retained")
		}
		facts := len(project.callableSubclasses.nodes)
		if facts == 0 {
			t.Fatal("explicit project's source facts were not populated")
		}
		standalone := helper.Ctx.Clone()
		if standalone.callableSubclasses != nil {
			t.Fatal("standalone helper unexpectedly owned project facts")
		}
		for attempt := 0; attempt < 2; attempt++ {
			if _, err := convertFileNode(helper.File, standalone); err != nil {
				t.Fatal(err)
			}
		}
		if standalone.callableSubclasses != nil || len(project.callableSubclasses.nodes) != facts {
			t.Fatal("standalone conversion retained or modified external project facts")
		}
	})
}

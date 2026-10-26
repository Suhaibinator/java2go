package transpiler

import (
	"go/ast"
	"testing"
)

// SourceInputSnapshot validates retained input only. Successful input validation
// cannot substitute for the independent allocation/class/executable certificate.
func TestCI213AnonymousFunctionalInputSnapshotControls(t *testing.T) {
	for _, name := range []string{"registered input positive", "stale filename", "stale parser source", "stale original source", "stale offset map"} {
		t.Run(name, func(t *testing.T) {
			const source = `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`
			ctx, _, _, info := anonymousFunctional209Lower(t, source, "Function<String,String>")
			snapshot := ctx.currentFile.SourceInputSnapshot()
			if snapshot == nil || snapshot.Validate() != nil {
				t.Fatal("actual registered input snapshot missing")
			}
			if name == "registered input positive" {
				return
			}
			switch name {
			case "stale filename":
				ctx.currentFile.SourceFilename = "Other.java"
			case "stale parser source":
				ctx.currentFile.Source[0] = ' '
			case "stale original source":
				ctx.currentFile.OriginalSource = []byte("foreign")
			case "stale offset map":
				ctx.currentFile.SourceOffsets = []uint32{1}
			}
			if snapshot.Validate() == nil {
				t.Fatal("input mutation was not detected by the existing snapshot API")
			}
			allocation := info.scope.Class.DeclarationNode
			before := len(*ctx.hoistedDecls)
			if result := lowerAnonymousClassToStruct(allocation, allocation.ChildByFieldName("type"), objectCreationClassBody(allocation), ctx.currentFile.Source, ctx); result != nil || len(*ctx.hoistedDecls) != before {
				t.Fatalf("anonymous functional allocation must refuse invalid current input: %s", name)
			}
		})
	}
}

// First-entry refusal must precede allocating an identity, registering a scope,
// emitting any declaration, or constructing a receiver. These controls have no
// prior synthesized info that could accidentally make refusal a cache property.
func TestCI213AnonymousFunctionalFreshInputSnapshotControls(t *testing.T) {
	for _, name := range []string{"registered input positive", "stale filename", "stale parser source", "stale original source", "stale offset map"} {
		t.Run(name, func(t *testing.T) {
			const source = `import java.util.function.Function;class Main{static Function<String,String> make(){return new Function<String,String>(){public String apply(String value){return value;}};}}`
			h := setupParseHelper(t, source)
			ctx := h.Ctx.Clone()
			ctx.currentClass = ctx.currentFile.FindClassScope("Main")
			if ctx.currentClass == nil {
				t.Fatal("actual Main declaration missing")
			}
			methods := ctx.currentClass.FindMethod().ByOriginalName("make")
			if len(methods) != 1 {
				t.Fatal("actual make declaration missing")
			}
			ctx.localScope = methods[0]
			ctx.className = ctx.currentClass.Class.Name
			var declarations []ast.Decl
			var counter int
			ctx.hoistedDecls = &declarations
			ctx.anonClassCounter = &counter
			ctx.anonymousClasses = make(map[anonymousClassKey]*anonymousClassInfo)
			ctx.localClasses = make(map[string]*localClassInfo)
			ctx.executionContextName = "execution"
			allocation := findNode(h.File.Ast, "object_creation_expression")
			if allocation == nil {
				t.Fatal("actual allocation missing")
			}
			typ, body := allocation.ChildByFieldName("type"), objectCreationClassBody(allocation)
			ctx.expectedType, ctx.expectedTypeRoot = "Function<String,String>", allocation
			snapshot := ctx.currentFile.SourceInputSnapshot()
			if snapshot == nil || snapshot.Validate() != nil {
				t.Fatal("actual registered input snapshot missing")
			}
			switch name {
			case "stale filename":
				ctx.currentFile.SourceFilename = "Other.java"
			case "stale parser source":
				ctx.currentFile.Source[0] = ' '
			case "stale original source":
				ctx.currentFile.OriginalSource = []byte("foreign")
			case "stale offset map":
				ctx.currentFile.SourceOffsets = []uint32{1}
			}
			if name != "registered input positive" && snapshot.Validate() == nil {
				t.Fatal("input mutation was not detected by the existing snapshot API")
			}
			if len(ctx.anonymousClasses) != 0 || len(declarations) != 0 || counter != 0 {
				t.Fatal("fresh synthesis premise mutated")
			}
			result := lowerAnonymousClassToStruct(allocation, typ, body, ctx.currentFile.Source, ctx)
			if name == "registered input positive" {
				key, ok := anonymousClassSourceKey(allocation)
				info := ctx.anonymousClasses[key]
				if !ok || result == nil || len(ctx.anonymousClasses) != 1 || info == nil || info.scope == nil || info.scope.Class == nil || info.scope.Class.DeclarationNode == nil || !info.scope.Class.DeclarationNode.Equal(allocation) || len(declarations) == 0 || counter != 1 {
					t.Fatal("fresh positive must synthesize the actual allocation")
				}
				return
			}
			if result != nil || len(ctx.anonymousClasses) != 0 || len(declarations) != 0 || counter != 0 {
				t.Fatalf("anonymous functional fresh synthesis must refuse invalid current input before map hoist counter or construction mutation: %s", name)
			}
		})
	}
}

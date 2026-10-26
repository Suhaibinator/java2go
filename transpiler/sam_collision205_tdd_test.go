package transpiler

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/NickyBoy89/java2go/symbol"
)

func TestCI205SourceSAMMetadataSelectorCollisions(t *testing.T) {
	rows := []struct {
		name, source, owner, implementation string
	}{
		{"direct", `package samcollision; interface Direct {int JavaDynamicTypeID();}`, "Direct", ""},
		{"default", `package samcollision; interface WithDefault {int value(); default int JavaDynamicTypeID(){return value()+2;}}`, "WithDefault", ""},
		{"inherited", `package samcollision; interface Parent {int JavaDynamicTypeID();} interface Child extends Parent {} class Impl implements Child {public int JavaDynamicTypeID(){return 17;}}`, "Parent", "Impl"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			h := setupParseHelper(t, row.source)
			owner := h.File.Symbols.FindClassScope(row.owner)
			if owner == nil {
				t.Fatal("actual interface declaration missing")
			}
			methods := owner.FindMethod().ByOriginalName("JavaDynamicTypeID")
			if len(methods) != 1 || methods[0].DeclarationNode == nil {
				t.Fatal("actual Java method declaration missing")
			}
			method := methods[0]
			adapter := symbol.GoIdentifier(owner.Class.Name + "FuncAdapter")
			ownerID := javaClassBinaryName(owner)
			if ownerID != "samcollision."+row.owner {
				t.Fatal("actual Java interface lost its original binary name")
			}
			if method.Name == "JavaDynamicTypeID" {
				t.Fatal("Java method family must retain a collision-safe selector independently of identity demand")
			}
			if row.implementation != "" {
				implementation := h.File.Symbols.FindClassScope(row.implementation)
				if implementation == nil {
					t.Fatal("actual implementing declaration missing")
				}
				actual := implementation.FindMethod().ByOriginalName("JavaDynamicTypeID")
				if len(actual) != 1 || actual[0].Name != method.Name {
					t.Fatal("inherited interface and implementing class lost the same Java method family selector")
				}
			}

			file, err := parser.ParseFile(token.NewFileSet(), "generated.go", renderGoFileFromJava(t, row.source), 0)
			if err != nil {
				t.Fatal(err)
			}
			samIdentity205RequireAdapter(t, file, adapter)
			metadata, sourceBodies := 0, 0
			selectors := make(map[string]bool)
			for _, declaration := range file.Decls {
				fn, ok := declaration.(*ast.FuncDecl)
				if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
					continue
				}
				receiver := samIdentity205ReceiverBase(fn.Recv.List[0].Type)
				key := receiver + "." + fn.Name.Name
				if selectors[key] {
					t.Fatalf("duplicate generated receiver selector %s", key)
				}
				selectors[key] = true
				if receiver == adapter && fn.Name.Name == "JavaDynamicTypeID" {
					metadata++
					samIdentity205RequireMetadata(t, file, fn, ownerID)
				}
				if fn.Name.Name == method.Name {
					sourceBodies++
					if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
						t.Fatal("Java method result missing")
					}
					result, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
					if !ok || result.Name != "int32" {
						t.Fatal("Java method lost its int result")
					}
				}
			}
			if metadata != 1 || sourceBodies == 0 {
				t.Fatalf("source method and adapter identity must coexist: metadata=%d sourceBodies=%d", metadata, sourceBodies)
			}
		})
	}
}

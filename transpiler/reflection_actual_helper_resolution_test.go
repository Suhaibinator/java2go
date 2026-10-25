package transpiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

func TestReflectionActualGsonHelperMethodResolution(t *testing.T) {
	base := filepath.Join("testdata", "accessible_annotation_gson_minimal", "project", "src", "main", "java")
	files := map[string]string{}
	err := filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		relative, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(bytes)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := ordinaryDemandTestContext(t, files, "MinimalWitness.java")
	ParseNode(ctx.currentFile.ParsedRoot(), ctx.currentFile.Source, ctx)
	ctx.localScope = ctx.currentClass.FindMethodByName("main", nil)
	var invocation *sitter.Node
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if node.Type() == "method_invocation" {
			object := node.ChildByFieldName("object")
			if object != nil && object.Content(ctx.currentFile.Source) == "ReflectionAccessFilterHelper" {
				invocation = node
				return
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child)
		}
	}
	visit(ctx.localScope.DeclarationNode)
	if invocation == nil {
		t.Fatal("frozen helper call absent")
	}
	object := invocation.ChildByFieldName("object")
	owner := resolveClassScopeByIdentifier(ctx, ctx.currentFile.Source, object)
	if owner == nil {
		t.Fatal("actual source helper owner absent")
	}
	args := nodeutil.SemanticNamedChildrenOf(invocation.ChildByFieldName("arguments"))
	for index, arg := range args {
		actual, known := inferExprJavaType(arg, ctx, ctx.currentFile.Source)
		t.Logf("actual%d=%q known=%v", index, actual, known)
	}
	for _, method := range owner.Methods {
		if method.OriginalName == "canAccess" {
			score, ok := scoreDeclaredMethodCandidate(method, owner, nil, args, ctx, ctx.currentFile.Source, nil, nil)
			t.Logf("candidate=%s static=%v score=%+v applicable=%v", method.Name, method.IsStatic, score, ok)
			for index, param := range method.Parameters {
				t.Logf("formal%d=%q original=%q", index, methodParameterReferenceType(param, method, owner, ctx), param.OriginalType)
			}
		}
	}
	selected := findBestMethodInHierarchy(owner, "canAccess", invocation.ChildByFieldName("arguments"), false, true, ctx, ctx.currentFile.Source)
	if selected == nil || selected.def == nil || !selected.def.IsStatic || selected.owner != owner {
		t.Fatal("real public static helper rejected before execution lowering")
	}
}

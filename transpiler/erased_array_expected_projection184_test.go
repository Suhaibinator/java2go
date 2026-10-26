package transpiler

import (
	"bytes"
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
	"go/format"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func closedArray184Render(t *testing.T, e ast.Expr) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), e); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func closedArray184Setup(t *testing.T, declarations, sourceView, target string) (*ParseHelper, Ctx, *sitter.Node) {
	t.Helper()
	h := setupParseHelper(t, declarations+`class Box<T>{T value;T get(){return value;}}class Use{static void probe(Box<`+sourceView+`>selected){Object erased=selected.value;}}`)
	ctx := h.Ctx.Clone()
	ctx.currentClass = h.File.Symbols.FindClassScope("Use")
	ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("probe")[0]
	var node *sitter.Node
	var walk func(*sitter.Node)
	walk = func(n *sitter.Node) {
		if node != nil {
			return
		}
		if n.Type() == "field_access" && n.Content(h.File.Source) == "selected.value" {
			node = n
			return
		}
		for i := 0; i < int(n.NamedChildCount()); i++ {
			walk(n.NamedChild(i))
		}
	}
	walk(ctx.localScope.DeclarationNode)
	if node == nil {
		t.Fatal("actual field read missing")
	}
	ctx.expectedType = target
	ctx.expectedTypeRoot = node
	return h, ctx, node
}
func TestErasedArrayExpected184ClosedPhysicalViews(t *testing.T) {
	for _, row := range []struct {
		view, target, physical string
		rank                   int
	}{
		{"boolean[]", "boolean[]", "*stdjava.PrimitiveArray[bool]", 1}, {"byte[]", "byte[]", "*stdjava.PrimitiveArray[int8]", 1}, {"short[]", "short[]", "*stdjava.PrimitiveArray[int16]", 1}, {"char[]", "char[]", "*stdjava.PrimitiveArray[rune]", 1}, {"int[]", "int[]", "*stdjava.PrimitiveArray[int32]", 1}, {"long[]", "long[]", "*stdjava.PrimitiveArray[int64]", 1}, {"float[]", "float[]", "*stdjava.PrimitiveArray[float32]", 1}, {"double[]", "double[]", "*stdjava.PrimitiveArray[float64]", 1}, {"Child[]", "Child[]", "*stdjava.ReferenceArray", 1}, {"Child[]", "Base[]", "*stdjava.ReferenceArray", 1}, {"int[][]", "int[][]", "*stdjava.ReferenceArray", 2}, {"Child[][]", "Base[][]", "*stdjava.ReferenceArray", 2},
	} {
		t.Run(row.view+"_to_"+row.target, func(t *testing.T) {
			h, ctx, node := closedArray184Setup(t, "class Base{}class Child extends Base{}", row.view, row.target)
			view, erasure, known := directOwnerFieldAccessView(node, ctx, h.File.Source)
			if !known || view != row.view {
				t.Fatalf("owned field proof missing %s/%s/%t", view, erasure, known)
			}
			once := &ast.CallExpr{Fun: ast.NewIdent("obtain")}
			got := coerceArgumentToExpectedType(once, node, row.target, ctx, h.File.Source)
			call, ok := got.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 || call.Args[0] != once {
				t.Fatalf("expected single evaluation checked view, got %s", closedArray184Render(t, got))
			}
			index, ok := call.Fun.(*ast.IndexExpr)
			if !ok || closedArray184Render(t, index.Index) != row.physical {
				t.Fatalf("wrong physical view %s", closedArray184Render(t, got))
			}
			if closedArray184Render(t, index.X) != "stdjava.ObjectView" || strings.Count(closedArray184Render(t, call.Args[1]), "ArrayTypeID(") != row.rank {
				t.Fatalf("wrong array descriptor %s", closedArray184Render(t, got))
			}
			if strings.Count(closedArray184Render(t, got), "obtain()") != 1 {
				t.Fatal("receiver evaluated multiple times")
			}
			if row.target != row.view && strings.Contains(closedArray184Render(t, call.Args[1]), `"Child"`) {
				t.Fatal("cast targeted logical Child[] instead of consuming Base[]")
			}
		})
	}
}
func TestErasedArrayExpected184NegativeAndOwnerGuards(t *testing.T) {
	for _, target := range []string{"Object", "Unknown[]", "java.util.List<String>[]", "? extends Base[]"} {
		t.Run(target, func(t *testing.T) {
			h, ctx, node := closedArray184Setup(t, "class Base{}", "byte[]", target)
			raw := ast.NewIdent("physical")
			if got := projectDirectOwnerErasedExpressionForExpected(raw, node, ctx, h.File.Source); got != raw {
				t.Fatalf("unproved/deferred target gained projection %s", closedArray184Render(t, got))
			}
		})
	}
	t.Run("expected root", func(t *testing.T) {
		h, ctx, node := closedArray184Setup(t, "", "byte[]", "byte[]")
		ctx.expectedTypeRoot = ctx.localScope.DeclarationNode
		ctx.importAliases = map[string]string{}
		ctx.usedImports = map[string]bool{}
		beforeAliases := map[string]string{}
		beforeImports := map[string]bool{}
		raw := ast.NewIdent("physical")
		if projectDirectOwnerErasedExpressionForExpected(raw, node, ctx, h.File.Source) != raw || !reflect.DeepEqual(ctx.importAliases, beforeAliases) || !reflect.DeepEqual(ctx.usedImports, beforeImports) {
			t.Fatal("mismatched target root changed expression or imports")
		}
	})
	t.Run("ordinary field", func(t *testing.T) {
		h := setupParseHelper(t, `class Plain{byte[] value;}class Use{static void read(Plain selected){Object v=selected.value;}}`)
		ctx := h.Ctx.Clone()
		ctx.currentClass = h.File.Symbols.FindClassScope("Use")
		ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("read")[0]
		node := findNode(ctx.localScope.DeclarationNode, "field_access")
		ctx.expectedType = "byte[]"
		ctx.expectedTypeRoot = node
		raw := ast.NewIdent("physical")
		if projectDirectOwnerErasedExpressionForExpected(raw, node, ctx, h.File.Source) != raw {
			t.Fatal("ordinary array storage gained erased-field projection")
		}
	})
	t.Run("caller binder", func(t *testing.T) {
		h := setupParseHelper(t, `class Box<T>{T value;T get(){return value;}}class Use{static<T>void read(Box<T[]>selected){T[]v=selected.value;}}`)
		ctx := h.Ctx.Clone()
		ctx.currentClass = h.File.Symbols.FindClassScope("Use")
		ctx.localScope = ctx.currentClass.FindMethod().ByOriginalName("read")[0]
		node := findNode(ctx.localScope.DeclarationNode, "field_access")
		ctx.expectedType = "T[]"
		ctx.expectedTypeRoot = node
		raw := ast.NewIdent("physical")
		if projectDirectOwnerErasedExpressionForExpected(raw, node, ctx, h.File.Source) != raw {
			t.Fatal("live caller binder array gained closed-target projection")
		}
	})
	for _, qualified := range []bool{false, true} {
		t.Run("source homonym", func(t *testing.T) {
			target := "ByteBuffer[]"
			if qualified {
				target = "java.nio.ByteBuffer[]"
			}
			h, ctx, node := closedArray184Setup(t, "class ByteBuffer{}", target, target)
			got := projectDirectOwnerErasedExpressionForExpected(ast.NewIdent("physical"), node, ctx, h.File.Source)
			out := closedArray184Render(t, got)
			if !strings.Contains(out, "ObjectView[*stdjava.ReferenceArray]") {
				t.Fatalf("closed source/canonical target missing %s", out)
			}
			if strings.Contains(out, `"java.nio.ByteBuffer"`) != qualified {
				t.Fatalf("source homonym stole canonical owner or vice versa: %s", out)
			}
		})
	}
}
func TestErasedArrayExpected184ExistingRankZeroAndOperand(t *testing.T) {
	h, ctx, node := closedArray184Setup(t, "", "java.nio.ByteBuffer", "java.nio.ByteBuffer")
	got := projectDirectOwnerErasedExpressionForExpected(ast.NewIdent("physical"), node, ctx, h.File.Source)
	if !strings.Contains(closedArray184Render(t, got), "ObjectView[*stdjava.ByteBuffer]") {
		t.Fatal("rank-zero path changed")
	}
	h, ctx, node = closedArray184Setup(t, "", "int[]", "Object")
	got = projectErasedFieldArrayOperand(ast.NewIdent("physical"), node, ctx, h.File.Source)
	if !strings.Contains(closedArray184Render(t, got), "JavaArrayCast[*stdjava.PrimitiveArray[int32]]") {
		t.Fatal("existing actual-origin operand path changed")
	}
}

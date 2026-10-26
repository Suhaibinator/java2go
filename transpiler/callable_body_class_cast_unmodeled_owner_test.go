package transpiler

import (
	"testing"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// Same-name classes inside unrepresented owner boundaries cannot supply the
// occurrence certificate for an ordinary registered member class.
func TestCallableBodyClassCastRejectsUnmodeledOwnerRebound(t *testing.T) {
	for _, which := range []string{"enumConstantBody", "instanceInitializerBlock"} {
		t.Run(which, func(t *testing.T) {
			source := `enum Outer{A{class Owner<T>{Object read(Object value){return (Class<T>)value;}}};class Owner<T>{Object read(Object value){return (Class<T>)value;}}}`
			boundaryKind := "enum_constant"
			if which == "instanceInitializerBlock" {
				source = `class Outer{{class Owner<T>{Object read(Object value){return (Class<T>)value;}}}class Owner<T>{Object read(Object value){return (Class<T>)value;}}}`
				boundaryKind = "block"
			}
			helper := setupParseHelper(t, source)
			outer := helper.File.Symbols.FindClassScope("Outer")
			if outer == nil || outer.Class == nil {
				t.Fatal("actual outer declaration absent")
			}
			owner := outer.FindClassScope("Owner")
			if owner == nil || owner.Class == nil || owner.Enclosing != outer || len(owner.OwnTypeParameters()) != 1 {
				t.Fatal("ordinary registered member class premise absent")
			}
			method := owner.FindMethodByName("read", nil)
			if method == nil || method.DeclarationNode == nil {
				t.Fatal("actual registered read method absent")
			}
			declaration := owner.OwnTypeParameters()[0].Declaration
			cast := findNode(method.DeclarationNode, "cast_expression")
			if !callableBodyCanonicalClassCast(owner, method, cast, declaration, helper.Ctx) {
				t.Fatal("unchanged ordinary member Class cast positive premise failed")
			}
			var replacement *sitter.Node
			var walk func(*sitter.Node)
			walk = func(node *sitter.Node) {
				if node == nil || replacement != nil {
					return
				}
				if node.Type() == "class_declaration" {
					name := node.ChildByFieldName("name")
					if name != nil && name.Content(helper.File.Source) == "Owner" && !node.Equal(owner.Class.DeclarationNode) {
						for parent := node.Parent(); parent != nil && !parent.Equal(outer.Class.DeclarationNode); parent = parent.Parent() {
							if parent.Type() == boundaryKind {
								replacement = node
								return
							}
						}
					}
				}
				for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
					walk(child)
				}
			}
			walk(outer.Class.DeclarationNode)
			if replacement == nil {
				t.Fatal("actual same-name class under unmodeled owner boundary absent")
			}
			replacementMethod := findNode(replacement, "method_declaration")
			if replacementMethod == nil || replacementMethod.ChildByFieldName("name") == nil || replacementMethod.ChildByFieldName("name").Content(helper.File.Source) != method.OriginalName {
				t.Fatal("actual same-name read declaration absent")
			}
			oldClassNode := owner.Class.DeclarationNode
			oldMethodNode := method.DeclarationNode
			t.Cleanup(func() {
				owner.Class.DeclarationNode = oldClassNode
				method.DeclarationNode = oldMethodNode
			})
			owner.Class.DeclarationNode = replacement
			method.DeclarationNode = replacementMethod
			cast = findNode(replacementMethod, "cast_expression")
			if owner.Enclosing != outer || declaration == nil || method.Parameters[0].TypeParameterBindings[declaration.SourceName] != declaration {
				t.Fatal("original enclosing metadata/binder premise changed")
			}
			if callableBodyCanonicalClassCast(owner, method, cast, declaration, helper.Ctx) {
				t.Error("unmodeled enum/initializer owner supplied the registered Class cast certificate")
			}
		})
	}
}

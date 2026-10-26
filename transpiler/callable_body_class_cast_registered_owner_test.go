package transpiler

import (
	"testing"
)

// A registered method pointer and same-tree byte ranges do not authorize a
// different class declaration, even when its method/header have equal text.
func TestCallableBodyClassCastRegisteredClassOwnership(t *testing.T) {
	for _, which := range []string{"pairedDifferentName", "pairedNestedSameName", "wrongEnclosingPointer"} {
		t.Run(which, func(t *testing.T) {
			source := `class Owner<T>{Object read(Object value){return (Class<T>)value;}}class Other<T>{Object read(Object value){return (Class<T>)value;}}`
			if which != "pairedDifferentName" {
				source = `class Outer<T>{class Owner<U>{Object read(Object value){return (Class<U>)value;}}}class Other<T>{class Owner<U>{Object read(Object value){return (Class<U>)value;}}}`
			}
			helper := setupParseHelper(t, source)
			owner := helper.File.Symbols.FindClassScope("Owner")
			other := helper.File.Symbols.FindClassScope("Other")
			if which != "pairedDifferentName" {
				outer := helper.File.Symbols.FindClassScope("Outer")
				if outer == nil || other == nil {
					t.Fatal("actual enclosing source classes absent")
				}
				owner = outer.FindClassScope("Owner")
			}
			if owner == nil || owner.Class == nil || other == nil || other.Class == nil || len(owner.OwnTypeParameters()) != 1 {
				t.Fatal("actual distinct owner/binder declarations absent")
			}
			method := owner.FindMethodByName("read", nil)
			if method == nil || method.DeclarationNode == nil {
				t.Fatal("actual registered read method absent")
			}
			cast := findNode(method.DeclarationNode, "cast_expression")
			declaration := owner.OwnTypeParameters()[0].Declaration
			if !callableBodyCanonicalClassCast(owner, method, cast, declaration, helper.Ctx) {
				t.Fatal("unchanged actual owned Class cast positive premise failed")
			}
			oldClassNode := owner.Class.DeclarationNode
			oldMethodNode := method.DeclarationNode
			oldEnclosing := owner.Enclosing
			t.Cleanup(func() {
				owner.Class.DeclarationNode = oldClassNode
				method.DeclarationNode = oldMethodNode
				owner.Enclosing = oldEnclosing
			})
			if which == "wrongEnclosingPointer" {
				if oldEnclosing == nil || oldEnclosing == other {
					t.Fatal("distinct actual enclosing-owner premise absent")
				}
				owner.Enclosing = other
			} else {
				replacement := other
				if which == "pairedNestedSameName" {
					replacement = other.FindClassScope("Owner")
				}
				if replacement == nil || replacement == owner || replacement.Class == nil {
					t.Fatal("different actual replacement class absent")
				}
				replacementMethod := replacement.FindMethodByName("read", nil)
				if replacementMethod == nil || replacementMethod.DeclarationNode == nil || replacement.Class.DeclarationNode.Equal(oldClassNode) {
					t.Fatal("distinct actual replacement class/method premise absent")
				}
				if which == "pairedNestedSameName" && replacement.Class.OriginalName != owner.Class.OriginalName {
					t.Fatal("same Java simple-name counterexample premise absent")
				}
				owner.Class.DeclarationNode = replacement.Class.DeclarationNode
				method.DeclarationNode = replacementMethod.DeclarationNode
				cast = findNode(method.DeclarationNode, "cast_expression")
			}
			// Keep the registered method, original metadata/binding identities,
			// and actual source tree. Only declaration ownership is corrupted.
			found := false
			for _, candidate := range owner.Methods {
				found = found || candidate == method
			}
			if !found || declaration == nil || method.Parameters[0].TypeParameterBindings[declaration.SourceName] != declaration {
				t.Fatal("retained registered method/binder premise changed")
			}
			if got := callableBodyCanonicalClassCast(owner, method, cast, declaration, helper.Ctx); got {
				t.Error("another source class/enclosing declaration supplied the owned Class cast certificate")
			}
		})
	}
}

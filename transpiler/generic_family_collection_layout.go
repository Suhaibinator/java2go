package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

// Collection arguments stay logical type information: its existing native
// reference protocol is JavaIterable for every argument. Prove that exact
// declaration at this occurrence before allowing a family to share storage.
func genericFamilyCollectionArgumentIndependent(javaType string, ctx Ctx, occurrences ...*sitter.Node) bool {
	component, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments, valid := reflectiveNamedTypeParts(component)
	if !valid || rank != 0 || stripJavaQualifier(base) != "Collection" || len(arguments) > 1 {
		return false
	}
	if len(occurrences) != 1 || occurrences[0] == nil || ctx.currentFile == nil {
		return false
	}
	root := occurrences[0]
	for root.Parent() != nil {
		root = root.Parent()
	}
	anchor := ctx.currentFile.ParsedRoot()
	if anchor == nil || !root.Equal(anchor) {
		return false
	}
	owner, known := canonicalArgumentIndependentOccurrenceOwner(base, ctx, occurrences...)
	return known && owner == "java.util.Collection"
}

// A nested Collection<E> declaration still has the JVM descriptor Collection,
// even when E belongs to this source owner. The complete layout must already
// be admitted, and the exact method/parameter must own the original type node.
func genericFamilyCollectionDefinitionErasure(owner *symbol.ClassScope, definition *symbol.Definition, ctx Ctx) (string, bool) {
	if owner == nil || definition == nil || canonicalGenericFamily(owner, ctx) == nil {
		return "", false
	}
	declaring := classScopeCtx(owner, ctx)
	if declaring.currentFile == nil {
		return "", false
	}
	for _, method := range owner.Methods {
		if method == nil || method.Constructor || method.IsStatic || len(method.TypeParameters) != 0 || method.DeclarationNode == nil || method.DeclarationNode.Type() != "method_declaration" {
			continue
		}
		var typ *sitter.Node
		if method == definition {
			typ = method.DeclarationNode.ChildByFieldName("type")
		} else {
			for index, parameter := range method.Parameters {
				if parameter != definition {
					continue
				}
				if executionParameterIsVariadic(method, index) {
					return "", false
				}
				nodes := nodeutil.SemanticNamedChildrenOf(method.DeclarationNode.ChildByFieldName("parameters"))
				if index >= len(nodes) {
					return "", false
				}
				typ, _ = nodeutil.JavaParameterNodes(nodes[index])
				break
			}
		}
		if typ == nil {
			continue
		}
		declaring.localScope = method
		if typ.Content(declaring.currentFile.Source) != strings.TrimSpace(definition.OriginalType) {
			return "", false
		}
		if genericFamilyCollectionArgumentIndependent(definition.OriginalType, declaring, typ) {
			return "java.util.Collection", true
		}
		return "", false
	}
	return "", false
}

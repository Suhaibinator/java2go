package transpiler

import (
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Admission records use canonical Java declarations, independently of source
// names and physical Go runtime types. These records promise refusal only.
var unimplementedPlatformConstructors = map[string]bool{}

type platformMemberContract struct {
	owner, name, result string
	arity               int
}

var unimplementedPlatformMembers = []platformMemberContract{
	{owner: "java.net.URI", name: "normalize", arity: 0, result: "java.net.URI"},
}

// Type syntax ignores value-only prefixes, but a type/binder owns the first
// segment even if a later member is missing. Pre-hoisting locals count too.
func platformTypeOwner(javaType string, occurrence *sitter.Node, ctx Ctx, source []byte) string {
	base, rank := javaArrayTypeParts(strings.TrimSpace(javaType))
	base, arguments := parseJavaTypeString(base)
	if rank != 0 || len(arguments) != 0 || base == "" {
		return ""
	}
	first := strings.SplitN(base, ".", 2)[0]
	if _, bound := resolveReferenceTypeParameter(symbol.JavaType{Original: first}, ctx); bound {
		return ""
	}
	if resolveClassScopeByQualifiedName(ctx, first) != nil || resolveClassScopeByQualifiedName(ctx, base) != nil {
		return ""
	}
	if first != base && ctx.currentFile != nil {
		if _, importedType := ctx.currentFile.Imports[first]; importedType {
			return ""
		}
	}
	if occurrence != nil && sourceGenericViewLocalTypeShadowed(occurrence, base, source) {
		return ""
	}
	owner, known := canonicalIntrinsicOwner(base, ctx)
	if !known {
		return ""
	}
	return owner
}

func unimplementedPlatformConstructorExpr(node *sitter.Node, javaType string, ctx Ctx, source []byte) ast.Expr {
	owner := platformTypeOwner(javaType, node, ctx, source)
	if !unimplementedPlatformConstructors[owner] {
		return nil
	}
	return unsupportedIntrinsicOwnerValue(owner, node, source, ctx)
}

// Recover a concrete source signature in its declaring namespace. External
// field/return origins currently retain a spelling but no declaring file;
// resolving that spelling in a consumer with a same-named binder is unsafe.
// Genuine substituted/captured binders are excluded before this lookup.
func platformReceiverDeclaration(node *sitter.Node, ctx Ctx, source []byte) (*symbol.Definition, Ctx) {
	switch node.Type() {
	case "identifier":
		name := node.Content(source)
		if ctx.localScope != nil {
			if parameter := ctx.localScope.ParameterByName(name); parameter != nil {
				return parameter, ctx
			}
			if variable := ctx.localScope.FindVariable(name); variable != nil {
				return variable, ctx
			}
		}
		if field := findFieldResolutionInHierarchy(ctx.currentClass, name, ctx); field != nil {
			return field.def, classScopeCtx(field.owner, ctx)
		}
		if imported := resolveStaticImportedField(name, ctx); imported.source != nil {
			return imported.source.def, classScopeCtx(imported.source.owner, ctx)
		}
	case "field_access":
		object, fieldNode := node.ChildByFieldName("object"), node.ChildByFieldName("field")
		if object == nil || fieldNode == nil {
			break
		}
		scope := resolveClassScopeByIdentifier(ctx, source, object)
		if target := resolveInvocationTarget(object, ctx, source); target != nil {
			scope = target.classScope
		}
		if field := findFieldResolutionInHierarchy(scope, fieldNode.Content(source), ctx); field != nil {
			return field.def, classScopeCtx(field.owner, ctx)
		}
	case "method_invocation":
		name, arguments := node.ChildByFieldName("name"), node.ChildByFieldName("arguments")
		if name == nil {
			break
		}
		var selected *methodResolution
		if object := node.ChildByFieldName("object"); object != nil {
			if target := resolveInvocationTarget(object, ctx, source); target != nil {
				selected, _ = findBestMethodForInvocationTarget(target, name.Content(source), arguments, true, true, ctx, source)
			}
		} else if ctx.currentClass != nil {
			selected = findBestMethodInHierarchy(ctx.currentClass, name.Content(source), arguments, true, true, ctx, source)
		}
		if selected != nil && selected.def != nil {
			declaring, _ := invocationMethodDeclarationContext(selected.def, ctx)
			return selected.def, declaring
		}
	}
	return nil, ctx
}

func platformReceiverOwner(node *sitter.Node, ctx Ctx, source []byte) string {
	if node == nil {
		return ""
	}
	// Parentheses preserve the original declaration instead of erasing its
	// origin via an inference path that does not forward origin outputs.
	if node.Type() == "parenthesized_expression" && nodeutil.SemanticNamedChildCount(node) == 1 {
		return platformReceiverOwner(nodeutil.SemanticNamedChild(node, 0), ctx, source)
	}
	var origin inferredJavaTypeOrigin
	javaType, known := inferExprJavaType(node, ctx, source, &origin)
	if !known || origin.unresolved || origin.parameter != nil || origin.nominalScope != nil {
		return ""
	}
	if definition, declaring := platformReceiverDeclaration(node, ctx, source); definition != nil {
		declared := definitionDeclaredJavaType(definition)
		if _, bound := resolveReferenceTypeParameter(declared, declaring); bound {
			return ""
		}
		qualified := qualifyDeclaredReferenceType(declared, declaring)
		// Occurrence-local source names belong to the consuming expression only
		// for its own declarations; imported signatures use their owning file.
		var occurrence *sitter.Node
		if declaring.currentFile == ctx.currentFile {
			occurrence = node
		}
		// Prefix shadowing is checked on the written declaration, not on the
		// synthetic qualified identity produced from an import. A member type
		// named java cannot steal an already selected import java.net.URI.
		owner := platformTypeOwner(declared.Original, occurrence, declaring, source)
		if owner == qualified {
			return owner
		}
		return ""
	}
	// Exact inferred external origins are safe; an unqualified spelling without
	// a recoverable declaration is not evidence of a platform type.
	if origin.javaType == javaType && strings.Contains(javaType, ".") {
		return platformTypeOwner(javaType, node, ctx, source)
	}
	return ""
}

func unimplementedPlatformMemberExpr(object *sitter.Node, method string, ctx Ctx, source []byte) ast.Expr {
	if object == nil || object.Parent() == nil || object.Parent().Type() != "method_invocation" {
		return nil
	}
	invocation := object.Parent()
	for _, contract := range unimplementedPlatformMembers {
		if method != contract.name || invocationArgumentCount(invocation) != contract.arity {
			continue
		}
		if platformReceiverOwner(object, ctx, source) == contract.owner {
			return unsupportedIntrinsicValue(invocation, contract.result, source, ctx)
		}
	}
	return nil
}

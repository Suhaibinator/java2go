package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func init() {
	registerIntrinsicOwner("java.util.Arrays", true)
	registerStaticNodeIntrinsic("Arrays", "copyOfRange", lowerArraysByteCopyRange)
	registerStaticIntrinsicExpectedArguments("Arrays", "copyOfRange", func(invocation *sitter.Node, ctx Ctx, source []byte) []string {
		if !arraysByteCopyRangeSelected(invocation, ctx, source) {
			return nil
		}
		return []string{"byte[]", "int", "int"}
	})
	registerStaticIntrinsicDerivedResultType("Arrays", "copyOfRange", func(invocation *sitter.Node, ctx Ctx, source []byte) (string, bool) {
		if !arraysByteCopyRangeSelected(invocation, ctx, source) {
			return "", false
		}
		return "byte[]", true
	})
	registerStaticIntrinsicImportSignature("Arrays", "copyOfRange", "byte[]", "int", "int")
}

// The historical default intrinsic owner is not declaration evidence. Require
// a written canonical qualifier/import in addition to the existing lexical and
// source-class guards before selecting this overload.
func arraysByteCopyRangeCanonicalOwner(invocation *sitter.Node, ctx Ctx, source []byte) bool {
	object := invocation.ChildByFieldName("object")
	if object == nil {
		resolved := resolveStaticImportedMethod(invocation, ctx, source)
		if resolved.problem != "" || resolved.source != nil || resolved.intrinsic != "Arrays" {
			return false
		}
		entries := staticMethodImports(ctx)
		explicit := false
		for _, entry := range entries {
			if !entry.wildcard && entry.member == "copyOfRange" {
				explicit = true
			}
		}
		for _, entry := range entries {
			if entry.owner == "java.util.Arrays" && ((!explicit && entry.wildcard) || (explicit && !entry.wildcard && entry.member == "copyOfRange")) {
				return true
			}
		}
		return false
	}
	class, ok := intrinsicStaticClassName(object, ctx, source)
	if !ok || class != "Arrays" {
		return false
	}
	name := object.Content(source)
	if name == "java.util.Arrays" {
		return true
	}
	if name != "Arrays" || ctx.currentFile == nil {
		return false
	}
	if pkg, present := ctx.currentFile.Imports[name]; present {
		return pkg == "java.util"
	}
	owner, known := canonicalIntrinsicOwner(name, ctx)
	if !known || owner != "java.util.Arrays" {
		return false
	}
	imported := false
	for _, pkg := range intrinsicOnDemandImports(ctx) {
		if pkg == "java.util" {
			imported = true
		}
		if pkg != "java.util" && resolveClassScopeByQualifiedName(ctx, pkg+".Arrays") != nil {
			return false
		}
	}
	return imported
}

func arraysByteCopyRangeSelected(invocation *sitter.Node, ctx Ctx, source []byte) bool {
	if invocation == nil || invocation.Type() != "method_invocation" || invocationArgumentCount(invocation) != 3 {
		return false
	}
	name := invocation.ChildByFieldName("name")
	if name == nil || name.Content(source) != "copyOfRange" || !arraysByteCopyRangeCanonicalOwner(invocation, ctx, source) {
		return false
	}
	actual, known := inferExprJavaType(invocationArgumentNode(invocation, 0), ctx, source)
	component, primitive := javaPrimitiveArrayComponent(actual)
	return known && primitive && component == "byte" && intrinsicInvocationConversionApplicable(invocationArgumentNode(invocation, 1), "int", ctx, source) && intrinsicInvocationConversionApplicable(invocationArgumentNode(invocation, 2), "int", ctx, source)
}

func lowerArraysByteCopyRange(invocation *sitter.Node, args []ast.Expr, ctx Ctx, source []byte) ast.Expr {
	if len(args) != 3 || !arraysByteCopyRangeSelected(invocation, ctx, source) {
		return nil
	}
	return stdjavaCall(ctx, "ArraysByteCopyOfRange", args...)
}

package transpiler

import (
	sitter "github.com/smacker/go-tree-sitter"
	"go/ast"
)

func enumInheritedTextSelected(object *sitter.Node, method string, ctx Ctx, source []byte) bool {
	if object == nil || method != "toString" || invocationArgumentCount(object.Parent()) != 0 {
		return false
	}
	if object.Type() == "super" {
		return ctx.currentClass != nil && ctx.currentClass.IsEnum
	}
	typ, known := inferExprJavaType(object, ctx, source)
	if !known || !enumReferenceType(typ, ctx) {
		return false
	}
	if target := resolveInvocationTarget(object, ctx, source); target != nil {
		selected, _ := findBestMethodForInvocationTarget(target, method, object.Parent().ChildByFieldName("arguments"), true, false, ctx, source)
		if selected != nil && selected.def != nil && selected.def.DeclarationNode != nil && len(selected.def.Parameters) == 0 {
			return false
		}
	}
	return true
}

func enumConstantMethodContext(ctx Ctx) bool {
	if ctx.localScope == nil {
		return false
	}
	for node := ctx.localScope.DeclarationNode; node != nil; node = node.Parent() {
		switch node.Type() {
		case "enum_constant":
			return true
		case "class_declaration", "enum_declaration", "interface_declaration", "record_declaration":
			return false
		}
	}
	return false
}

func enumInheritedTextInvocation(object *sitter.Node, method string, ctx Ctx, source []byte) ast.Expr {
	if !enumInheritedTextSelected(object, method, ctx, source) {
		return nil
	}
	if object.Type() == "super" {
		receiver := ast.NewIdent(ShortName(ctx.className))
		if enumConstantMethodContext(ctx) {
			if declared := findDeclaredToStringMethod(ctx.currentClass); declared != nil {
				// The enclosing enum is a constant body's superclass; bypass its virtual wrapper.
				return callIdent("_"+ctx.className+"_"+declared.Name+"_default", intrinsicExecutionExpr(ctx), receiver)
			}
		}
		return stdjavaCall(ctx, "EnumNameJavaString", receiver)
	}
	return stdjavaCall(ctx, "JavaStringValueOfExecution", intrinsicExecutionExpr(ctx), stdjavaCall(ctx, "ReferenceRequireNonNull", ParseExpr(object, source, ctx)))
}

func implicitEnumInheritedTextSelected(node *sitter.Node, selected *methodResolution, ctx Ctx, source []byte) bool {
	if node == nil || node.ChildByFieldName("object") != nil || ctx.currentClass == nil || !ctx.currentClass.IsEnum || ctx.localScope == nil || ctx.localScope.IsStatic || invocationArgumentCount(node) != 0 {
		return false
	}
	name := node.ChildByFieldName("name")
	if name == nil || name.Content(source) != "toString" {
		return false
	}
	return selected == nil || selected.def == nil || len(selected.def.Parameters) != 0 || selected.def.DeclarationNode == nil
}

func enumInheritedTextResult(node *sitter.Node, ctx Ctx, source []byte) bool {
	if node == nil || node.ChildByFieldName("name") == nil {
		return false
	}
	if object := node.ChildByFieldName("object"); object != nil {
		return enumInheritedTextSelected(object, node.ChildByFieldName("name").Content(source), ctx, source)
	}
	if ctx.currentClass == nil {
		return false
	}
	selected := findBestMethodInHierarchy(ctx.currentClass, node.ChildByFieldName("name").Content(source), node.ChildByFieldName("arguments"), ctx.localScope != nil && !ctx.localScope.IsStatic, true, ctx, source)
	return implicitEnumInheritedTextSelected(node, selected, ctx, source)
}

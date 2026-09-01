package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

func compileTimeConstantJavaType(javaType string) bool {
	javaType = strings.TrimSpace(javaType)
	if strings.HasSuffix(javaType, "[]") {
		return false
	}
	base, _ := parseJavaTypeString(javaType)
	switch stripJavaQualifier(base) {
	case "byte", "short", "int", "long", "char", "float", "double", "boolean", "String":
		return true
	default:
		return false
	}
}

func resolveCompileTimeConstantsForClass(scope *symbol.ClassScope, source []byte, ctx Ctx) {
	if scope == nil {
		return
	}
	visiting := make(map[*symbol.Definition]bool)
	for _, field := range scope.Fields {
		isCompileTimeConstantField(field, scope, source, ctx, visiting)
	}
}

func isCompileTimeConstantField(
	field *symbol.Definition,
	owner *symbol.ClassScope,
	fallbackSource []byte,
	ctx Ctx,
	visiting map[*symbol.Definition]bool,
) bool {
	if field == nil || owner == nil {
		return false
	}
	if field.IsCompileTimeConstant {
		return true
	}
	if !field.IsStatic || !field.IsFinal || !compileTimeConstantJavaType(field.OriginalType) || field.DeclarationNode == nil {
		return false
	}
	if visiting[field] {
		return false
	}
	visiting[field] = true
	defer delete(visiting, field)

	fieldCtx := ctx.Clone()
	fieldCtx.currentClass = owner
	if owner.Class != nil {
		fieldCtx.className = owner.Class.Name
	}
	source := fallbackSource
	if file := findFileScopeForClassScope(owner); file != nil {
		fieldCtx.currentFile = file
		if len(file.Source) > 0 {
			source = file.Source
		}
	}
	declarator := declaratorForField(field, source)
	if declarator == nil {
		return false
	}
	initializer := declarator.ChildByFieldName("value")
	if initializer == nil {
		return false
	}
	if !compileTimeConstantExpression(initializer, source, fieldCtx, visiting) {
		return false
	}
	field.IsCompileTimeConstant = true
	return true
}

func declaratorForField(field *symbol.Definition, source []byte) *sitter.Node {
	if field == nil || field.DeclarationNode == nil {
		return nil
	}
	for _, declarator := range nodeutil.VariableDeclarators(field.DeclarationNode) {
		name := declarator.ChildByFieldName("name")
		if name != nil && name.Content(source) == field.OriginalName {
			return declarator
		}
	}
	return nil
}

func compileTimeConstantExpression(
	node *sitter.Node,
	source []byte,
	ctx Ctx,
	visiting map[*symbol.Definition]bool,
) bool {
	if node == nil {
		return false
	}
	switch node.Type() {
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal",
		"decimal_floating_point_literal", "hex_floating_point_literal",
		"character_literal", "string_literal", "true", "false":
		return true
	case "parenthesized_expression", "unary_expression":
		return node.NamedChildCount() == 1 &&
			compileTimeConstantExpression(node.NamedChild(0), source, ctx, visiting)
	case "cast_expression":
		return node.NamedChildCount() == 2 &&
			compileTimeConstantJavaType(node.NamedChild(0).Content(source)) &&
			compileTimeConstantExpression(node.NamedChild(1), source, ctx, visiting)
	case "binary_expression":
		if node.ChildCount() < 3 || !compileTimeConstantBinaryOperator(node.Child(1).Content(source)) {
			return false
		}
		return compileTimeConstantExpression(node.Child(0), source, ctx, visiting) &&
			compileTimeConstantExpression(node.Child(2), source, ctx, visiting)
	case "ternary_expression":
		return node.NamedChildCount() == 3 &&
			compileTimeConstantExpression(node.NamedChild(0), source, ctx, visiting) &&
			compileTimeConstantExpression(node.NamedChild(1), source, ctx, visiting) &&
			compileTimeConstantExpression(node.NamedChild(2), source, ctx, visiting)
	case "identifier":
		name := node.Content(source)
		for scope := ctx.currentClass; scope != nil; scope = scope.Enclosing {
			resolution := findFieldResolutionInHierarchy(scope, name, ctx)
			if resolution == nil {
				continue
			}
			return isCompileTimeConstantField(resolution.def, resolution.owner, source, ctx, visiting)
		}
		return false
	case "field_access":
		object := node.ChildByFieldName("object")
		if object == nil {
			return false
		}
		_, typeQualifier := staticFieldQualifierScope(object, source, ctx)
		if !typeQualifier {
			return false
		}
		access, ok := resolveStaticFieldAccess(node, source, ctx)
		if !ok || access.resolution == nil {
			return false
		}
		return isCompileTimeConstantField(
			access.resolution.def,
			access.resolution.owner,
			source,
			ctx,
			visiting,
		)
	default:
		return false
	}
}

func compileTimeConstantBinaryOperator(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", "<<", ">>", ">>>",
		"<", "<=", ">", ">=", "==", "!=", "&", "^", "|", "&&", "||":
		return true
	default:
		return false
	}
}

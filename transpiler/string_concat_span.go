package transpiler

import (
	"bytes"
	"go/ast"
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

type stringConcatSpanLeaf struct {
	node     *sitter.Node
	javaType string
	text     bool
}

// Parent checks are structural: refusing a maximal adjacent plus expression
// also refuses all its child prefixes, without mutable state in Ctx.
func stringConcatSpanHasPlusParent(node *sitter.Node, source []byte) bool {
	parent := node.Parent()
	for parent != nil && parent.Type() == "parenthesized_expression" {
		parent = parent.Parent()
	}
	return parent != nil && parent.Type() == "binary_expression" &&
		parent.ChildByFieldName("operator") != nil && parent.ChildByFieldName("operator").Content(source) == "+"
}

// Require actual file/tree, callable and declaring-class identities. A donor
// Ctx from a same-spelled method or byte-identical second tree proves nothing.
func stringConcatSpanContext(node *sitter.Node, source []byte, ctx Ctx) bool {
	if node == nil || node.HasError() || ctx.erasedGenericMethodBody != nil || ctx.genericInterfaceMethodBody != nil || ctx.rawFunctionTarget != nil || ctx.currentFile == nil || ctx.currentFile.ParsedRoot() == nil ||
		!bytes.Equal(source, ctx.currentFile.Source) || node.EndByte() > uint32(len(source)) ||
		ctx.localScope == nil || ctx.localScope.DeclarationNode == nil || ctx.currentClass == nil ||
		ctx.currentClass.HasAnonymousSourceOrigin() || ctx.currentClass.Class == nil || ctx.currentClass.Class.DeclarationNode == nil {
		return false
	}
	var callable *sitter.Node
	foundOwner, foundRoot := false, false
	for parent := node; parent != nil; parent = parent.Parent() {
		if parent.Type() == "lambda_expression" && callable == nil {
			return false
		}
		if callable == nil && (parent.Type() == "method_declaration" || parent.Type() == "constructor_declaration") {
			if parent.HasError() {
				return false
			}
			callable = parent
		}
		if !foundOwner && sourceGenericViewTypeDeclaration(parent) {
			if callable == nil || parent.HasError() || !parent.Equal(ctx.currentClass.Class.DeclarationNode) {
				return false
			}
			foundOwner = true
		}
		if parent.Equal(ctx.currentFile.ParsedRoot()) {
			foundRoot = true
		}
	}
	return foundOwner && foundRoot && callable != nil && callable.Equal(ctx.localScope.DeclarationNode)
}

func stringConcatSpanMentions(node *sitter.Node, name string, source []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "identifier" && node.Content(source) == name {
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if stringConcatSpanMentions(child, name, source) {
			return true
		}
	}
	return false
}

func stringConcatSpanHasDimensions(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	if node.Type() == "dimensions" || node.Type() == "array_type" || node.Type() == "spread_parameter" {
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if stringConcatSpanHasDimensions(child) {
			return true
		}
	}
	return false
}

// Unsupported nearest declarations are a found refusal, never permission to
// continue toward a convenient outer parameter or flattened symbol inventory.
func stringConcatSpanLocal(declaration, use *sitter.Node, name string, source []byte) (string, bool, bool) {
	if declaration == nil || declaration.Type() != "local_variable_declaration" {
		return "", false, false
	}
	for _, declarator := range nodeutil.NamedChildrenOf(declaration) {
		if declarator.Type() != "variable_declarator" {
			continue
		}
		ident := declarator.ChildByFieldName("name")
		if ident == nil || ident.Content(source) != name || ident.EndByte() > use.StartByte() {
			continue
		}
		value := declarator.ChildByFieldName("value")
		if receiverNodeWithin(use, value) || stringConcatSpanHasDimensions(declarator) {
			return "", true, false
		}
		typ := declaration.ChildByFieldName("type")
		if typ == nil || stringConcatSpanHasDimensions(typ) || typ.Content(source) == "var" {
			return "", true, false
		}
		return strings.TrimSpace(typ.Content(source)), true, true
	}
	return "", false, false
}

func stringConcatSpanIdentifierType(use *sitter.Node, source []byte) (string, bool) {
	name := use.Content(source)
	for child, parent := use, use.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		switch parent.Type() {
		case "local_variable_declaration":
			if typ, found, supported := stringConcatSpanLocal(parent, use, name, source); found {
				return typ, supported
			}
		case "block", "constructor_body":
			children := nodeutil.NamedChildrenOf(parent)
			for index := len(children) - 1; index >= 0; index-- {
				declaration := children[index]
				if declaration.EndByte() > child.StartByte() {
					continue
				}
				if typ, found, supported := stringConcatSpanLocal(declaration, use, name, source); found {
					return typ, supported
				}
			}
		case "for_statement":
			if stringConcatSpanMentions(parent.ChildByFieldName("init"), name, source) {
				return "", false
			}
		case "enhanced_for_statement":
			if receiverDeclarationName(parent.ChildByFieldName("name"), name, source) {
				return "", false
			}
		case "catch_clause":
			for _, parameter := range nodeutil.NamedChildrenOf(parent) {
				if parameter.Type() == "catch_formal_parameter" && stringConcatSpanMentions(parameter, name, source) {
					return "", false
				}
			}
		case "try_with_resources_statement":
			if stringConcatSpanMentions(parent.ChildByFieldName("resources"), name, source) {
				return "", false
			}
		case "lambda_expression", "class_body", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			return "", false
		case "method_declaration", "constructor_declaration":
			parameters := parent.ChildByFieldName("parameters")
			if parameters == nil {
				return "", false
			}
			for _, parameter := range nodeutil.NamedChildrenOf(parameters) {
				ident := parameter.ChildByFieldName("name")
				if parameter.Type() == "formal_parameter" && receiverDeclarationName(ident, name, source) {
					typ := parameter.ChildByFieldName("type")
					if typ == nil || stringConcatSpanHasDimensions(parameter) || strings.Contains(parameter.Content(source), "...") {
						return "", false
					}
					return strings.TrimSpace(typ.Content(source)), true
				}
				if stringConcatSpanMentions(parameter, name, source) {
					return "", false
				}
			}
			return "", false
		}
	}
	return "", false
}

func stringConcatSpanHasSourceType(node *sitter.Node, name string, source []byte) bool {
	if node == nil {
		return false
	}
	if sourceGenericViewTypeDeclaration(node) && receiverDeclarationName(node.ChildByFieldName("name"), name, source) {
		return true
	}
	for _, child := range nodeutil.NamedChildrenOf(node) {
		if stringConcatSpanHasSourceType(child, name, source) {
			return true
		}
	}
	return false
}

// Absence of a modeled source class alone cannot resolve an unknown on-demand
// or static type import. These uncommon declaration contexts retain fallback.
func stringConcatSpanCanonicalString(typ string, use *sitter.Node, source []byte, ctx Ctx) bool {
	if !isBuiltinJavaString(typ, ctx) {
		return false
	}
	root := ctx.currentFile.ParsedRoot()
	name := "String"
	if typ == "java.lang.String" {
		name = "java"
		if resolveClassScopeByQualifiedName(ctx, name) != nil {
			return false
		}
	} else if typ != "String" {
		return false
	}
	if stringConcatSpanHasSourceType(root, name, source) {
		return false
	}
	for parent := use.Parent(); parent != nil; parent = parent.Parent() {
		parameters := parent.ChildByFieldName("type_parameters")
		var mentionsType func(*sitter.Node) bool
		mentionsType = func(node *sitter.Node) bool {
			if node == nil {
				return false
			}
			if node.Type() == "type_identifier" && node.Content(source) == name {
				return true
			}
			for _, child := range nodeutil.NamedChildrenOf(node) {
				if mentionsType(child) {
					return true
				}
			}
			return false
		}
		if mentionsType(parameters) {
			return false
		}
	}
	if typ == "String" {
		for _, declaration := range nodeutil.NamedChildrenOf(root) {
			if declaration.Type() != "import_declaration" {
				continue
			}
			for index := 0; index < int(declaration.ChildCount()); index++ {
				part := declaration.Child(index)
				if part.Type() == "static" || part.Content(source) == "*" {
					return false
				}
			}
		}
	}
	return true
}

func stringConcatSpanLeafType(node *sitter.Node, source []byte, ctx Ctx) (stringConcatSpanLeaf, bool) {
	leaf := stringConcatSpanLeaf{node: node}
	if node == nil || node.HasError() {
		return leaf, false
	}
	switch node.Type() {
	case "string_literal":
		leaf.javaType, leaf.text = "java.lang.String", true
	case "character_literal":
		leaf.javaType = "char"
	case "true", "false":
		leaf.javaType = "boolean"
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal":
		leaf.javaType = "int"
		if strings.HasSuffix(strings.ToLower(node.Content(source)), "l") {
			leaf.javaType = "long"
		}
	case "decimal_floating_point_literal", "hex_floating_point_literal":
		leaf.javaType = "double"
		if strings.HasSuffix(strings.ToLower(node.Content(source)), "f") {
			leaf.javaType = "float"
		}
	case "identifier":
		typ, found := stringConcatSpanIdentifierType(node, source)
		if !found {
			return leaf, false
		}
		if primitive, ok := javaPrimitiveType(typ); ok {
			leaf.javaType = primitive
		} else if stringConcatSpanCanonicalString(typ, node, source, ctx) {
			leaf.javaType, leaf.text = typ, true
		} else {
			return leaf, false
		}
	default:
		return leaf, false
	}
	return leaf, true
}

func planCanonicalStringConcatSpan(node *sitter.Node, source []byte, ctx Ctx) ([]stringConcatSpanLeaf, bool) {
	if !stringConcatSpanContext(node, source, ctx) || stringConcatSpanHasPlusParent(node, source) {
		return nil, false
	}
	var reversed []*sitter.Node
	left := node
	for left != nil && left.Type() == "binary_expression" {
		op := left.ChildByFieldName("operator")
		if op == nil || op.Content(source) != "+" {
			return nil, false
		}
		reversed = append(reversed, left)
		left = left.ChildByFieldName("left")
	}
	if len(reversed) < 2 {
		return nil, false
	}
	first, ok := stringConcatSpanLeafType(left, source, ctx)
	if !ok {
		return nil, false
	}
	leaves := []stringConcatSpanLeaf{first}
	text := first.text
	for index := len(reversed) - 1; index >= 0; index-- {
		plus := reversed[index]
		right, valid := stringConcatSpanLeafType(plus.ChildByFieldName("right"), source, ctx)
		if !valid || (!text && !right.text) || compileTimeConstantExpression(plus, source, ctx, map[*symbol.Definition]bool{}) {
			return nil, false
		}
		text = true
		leaves = append(leaves, right)
	}
	return leaves, true
}

func lowerCanonicalStringConcatSpan(node *sitter.Node, source []byte, ctx Ctx) (ast.Expr, bool) {
	leaves, ok := planCanonicalStringConcatSpan(node, source, ctx)
	if !ok {
		return nil, false
	}
	used := affineLoopUsedNames(node, source, ctx)
	used[ctx.executionContextName] = struct{}{}
	var body []ast.Stmt
	var converted []ast.Expr
	for _, leaf := range leaves {
		valueName := synchronizedUniqueLocalName("_javaConcatValue", used)
		textName := synchronizedUniqueLocalName("_javaConcatText", used)
		var typ ast.Expr
		if leaf.text {
			typ = javaStringReferenceType(ctx)
		} else if leaf.javaType == "boolean" {
			typ = ast.NewIdent("bool")
		} else {
			typ = ast.NewIdent(goPrimitiveConversionName(leaf.javaType))
		}
		body = append(body, typedLocalDeclaration(valueName, typ, ParseExpr(leaf.node, source, ctx)))
		body = append(body, typedLocalDeclaration(textName, javaStringReferenceType(ctx), canonicalStringValueOf(leaf.javaType, ast.NewIdent(valueName), true, ctx)))
		converted = append(converted, ast.NewIdent(textName))
	}
	body = append(body, &ast.ReturnStmt{Results: []ast.Expr{stdjavaCall(ctx, "ConcatJavaStringSpan", converted...)}})
	return &ast.CallExpr{Fun: &ast.FuncLit{Type: &ast.FuncType{Results: &ast.FieldList{List: []*ast.Field{{Type: javaStringReferenceType(ctx)}}}}, Body: &ast.BlockStmt{List: body}}}, true
}

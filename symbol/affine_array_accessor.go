package symbol

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	sitter "github.com/smacker/go-tree-sitter"
)

// TrivialArrayAccessorKind identifies the only accessor bodies that the affine
// loop optimizer may replace. Each kind is defined structurally below; method
// names are deliberately irrelevant.
type TrivialArrayAccessorKind uint8

const (
	TrivialArrayAccessorGet TrivialArrayAccessorKind = iota + 1
	TrivialArrayAccessorSet
	TrivialArrayAccessorAdd
)

// AffineArrayView describes one private final flat array plus its private final
// int row stride. HelperName is assigned after ordinary symbol renaming so a
// synthesized cross-package view method cannot collide with user methods.
type AffineArrayView struct {
	ArrayField *Definition
	SizeField  *Definition
	HelperName string
}

// TrivialArrayAccessor records how parameters feed a row-major array operation.
// Parameter positions refer to Definition.Parameters and remain valid after
// identifier renaming.
type TrivialArrayAccessor struct {
	Kind            TrivialArrayAccessorKind
	View            *AffineArrayView
	RowParameter    int
	ColumnParameter int
	ValueParameter  int
	// ValueFirst preserves source operand order for add accessors. Floating-point
	// addition is not safely commutative when NaN payloads are observable.
	ValueFirst bool
}

type arrayAccessorCandidate struct {
	kind            TrivialArrayAccessorKind
	arrayFieldName  string
	sizeFieldName   string
	rowParameter    int
	columnParameter int
	valueParameter  int
	valueFirst      bool
}

func discoverTrivialArrayAccessors(scope *ClassScope, source []byte) {
	if scope == nil || scope.Class == nil || !scope.Class.IsFinal || scope.IsInterface || scope.IsEnum || len(scope.TypeParameters) != 0 ||
		hasJavaModifier(scope.Class.DeclarationNode, "strictfp") {
		return
	}

	views := make(map[string]*AffineArrayView)
	for _, method := range scope.Methods {
		candidate, ok := recognizeTrivialArrayAccessor(method, source)
		if !ok {
			continue
		}

		arrayField := scope.FindFieldByName(candidate.arrayFieldName)
		sizeField := scope.FindFieldByName(candidate.sizeFieldName)
		if !validAffineBackingFields(arrayField, sizeField) || !validAccessorSignature(method, candidate, arrayField) {
			continue
		}

		key := arrayField.OriginalName + "\x00" + sizeField.OriginalName
		view := views[key]
		if view == nil {
			view = &AffineArrayView{ArrayField: arrayField, SizeField: sizeField}
			views[key] = view
			scope.AffineArrayViews = append(scope.AffineArrayViews, view)
		}
		method.TrivialArrayAccessor = &TrivialArrayAccessor{
			Kind:            candidate.kind,
			View:            view,
			RowParameter:    candidate.rowParameter,
			ColumnParameter: candidate.columnParameter,
			ValueParameter:  candidate.valueParameter,
			ValueFirst:      candidate.valueFirst,
		}
	}
}

func validAffineBackingFields(arrayField, sizeField *Definition) bool {
	if arrayField == nil || sizeField == nil || arrayField.IsStatic || sizeField.IsStatic {
		return false
	}
	if !arrayField.IsPrivate || !arrayField.IsFinal || !sizeField.IsPrivate || !sizeField.IsFinal {
		return false
	}
	arrayType := strings.TrimSpace(arrayField.OriginalType)
	if !strings.HasSuffix(arrayType, "[]") || strings.HasSuffix(strings.TrimSuffix(arrayType, "[]"), "[]") {
		return false
	}
	elementType := strings.TrimSpace(strings.TrimSuffix(arrayType, "[]"))
	switch elementType {
	case "byte", "short", "int", "long", "char", "float", "double":
	default:
		return false
	}
	return strings.TrimSpace(sizeField.OriginalType) == "int"
}

func validAccessorSignature(method *Definition, candidate arrayAccessorCandidate, arrayField *Definition) bool {
	if method == nil || method.IsStatic || !method.HasBody || method.Constructor {
		return false
	}
	if candidate.rowParameter < 0 || candidate.columnParameter < 0 || candidate.rowParameter == candidate.columnParameter {
		return false
	}
	if candidate.rowParameter >= len(method.Parameters) || candidate.columnParameter >= len(method.Parameters) {
		return false
	}
	if strings.TrimSpace(method.Parameters[candidate.rowParameter].OriginalType) != "int" || strings.TrimSpace(method.Parameters[candidate.columnParameter].OriginalType) != "int" {
		return false
	}
	elementType := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(arrayField.OriginalType), "[]"))
	switch candidate.kind {
	case TrivialArrayAccessorGet:
		return len(method.Parameters) == 2 && strings.TrimSpace(method.OriginalType) == elementType
	case TrivialArrayAccessorSet, TrivialArrayAccessorAdd:
		return len(method.Parameters) == 3 && candidate.valueParameter >= 0 && candidate.valueParameter < len(method.Parameters) &&
			strings.TrimSpace(method.Parameters[candidate.valueParameter].OriginalType) == elementType && strings.TrimSpace(method.OriginalType) == "void"
	default:
		return false
	}
}

func recognizeTrivialArrayAccessor(method *Definition, source []byte) (arrayAccessorCandidate, bool) {
	if method == nil || method.DeclarationNode == nil || method.DeclarationNode.Type() != "method_declaration" {
		return arrayAccessorCandidate{}, false
	}
	// Inlining either modifier would move work across a Java semantic boundary:
	// synchronized supplies monitor acquisition, while pre-Java-17 strictfp can
	// change floating-point rounding at method entry/exit.
	if hasJavaModifier(method.DeclarationNode, "synchronized") || hasJavaModifier(method.DeclarationNode, "strictfp") {
		return arrayAccessorCandidate{}, false
	}
	body := method.DeclarationNode.ChildByFieldName("body")
	statements := meaningfulNamedChildren(body)
	if len(statements) == 1 && statements[0].Type() == "return_statement" {
		return recognizeArrayGetter(method, statements[0], source)
	}
	if len(statements) == 1 && statements[0].Type() == "expression_statement" {
		return recognizeDirectArrayMutation(method, statements[0], source)
	}
	if len(statements) == 2 && statements[0].Type() == "local_variable_declaration" && statements[1].Type() == "expression_statement" {
		return recognizeIndexedArrayAdd(method, statements[0], statements[1], source)
	}
	return arrayAccessorCandidate{}, false
}

func meaningfulNamedChildren(node *sitter.Node) []*sitter.Node {
	if node == nil {
		return nil
	}
	children := make([]*sitter.Node, 0, node.NamedChildCount())
	for _, child := range nodeutil.NamedChildrenOf(node) {
		switch child.Type() {
		case "comment", "line_comment", "block_comment":
			continue
		}
		children = append(children, child)
	}
	return children
}

func recognizeArrayGetter(method *Definition, returnStmt *sitter.Node, source []byte) (arrayAccessorCandidate, bool) {
	if returnStmt == nil || returnStmt.NamedChildCount() != 1 {
		return arrayAccessorCandidate{}, false
	}
	arrayField, index, ok := matchThisArrayAccess(returnStmt.NamedChild(0), source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}
	row, column, sizeField, ok := matchAffineIndex(index, method, source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}
	return arrayAccessorCandidate{
		kind:            TrivialArrayAccessorGet,
		arrayFieldName:  arrayField,
		sizeFieldName:   sizeField,
		rowParameter:    row,
		columnParameter: column,
		valueParameter:  -1,
	}, true
}

func recognizeDirectArrayMutation(method *Definition, expressionStmt *sitter.Node, source []byte) (arrayAccessorCandidate, bool) {
	assignment := singleAssignmentExpression(expressionStmt)
	if assignment == nil || assignment.ChildCount() < 3 || assignment.Child(1).Content(source) != "=" {
		return arrayAccessorCandidate{}, false
	}
	arrayField, index, ok := matchThisArrayAccess(assignment.Child(0), source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}
	row, column, sizeField, ok := matchAffineIndex(index, method, source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}

	rhs := unwrapJavaParens(assignment.Child(2))
	if valueParameter, ok := matchParameterIdentifier(rhs, method, source); ok {
		return arrayAccessorCandidate{
			kind:            TrivialArrayAccessorSet,
			arrayFieldName:  arrayField,
			sizeFieldName:   sizeField,
			rowParameter:    row,
			columnParameter: column,
			valueParameter:  valueParameter,
		}, true
	}

	left, right, ok := matchJavaBinary(rhs, "+", source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}
	for operandIndex, operands := range [][2]*sitter.Node{{left, right}, {right, left}} {
		valueParameter, valueOK := matchParameterIdentifier(operands[1], method, source)
		otherArray, otherIndex, arrayOK := matchThisArrayAccess(operands[0], source)
		if !valueOK || !arrayOK || otherArray != arrayField || !sameAffineIndex(otherIndex, method, row, column, sizeField, source) {
			continue
		}
		return arrayAccessorCandidate{
			kind:            TrivialArrayAccessorAdd,
			arrayFieldName:  arrayField,
			sizeFieldName:   sizeField,
			rowParameter:    row,
			columnParameter: column,
			valueParameter:  valueParameter,
			valueFirst:      operandIndex == 1,
		}, true
	}
	return arrayAccessorCandidate{}, false
}

func recognizeIndexedArrayAdd(method *Definition, declaration, expressionStmt *sitter.Node, source []byte) (arrayAccessorCandidate, bool) {
	typeNode := declaration.ChildByFieldName("type")
	declarator := declaration.ChildByFieldName("declarator")
	if typeNode == nil || strings.TrimSpace(typeNode.Content(source)) != "int" || declarator == nil {
		return arrayAccessorCandidate{}, false
	}
	nameNode := declarator.ChildByFieldName("name")
	valueNode := declarator.ChildByFieldName("value")
	if nameNode == nil || valueNode == nil {
		return arrayAccessorCandidate{}, false
	}
	row, column, sizeField, ok := matchAffineIndex(valueNode, method, source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}
	localIndexName := nameNode.Content(source)

	assignment := singleAssignmentExpression(expressionStmt)
	if assignment == nil || assignment.ChildCount() < 3 || assignment.Child(1).Content(source) != "=" {
		return arrayAccessorCandidate{}, false
	}
	arrayField, lhsIndex, ok := matchThisArrayAccess(assignment.Child(0), source)
	if !ok || !matchesIdentifier(lhsIndex, localIndexName, source) {
		return arrayAccessorCandidate{}, false
	}
	left, right, ok := matchJavaBinary(assignment.Child(2), "+", source)
	if !ok {
		return arrayAccessorCandidate{}, false
	}
	for operandIndex, operands := range [][2]*sitter.Node{{left, right}, {right, left}} {
		valueParameter, valueOK := matchParameterIdentifier(operands[1], method, source)
		otherArray, otherIndex, arrayOK := matchThisArrayAccess(operands[0], source)
		if !valueOK || !arrayOK || otherArray != arrayField || !matchesIdentifier(otherIndex, localIndexName, source) {
			continue
		}
		return arrayAccessorCandidate{
			kind:            TrivialArrayAccessorAdd,
			arrayFieldName:  arrayField,
			sizeFieldName:   sizeField,
			rowParameter:    row,
			columnParameter: column,
			valueParameter:  valueParameter,
			valueFirst:      operandIndex == 1,
		}, true
	}
	return arrayAccessorCandidate{}, false
}

func singleAssignmentExpression(expressionStmt *sitter.Node) *sitter.Node {
	if expressionStmt == nil || expressionStmt.Type() != "expression_statement" || expressionStmt.NamedChildCount() != 1 {
		return nil
	}
	expression := expressionStmt.NamedChild(0)
	if expression.Type() != "assignment_expression" {
		return nil
	}
	return expression
}

func matchThisArrayAccess(node *sitter.Node, source []byte) (string, *sitter.Node, bool) {
	node = unwrapJavaParens(node)
	if node == nil || node.Type() != "array_access" {
		return "", nil, false
	}
	arrayNode := node.ChildByFieldName("array")
	indexNode := node.ChildByFieldName("index")
	if arrayNode == nil && node.NamedChildCount() > 0 {
		arrayNode = node.NamedChild(0)
	}
	if indexNode == nil && node.NamedChildCount() > 1 {
		indexNode = node.NamedChild(1)
	}
	fieldName, ok := matchThisField(arrayNode, source)
	return fieldName, indexNode, ok && indexNode != nil
}

func matchThisField(node *sitter.Node, source []byte) (string, bool) {
	node = unwrapJavaParens(node)
	if node == nil || node.Type() != "field_access" {
		return "", false
	}
	object := node.ChildByFieldName("object")
	field := node.ChildByFieldName("field")
	if object == nil && node.NamedChildCount() > 0 {
		object = node.NamedChild(0)
	}
	if field == nil && node.NamedChildCount() > 1 {
		field = node.NamedChild(int(node.NamedChildCount()) - 1)
	}
	if object == nil || object.Type() != "this" || field == nil || field.Type() != "identifier" {
		return "", false
	}
	return field.Content(source), true
}

func matchAffineIndex(node *sitter.Node, method *Definition, source []byte) (int, int, string, bool) {
	left, right, ok := matchJavaBinary(node, "+", source)
	if !ok {
		return -1, -1, "", false
	}
	for _, operands := range [][2]*sitter.Node{{left, right}, {right, left}} {
		column, columnOK := matchParameterIdentifier(operands[1], method, source)
		productLeft, productRight, productOK := matchJavaBinary(operands[0], "*", source)
		if !columnOK || !productOK {
			continue
		}
		for _, productOperands := range [][2]*sitter.Node{{productLeft, productRight}, {productRight, productLeft}} {
			row, rowOK := matchParameterIdentifier(productOperands[0], method, source)
			sizeField, sizeOK := matchThisField(productOperands[1], source)
			if rowOK && sizeOK && row != column {
				return row, column, sizeField, true
			}
		}
	}
	return -1, -1, "", false
}

func sameAffineIndex(node *sitter.Node, method *Definition, row, column int, sizeField string, source []byte) bool {
	gotRow, gotColumn, gotSize, ok := matchAffineIndex(node, method, source)
	return ok && gotRow == row && gotColumn == column && gotSize == sizeField
}

func matchParameterIdentifier(node *sitter.Node, method *Definition, source []byte) (int, bool) {
	node = unwrapJavaParens(node)
	if node == nil || node.Type() != "identifier" || method == nil {
		return -1, false
	}
	name := node.Content(source)
	for index, parameter := range method.Parameters {
		if parameter != nil && parameter.OriginalName == name {
			return index, true
		}
	}
	return -1, false
}

func matchJavaBinary(node *sitter.Node, operator string, source []byte) (*sitter.Node, *sitter.Node, bool) {
	node = unwrapJavaParens(node)
	if node == nil || node.Type() != "binary_expression" || node.ChildCount() < 3 || node.Child(1).Content(source) != operator {
		return nil, nil, false
	}
	return node.Child(0), node.Child(2), true
}

func matchesIdentifier(node *sitter.Node, name string, source []byte) bool {
	node = unwrapJavaParens(node)
	return node != nil && node.Type() == "identifier" && node.Content(source) == name
}

func unwrapJavaParens(node *sitter.Node) *sitter.Node {
	for node != nil && node.Type() == "parenthesized_expression" && node.NamedChildCount() == 1 {
		node = node.NamedChild(0)
	}
	return node
}

func hasJavaModifier(declaration *sitter.Node, modifier string) bool {
	if declaration == nil || declaration.NamedChildCount() == 0 {
		return false
	}
	modifiers := declaration.NamedChild(0)
	if modifiers == nil || modifiers.Type() != "modifiers" {
		return false
	}
	for index := 0; index < int(modifiers.ChildCount()); index++ {
		if child := modifiers.Child(index); child != nil && child.Type() == modifier {
			return true
		}
	}
	return false
}

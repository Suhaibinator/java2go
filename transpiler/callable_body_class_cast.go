package transpiler

import (
	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// A canonical Class cast has one physical reference layout regardless of its
// represented argument. Its body audit can ignore that owned TYPE subtree;
// the operand and every other body node still require their ordinary proofs.
func callableBodyCanonicalClassCast(owner *symbol.ClassScope, method *symbol.Definition, cast *sitter.Node, declaration *symbol.TypeParamDeclaration, ctx Ctx) bool {
	if owner == nil || owner.Class == nil || method == nil || declaration == nil || cast == nil || cast.Type() != "cast_expression" {
		return false
	}
	registered, found := genericMethodOwnerContext(method)
	if !found || registered.currentClass != owner || registered.currentFile == nil {
		return false
	}
	file := registered.currentFile
	classNode := owner.Class.DeclarationNode
	methodNode := method.DeclarationNode
	if classNode == nil || methodNode == nil || file.ParsedRoot() == nil || !threadSourceNodeWithin(classNode, file.ParsedRoot()) || !threadSourceNodeWithin(methodNode, file.ParsedRoot()) || methodNode.HasError() || cast.HasError() {
		return false
	}
	if (method.Constructor && methodNode.Type() != "constructor_declaration") || (!method.Constructor && methodNode.Type() != "method_declaration") {
		return false
	}
	if !callableClassCastNearestNode(methodNode, classNode) || !callableClassCastNearestNode(cast, methodNode) {
		return false
	}
	body := methodNode.ChildByFieldName("body")
	typ := cast.ChildByFieldName("type")
	value := cast.ChildByFieldName("value")
	if body == nil || typ == nil || value == nil || !threadSourceNodeWithin(cast, body) || typ.Parent() == nil || !typ.Parent().Equal(cast) || value.Parent() == nil || !value.Parent().Equal(cast) || cast.EndByte() > uint32(len(file.Source)) {
		return false
	}
	name := methodNode.ChildByFieldName("name")
	if name == nil || name.Content(file.Source) != method.OriginalName {
		return false
	}
	// Validate declaration roles before using any binding map. A same-spelled
	// method parameter must not stand in for a class parameter, or vice versa.
	seen := map[*symbol.TypeParamDeclaration]bool{}
	visitedOwners := map[*symbol.ClassScope]bool{}
	owned := false
	for scope := owner; scope != nil; scope = scope.Enclosing {
		if scope.Class == nil || visitedOwners[scope] {
			return false
		}
		visitedOwners[scope] = true
		node := scope.Class.DeclarationNode
		if node == nil || !threadSourceNodeWithin(node, file.ParsedRoot()) {
			return false
		}
		switch node.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
		default:
			return false
		}
		className := node.ChildByFieldName("name")
		if className == nil || className.Content(file.Source) != scope.Class.OriginalName {
			return false
		}
		actualEnclosing, coherent := callableClassCastEnclosingNode(node, file.ParsedRoot())
		if !coherent {
			return false
		}
		if scope.Enclosing == nil {
			if actualEnclosing != nil {
				return false
			}
		} else if scope.Enclosing.Class == nil || scope.Enclosing.Class.DeclarationNode == nil || actualEnclosing == nil || !actualEnclosing.Equal(scope.Enclosing.Class.DeclarationNode) {
			return false
		}
		header := &symbol.Definition{DeclarationNode: node, TypeParameters: scope.OwnTypeParameters()}
		if !genericMethodClassHeaderOwned(header, file.Source) {
			return false
		}
		for _, p := range scope.OwnTypeParameters() {
			if p.Declaration == nil || seen[p.Declaration] || p.Declaration.SourceName != p.Name {
				return false
			}
			seen[p.Declaration] = true
			owned = owned || p.Declaration == declaration
		}
	}
	if !owned || !genericMethodClassHeaderOwned(method, file.Source) {
		return false
	}
	for _, p := range method.TypeParameters {
		if p.Declaration == nil || seen[p.Declaration] || p.Declaration.SourceName != p.Name {
			return false
		}
		seen[p.Declaration] = true
	}
	declaring := classScopeCtx(owner, ctx).Clone()
	declaring.localScope = method
	declaring.expectedType = ""
	declaring.expectedTypeRoot = nil
	if declaring.currentFile != file || visibleTypeParameterDeclarationForJavaType(declaration.SourceName, declaring) != declaration {
		return false
	}
	expected := symbol.VisibleTypeParamBindings(symbol.MergeTypeParams(owner.TypeParameters, method.TypeParameters))
	formals := nodeutil.SemanticNamedChildrenOf(methodNode.ChildByFieldName("parameters"))
	if len(formals) != len(method.Parameters) {
		return false
	}
	for i, formal := range formals {
		if formal.Type() != "formal_parameter" {
			return false
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(formal) {
			if child.Type() == "dimensions" {
				return false
			}
		}
		ft, fn := nodeutil.JavaParameterNodes(formal)
		p := method.Parameters[i]
		if p == nil || ft == nil || fn == nil || ft.Content(file.Source) != p.OriginalType || fn.Content(file.Source) != p.OriginalName || len(p.TypeParameterBindings) != len(expected) {
			return false
		}
		for name, binding := range expected {
			if binding == nil || p.TypeParameterBindings[name] != binding || visibleTypeParameterDeclarationForJavaType(name, declaring) != binding {
				return false
			}
		}
	}
	return canonicalClassArgumentIndependent(typ.Content(file.Source), declaring, typ)
}

// Node.Equal ancestry is the authority. Crossing another executable or type
// declaration means the target belongs to a different lexical body/owner.
func callableClassCastNearestNode(node, target *sitter.Node) bool {
	if node == nil || target == nil {
		return false
	}
	for p := node.Parent(); p != nil; p = p.Parent() {
		if p.Equal(target) {
			return true
		}
		switch p.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration", "object_creation_expression", "method_declaration", "constructor_declaration", "lambda_expression":
			return false
		}
	}
	return false
}

// Each metadata enclosing edge must follow the actual parsed type ancestry.
// A local or anonymous executable cannot supply another class's certificate.
func callableClassCastEnclosingNode(node, root *sitter.Node) (*sitter.Node, bool) {
	if node == nil || root == nil {
		return nil, false
	}
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Equal(root) {
			return nil, true
		}
		switch parent.Type() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			return parent, true
		case "class_body", "interface_body", "enum_body", "enum_body_declarations", "annotation_type_body":
			// Only direct named type-body containers preserve enclosing ownership.
		default:
			return nil, false
		}
	}
	return nil, false
}

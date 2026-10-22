package transpiler

import (
	"go/ast"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// canonicalArrayNullOperand proves declaration storage, not an inferred Java
// use-site view. An erased field or call can look like an array while remaining
// an interface whose typed nil still needs JavaReferenceEqual.
func canonicalArrayNullOperand(node *sitter.Node, source []byte, ctx Ctx) bool {
	if ctx.localScope == nil || ctx.localScope.DeclarationNode == nil || ctx.currentClass == nil ||
		ctx.currentClass.IsInterface || ctx.erasedGenericMethodBody != nil || ctx.genericInterfaceMethodBody != nil ||
		ctx.rawFunctionTarget != nil || ctx.dependentTypeWitnesses != nil ||
		genericMethodHasErasedEntry(ctx.localScope) {
		return false
	}
	// Proof-only type rendering must not publish speculative imports.
	proof := ctx.Clone()
	proof.importAliases = make(map[string]string, len(ctx.importAliases))
	for path, alias := range ctx.importAliases {
		proof.importAliases[path] = alias
	}
	proof.usedImports = make(map[string]bool, len(ctx.usedImports))
	for path, used := range ctx.usedImports {
		proof.usedImports[path] = used
	}
	return canonicalArrayNullStorage(node, source, proof, map[uint32]bool{}) != nil
}

func canonicalArrayNullPointer(javaType string, ctx Ctx) ast.Expr {
	_, rank := javaArrayTypeParts(javaType)
	if rank == 0 {
		return nil
	}
	physical := javaTypeStringToGoTypeExpr(javaType, inScopeTypeParameters(ctx), ctx)
	if primitive, ok := primitiveArrayTypeExpr(javaType, ctx); ok {
		if symbol.NodeToStr(physical) == symbol.NodeToStr(primitive) {
			return physical
		}
		return nil
	}
	if _, _, reified := reifiedSourceReferenceArrayComponent(javaType, ctx); reified &&
		symbol.NodeToStr(physical) == symbol.NodeToStr(reifiedReferenceArrayTypeExpr(ctx)) {
		return physical
	}
	return nil
}

// Resolve only preceding lexical declarations. The flattened symbol children
// also contain later/sibling locals and cannot prove which storage is read.
func canonicalArrayNullStorage(node *sitter.Node, source []byte, ctx Ctx, visiting map[uint32]bool) ast.Expr {
	node = unwrapParenthesizedExpressionNode(node)
	if node == nil || node.Type() != "identifier" {
		return nil
	}
	name := node.Content(source)
	inspect := func(declaration *sitter.Node) (ast.Expr, bool) {
		if declaration == nil || declaration.Type() != "local_variable_declaration" {
			return nil, false
		}
		for _, declarator := range nodeutil.VariableDeclarators(declaration) {
			ident := declarator.ChildByFieldName("name")
			if ident == nil || ident.Content(source) != name || ident.StartByte() >= node.StartByte() {
				continue
			}
			if visiting[ident.StartByte()] {
				return nil, true
			}
			typ := declaration.ChildByFieldName("type")
			if typ == nil {
				return nil, true
			}
			physical := canonicalArrayNullPointer(typ.Content(source), ctx)
			if physical == nil {
				return nil, true
			}
			initializer := unwrapParenthesizedExpressionNode(declarator.ChildByFieldName("value"))
			if initializer == nil || initializer.Type() == "null_literal" {
				if symbol.NodeToStr(explicitLocalVariableType(typ.Content(source), ctx)) == symbol.NodeToStr(physical) {
					return physical, true
				}
				return nil, true
			}
			// A short declaration inherits the initializer's physical type. Admit only
			// an already proved array binding; fields/calls/casts remain conservative.
			visiting[ident.StartByte()] = true
			input := canonicalArrayNullStorage(initializer, source, ctx, visiting)
			delete(visiting, ident.StartByte())
			if input != nil && symbol.NodeToStr(input) == symbol.NodeToStr(physical) {
				return physical, true
			}
			return nil, true
		}
		return nil, false
	}
	for child, parent := node, node.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		switch parent.Type() {
		case "local_variable_declaration":
			if typ, found := inspect(parent); found {
				return typ
			}
		case "block", "constructor_body":
			for index := int(parent.NamedChildCount()) - 1; index >= 0; index-- {
				sibling := parent.NamedChild(index)
				if sibling.StartByte() >= child.StartByte() {
					continue
				}
				if typ, found := inspect(sibling); found {
					return typ
				}
			}
		case "for_statement":
			if typ, found := inspect(parent.ChildByFieldName("init")); found {
				return typ
			}
		case "lambda_expression", "enhanced_for_statement", "catch_clause", "class_declaration", "interface_declaration":
			return nil
		case "method_declaration", "constructor_declaration":
			declaration := ctx.localScope.DeclarationNode
			if parent.StartByte() != declaration.StartByte() || parent.EndByte() != declaration.EndByte() || parent.Type() != declaration.Type() {
				return nil
			}
			parameter := ctx.localScope.ParameterByName(name)
			if parameter == nil {
				return nil
			}
			for index, candidate := range ctx.localScope.Parameters {
				if candidate != parameter || executionParameterIsVariadic(ctx.localScope, index) {
					continue
				}
				physical := canonicalArrayNullPointer(parameter.OriginalType, ctx)
				if physical == nil {
					return nil
				}
				// Formal emission applies this declaration's physical rewrite before
				// choosing its slot. A wildcard component does not erase the outer
				// array pointer; a scalar/other carrier still fails the exact proof.
				emittedJavaType := parameter.OriginalType
				if rewritten, ok := ctx.rawGenericParameterTypes[parameter.OriginalName]; ok {
					emittedJavaType = rewritten
				} else if rewritten, ok := ctx.rawGenericParameterTypes[parameter.Name]; ok {
					emittedJavaType = rewritten
				}
				emitted := executionParameterTypeExpr(ctx.localScope, index, emittedJavaType, inScopeTypeParameters(ctx), ctx)
				emitted = directOwnerTypeParameterMethodParameterType(ctx.currentClass, ctx.localScope, index, emitted, ctx)
				if symbol.NodeToStr(emitted) == symbol.NodeToStr(physical) {
					return physical
				}
				return nil
			}
			return nil
		}
	}
	return nil
}

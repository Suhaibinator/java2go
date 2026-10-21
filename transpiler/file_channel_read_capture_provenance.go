package transpiler

import (
	"strconv"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Binding identity is the source declaration, independent of a closure's Go
// storage and of the flattened parameter list used for ordinary Java lookup.
type fileChannelReadCaptureBinding struct {
	declaration *sitter.Node
	name        string
}

func fileChannelReadCaptureParameters(parameters *sitter.Node, source []byte) []fileChannelReadCaptureBinding {
	if parameters == nil {
		return nil
	}
	nodes := nodeutil.SemanticNamedChildrenOf(parameters)
	if parameters.Type() != "formal_parameters" && parameters.Type() != "inferred_parameters" {
		nodes = []*sitter.Node{parameters}
	}
	var result []fileChannelReadCaptureBinding
	for _, parameter := range nodes {
		name := parameter.ChildByFieldName("name")
		if parameter.Type() == "identifier" {
			name = parameter
		}
		if name != nil {
			result = append(result, fileChannelReadCaptureBinding{parameter, name.Content(source)})
		}
	}
	return result
}

// Resolve only declarations actually visible at this occurrence. In particular,
// a later declaration or a local in a sibling block cannot become a capture.
func fileChannelReadCaptureDeclaration(node *sitter.Node, source []byte) (fileChannelReadCaptureBinding, bool) {
	name := node.Content(source)
	inspect := func(declaration *sitter.Node) (fileChannelReadCaptureBinding, bool) {
		if declaration == nil || declaration.Type() != "local_variable_declaration" {
			return fileChannelReadCaptureBinding{}, false
		}
		children := nodeutil.SemanticNamedChildrenOf(declaration)
		for i := len(children) - 1; i >= 0; i-- {
			declarator := children[i]
			ident := declarator.ChildByFieldName("name")
			if declarator.Type() == "variable_declarator" && ident != nil && ident.Content(source) == name && ident.EndByte() <= node.StartByte() {
				return fileChannelReadCaptureBinding{declarator, name}, true
			}
		}
		return fileChannelReadCaptureBinding{}, false
	}
	for child, parent := node, node.Parent(); parent != nil; child, parent = parent, parent.Parent() {
		switch parent.Type() {
		case "local_variable_declaration":
			if binding, ok := inspect(parent); ok {
				return binding, true
			}
		case "block", "constructor_body":
			siblings := nodeutil.SemanticNamedChildrenOf(parent)
			for i := len(siblings) - 1; i >= 0; i-- {
				if siblings[i].StartByte() >= child.StartByte() {
					continue
				}
				if binding, ok := inspect(siblings[i]); ok {
					return binding, true
				}
			}
		case "for_statement":
			if binding, ok := inspect(parent.ChildByFieldName("init")); ok {
				return binding, true
			}
		case "enhanced_for_statement":
			ident := parent.ChildByFieldName("name")
			if ident != nil && ident.Content(source) == name && receiverNodeWithin(node, parent.ChildByFieldName("body")) {
				return fileChannelReadCaptureBinding{parent, name}, true
			}
		case "catch_clause":
			if receiverNodeWithin(node, parent.ChildByFieldName("body")) {
				for _, parameter := range nodeutil.SemanticNamedChildrenOf(parent) {
					ident := parameter.ChildByFieldName("name")
					if parameter.Type() == "catch_formal_parameter" && ident != nil && ident.Content(source) == name {
						return fileChannelReadCaptureBinding{parameter, name}, true
					}
				}
			}
		case "try_with_resources_statement":
			resources := parent.ChildByFieldName("resources")
			if receiverNodeWithin(node, resources) || receiverNodeWithin(node, parent.ChildByFieldName("body")) {
				for _, resource := range nodeutil.SemanticNamedChildrenOf(resources) {
					ident := resource.ChildByFieldName("name")
					if ident != nil && ident.Content(source) == name && ident.EndByte() <= node.StartByte() {
						return fileChannelReadCaptureBinding{resource, name}, true
					}
				}
			}
		case "lambda_expression", "method_declaration", "constructor_declaration":
			for _, binding := range fileChannelReadCaptureParameters(parent.ChildByFieldName("parameters"), source) {
				if binding.name == name {
					return binding, true
				}
			}
			if parent.Type() != "lambda_expression" {
				return fileChannelReadCaptureBinding{}, false
			}
		case "class_body", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			return fileChannelReadCaptureBinding{}, false
		}
	}
	return fileChannelReadCaptureBinding{}, false
}

func fileChannelReadCaptureIdentifierUse(node *sitter.Node) bool {
	parent := node.Parent()
	if !identifierIsSyntheticCaptureRead(node, parent) {
		return false
	}
	if parent == nil {
		return false
	}
	switch parent.Type() {
	case "variable_declarator", "formal_parameter", "spread_parameter", "catch_formal_parameter", "enhanced_for_statement", "resource":
		return !sameSourceNode(node, parent.ChildByFieldName("name"))
	case "inferred_parameters", "break_statement", "continue_statement":
		return false
	case "lambda_expression":
		return !sameSourceNode(node, parent.ChildByFieldName("parameters"))
	case "labeled_statement":
		return parent.NamedChildCount() == 0 || !sameSourceNode(node, parent.NamedChild(0))
	case "method_reference":
		children := nodeutil.SemanticNamedChildrenOf(parent)
		return len(children) > 0 && !sameSourceNode(node, children[len(children)-1])
	}
	return true
}

// javac's synthetic capture signature is determined before constant-dead
// statement elimination. Nested lambda free uses also require captures at the
// outer creation boundary, but each inner invocation has its own signature.
func fileChannelReadLambdaCaptures(lambda *sitter.Node, ctx Ctx, source []byte) ([]fileChannelReadCaptureBinding, bool) {
	var result []fileChannelReadCaptureBinding
	seen := map[[2]uint32]bool{}
	proved := true
	var walk func(*sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || !proved {
			return
		}
		switch node.Type() {
		case "class_body", "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
			proved = false
			return
		}
		if node.Type() == "identifier" && fileChannelReadCaptureIdentifierUse(node) {
			if binding, found := fileChannelReadCaptureDeclaration(node, source); found && !receiverNodeWithin(binding.declaration, lambda) {
				if fileChannelReadCaptureFinalInferredType(binding, source) {
					proved = false
					return
				}
				if constant, local := javaLocalConstantExpression(node, source, ctx, map[*symbol.Definition]bool{}); local && constant {
					return
				}
				key := [2]uint32{binding.declaration.StartByte(), binding.declaration.EndByte()}
				if !seen[key] {
					seen[key] = true
					result = append(result, binding)
				}
			}
		}
		for _, child := range nodeutil.SemanticNamedChildrenOf(node) {
			walk(child)
		}
	}
	walk(lambda.ChildByFieldName("body"))
	return result, proved
}

// bound distinguishes a real lexical declaration from the preexisting field
// and selected-return descriptions. A selected but unproved local must not fall
// through to the consumer's flattened enclosing-method parameter indexes.
func fileChannelReadLambdaIdentifierDescription(node *sitter.Node, ctx Ctx, source []byte) (description string, bound, described bool) {
	var lambda *sitter.Node
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Type() == "lambda_expression" {
			lambda = parent
			break
		}
		if parent.Type() == "method_declaration" || parent.Type() == "constructor_declaration" || parent.Type() == "class_body" {
			break
		}
	}
	if lambda == nil {
		return "", false, false
	}
	binding, found := fileChannelReadCaptureDeclaration(node, source)
	if !found {
		return "", false, false
	}
	if ctx.currentClass == nil || ctx.currentClass.Class == nil || ctx.currentFile == nil || findFileScopeForClassScope(ctx.currentClass, ctx) != ctx.currentFile || !receiverNodeWithin(node, ctx.currentClass.Class.DeclarationNode) {
		return "", true, false
	}
	captures, proved := fileChannelReadLambdaCaptures(lambda, ctx, source)
	if !proved {
		return "", true, false
	}
	parameters := append(captures, fileChannelReadCaptureParameters(lambda.ChildByFieldName("parameters"), source)...)
	for index, parameter := range parameters {
		if sameSourceNode(parameter.declaration, binding.declaration) {
			return strconv.Quote("<parameter" + strconv.Itoa(index+1) + ">"), true, true
		}
	}
	// A body-local receiver requires a bytecode local-slot model, which this
	// capture/signature proof intentionally does not pretend to provide.
	return "", true, false
}

// The existing local constant proof consumes a declared primitive/String type.
// An inferred final type needs its own attribution proof before we can decide
// whether javac omits this reference from the synthetic capture signature.
func fileChannelReadCaptureFinalInferredType(binding fileChannelReadCaptureBinding, source []byte) bool {
	if binding.declaration.Type() != "variable_declarator" {
		return false
	}
	declaration := binding.declaration.Parent()
	if declaration == nil || declaration.Type() != "local_variable_declaration" {
		return false
	}
	typ := declaration.ChildByFieldName("type")
	if typ == nil || typ.Content(source) != "var" {
		return false
	}
	for _, part := range nodeutil.SemanticNamedChildrenOf(declaration) {
		if part.Type() == "modifiers" {
			for i := 0; i < int(part.ChildCount()); i++ {
				if part.Child(i).Type() == "final" {
					return true
				}
			}
		}
	}
	return false
}

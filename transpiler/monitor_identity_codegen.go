package transpiler

import (
	"strings"

	"github.com/NickyBoy89/java2go/nodeutil"
	"github.com/NickyBoy89/java2go/symbol"
	sitter "github.com/smacker/go-tree-sitter"
)

// Monitor receivers need the same allocation identity through every superclass
// view. Reuse the reference-identity hierarchy closure so constructor setup and
// casts cannot install separate monitor identities for one Java allocation.
func addMonitorIdentitySeeds(scope *symbol.ClassScope, ctx Ctx, seeds map[*symbol.ClassScope]struct{}, erased *bool) {
	if scope == nil || scope.Class == nil || scope.Class.DeclarationNode == nil {
		return
	}
	file := findFileScopeForClassScope(scope)
	if file == nil {
		return
	}
	ctx = classScopeCtx(scope, ctx)
	root := scope.Class.DeclarationNode
	var visit func(*sitter.Node, Ctx)
	visit = func(node *sitter.Node, current Ctx) {
		if node == nil {
			return
		}
		if node.StartByte() != root.StartByte() {
			switch node.Type() {
			case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration":
				return
			}
		}
		switch node.Type() {
		case "method_declaration", "constructor_declaration":
			for _, method := range scope.Methods {
				if method.DeclarationNode != nil && method.DeclarationNode.StartByte() == node.StartByte() {
					current = current.Clone()
					current.localScope = method
					break
				}
			}
			if declarationHasModifier(node, "synchronized") && !declarationHasModifier(node, "static") {
				seeds[scope] = struct{}{}
			}
		case "synchronized_statement":
			addMonitorReceiverSeed(node.NamedChild(0), scope, current, file.Source, seeds, erased)
		case "method_invocation":
			name := node.ChildByFieldName("name")
			if name != nil {
				switch name.Content(file.Source) {
				case "wait", "notify", "notifyAll":
					addMonitorReceiverSeed(node.ChildByFieldName("object"), scope, current, file.Source, seeds, erased)
				case "holdsLock":
					args := node.ChildByFieldName("arguments")
					if args != nil && args.NamedChildCount() == 1 {
						addMonitorReceiverSeed(args.NamedChild(0), scope, current, file.Source, seeds, erased)
					}
				}
			}
		}
		for _, child := range nodeutil.NamedChildrenOf(node) {
			visit(child, current)
		}
	}
	visit(root, ctx)
}

func addMonitorReceiverSeed(node *sitter.Node, owner *symbol.ClassScope, ctx Ctx, source []byte, seeds map[*symbol.ClassScope]struct{}, erased *bool) {
	if node == nil {
		seeds[owner] = struct{}{}
		return
	}
	for node.Type() == "parenthesized_expression" && node.NamedChildCount() == 1 {
		node = node.NamedChild(0)
	}
	if node.Type() == "this" || node.Type() == "super" {
		seeds[owner] = struct{}{}
		return
	}
	javaType, ok := inferExprJavaType(node, ctx, source)
	if !ok {
		*erased = true
		return
	}
	base, _ := parseJavaTypeString(javaType)
	if target := resolveClassScopeByQualifiedName(ctx, base); target != nil {
		seeds[target] = struct{}{}
		return
	}
	// Array wrappers already have allocation identity. Object or an unresolved
	// type variable can carry any source hierarchy and needs conservative coverage.
	if strings.HasSuffix(base, "[]") {
		return
	}
	_, parameter := javaTypeParameterErasure(base, ctx)
	if stripJavaQualifier(base) == "Object" || parameter {
		*erased = true
	}
}

// These Object methods cannot be overridden by Java source classes. Other
// overloads (including timed wait) retain their existing resolution path.
func isObjectMonitorInvocation(receiver *sitter.Node, method string) bool {
	if receiver == nil {
		return false
	}
	switch method {
	case "wait", "notify", "notifyAll":
	default:
		return false
	}
	invocation := receiver.Parent()
	if invocation == nil || invocation.Type() != "method_invocation" {
		return false
	}
	arguments := invocation.ChildByFieldName("arguments")
	return arguments != nil && arguments.NamedChildCount() == 0
}
